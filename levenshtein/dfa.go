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
	"unicode"

	"github.com/couchbaselabs/vellum/utf8"
)

type dfa struct {
	states statesStack
}

type state struct {
	next  []*uint
	match bool
}

func (s *state) String() string {
	rv := "  |"
	for i := 0; i < 16; i++ {
		rv += fmt.Sprintf("% 5x", i)
	}
	rv += "\n"
	for i := 0; i < len(s.next); i++ {
		if i%16 == 0 {
			rv += fmt.Sprintf("%x |", i/16)
		}
		if s.next[i] != nil {
			rv += fmt.Sprintf("% 5d", *s.next[i])
		} else {
			rv += "    -"
		}
		if i%16 == 15 {
			rv += "\n"
		}
	}
	return rv
}

type dfaBuilder struct {
	dfa   *dfa
	lev   *dynamicLevenshtein
	cache map[string]uint
}

func newDfaBuilder(lev *dynamicLevenshtein) *dfaBuilder {
	return &dfaBuilder{
		dfa: &dfa{
			states: make([]*state, 0, 16),
		},
		lev:   lev,
		cache: make(map[string]uint, 1024),
	}
}

func (b *dfaBuilder) build() (*dfa, error) {
	var stack uintsStack
	stack = stack.Push(b.lev.start())
	seen := make(map[uint]struct{})

	var levState []uint
	stack, levState = stack.Pop()
	for levState != nil {
		dfaSi := b.cachedState(levState)
		mmToSi, mmMismatchState, err := b.addMismatchUtf8States(*dfaSi, levState)
		if err != nil {
			return nil, err
		}
		if mmToSi != nil {
			if _, ok := seen[*mmToSi]; !ok {
				seen[*mmToSi] = struct{}{}
				stack = stack.Push(mmMismatchState)
			}
		}

		i := 0
		for _, r := range b.lev.query {
			if levState[i] > b.lev.distance {
				i++
				continue
			}
			levNext := b.lev.accept(levState, &r)
			nextSi := b.cachedState(levNext)
			if nextSi != nil {
				err = b.addUtf8Sequences(true, *dfaSi, *nextSi, r, r)
				if err != nil {
					return nil, err
				}
				if _, ok := seen[*nextSi]; !ok {
					seen[*nextSi] = struct{}{}
					stack = stack.Push(levNext)
				}
			}
			i++
		}

		if len(b.dfa.states) > StateLimit {
			return nil, ErrTooManyStates
		}

		stack, levState = stack.Pop()
	}

	return b.dfa, nil
}

func (b *dfaBuilder) cachedState(levState []uint) *uint {
	rv, _ := b.cached(levState)
	return rv
}

func (b *dfaBuilder) cached(levState []uint) (*uint, bool) {
	if !b.lev.canMatch(levState) {
		return nil, true
	}
	k := fmt.Sprintf("%v", levState)
	v, ok := b.cache[k]
	if ok {
		return &v, true
	}
	match := b.lev.isMatch(levState)
	b.dfa.states = b.dfa.states.Push(&state{
		next:  make([]*uint, 256),
		match: match,
	})
	newV := uint(len(b.dfa.states) - 1)
	b.cache[k] = newV
	return &newV, false
}

func (b *dfaBuilder) addMismatchUtf8States(fromSi uint, levState []uint) (*uint, []uint, error) {
	mmState := b.lev.accept(levState, nil)
	toSi, _ := b.cached(mmState)
	if toSi == nil {
		return nil, nil, nil
	}
	err := b.addUtf8Sequences(false, fromSi, *toSi, 0, unicode.MaxRune)
	if err != nil {
		return nil, nil, err
	}
	return toSi, mmState, nil
}

func (b *dfaBuilder) addUtf8Sequences(overwrite bool, fromSi, toSi uint, fromChar, toChar rune) error {
	sequences, err := utf8.NewSequences(fromChar, toChar)
	if err != nil {
		return err
	}
	for _, seq := range sequences {
		fsi := fromSi
		for _, utf8r := range seq[:len(seq)-1] {
			tsi := b.newState(false)
			b.addUtf8Range(overwrite, fsi, tsi, utf8r)
			fsi = tsi
		}
		b.addUtf8Range(overwrite, fsi, toSi, seq[len(seq)-1])
	}
	return nil
}

func (b *dfaBuilder) addUtf8Range(overwrite bool, from, to uint, rang *utf8.Range) {
	for by := rang.Start; by <= rang.End; by++ {
		if overwrite || b.dfa.states[from].next[by] == nil {

			b.dfa.states[from].next[by] = &to
		}
	}
}

func (b *dfaBuilder) newState(match bool) uint {
	b.dfa.states = append(b.dfa.states, &state{
		next:  make([]*uint, 256),
		match: match,
	})
	return uint(len(b.dfa.states) - 1)
}
