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
	"io/ioutil"
	"testing"

	pkg "github.com/cloudprivacylabs/leap/pkg/funcs"
	"github.com/cloudprivacylabs/lpg"
	"github.com/cloudprivacylabs/opencypher"
)

func TestReadDir(t *testing.T) {
	opencypher.RegisterGlobalFunc(opencypher.Function{
		Name:      "age",
		MinArgs:   0,
		MaxArgs:   2,
		ValueFunc: pkg.AgeFunc,
	})
	x, err := opencypher.ParseAndEvaluate(`return age("01/01/2022", "09/01/2022")`, opencypher.NewEvalContext(lpg.NewGraph()))
	fmt.Println(x)
	ch, err := ReadDir(context.Background(), "testdata/dir")
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
	if n != 10 {
		t.Errorf("Wrong count")
	}
}
