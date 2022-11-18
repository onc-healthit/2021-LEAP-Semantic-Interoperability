package pkg

import (
	"testing"

	"github.com/cloudprivacylabs/lpg"
	"github.com/cloudprivacylabs/opencypher"
)

func TestAge(t *testing.T) {
	// return age(parseDate("01/01/2022","MM/DD/YYYY"))
	v1, err := opencypher.ParseAndEvaluate(`return age(parseDate("01/01/2000", "MM/DD/YYYY"))`, opencypher.NewEvalContext(lpg.NewGraph()))
	if err != nil {
		t.Error(err)
	}
	age1 := v1.Get().(opencypher.ResultSet).Rows[0]["1"].Get()
	if age1 != 22 {
		t.Errorf("Got age: %d, expected age: %d", age1, 22)
	}
	// max dob - 1678
	v2, err := opencypher.ParseAndEvaluate(`return age(parseDate("01/01/1900", "MM/DD/YYYY"), parseDate("01/01/2022", "MM/DD/YYYY"))`, opencypher.NewEvalContext(lpg.NewGraph()))
	if err != nil {
		t.Error(err)
	}
	age2 := v2.Get().(opencypher.ResultSet).Rows[0]["1"].Get()
	if age2 != 122 {
		t.Errorf("Got age: %d, expected age: %d", age2, 122)
	}
	v3, err := opencypher.ParseAndEvaluate(`return age(parseDate("01/01/1678", "MM/DD/YYYY"), parseDate("01/01/2022", "MM/DD/YYYY"))`, opencypher.NewEvalContext(lpg.NewGraph()))
	if err != nil {
		t.Error(err)
	}
	age3 := v3.Get().(opencypher.ResultSet).Rows[0]["1"].Get()
	if age3 != 344 {
		t.Errorf("Got age: %d, expected age: %d", age3, 344)
	}
	v4, err := opencypher.ParseAndEvaluate(`return age(parseDate("01/01/1677", "MM/DD/YYYY"), parseDate("01/01/2022", "MM/DD/YYYY"))`, opencypher.NewEvalContext(lpg.NewGraph()))
	if err != nil {
		t.Error(err)
	}
	age4 := v4.Get().(opencypher.ResultSet).Rows[0]["1"].Get()
	if age4 != 345 {
		t.Errorf("Got age: %d, expected age: %d", age4, 345)
	}
}
