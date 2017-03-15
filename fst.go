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

import "io"

// FST is an in-memory representation of a finite state transducer,
// capable of returning the uint64 value associated with
// each []byte key stored, as well as enumerating all of the keys
// in order.
type FST struct {
	f       io.Closer
	ver     int
	len     int
	data    []byte
	decoder decoder
}

func newFST(data []byte) (*FST, error) {
	rv := &FST{
		data: data,
	}

	err := rv.initFST()
	if err != nil {
		return nil, err
	}

	return rv, nil
}

func (f *FST) initFST() error {
	var err error
	f.ver, _, err = decodeHeader(f.data)
	if err != nil {
		return err
	}

	f.decoder, err = loadDecoder(f.ver, f.data)
	if err != nil {
		return err
	}

	err = f.decoder.start(f)
	if err != nil {
		return err
	}

	f.len = f.decoder.getLen()

	return nil
}

// Contains returns true if this FST contains the specified key.
func (f *FST) Contains(val []byte) (bool, error) {
	_, exists, err := f.Get(val)
	return exists, err
}

// Get returns the value associated with the key.  NOTE: a value of zero
// does not imply the key does not exist, you must consult the second
// return value as well.
func (f *FST) Get(input []byte) (uint64, bool, error) {

	var total uint64
	curr := f.decoder.getRoot()
	state, err := f.decoder.stateAt(curr)
	if err != nil {
		return 0, false, err
	}
	for i := range input {
		_, curr, output := state.TransitionFor(input[i])
		if curr < 0 {
			return 0, false, nil
		}

		state, err = f.decoder.stateAt(curr)
		if err != nil {
			return 0, false, err
		}

		total += output
	}

	if state.Final() {
		total += state.FinalOutput()
		return total, true, nil
	}
	return 0, false, nil
}

// Version returns the encoding version used by this FST instance.
func (f *FST) Version() int {
	return f.ver
}

// Len returns the number of entries in this FST instance.
func (f *FST) Len() int {
	return f.len
}

// Close will unmap any mmap'd data (if managed by vellum) and it will close
// the backing file (if managed by vellum).  You MUST call Close() for any
// FST instance that is created.
func (f *FST) Close() error {
	if f.f != nil {
		err := f.f.Close()
		if err != nil {
			return err
		}
	}
	f.data = nil
	f.decoder = nil
	return nil
}

// Iterator returns a new Iterator capable of enumerating the key/value pairs
// between the provided startKeyInclusive and endKeyExclusive.
func (f *FST) Iterator(startKeyInclusive, endKeyExclusive []byte) (*Iterator, error) {
	return newIterator(f, startKeyInclusive, endKeyExclusive)
}

// DebugDump is only intended for debug purproses, it simply asks the underlying
// decoder to output a debug representation to the provided Writer.
func (f *FST) DebugDump(w io.Writer) error {
	return f.decoder.debugDump(w)
}
