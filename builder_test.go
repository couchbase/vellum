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

package vellum

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestCommonPrefixLen(t *testing.T) {
	tests := []struct {
		desc string
		a    []byte
		b    []byte
		want int
	}{
		{
			"both slices nil",
			nil,
			nil,
			0,
		},
		{
			"slice a nil, slice b not",
			nil,
			[]byte("anything"),
			0,
		},
		{
			"slice b nil, slice a not",
			[]byte("anything"),
			nil,
			0,
		},
		{
			"both slices empty",
			[]byte(""),
			[]byte(""),
			0,
		},
		{
			"slice a empty, slice b not",
			[]byte(""),
			[]byte("anything"),
			0,
		},
		{
			"slice b nil, slice a not",
			[]byte("anything"),
			[]byte(""),
			0,
		},
		{
			"slices a and b the same",
			[]byte("anything"),
			[]byte("anything"),
			8,
		},
		{
			"slice a substring of b",
			[]byte("any"),
			[]byte("anything"),
			3,
		},
		{
			"slice b substring of a",
			[]byte("anything"),
			[]byte("any"),
			3,
		},
		{
			"slice a starts with prefix of b",
			[]byte("anywhere"),
			[]byte("anything"),
			3,
		},
		{
			"slice b starts with prefix of a",
			[]byte("anything"),
			[]byte("anywhere"),
			3,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := commonPrefixLen(test.a, test.b)
			if got != test.want {
				t.Errorf("wanted: %d, got: %d", test.want, got)
			}
		})
	}
}

func TestBuilderStateHash(t *testing.T) {
	tests := []struct {
		desc  string
		state *builderState
		want  string
	}{
		{
			"non-final state with no transitions",
			&builderState{},
			"",
		},
		{
			"final state with no transitions",
			&builderState{
				final: true,
			},
			"f",
		},
		{
			"final state with a transitions",
			&builderState{
				final: true,
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
				},
			},
			"fa1",
		},
		{
			"final state with multiple transitions",
			&builderState{
				final: true,
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 3,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 2,
						},
					},
				},
			},
			"fa1b3c2",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := test.state.hash()
			if got != test.want {
				t.Errorf("wanted: %s, got: %s", test.want, got)
			}
		})
	}
}

func TestBuilderStateHasTransitions(t *testing.T) {
	tests := []struct {
		desc  string
		state *builderState
		want  bool
	}{
		{
			"no transitions",
			&builderState{},
			false,
		},
		{
			"some transitions",
			&builderState{
				transitions: []*transition{
					&transition{
						key:  'a',
						dest: &builderState{},
					},
					&transition{
						key:  'b',
						dest: &builderState{},
					},
				},
			},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := test.state.hasTransitions()
			if got != test.want {
				t.Errorf("wanted: %t, got: %t", test.want, got)
			}
		})
	}
}

func TestBuilderStateFindTransition(t *testing.T) {
	tests := []struct {
		desc  string
		state *builderState
		key   byte
		want  int
	}{
		{
			"no transitions",
			&builderState{},
			'x',
			-1,
		},
		{
			"some transitions, exists",
			&builderState{
				transitions: []*transition{
					&transition{
						key:  'a',
						dest: &builderState{},
					},
					&transition{
						key:  'b',
						dest: &builderState{},
					},
					&transition{
						key:  'c',
						dest: &builderState{},
					},
				},
			},
			'b',
			1,
		},
		{
			"some transitions, does not exist",
			&builderState{
				transitions: []*transition{
					&transition{
						key:  'a',
						dest: &builderState{},
					},
					&transition{
						key:  'b',
						dest: &builderState{},
					},
					&transition{
						key:  'c',
						dest: &builderState{},
					},
				},
			},
			'x',
			-1,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := test.state.findTransition(test.key)
			if got != test.want {
				t.Errorf("wanted: %d, got: %d", test.want, got)
			}
		})
	}
}

func TestBuilderStateTransitionFor(t *testing.T) {
	tests := []struct {
		desc  string
		state *builderState
		key   byte
		want  *transition
	}{
		{
			"no transitions",
			&builderState{},
			'x',
			nil,
		},
		{
			"some transitions, exists",
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 2,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
				},
			},
			'b',
			&transition{key: 'b', dest: &builderState{id: 2}},
		},
		{
			"some transitions, does not exist",
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 2,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
				},
			},
			'x',
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := test.state.transitionFor(test.key)
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("wanted: %+v, got: %+v", test.want, got)
			}
		})
	}
}

