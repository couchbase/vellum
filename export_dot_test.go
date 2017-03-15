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
	"io/ioutil"
	"reflect"
	"testing"
)

func TestExportDot(t *testing.T) {
	expected := []byte(`digraph g {
rankdir=LR
0 -> 1 [label="c/3"]


1 -> 2 [label="a"]


2 -> 3 [label="t"]


3 [label="3 (2)"]
3 [shape=doublecircle]
3 -> 4 [label="c"]


4 -> 5 [label="h"]


5 [shape=doublecircle]


}
`)

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
	var buf bytes.Buffer
	err = ExportBuilderDot(b, &buf)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expected, buf.Bytes()) {
		t.Errorf("expected: '%s', got '%s'", expected, buf.Bytes())
	}
}
