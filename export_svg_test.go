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

// +build havedot

package vellum

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestExportBuilderSVGFile(t *testing.T) {
	b, err := New(ioutil.Discard, nil)
	if err != nil {
		t.Fatalf("error creating new builder: %v", err)
	}
	err = b.Insert([]byte("cat"), 5)
	if err != nil {
		t.Fatal(err)
	}
	err = b.Insert([]byte("catch"), 3)
	if err != nil {
		t.Fatal(err)
	}
	err = b.Close()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir, err := ioutil.TempDir("", "vellum-svg")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	path := tmpDir + string(os.PathSeparator) + "tmp.svg"

	err = ExportBuilderSVGFile(b, path)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	finfo, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}

	if finfo.Size() == 0 {
		t.Fatalf("expected non-zero file size, got 0")
	}
}
