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
	"encoding/binary"
	"fmt"
	"io"
)

const versionV1 = 1
const oneTransition = 1 << 7
const transitionNext = 1 << 6
const stateFinal = 1 << 6
const footerSizeV1 = 16

func init() {
	registerEncoder(versionV1, func(w io.Writer) encoder {
		return newEncoderV1(w)
	})
}

type encoderV1 struct {
	bw        *writer
	lastState int
}

func newEncoderV1(w io.Writer) *encoderV1 {
	return &encoderV1{
		bw:        newWriter(w),
		lastState: -1,
	}
}

func (e *encoderV1) start(s *Builder) error {
	header := make([]byte, headerSize)
	binary.LittleEndian.PutUint64(header, versionV1)
	binary.LittleEndian.PutUint64(header[8:], uint64(0)) // type
	n, err := e.bw.Write(header)
	if err != nil {
		return err
	}
	if n != headerSize {
		return fmt.Errorf("short write of header %d/%d", n, headerSize)
	}
	return nil
}

func (e *encoderV1) encodeState(s *builderState) error {
	if len(s.transitions) == 0 && s.final && s.finalVal == 0 {
		return nil
	} else if len(s.transitions) != 1 || s.final {
		return e.encodeStateMany(s)
	} else if !s.final && s.transitions[0].val == 0 && s.transitions[0].dest.offset == e.lastState {
		return e.encodeStateOneFinish(s, transitionNext)
	}
	return e.encodeStateOne(s)
}

func (e *encoderV1) encodeStateOne(s *builderState) error {
	start := uint64(e.bw.counter)
	outPackSize := 0
	if s.transitions[0].val != 0 {
		outPackSize = packedSize(s.transitions[0].val)
		err := e.bw.WritePackedUintIn(s.transitions[0].val, outPackSize)
		if err != nil {
			return err
		}
	}
	delta := deltaAddr(start, uint64(s.transitions[0].dest.offset))
	transPackSize := packedSize(delta)
	err := e.bw.WritePackedUintIn(delta, transPackSize)
	if err != nil {
		return err
	}

	packSize := encodePackSize(transPackSize, outPackSize)
	err = e.bw.WriteByte(packSize)
	if err != nil {
		return err
	}

	return e.encodeStateOneFinish(s, 0)
}

func (e *encoderV1) encodeStateOneFinish(s *builderState, next byte) error {
	enc := encodeCommon(s.transitions[0].key)

	// not a common input
	if enc == 0 {
		err := e.bw.WriteByte(s.transitions[0].key)
		if err != nil {
			return err
		}
	}
	err := e.bw.WriteByte(oneTransition | next | enc)
	if err != nil {
		return err
	}

	s.offset = e.bw.counter - 1
	e.lastState = s.offset
	return nil
}

func (e *encoderV1) encodeStateMany(s *builderState) error {
	start := uint64(e.bw.counter)
	transPackSize := 0
	outPackSize := packedSize(s.finalVal)
	anyOutputs := s.finalVal != 0
	for i := range s.transitions {
		delta := deltaAddr(start, uint64(s.transitions[i].dest.offset))
		tsize := packedSize(delta)
		if tsize > transPackSize {
			transPackSize = tsize
		}
		osize := packedSize(s.transitions[i].val)
		if osize > outPackSize {
			outPackSize = osize
		}
		anyOutputs = anyOutputs || s.transitions[i].val != 0
	}
	if !anyOutputs {
		outPackSize = 0
	}

	if anyOutputs {
		// output final value
		if s.final {
			err := e.bw.WritePackedUintIn(s.finalVal, outPackSize)
			if err != nil {
				return err
			}
		}
		// output transition values (in reverse)
		for j := len(s.transitions) - 1; j >= 0; j-- {
			err := e.bw.WritePackedUintIn(s.transitions[j].val, outPackSize)
			if err != nil {
				return err
			}
		}
	}

	// output transition dests (in reverse)
	for j := len(s.transitions) - 1; j >= 0; j-- {
		delta := deltaAddr(start, uint64(s.transitions[j].dest.offset))
		err := e.bw.WritePackedUintIn(delta, transPackSize)
		if err != nil {
			return err
		}
	}

	// output transition keys (in reverse)
	for j := len(s.transitions) - 1; j >= 0; j-- {
		err := e.bw.WriteByte(s.transitions[j].key)
		if err != nil {
			return err
		}
	}

	packSize := encodePackSize(transPackSize, outPackSize)
	err := e.bw.WriteByte(packSize)
	if err != nil {
		return err
	}

	numTrans := encodeNumTrans(len(s.transitions))

	// if number of transitions wont fit in edge header byte
	// write out separately
	if numTrans == 0 {
		if len(s.transitions) == 256 {
			// this wouldn't fit in single byte, but reuse value 1
			// which would have always fit in the edge header instead
			err = e.bw.WriteByte(1)
			if err != nil {
				return err
			}
		} else {
			err = e.bw.WriteByte(byte(len(s.transitions)))
			if err != nil {
				return err
			}
		}
	}

	// finally write edge header
	if s.final {
		numTrans |= stateFinal
	}
	err = e.bw.WriteByte(numTrans)
	if err != nil {
		return err
	}

	s.offset = e.bw.counter - 1
	e.lastState = s.offset

	return nil
}

func (e *encoderV1) finish(s *Builder) error {
	footer := make([]byte, footerSizeV1)
	binary.LittleEndian.PutUint64(footer, uint64(s.len))             // root addr
	binary.LittleEndian.PutUint64(footer[8:], uint64(s.root.offset)) // root addr
	n, err := e.bw.Write(footer)
	if err != nil {
		return err
	}
	if n != footerSizeV1 {
		return fmt.Errorf("short write of footer %d/%d", n, footerSizeV1)
	}
	err = e.bw.Flush()
	if err != nil {
		return err
	}
	return nil
}
