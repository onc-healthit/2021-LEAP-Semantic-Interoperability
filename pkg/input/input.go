// Copyright 2021 Cloud Privacy Labs, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package input

import (
	"io"
	"os"

	"github.com/cloudprivacylabs/lsa/pkg/ls"
)

// Entry represents an input element that contains either a stream, or
// an error for the named input file
type Entry struct {
	Name   string
	Err    error
	Stream io.Reader

	Closer func(Entry)
}

func (e Entry) GetName() string {
	return e.Name
}

func (e Entry) Close() {
	if e.Closer != nil && e.Stream != nil {
		e.Closer(e)
	}
}

// ReadMany combines multiple input channels sequentially
func ReadMany(ch ...<-chan Entry) <-chan Entry {
	ret := make(chan Entry)
	go func() {
		for _, input := range ch {
			for entry := range input {
				ret <- entry
			}
		}
	}()
	return ret
}

// ReadPaths reads files/directories
func ReadPaths(ctx *ls.Context, paths ...string) (<-chan Entry, error) {
	dirs := make([]string, 0)
	files := make([]string, 0)
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if fi.IsDir() {
			dirs = append(dirs, p)
		} else {
			files = append(files, p)
		}
	}
	entries := make(chan Entry)
	fch, err := ReadFiles(ctx, files...)
	if err != nil {
		return entries, err
	}
	go func() {
		defer close(entries)
		for f := range fch {
			entries <- f
		}
		for _, dir := range dirs {
			dirCh, err := ReadDir(ctx, dir)
			if err != nil {
				entries <- Entry{Err: err}
				return
			}
			for dir := range dirCh {
				entries <- dir
			}
		}
	}()
	return entries, nil
}
