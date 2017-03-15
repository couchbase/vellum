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
	"io"
)

const headerSize = 16

type encoderConstructor func(w io.Writer) encoder

var encoders = map[int]encoderConstructor{}

type encoder interface {
	start(s *Builder) error
	encodeState(s *builderState) error
	finish(s *Builder) error
}

func loadEncoder(ver int, w io.Writer) (encoder, error) {
	if cons, ok := encoders[ver]; ok {
		return cons(w), nil
	}
	return nil, fmt.Errorf("no encoder for version %d registered", ver)
}

func registerEncoder(ver int, cons encoderConstructor) {
	encoders[ver] = cons
}
