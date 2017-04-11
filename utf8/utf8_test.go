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

package utf8

import (
	"fmt"
	"reflect"
	"testing"
	"unicode/utf8"
)

func TestUtf8Sequences(t *testing.T) {

	want := Utf8Sequences{
		Utf8Sequence{
			&Utf8Range{0x0, 0x7f},
		},
		Utf8Sequence{
			&Utf8Range{0xc2, 0xdf},
			&Utf8Range{0x80, 0xbf},
		},
		Utf8Sequence{
			&Utf8Range{0xe0, 0xe0},
			&Utf8Range{0xa0, 0xbf},
			&Utf8Range{0x80, 0xbf},
		},
		Utf8Sequence{
			&Utf8Range{0xe1, 0xec},
			&Utf8Range{0x80, 0xbf},
			&Utf8Range{0x80, 0xbf},
		},
		Utf8Sequence{
			&Utf8Range{0xed, 0xed},
			&Utf8Range{0x80, 0x9f},
			&Utf8Range{0x80, 0xbf},
		},
		Utf8Sequence{
			&Utf8Range{0xee, 0xef},
			&Utf8Range{0x80, 0xbf},
			&Utf8Range{0x80, 0xbf},
		},
	}

	got, err := NewUtf8Sequences(0, 0xffff)
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
	sequences, err := NewUtf8Sequences(start, end)
	if err != nil {
		return err
	}
	for i := start; i < end; i++ {
		n := utf8.EncodeRune(buf, i)
		for _, seq := range sequences {
			if seq.Matches(buf[:n]) {
				return fmt.Errorf("utf8 seq: %v matches surrogate %d", seq, i)
			}
		}
	}
	return nil
}
