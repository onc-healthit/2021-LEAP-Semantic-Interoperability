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
	"io/ioutil"
	"testing"
)

func TestReadFiles(t *testing.T) {
	ch, err := ReadFiles(context.Background(), "testdata/dir/1", "testdata/dir/2", "testdata/dir/3", "testdata/dir/4")
	if err != nil {
		t.Error(err)
		return
	}
	n := 0
	for x := range ch {
		n++
		data, err := ioutil.ReadAll(x.Stream)
		if err != nil {
			t.Error(err)
		}
		if len(data) != 5 {
			t.Errorf("Wrong size: %d", len(data))
		}
		x.Close()
	}
	if n != 4 {
		t.Errorf("Wrong count")
	}
}
