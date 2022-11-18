package pkg

import (
	"time"

	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/pkg/types"
	oc "github.com/cloudprivacylabs/opencypher"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

func init() {
	oc.RegisterGlobalFunc(oc.Function{
		Name:      "age",
		MinArgs:   1,
		MaxArgs:   2,
		ValueFunc: AgeFunc,
	})
}

func AgeFunc(ctx *oc.EvalContext, args []oc.Value) (oc.Value, error) {
	if args[0].Get() == nil {
		return oc.RValue{}, nil
	}
	var n4jDate neo4j.Date
	var n4jDateTime neo4j.LocalDateTime
	var err error
	// check if argument a neo4j.Date or neo4j.LocalDateTime
	if n4jDate, err = valueAsNeo4jDate(args[0]); err != nil {
		return nil, err
	}
	if n4jDate == (neo4j.Date{}) {
		if n4jDateTime, err = valueAsNeo4jDateTime(args[0]); err != nil {
			return nil, err
		}
	}
	var dob interface{}
	if n4jDateTime != (neo4j.LocalDateTime{}) {
		dob = neo.Neo4jValueToNativeValue(n4jDateTime)
	} else {
		dob = neo.Neo4jValueToNativeValue(n4jDate)
	}
	var tDob types.Date
	var tDobDt types.DateTime
	tDob, ok := dob.(types.Date)
	if !ok {
		tDobDt, ok = dob.(types.DateTime)
		if !ok {
			panic("")
		}
	}
	var tm interface{}
	if len(args) > 1 && args[1].Get() != nil {
		if n4jDate, err = valueAsNeo4jDate(args[1]); err != nil {
			return nil, err
		}
		if n4jDate == (neo4j.Date{}) {
			if n4jDateTime, err = valueAsNeo4jDateTime(args[1]); err != nil {
				return nil, err
			}
		}
		if n4jDateTime != (neo4j.LocalDateTime{}) {
			tm = neo.Neo4jValueToNativeValue(n4jDateTime)
		} else {
			tm = neo.Neo4jValueToNativeValue(n4jDate)
		}
	} else {
		tm = time.Now()
	}
	// determine second argument type, default to time.Time
	var tDate types.Date
	var tDateTime types.DateTime
	var tTime time.Time
	switch t := tm.(type) {
	case types.Date:
		tDate = t
	case types.DateTime:
		tDateTime = t
	default:
		tTime = tm.(time.Time)
	}
	var age int
	if tDate.Year != 0 && tDob.Year != 0 {
		age = tDate.Year - tDob.Year
	} else if tDateTime.Year != 0 && tDobDt.Year != 0 {
		age = tDateTime.Year - tDobDt.Year
	} else if tDobDt.Year != 0 {
		age = tTime.Year() - tDobDt.Year
	} else {
		age = tTime.Year() - tDob.Year
	}
	return oc.RValue{Value: age}, nil
}

// valueAsNeo4jDate returns the neo4j.Date value. If value is not neo4j.Date,
// rReturns ErrStringValueRequired. If an err argument is given, that error is returned.
func valueAsNeo4jDate(v oc.Value, err ...error) (neo4j.Date, error) {
	if len(err) != 0 {
		return neo4j.Date{}, err[0]
	}
	s, ok := v.Get().(neo4j.Date)
	if !ok {
		return neo4j.Date{}, oc.ErrNeo4jDateValueRequired
	}
	return s, nil
}

// valueAsNeo4jDateTime returns the neo4j.LocalDateTime value. If value is not neo4j.LocalDateTime,
// rReturns ErrStringValueRequired. If an err argument is given, that error is returned.
func valueAsNeo4jDateTime(v oc.Value, err ...error) (neo4j.LocalDateTime, error) {
	if len(err) != 0 {
		return neo4j.LocalDateTime{}, err[0]
	}
	s, ok := v.Get().(neo4j.LocalDateTime)
	if !ok {
		return neo4j.LocalDateTime{}, oc.ErrNeo4jDateTimeValueRequired
	}
	return s, nil
}
