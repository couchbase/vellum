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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

// ExportBuilderSVGFile will invoke ExportBuilderSVG and send the output
// to a new file at the provided path.
func ExportBuilderSVGFile(b *Builder, path string) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()
	return ExportBuilderSVG(b, file)
}

// ExportBuilderSVG will take the provided Builder and generate an SVG
// representation of the FST in it's current state.  This SVG will be
// streamed to the provided writer.
func ExportBuilderSVG(b *Builder, w io.Writer) error {
	pr, pw := io.Pipe()
	go func() {
		defer func() {
			_ = pw.Close()
		}()
		_ = ExportBuilderDot(b, pw)
	}()
	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = pr
	cmd.Stdout = w
	cmd.Stderr = ioutil.Discard
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
