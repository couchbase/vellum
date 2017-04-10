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

import "unicode/utf8"

// dynamicLevenshtein is the rune-based automaton, which is used
// during the building of the ut8-aware byte-based automaton
type dynamicLevenshtein struct {
	query    string
	distance uint
}

func (d *dynamicLevenshtein) start() []uint {
	runeCount := utf8.RuneCountInString(d.query)
	rv := make([]uint, runeCount+1)
	for i := 0; i < runeCount+1; i++ {
		rv[i] = uint(i)
	}
	return rv
}

func (d *dynamicLevenshtein) isMatch(state []uint) bool {
	last := state[len(state)-1]
	if last <= d.distance {
		return true
	}
	return false
}

func (d *dynamicLevenshtein) canMatch(state []uint) bool {
	if len(state) > 0 {
		min := state[0]
		for i := 1; i < len(state); i++ {
			if state[i] < min {
				min = state[i]
			}
		}
		if min <= d.distance {
			return true
		}
	}
	return false
}

func (d *dynamicLevenshtein) accept(state []uint, r *rune) []uint {
	next := []uint{state[0] + 1}
	i := 0
	for _, c := range d.query {
		var cost uint
		if r == nil || c != *r {
			cost = 1
		}
		v := min(min(next[i]+1, state[i+1]+1), state[i]+cost)
		next = append(next, min(v, d.distance+1))
		i++
	}
	return next
}

func min(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
