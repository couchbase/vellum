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

import "testing"

// FIXME add tests for MRU

func TestRegistry(t *testing.T) {
	r := newRegistry(10, 1)

	n1 := &builderState{
		transitions: []*transition{
			&transition{
				key:  'a',
				dest: &builderState{id: 1},
			},
			&transition{
				key:  'b',
				dest: &builderState{id: 2},
			},
			&transition{
				key:  'c',
				dest: &builderState{id: 3},
			},
		},
	}

	// first look, doesn't exist
	equiv := r.entry(n1)
	if equiv != nil {
		t.Errorf("expected empty registry to not have equivalent")
	}

	// second look, does
	equiv = r.entry(n1)
	if equiv == nil {
		t.Errorf("expected to find equivalent after registering it")
	}
}