func TestBuilderStateReplaceTransition(t *testing.T) {
	tests := []struct {
		desc        string
		state       *builderState
		replacement *transition
		want        *builderState
	}{
		{
			"no transitions",
			&builderState{},
			&transition{key: 'x', dest: &builderState{id: 5}},
			&builderState{},
		},
		{
			"some transitions, replacement exists",
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 2,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
				},
			},
			&transition{key: 'b', dest: &builderState{id: 5}},
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 5,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
				},
			},
		},
		{
			"some transitions, does not exist",
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 2,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
				},
			},
			&transition{key: 'x', dest: &builderState{id: 5}},
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 2,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test.state.replaceTransition(test.replacement)
			if !reflect.DeepEqual(test.want, test.state) {
				t.Errorf("wanted: %+v, got: %+v", test.want, test.state)
			}
		})
	}
}

func TestBuilderStateLastTransition(t *testing.T) {
	tests := []struct {
		desc  string
		state *builderState
		want  *transition
	}{
		{
			"no transitions",
			&builderState{},
			nil,
		},
		{
			"some transitions",
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 2,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
				},
			},
			&transition{
				key: 'c',
				dest: &builderState{
					id: 3,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := test.state.lastTransition()
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("wanted transition: %+v, got transition: %+v", test.want, got)
			}
		})
	}
}

func TestBuilderStateAddTransition(t *testing.T) {
	tests := []struct {
		desc  string
		state *builderState
		add   *transition
		want  *builderState
	}{
		{
			"no transitions",
			&builderState{},
			&transition{
				key: 'x',
				dest: &builderState{
					id: 5,
				},
			},
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'x',
						dest: &builderState{
							id: 5,
						},
					},
				},
			},
		},
		{
			"some transitions, replacement exists",
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 2,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
				},
			},
			&transition{
				key: 'x',
				dest: &builderState{
					id: 5,
				},
			},
			&builderState{
				transitions: []*transition{
					&transition{
						key: 'a',
						dest: &builderState{
							id: 1,
						},
					},
					&transition{
						key: 'b',
						dest: &builderState{
							id: 2,
						},
					},
					&transition{
						key: 'c',
						dest: &builderState{
							id: 3,
						},
					},
					&transition{
						key: 'x',
						dest: &builderState{
							id: 5,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test.state.addTransition(test.add)
			if !reflect.DeepEqual(test.want, test.state) {
				t.Errorf("wanted: %+v, got: %+v", test.want, test.state)
			}
		})
	}
}

func TestBuilderAddSuffix(t *testing.T) {
	b, err := New(ioutil.Discard, nil)
	if err != nil {
		t.Fatalf("error creating builder: %v", err)
	}
	b.addSuffix(b.root, []byte{'a'}, 0)
	if b.nodeCount != 2 {
		t.Errorf("expected node count to be 2, got %d", b.nodeCount)
	}
	if !b.root.hasTransitions() {
		t.Fatalf("expected root node to have transitions, got none")
	}
	tstate := b.root.transitionFor('a')
	if tstate == nil {
		t.Fatal("expected root node to have transition for 'a', got nil")
	}
	if tstate.dest.id != 1 {
		t.Errorf("expected new state to have id 1, got %d", tstate.dest.id)
	}
}

// this simple test case only has a shared final state
// it also tests out of order insert
func TestBuilderSimple(t *testing.T) {
	b, err := New(ioutil.Discard, nil)
	if err != nil {
		t.Fatalf("error creating builder: %v", err)
	}

	// add our first string
	err = b.Insert([]byte("jul"), 0)
	if err != nil {
		t.Errorf("got error inserting string: %v", err)
	}
	// expect node count to be one for each letter, plus the root
	if b.nodeCount != 4 {
		t.Errorf("expected node count to be 4, got %v", b.nodeCount)
	}

	// try to add a value out of order (not allowed)
	err = b.Insert([]byte("abc"), 0)
	if err != ErrOutOfOrder {
		t.Errorf("expected %v, got %v", ErrOutOfOrder, err)
	}

	// add a second string
	err = b.Insert([]byte("mar"), 0)
	if err != nil {
		t.Errorf("got error inserting string: %v", err)
	}
	// expect node count to grow by the number of chars (no shared prefix)
	if b.nodeCount != 7 {
		t.Errorf("expected node count to be 7, got %v", b.nodeCount)
	}

	// now close the builder
	err = b.Close()
	if err != nil {
		t.Errorf("got error closing set builder: %v", err)
	}
	// expect the node count to go down by one,
	// accounting for the now shard final state
	if b.nodeCount != 6 {
		t.Errorf("expected node count to be 6, got %d", b.nodeCount)
	}
}

func TestBuilderSharedPrefix(t *testing.T) {
	b, err := New(ioutil.Discard, nil)
	if err != nil {
		t.Fatalf("error creating builder: %v", err)
	}

	// add our first string
	err = b.Insert([]byte("car"), 0)
	if err != nil {
		t.Errorf("got error inserting string: %v", err)
	}
	// expect node count to be one for each letter, plus the root
	if b.nodeCount != 4 {
		t.Errorf("expected node count to be 4, got %v", b.nodeCount)
	}

	// add a second string
	err = b.Insert([]byte("cat"), 0)
	if err != nil {
		t.Errorf("got error inserting string: %v", err)
	}
	// expect node count to grow by one (only one char not shared prefix)
	if b.nodeCount != 5 {
		t.Errorf("expected node count to be 5, got %v", b.nodeCount)
	}

	// now close the builder
	err = b.Close()
	if err != nil {
		t.Errorf("got error closing set builder: %v", err)
	}
	// expect the node count to go down by one,
	// accounting for the now shard final state
	if b.nodeCount != 4 {
		t.Errorf("expected node count to be 4, got %d", b.nodeCount)
	}
}

