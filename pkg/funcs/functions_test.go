package pkg

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/cloudprivacylabs/lpg"
	"github.com/cloudprivacylabs/lsa/pkg/ls"
	"github.com/cloudprivacylabs/opencypher"
)

type ageTest struct {
	query    string
	expected int
}

func TestAge(t *testing.T) {
	rGetter := func(v opencypher.Value) interface{} {
		return v.Get().(opencypher.ResultSet).Rows[0]["1"].Get()
	}
	oldNow := GetNow
	defer func() {
		GetNow = oldNow
	}()
	GetNow = func() time.Time { return time.Now() }
	tm := GetNow()
	ageTests := []ageTest{
		{
			query:    fmt.Sprintf(`return age(parseDate("01/01/2000", "MM/DD/YYYY"), parseDate("%d/%d/%d", "MM/DD/YYYY"))`, tm.Month(), tm.Day(), tm.Year()),
			expected: 22,
		},
		{
			query:    `return age(parseDate("01/01/1900", "MM/DD/YYYY"), parseDate("01/01/2022", "MM/DD/YYYY"))`,
			expected: 123,
		},
		{
			query:    `return age(parseDate("01/01/1900", "MM/DD/YYYY"), parseDate("01/02/2022", "MM/DD/YYYY"))`,
			expected: 122,
		},
		{
			query:    `return age(parseDate("01/01/1678", "MM/DD/YYYY"), parseDate("01/01/2000", "MM/DD/YYYY"))`,
			expected: 323,
		},
		{
			query:    `return age(parseDate("01/01/1678", "MM/DD/YYYY"), parseDate("01/31/2000", "MM/DD/YYYY"))`,
			expected: 322,
		},
		{
			query:    `return age(parseDate("01/01/1678", "MM/DD/YYYY"))`,
			expected: 344,
		},
		{
			query:    `return age(parseDate("01/01/1922", "MM/DD/YYYY"), parseDate("01/01/2022", "MM/DD/YYYY"))`,
			expected: 101,
		},
		{
			query:    `return age(parseDate("03/03/2001", "MM/DD/YYYY"), parseDate("03/03/2010", "MM/DD/YYYY"))`,
			expected: 10,
		},
		{
			query:    `return age(parseDate("03/03/2001", "MM/DD/YYYY"), parseDate("03/02/2010", "MM/DD/YYYY"))`,
			expected: 9,
		},
	}
	ctx := opencypher.NewEvalContext(lpg.NewGraph())
	for _, tt := range ageTests {
		v, err := opencypher.ParseAndEvaluate(tt.query, ctx)
		if err != nil {
			t.Error(err)
		}
		if rGetter(v) != tt.expected {
			t.Errorf("Got age: %d, expected age: %d", rGetter(v), tt.expected)
		}
	}
}

func TestValuesetLookup(t *testing.T) {
	// lookupValueset({tableId: tableId, param1: x, ...})
	rGetter := func(v opencypher.Value) interface{} {
		return v.Get().(opencypher.ResultSet).Rows[0]["1"].Get()
	}
	vsTests := []struct {
		query    string
		expected map[string]string
	}{
		{
			query:    `return lookupValueset({tableId: "tableId", param1: "x1", param2: "x2"})`,
			expected: map[string]string{"tableId": "tableId", "param1": "x1", "param2": "x2"},
		},
	}
	ctx := opencypher.NewEvalContext(lpg.NewGraph())
	ValuesetLookupFunc = func(ctx *ls.Context, vsreq ls.ValuesetLookupRequest) (ls.ValuesetLookupResponse, error) {
		return ls.ValuesetLookupResponse{KeyValues: vsreq.KeyValues}, nil
	}
	for _, tt := range vsTests {
		v, err := opencypher.ParseAndEvaluate(tt.query, ctx)
		if err != nil {
			t.Error(err)
		}
		mp := make(map[string]string)
		for k, v := range rGetter(v).(map[string]opencypher.Value) {
			mp[k] = v.Get().(string)
		}
		if !reflect.DeepEqual(mp, tt.expected) {
			t.Errorf("Got valueset: %v, expected valueset: %v", rGetter(v), tt.expected)
		}
	}
}
