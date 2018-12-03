//  Copyright (c) 2018 Couchbase, Inc.
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

package levenshtein2

// LevenshteinAutomatonBuilder wraps a precomputed
// datastructure that allows to produce small (but not minimal) DFA.
type LevenshteinAutomatonBuilder struct {
	pDfa *ParametricDFA
}

// Creates a Levenshtein automaton builder.
// `maxDistance` - maximum distance considered by the automaton.
// `transposition` - assign a distance of 1 for transposition
//
// Building this automaton builder is computationally intensive.
// While it takes only a few milliseconds for `d=2`, it grows exponentially
// with `d`. It is only reasonable to `d <= 5`.
func NewLevenshteinAutomatonBuilder(maxDistance uint8,
	transposition bool) LevenshteinAutomatonBuilder {
	lnfa := newLevenshtein(maxDistance, transposition)

	return LevenshteinAutomatonBuilder{
		pDfa: fromNfa(lnfa),
	}
}

func (lab *LevenshteinAutomatonBuilder) BuildDfa(query string, fuzziness uint8) *DFA {
	return lab.pDfa.buildDfa(query, fuzziness, false)
}

func (lab *LevenshteinAutomatonBuilder) MaxDistance() uint8 {
	return lab.pDfa.maxDistance
}
