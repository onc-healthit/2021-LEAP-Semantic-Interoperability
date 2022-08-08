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
	"os"
)

// ReadFiles reads the given files
func ReadFiles(ctx context.Context, files ...string) (chan Entry, error) {
	ch := make(chan Entry)
	go func() {
		defer close(ch)
		select {
		case <-ctx.Done():
			return
		default:
		}
		for _, file := range files {
			fEntry := Entry{
				Name: file,
				Closer: func(e Entry) {
					if e.Stream != nil {
						e.Stream.(*os.File).Close()
					}
				},
			}
			f, err := os.Open(fEntry.Name)
			if err != nil {
				fEntry.Err = err
			} else {
				fEntry.Stream = f
			}
			ch <- fEntry
		}
	}()
	return ch, nil
}
