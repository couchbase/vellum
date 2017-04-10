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
	"reflect"
	"testing"
	"unicode/utf8"
)

func TestUtf8Sequences(t *testing.T) {

	want := utf8Sequences{
		utf8Sequence{
			&utf8Range{0x0, 0x7f},
		},
		utf8Sequence{
			&utf8Range{0xc2, 0xdf},
			&utf8Range{0x80, 0xbf},
		},
		utf8Sequence{
			&utf8Range{0xe0, 0xe0},
			&utf8Range{0xa0, 0xbf},
			&utf8Range{0x80, 0xbf},
		},
		utf8Sequence{
			&utf8Range{0xe1, 0xec},
			&utf8Range{0x80, 0xbf},
			&utf8Range{0x80, 0xbf},
		},
		utf8Sequence{
			&utf8Range{0xed, 0xed},
			&utf8Range{0x80, 0x9f},
			&utf8Range{0x80, 0xbf},
		},
		utf8Sequence{
			&utf8Range{0xee, 0xef},
			&utf8Range{0x80, 0xbf},
			&utf8Range{0x80, 0xbf},
		},
	}

	got, err := newUtf8Sequences(0, 0xffff)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted: %v, got %v", want, got)
	}
}

func TestCodepointsNoSurrogates(t *testing.T) {
	neverAcceptsSurrogateCodepoints(0x0, 0xFFFF)
	neverAcceptsSurrogateCodepoints(0x0, 0x10FFFF)
	neverAcceptsSurrogateCodepoints(0x0, 0x10FFFE)
	neverAcceptsSurrogateCodepoints(0x80, 0x10FFFF)
	neverAcceptsSurrogateCodepoints(0xD7FF, 0xE000)
}

func neverAcceptsSurrogateCodepoints(start, end rune) error {
	var buf = make([]byte, utf8.UTFMax)
	sequences, err := newUtf8Sequences(start, end)
	if err != nil {
		return err
	}
	for i := start; i < end; i++ {
		n := utf8.EncodeRune(buf, i)
		for _, seq := range sequences {
			if seq.matches(buf[:n]) {
				return fmt.Errorf("utf8 seq: %v matches surrogate %d", seq, i)
			}
		}
	}
	return nil
}
