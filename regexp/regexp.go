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
	"fmt"
	"regexp/syntax"
)

// ErrNoEmpty returned when "zero width assertions" are used
var ErrNoEmpty = fmt.Errorf("zero width assertions not allowed")

// ErrNoWordBoundary returned when word boundaries are used
var ErrNoWordBoundary = fmt.Errorf("word boundaries are not allowed")

// ErrNoBytes returned when byte literals are used
var ErrNoBytes = fmt.Errorf("byte literals are not allowed")

// ErrNoLazy returned when lazy quantifiers are used
var ErrNoLazy = fmt.Errorf("lazy quantifiers are not allowed")

// ErrCompiledTooBig returned when regular expression parses into
// too many instructions
var ErrCompiledTooBig = fmt.Errorf("too many instructions")

// Regexp implements the vellum.Automaton interface for matcing a user
// specified regular expression.
type Regexp struct {
	orig string
	dfa  *dfa
}

// NewRegexp creates a new Regular Expression automaton with the specified
// expression.  By default it is limited to approximately 10MB for the
// compiled finite state automaton.  If this size is exceeded,
// ErrCompiledTooBig will be returned.
func NewRegexp(expr string) (*Regexp, error) {
	return NewRegexpWithLimit(expr, 10*(1<<20))
}

// NewRegexpWithLimit creates a new Regular Expression automaton with
// the specified expression.  The size of the compiled finite state
// automaton exceeds the user specified size,  ErrCompiledTooBig will be
// returned.
func NewRegexpWithLimit(expr string, size uint) (*Regexp, error) {
	parsed, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return nil, err
	}
	compiler := newCompiler(size)
	insts, err := compiler.compile(parsed)
	if err != nil {
		return nil, err
	}
	dfaBuilder := newDfaBuilder(insts)
	dfa, err := dfaBuilder.build()
	if err != nil {
		return nil, err
	}
	return &Regexp{
		orig: expr,
		dfa:  dfa,
	}, nil
}

// Start returns the start state of this automaton.
func (r *Regexp) Start() interface{} {
	var zero uint
	return &zero
}

// IsMatch returns if the specified state is a matching state.
func (r *Regexp) IsMatch(s interface{}) bool {
	if state, ok := s.(*uint); ok {
		return r.dfa.states[*state].match
	}
	return false
}

// CanMatch returns if the specified state can ever transition to a matching
// state.
func (r *Regexp) CanMatch(s interface{}) bool {
	if v, ok := s.(*uint); ok && v != nil {
		return true
	}
	return false
}

// WillAlwaysMatch returns if the specified state will always end in a
// matching state.
func (r *Regexp) WillAlwaysMatch(interface{}) bool {
	return false
}

// Accept returns the new state, resulting from the transite byte b
// when currently in the state s.
func (r *Regexp) Accept(s interface{}, b byte) interface{} {
	if state, ok := s.(*uint); ok {
		next := r.dfa.states[*state].next[b]
		if next == nil {
			return nil
		}
		return next
	}
	return nil
}
