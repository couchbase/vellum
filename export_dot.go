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
	"bufio"
	"bytes"
	"fmt"
	"io"
)

var dotHeader = `digraph g {
rankdir=LR
`

var dotFooter = `}
`

// ExportBuilderDot will export the contents of the provided Builder into
// the GraphViz (dot) file format.
func ExportBuilderDot(b *Builder, w io.Writer) error {
	bw := bufio.NewWriter(w)

	_, err := bw.WriteString(dotHeader)
	if err != nil {
		return err
	}

	err = exportBuilderStateDot(b.root, bw, map[int]struct{}{})
	if err != nil {
		return err
	}

	_, err = bw.WriteString(dotFooter)
	if err != nil {
		return err
	}

	return bw.Flush()

}

func exportBuilderStateDot(s *builderState, bw *bufio.Writer, seen map[int]struct{}) error {
	if _, already := seen[s.id]; already {
		return nil
	}
	seen[s.id] = struct{}{}

	var buf bytes.Buffer
	if s.finalVal != 0 {
		_, _ = buf.WriteString(fmt.Sprintf("%d [label=\"%d (%d)\"]\n", s.id, s.id, s.finalVal))
	}
	if s.final {
		_, _ = buf.WriteString(fmt.Sprintf("%d [shape=doublecircle]\n", s.id))
	}
	for _, trans := range s.transitions {
		next := trans.dest
		if trans.val != 0 {
			_, _ = buf.WriteString(fmt.Sprintf("%d -> %d [label=\"%s/%d\"]\n", s.id, next.id, string(trans.key), trans.val))
		} else {
			_, _ = buf.WriteString(fmt.Sprintf("%d -> %d [label=\"%s\"]\n", s.id, next.id, string(trans.key)))
		}
	}
	_, _ = buf.WriteString("\n\n")

	_, err := bw.Write(buf.Bytes())
	if err != nil {
		return err
	}

	for _, next := range s.transitions {
		err = exportBuilderStateDot(next.dest, bw, seen)
		if err != nil {
			return err
		}
	}
	return nil
}
