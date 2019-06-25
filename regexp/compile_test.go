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

package regexp

import (
	"reflect"
	"regexp/syntax"
	"testing"
)

func TestCompiler(t *testing.T) {

	tests := []struct {
		query     string
		wantInsts prog
		wantErr   error
	}{
		{
			query: "",
			wantInsts: []*inst{
				{op: OpMatch},
			},
			wantErr: nil,
		},
		{
			query:   "^",
			wantErr: ErrNoEmpty,
		},
		{
			query:   `\b`,
			wantErr: ErrNoWordBoundary,
		},
		{
			query:   `.*?`,
			wantErr: ErrNoLazy,
		},
		{
			query: `a`,
			wantInsts: []*inst{
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpMatch},
			},
		},
		{
			query: `[a-c]`,
			wantInsts: []*inst{
				{op: OpRange, rangeStart: 'a', rangeEnd: 'c'},
				{op: OpMatch},
			},
		},
		{
			query: `(a)`,
			wantInsts: []*inst{
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpMatch},
			},
		},
		{
			query: `a?`,
			wantInsts: []*inst{
				{op: OpSplit, splitA: 1, splitB: 2},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpMatch},
			},
		},
		{
			query: `a*`,
			wantInsts: []*inst{
				{op: OpSplit, splitA: 1, splitB: 3},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpJmp, to: 0},
				{op: OpMatch},
			},
		},
		{
			query: `a+`,
			wantInsts: []*inst{
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpSplit, splitA: 0, splitB: 2},
				{op: OpMatch},
			},
		},
		{
			query: `a{2,4}`,
			wantInsts: []*inst{
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpSplit, splitA: 3, splitB: 6},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpSplit, splitA: 5, splitB: 6},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpMatch},
			},
		},
		{
			query: `a{3,}`,
			wantInsts: []*inst{
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpSplit, splitA: 4, splitB: 6},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpJmp, to: 3},
				{op: OpMatch},
			},
		},
		{
			query: `a+|b+`,
			wantInsts: []*inst{
				{op: OpSplit, splitA: 1, splitB: 4},
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpSplit, splitA: 1, splitB: 3},
				{op: OpJmp, to: 6},
				{op: OpRange, rangeStart: 'b', rangeEnd: 'b'},
				{op: OpSplit, splitA: 4, splitB: 6},
				{op: OpMatch},
			},
		},
		{
			query: `a+b+`,
			wantInsts: []*inst{
				{op: OpRange, rangeStart: 'a', rangeEnd: 'a'},
				{op: OpSplit, splitA: 0, splitB: 2},
				{op: OpRange, rangeStart: 'b', rangeEnd: 'b'},
				{op: OpSplit, splitA: 2, splitB: 4},
				{op: OpMatch},
			},
		},
		{
			query: `.`,
			wantInsts: []*inst{
				{op: OpSplit, splitA: 1, splitB: 3},
				{op: OpRange, rangeStart: 0, rangeEnd: 0x09},
				{op: OpJmp, to: 46}, // match ascii, less than 0x0a
				{op: OpSplit, splitA: 4, splitB: 6},
				{op: OpRange, rangeStart: 0x0b, rangeEnd: 0x7f},
				{op: OpJmp, to: 46}, // match rest ascii
				{op: OpSplit, splitA: 7, splitB: 10},
				{op: OpRange, rangeStart: 0xc2, rangeEnd: 0xdf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpJmp, to: 46}, // match
				{op: OpSplit, splitA: 11, splitB: 15},
				{op: OpRange, rangeStart: 0xe0, rangeEnd: 0xe0},
				{op: OpRange, rangeStart: 0xa0, rangeEnd: 0xbf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpJmp, to: 46}, // match
				{op: OpSplit, splitA: 16, splitB: 20},
				{op: OpRange, rangeStart: 0xe1, rangeEnd: 0xec},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpJmp, to: 46}, // match
				{op: OpSplit, splitA: 21, splitB: 25},
				{op: OpRange, rangeStart: 0xed, rangeEnd: 0xed},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0x9f},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpJmp, to: 46}, // match
				{op: OpSplit, splitA: 26, splitB: 30},
				{op: OpRange, rangeStart: 0xee, rangeEnd: 0xef},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpJmp, to: 46}, // match
				{op: OpSplit, splitA: 31, splitB: 36},
				{op: OpRange, rangeStart: 0xf0, rangeEnd: 0xf0},
				{op: OpRange, rangeStart: 0x90, rangeEnd: 0xbf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpJmp, to: 46}, // match
				{op: OpSplit, splitA: 37, splitB: 42},
				{op: OpRange, rangeStart: 0xf1, rangeEnd: 0xf3},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpJmp, to: 46}, // match
				{op: OpRange, rangeStart: 0xf4, rangeEnd: 0xf4},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0x8f},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpRange, rangeStart: 0x80, rangeEnd: 0xbf},
				{op: OpMatch},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			p, err := syntax.Parse(test.query, syntax.Perl)
			if err != nil {
				t.Fatalf("error parsing regexp: %v", err)
			}
			c := newCompiler(10000)
			gotInsts, gotErr := c.compile(p)
			if !reflect.DeepEqual(test.wantErr, gotErr) {
				t.Errorf("expected error: %v, got error: %v", test.wantErr, gotErr)
			}
			if !reflect.DeepEqual(test.wantInsts, gotInsts) {
				t.Errorf("expected insts: %v, got insts:%v", test.wantInsts, gotInsts)
			}
		})
	}
}
