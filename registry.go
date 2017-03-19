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
	"hash"
	"hash/fnv"
)

type registry struct {
	table     []*builderState
	tableSize uint
	mruSize   uint
	hasher    hash.Hash64
}

func newRegistry(tableSize, mruSize int) *registry {
	nsize := tableSize * mruSize
	rv := &registry{
		table:     make([]*builderState, nsize),
		tableSize: uint(tableSize),
		mruSize:   uint(mruSize),
		hasher:    fnv.New64a(),
	}
	return rv
}

func (r *registry) entry(node *builderState) *builderState {
	if len(r.table) == 0 {
		return nil
	}
	bucket := r.hash(node)
	start := r.mruSize * uint(bucket)
	end := start + r.mruSize
	rc := registryCache(r.table[start:end])
	return rc.entry(node)
}

const fnvPrime = 1099511628211

func (r *registry) hash(b *builderState) int {
	var final uint64
	if b.final {
		final = 1
	}

	var h uint64 = 14695981039346656037
	h ^= (final * fnvPrime)
	h ^= (b.finalVal * fnvPrime)
	for _, t := range b.transitions {
		h ^= (uint64(t.key) * fnvPrime)
		h ^= (t.val * fnvPrime)
		h ^= (uint64(t.dest.id) * fnvPrime)
	}
	return int(h % uint64(r.tableSize))
}

type registryCache []*builderState

func (r registryCache) entry(node *builderState) *builderState {
	if len(r) == 1 {
		cell := r[0]
		if cell != nil && cell.equiv(node) {
			return cell
		}
		r[0] = node
		return nil
	}
	for i, ent := range r {
		if ent != nil && ent.equiv(node) {
			r.promote(i)
			return ent
		}
	}
	// no match
	last := len(r) - 1
	r[last] = node // discard LRU
	r.promote(last)
	return nil

}

func (r registryCache) promote(i int) {
	for i > 0 {
		r.swap(i-1, i)
		i--
	}
}

func (r registryCache) swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
