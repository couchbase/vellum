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
	"bytes"
	"io"
	"strconv"
)

var defaultBuilderOpts = &BuilderOpts{
	Encoder:           1,
	RegistryTableSize: 10000,
	RegistryMRUSize:   2,
}

// A Builder is used to build a new FST.  When possible data is
// streamed out to the underlying Writer as soon as possible.
type Builder struct {
	opts      *BuilderOpts
	root      *builderState
	nextID    int
	nodeCount uint
	registry  *registry
	lastVal   []byte
	encoder   encoder
	len       int

	implicitFinal *builderState
}

// NewBuilder returns a new Builder which will stream out the
// underlying representation to the provided Writer as the set is built.
func newBuilder(w io.Writer, opts *BuilderOpts) (*Builder, error) {
	if opts == nil {
		opts = defaultBuilderOpts
	}
	rv := &Builder{
		nextID:    1,
		registry:  newRegistry(opts.RegistryTableSize, opts.RegistryMRUSize),
		root:      &builderState{},
		nodeCount: 1,
		opts:      opts,
	}

	var err error
	rv.encoder, err = loadEncoder(opts.Encoder, w)
	if err != nil {
		return nil, err
	}
	err = rv.encoder.start(rv)
	if err != nil {
		return nil, err
	}
	return rv, nil
}

// Insert the provided value to the set being built.
// NOTE: values must be inserted in lexicographical order.
func (s *Builder) Insert(key []byte, val uint64) error {
	// ensure items are added in lexicographic order
	if bytes.Compare(key, s.lastVal) < 0 {
		return ErrOutOfOrder
	}
	// identify the common prefix between this val and the last
	commonLen := commonPrefixLen(s.lastVal, key)
	commonPrefix := key[:commonLen]
	// update last val
	s.lastVal = key
	// find the optimization point, the last common state
	optState, val := s.traverseInsert(commonPrefix, val)
	// optimize the portion that can be updated
	err := s.optimize(optState)
	if err != nil {
		return err
	}
	// add the remaining bytes
	s.addSuffix(optState, key[commonLen:], val)
	s.len++
	return nil
}

// Close MUST be called after inserting all values.
func (s *Builder) Close() error {
	s.lastVal = nil
	err := s.optimize(s.root)
	if err != nil {
		return err
	}
	err = s.encoder.encodeState(s.root)
	if err != nil {
		return err
	}
	err = s.encoder.finish(s)
	if err != nil {
		return err
	}
	return nil
}

func (s *Builder) traverseInsert(key []byte, val uint64) (*builderState, uint64) {
	state := s.root
	var next *transition
	for i := range key {
		var adjustment uint64
		next = state.transitionFor(key[i])
		if next != nil {
			if next.val > val {
				diff := next.val - val
				adjustment += diff
				next.val -= diff
				if next.dest.final {
					next.dest.finalVal += diff
				}
				val = 0
			} else {
				val = val - next.val
			}

			// push down adjustment to all descendants of the current dest
			for j := range next.dest.transitions {
				next.dest.transitions[j].val += adjustment
			}

			state = next.dest
		} else {
			// should never happen during insert, as we already established
			// the common prefix, look for way to eliminate this
			return nil, val
		}
	}
	return state, val
}

func (s *Builder) traverse(key []byte) (*builderState, uint64) {
	var next *transition
	state := s.root
	var val uint64
	for i := range key {
		next = state.transitionFor(key[i])
		if next != nil {
			val += next.val
			state = next.dest
		} else {
			return nil, 0
		}
	}
	if next != nil && next.dest.final {
		val += next.dest.finalVal
	}
	return state, val
}

func (s *Builder) optimize(state *builderState) error {
	if !state.hasTransitions() {
		return nil
	}

	lastTransition := state.lastTransition()
	err := s.optimize(lastTransition.dest)
	if err != nil {
		return err
	}

	// we don't want to waste a slot in the cache for the implicit final state
	// instead, for now we track it explicitly.  this should be cleaned up
	// further in the future, as all we really care is that the file offset
	// becomes 0, but for now this is required.
	if lastTransition.dest.final && !lastTransition.dest.hasTransitions() &&
		lastTransition.dest.finalVal == 0 {

		// the first time we've encountered this situation, remember the state
		if s.implicitFinal == nil {
			s.implicitFinal = lastTransition.dest
			return nil
		}

		// replace ourselves with the implicit final state
		state.replaceTransition(&transition{key: lastTransition.key, dest: s.implicitFinal, val: lastTransition.val})
		s.nodeCount--
		return nil
	}

	if equiv := s.registry.entry(lastTransition.dest); equiv != nil {
		state.replaceTransition(&transition{key: lastTransition.key, dest: equiv, val: lastTransition.val})
		s.nodeCount--
	} else {
		err := s.encoder.encodeState(lastTransition.dest)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Builder) addSuffix(start *builderState, suffix []byte, val uint64) {
	node := start
	for i, char := range suffix {
		newNode := &builderState{
			id: s.nextID,
		}
		transition := &transition{key: char, dest: newNode}
		if i == 0 {
			transition.val = val
		}
		node.addTransition(transition)
		node = newNode
		s.nextID++
		s.nodeCount++
	}
	node.final = true
}

type builderState struct {
	transitions []*transition
	id          int
	final       bool
	finalVal    uint64
	offset      int
}

func (s *builderState) equiv(o *builderState) bool {
	if s.final != o.final {
		return false
	}
	if s.finalVal != o.finalVal {
		return false
	}
	if len(s.transitions) != len(o.transitions) {
		return false
	}
	for i := range s.transitions {
		if !s.transitions[i].equiv(o.transitions[i]) {
			return false
		}
	}
	return true
}

func (s *builderState) hasTransitions() bool {
	return len(s.transitions) > 0
}

func (s *builderState) findTransition(t byte) int {
	for i := range s.transitions {
		if t == s.transitions[i].key {
			return i
		}
	}
	return -1
}

func (s *builderState) transitionFor(t byte) *transition {
	pos := s.findTransition(t)
	if pos < 0 {
		return nil
	}
	return s.transitions[pos]
}

func (s *builderState) replaceTransition(replacement *transition) {
	pos := s.findTransition(replacement.key)
	if pos < 0 {
		return
	}
	s.transitions[pos] = replacement
}

func (s *builderState) lastTransition() *transition {
	if len(s.transitions) < 1 {
		return nil
	}
	return s.transitions[len(s.transitions)-1]
}

func (s *builderState) addTransition(transition *transition) {
	s.transitions = append(s.transitions, transition)
}

func (s *builderState) hash() string {
	var hash string
	if s.final {
		hash += "f"
	}

	for i := range s.transitions {
		transitionState := s.transitions[i].dest
		hash += string(s.transitions[i].key) + strconv.Itoa(transitionState.id)
	}

	return hash
}

type transition struct {
	key  byte
	dest *builderState
	val  uint64
}

func (t *transition) equiv(o *transition) bool {
	if t.key != o.key {
		return false
	}
	if t.dest != o.dest {
		return false
	}
	if t.val != o.val {
		return false
	}
	return true
}

// commonPrefixLen is a helper method used in several places to find
// the common prefix length of two byte slices
func commonPrefixLen(a, b []byte) int {
	prefixLen := 0
	lim := len(a)
	if len(b) < lim {
		lim = len(b)
	}
	for i := 0; i < lim; i++ {
		if a[i] != b[i] {
			break
		}
		prefixLen++
	}
	return prefixLen
}
