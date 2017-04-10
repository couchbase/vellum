//  Copyright (c) 2017 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package levenshtein

import (
	"fmt"
	"unicode/utf8"
)

type utf8Sequences []utf8Sequence

func newUtf8Sequences(start, end rune) (utf8Sequences, error) {
	var rv utf8Sequences

	var rangeStack rangeStack
	rangeStack = rangeStack.Push(&scalarRange{start, end})

	rangeStack, r := rangeStack.Pop()
TOP:
	for r != nil {
	INNER:
		for {
			r1, r2 := r.split()
			if r1 != nil {
				rangeStack = rangeStack.Push(&scalarRange{r2.start, r2.end})
				r.start = r1.start
				r.end = r1.end
				continue INNER
			}
			if !r.valid() {
				rangeStack, r = rangeStack.Pop()
				continue TOP
			}
			for i := 1; i < utf8.UTFMax; i++ {
				max := maxScalarValue(i)
				if r.start <= max && max < r.end {
					rangeStack = rangeStack.Push(&scalarRange{max + 1, r.end})
					r.end = max
					continue INNER
				}
			}
			asciiRange := r.ascii()
			if asciiRange != nil {
				rv = append(rv, utf8Sequence{
					asciiRange,
				})
				rangeStack, r = rangeStack.Pop()
				continue TOP
			}
			for i := uint(1); i < utf8.UTFMax; i++ {
				m := rune((1 << (6 * i)) - 1)
				if (r.start & ^m) != (r.end & ^m) {
					if (r.start & m) != 0 {
						rangeStack = rangeStack.Push(&scalarRange{(r.start | m) + 1, r.end})
						r.end = r.start | m
						continue INNER
					}
					if (r.end & m) != m {
						rangeStack = rangeStack.Push(&scalarRange{r.end & ^m, r.end})
						r.end = (r.end & ^m) - 1
						continue INNER
					}
				}
			}
			start := make([]byte, utf8.UTFMax)
			end := make([]byte, utf8.UTFMax)
			n, m := r.encode(start, end)
			seq, err := utf8SequenceFromEncodedRange(start[0:n], end[0:m])
			if err != nil {
				return nil, err
			}
			rv = append(rv, seq)
			rangeStack, r = rangeStack.Pop()
			continue TOP
		}
	}

	return rv, nil
}

type utf8Sequence []*utf8Range

// utf8SequenceFromEncodedRange creates utf-8 sequence from the encoded bytes
func utf8SequenceFromEncodedRange(start, end []byte) (utf8Sequence, error) {
	if len(start) != len(end) {
		return nil, fmt.Errorf("byte slices must be the same length")
	}
	switch len(start) {
	case 2:
		return utf8Sequence{
			&utf8Range{start[0], end[0]},
			&utf8Range{start[1], end[1]},
		}, nil
	case 3:
		return utf8Sequence{
			&utf8Range{start[0], end[0]},
			&utf8Range{start[1], end[1]},
			&utf8Range{start[2], end[2]},
		}, nil
	case 4:
		return utf8Sequence{
			&utf8Range{start[0], end[0]},
			&utf8Range{start[1], end[1]},
			&utf8Range{start[2], end[2]},
			&utf8Range{start[3], end[3]},
		}, nil
	}

	return nil, fmt.Errorf("invalid encoded byte length")
}

func (u utf8Sequence) matches(bytes []byte) bool {
	if len(bytes) < len(u) {
		return false
	}
	for i := 0; i < len(u); i++ {
		if !u[i].matches(bytes[i]) {
			return false
		}
	}
	return true
}

func (u utf8Sequence) String() string {
	switch len(u) {
	case 1:
		return fmt.Sprintf("%v", u[0])
	case 2:
		return fmt.Sprintf("%v%v", u[0], u[1])
	case 3:
		return fmt.Sprintf("%v%v%v", u[0], u[1], u[2])
	case 4:
		return fmt.Sprintf("%v%v%v%v", u[0], u[1], u[2], u[3])
	default:
		return fmt.Sprintf("invalid utf8 sequence")
	}
}

type utf8Range struct {
	start byte
	end   byte
}

func (u utf8Range) matches(b byte) bool {
	if u.start <= b && b <= u.end {
		return true
	}
	return false
}

func (u utf8Range) String() string {
	if u.start == u.end {
		return fmt.Sprintf("[%X]", u.start)
	}
	return fmt.Sprintf("[%X-%X]", u.start, u.end)
}

type scalarRange struct {
	start rune
	end   rune
}

func (s *scalarRange) String() string {
	return fmt.Sprintf("ScalarRange(%d,%d)", s.start, s.end)
}

// split this scalar range if it overlaps with a surrogate codepoint
func (s *scalarRange) split() (*scalarRange, *scalarRange) {
	if s.start < 0xe000 && s.end > 0xd7ff {
		return &scalarRange{
				start: s.start,
				end:   0xd7ff,
			},
			&scalarRange{
				start: 0xe000,
				end:   s.end,
			}
	}
	return nil, nil
}

func (s *scalarRange) valid() bool {
	return s.start <= s.end
}

func (s *scalarRange) ascii() *utf8Range {
	if s.valid() && s.end <= 0x7f {
		return &utf8Range{
			start: byte(s.start),
			end:   byte(s.end),
		}
	}
	return nil
}

// start and end MUST have capacity for utf8.UTFMax bytes
func (s *scalarRange) encode(start, end []byte) (int, int) {
	n := utf8.EncodeRune(start, s.start)
	m := utf8.EncodeRune(end, s.end)
	return n, m
}

type rangeStack []*scalarRange

func (s rangeStack) Push(v *scalarRange) rangeStack {
	return append(s, v)
}

func (s rangeStack) Pop() (rangeStack, *scalarRange) {
	l := len(s)
	if l < 1 {
		return s, nil
	}
	return s[:l-1], s[l-1]
}

func maxScalarValue(nbytes int) rune {
	switch nbytes {
	case 1:
		return 0x007f
	case 2:
		return 0x07FF
	case 3:
		return 0xFFFF
	default:
		return 0x10FFFF
	}
}
