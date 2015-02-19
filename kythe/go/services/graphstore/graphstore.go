/*
 * Copyright 2015 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package graphstore defines the Service and Sharded interfaces, and provides
// some useful utility functions.
package graphstore

import (
	"strings"

	"kythe/go/services/graphstore/compare"

	spb "kythe/proto/storage_proto"
)

// An EntryFunc is a callback from the implementation of a Service to deliver
// entry messages. If the callback returns an error, the operation stops.  If
// the error is io.EOF, the operation returns nil; otherwise it returns the
// error value from the callback.
type EntryFunc func(*spb.Entry) error

// Service refers to an open Kythe graph store.
type Service interface {
	// Read calls f with each entry having the ReadRequest's given source
	// VName, subject to the following rules:
	//
	//  |----------+---------------------------------------------------------|
	//  | EdgeKind | Result                                                  |
	//  |----------+---------------------------------------------------------|
	//  | ø        | All entries with kind and target empty (node entries).  |
	//  | "*"      | All entries (node and edge, regardless of kind/target). |
	//  | "kind"   | All edge entries with the given edge kind.              |
	//  |----------+---------------------------------------------------------|
	//
	// Read returns when there are no more entries to send. The Read operation should be
	// implemented with time complexity proportional to the size of the return set.
	Read(req *spb.ReadRequest, f EntryFunc) error

	// Scan calls f with each entries having the specified target VName, kind,
	// and fact label prefix. If a field is empty, any entry value for that
	// field matches and will be returned. Scan returns when there are no more
	// entries to send. Scan is similar to Read, but with no time complexity
	// restrictions.
	Scan(req *spb.ScanRequest, f EntryFunc) error

	// Write atomically inserts or updates a collection of entries into the store.
	// Each update is a tuple of the form (kind, target, fact, value). For each such
	// update, an entry (source, kind, target, fact, value) is written into the store,
	// replacing any existing entry (source, kind, target, fact, value') that may
	// exist. Note that this operation cannot delete any data from the store; entries are
	// only ever inserted or updated. Apart from acting atomically, no other constraints
	// are placed on the implementation.
	Write(req *spb.WriteRequest) error

	// Close and release any underlying resources used by the store.
	// No operations may be used on the store after this has been called.
	Close() error
}

// Sharded represents a store that can be arbitrarily sharded for parallel
// processing.  Depending on the implementation, these methods may not return
// consistent results when the store is being written to.  Shards are indexed
// from 0.
type Sharded interface {
	Service

	// Count returns the number of entries in the given shard.
	Count(req *spb.CountRequest) (int64, error)

	// Shard calls f with each entry in the given shard.
	Shard(req *spb.ShardRequest, f EntryFunc) error
}

// EntryMatchesScan reports whether entry belongs in the result set for req.
func EntryMatchesScan(req *spb.ScanRequest, entry *spb.Entry) bool {
	return (req.GetTarget() == nil || compare.VNamesEqual(entry.Target, req.Target)) &&
		(req.GetEdgeKind() == "" || entry.GetEdgeKind() == req.GetEdgeKind()) &&
		strings.HasPrefix(entry.GetFactName(), req.GetFactPrefix())
}

// BatchWrites returns a channel of WriteRequests for the given entries.
// Consecutive entries with the same Source will be collected in the same
// WriteRequest, with each request containing up to maxSize updates.
func BatchWrites(entries <-chan *spb.Entry, maxSize int) <-chan *spb.WriteRequest {
	ch := make(chan *spb.WriteRequest)
	go func() {
		defer close(ch)
		var req *spb.WriteRequest
		for entry := range entries {
			update := &spb.WriteRequest_Update{
				EdgeKind:  entry.EdgeKind,
				Target:    entry.Target,
				FactName:  entry.FactName,
				FactValue: entry.FactValue,
			}

			if req != nil && (!compare.VNamesEqual(req.Source, entry.Source) || len(req.Update) >= maxSize) {
				ch <- req
				req = nil
			}

			if req == nil {
				req = &spb.WriteRequest{
					Source: entry.Source,
					Update: []*spb.WriteRequest_Update{update},
				}
			} else {
				req.Update = append(req.Update, update)
			}
		}
		if req != nil {
			ch <- req
		}
	}()
	return ch
}