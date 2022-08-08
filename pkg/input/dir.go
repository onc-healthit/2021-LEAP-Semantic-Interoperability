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
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ReadDir reads the files in the given directory
func ReadDir(ctx context.Context, path string) (<-chan Entry, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fmt.Println("Opened", path)
	ch := make(chan Entry)
	go func() {
		defer close(ch)
		defer dir.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			fmt.Println("Reading dir")
			entries, err := dir.ReadDir(1024)
			fmt.Println("entries", len(entries))
			for _, entry := range entries {
				if (entry.Type() & fs.ModeType) == 0 {
					fEntry := Entry{
						Name: filepath.Join(path, entry.Name()),
						Closer: func(e Entry) {
							if e.Stream != nil {
								e.Stream.(*os.File).Close()
							}
						},
					}
					fmt.Println("open", fEntry)
					f, err := os.Open(fEntry.Name)
					if err != nil {
						fEntry.Err = err
					} else {
						fEntry.Stream = f
					}
					fmt.Println("sending")
					ch <- fEntry
					fmt.Println("Sent")
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				ch <- Entry{Err: err}
			}
		}
	}()
	return ch, nil
}
