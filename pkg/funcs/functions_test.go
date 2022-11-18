package pkg

import (
	"testing"

	"github.com/cloudprivacylabs/lpg"
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
	ageTests := []ageTest{
		{
			query:    `return age(parseDate("01/01/2000", "MM/DD/YYYY"))`,
			expected: 22,
		},
		{
			query:    `return age(parseDate("01/01/1900", "MM/DD/YYYY"), parseDate("01/01/2022", "MM/DD/YYYY"))`,
			expected: 122,
		},
		{
			query:    `return age(parseDate("01/01/1678", "MM/DD/YYYY"), parseDate("01/01/2000", "MM/DD/YYYY"))`,
			expected: 322,
		},
		{
			query:    `return age(parseDate("01/01/1678", "MM/DD/YYYY"))`,
			expected: 344,
		},
		{
			query:    `return age(parseDate("01/01/1922", "MM/DD/YYYY"), parseDate("01/01/2022", "MM/DD/YYYY"))`,
			expected: 100,
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