func TestBuilderTraverseToNonexistant(t *testing.T) {
	b, err := New(ioutil.Discard, nil)
	if err != nil {
		t.Fatalf("error creating builder: %v", err)
	}

	// add our first string
	err = b.Insert([]byte("car"), 0)
	if err != nil {
		t.Errorf("got error inserting string: %v", err)
	}
	// expect node count to be one for each letter, plus the root
	if b.nodeCount != 4 {
		t.Errorf("expected node count to be 4, got %d", b.nodeCount)
	}

	state, _ := b.traverse([]byte("cow"))
	if state != nil {
		t.Errorf("expected nil state, got %+v", state)
	}
}

func randomValues(list []string) []uint64 {
	rv := make([]uint64, len(list))
	for i := range list {
		rv[i] = uint64(rand.Uint64())
	}
	return rv
}

func insertStrings(b *Builder, list []string, vals []uint64) error {
	for i, item := range list {
		err := b.Insert([]byte(item), vals[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func builderContains(b *Builder, key string) (bool, uint64) {
	state, val := b.traverse([]byte(key))
	return (state != nil && state.final), val
}

func checkStrings(b *Builder, list []string, vals []uint64) error {
	for i, item := range list {
		contains, val := builderContains(b, item)
		if !contains {
			return fmt.Errorf("expected set to contain %s, does not", item)
		}
		if val != vals[i] {
			return fmt.Errorf("expected val %d, got %d for word %s", vals[i], val, item)
		}
	}
	return nil
}

func checkStringNotWithSuffix(b *Builder, list []string) error {
	for _, item := range list {
		if contains, _ := builderContains(b, item+"0"); contains {
			return fmt.Errorf("expected set to not contain %s, does", item)
		}
	}
	return nil
}

func TestOneThousandWords(t *testing.T) {
	dataset := thousandTestWords
	randomThousandVals := randomValues(dataset)
	b, err := New(ioutil.Discard, nil)
	if err != nil {
		t.Fatalf("error creating builder: %v", err)
	}
	err = insertStrings(b, dataset, randomThousandVals)
	if err != nil {
		t.Fatalf("error inserting thousand words: %v", err)
	}
	err = checkStrings(b, dataset, randomThousandVals)
	if err != nil {
		t.Fatalf("error checking thousand words: %v", err)
	}
	err = checkStringNotWithSuffix(b, dataset)
	if err != nil {
		t.Fatalf("error checking thousand words with suffix: %v", err)
	}
}

var smallSample = map[string]uint64{
	"mon":   2,
	"tues":  3,
	"thurs": 5,
	"tye":   99,
}

func insertStringMap(b *Builder, m map[string]uint64) error {
	// make list of keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// sort it
	sort.Strings(keys)
	// insert in sorted order
	for _, k := range keys {
		err := b.Insert([]byte(k), m[k])
		if err != nil {
			return err
		}
	}
	return nil
}

func TestBuilderStateEquiv(t *testing.T) {
	tests := []struct {
		desc string
		a    *builderState
		b    *builderState
		want bool
	}{
		{
			"both states final",
			&builderState{
				final: true,
			},
			&builderState{
				final: true,
			},
			true,
		},
		{
			"both states final, different final cal",
			&builderState{
				final:    true,
				finalVal: 7,
			},
			&builderState{
				final:    true,
				finalVal: 9,
			},
			false,
		},
		{
			"both states final, same transitions, but different trans val",
			&builderState{
				final: true,
				transitions: []*transition{
					&transition{key: 'a', val: 7},
				},
			},
			&builderState{
				final: true,
				transitions: []*transition{
					&transition{key: 'a', val: 9},
				},
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := test.a.equiv(test.b)
			if got != test.want {
				t.Errorf("wanted: %t, got: %t", test.want, got)
			}
		})
	}
}

func BenchmarkBuilder(b *testing.B) {
	dataset := thousandTestWords
	randomThousandVals := randomValues(dataset)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		builder, err := New(ioutil.Discard, nil)
		if err != nil {
			b.Fatalf("error creating builder: %v", err)
		}
		err = insertStrings(builder, dataset, randomThousandVals)
		if err != nil {
			b.Fatalf("error inserting thousand words: %v", err)
		}
		err = builder.Close()
		if err != nil {
			b.Fatalf("error closing builder: %v", err)
		}
	}
}
