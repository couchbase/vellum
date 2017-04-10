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

// Automaton represents the general contract of a byte-based finite automaton
type Automaton interface {

	// Start returns the start state
	Start() interface{}

	// IsMatch returns true if and only if the state is a match
	IsMatch(interface{}) bool

	// CanMatch returns true if and only if it is possible to reach a match
	// in zero or more steps
	CanMatch(interface{}) bool

	// WillAlwaysMatch returns true if and only if the current state matches
	// and will always match no matter what steps are taken
	WillAlwaysMatch(interface{}) bool

	// Accept returns the next state given the input to the specified state
	Accept(interface{}, byte) interface{}
}

// AlwaysMatch is an Automaton implementation which always matches
type AlwaysMatch struct{}

// Start returns the AlwaysMatch start state
func (m *AlwaysMatch) Start() interface{} {
	return nil
}

// IsMatch always returns true
func (m *AlwaysMatch) IsMatch(interface{}) bool {
	return true
}

// CanMatch always returns true
func (m *AlwaysMatch) CanMatch(interface{}) bool {
	return true
}

// WillAlwaysMatch always returns true
func (m *AlwaysMatch) WillAlwaysMatch(interface{}) bool {
	return true
}

// Accept returns the next AlwaysMatch state
func (m *AlwaysMatch) Accept(interface{}, byte) interface{} {
	return nil
}
