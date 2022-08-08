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
	"fmt"
	"io"
	"os"
	"path/filepath"

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
			dirs = append(dirs, filepath.Join(p, fi.Name()))
		} else {
			files = append(files, p)
		}
	}
	ctx.GetLogger().Debug(map[string]interface{}{
		"readPaths.dirs":  len(dirs),
		"readPaths.files": len(files),
	})
	entries := make(chan Entry)
	errors := make(chan error)
	fch, err := ReadFiles(ctx, files...)
	if err != nil {
		return entries, err
	}
	go func() {
		defer close(entries)
		defer close(errors)
		for f := range fch {
			ctx.GetLogger().Debug(map[string]interface{}{
				"readPaths.file": f.Name,
			})
			entries <- f
		}
		for _, dir := range dirs {
			ctx.GetLogger().Debug(map[string]interface{}{
				"readPaths.dir": dir,
			})
			dirCh, err := ReadDir(ctx, dir)
			if err != nil {
				errors <- err
				return
			}
			for dir := range dirCh {
				ctx.GetLogger().Debug(map[string]interface{}{
					"readPaths.atdir": dir,
				})
				entries <- dir
			}
		}
	}()
	select {
	case err, ok := <-errors:
		if ok {
			return entries, fmt.Errorf("%s", err.Error())
		}
	default:
	}
	return entries, nil
}
