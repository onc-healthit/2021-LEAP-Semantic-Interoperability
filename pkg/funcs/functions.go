package pkg

import (
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/araddon/dateparse"
	oc "github.com/cloudprivacylabs/opencypher"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

func init() {
	oc.RegisterGlobalFunc(oc.Function{
		Name:      "age",
		MinArgs:   0,
		MaxArgs:   2,
		ValueFunc: ageFunc,
	})
}

func ageFunc(ctx *oc.EvalContext, args []oc.Value) (oc.Value, error) {
	if args[0].Get() == nil {
		return oc.RValue{}, nil
	}
	var n4jDate neo4j.Date
	var n4jDateTime neo4j.LocalDateTime
	var err error
	// check if argument a neo4j.Date or neo4j.LocalDateTime
	if n4jDate, err = valueAsNeo4jDate(args[0]); err != nil {
		return nil, err
	} else if n4jDateTime, err = valueAsNeo4jDateTime(args[0]); err != nil {
		return nil, err
	}

	var tDob time.Time
	if n4jDateTime != (neo4j.LocalDateTime{}) {
		tDob, err = dateparse.ParseAny(n4jDateTime.String())
		if err != nil {
			return nil, err
		}
	} else {
		tDob, err = dateparse.ParseAny(n4jDate.String())
		if err != nil {
			return nil, err
		}
	}
	tDate := time.Now()
	if len(args) > 1 && args[1].Get() != nil {
		if n4jDate, err = valueAsNeo4jDate(args[1]); err != nil {
			return nil, err
		} else if n4jDateTime, err = valueAsNeo4jDateTime(args[1]); err != nil {
			return nil, err
		}
		if n4jDateTime != (neo4j.LocalDateTime{}) {
			tDate, err = dateparse.ParseAny(n4jDateTime.String())
			if err != nil {
				return nil, err
			}
		} else {
			tDate, err = dateparse.ParseAny(n4jDate.String())
			if err != nil {
				return nil, err
			}
		}

	}
	// age will only represent the year
	deltaMonths := (tDob.Year()*12 + int(tDob.Month()) - 1) - (tDate.Year()*12 + int(tDate.Month()) - 1)
	age := int(math.Floor(float64(deltaMonths / 12)))
	return oc.RValue{Value: age}, nil
}

// valueAsNeo4jDate returns the neo4j.Date value. If value is not neo4j.Date,
// rReturns ErrStringValueRequired. If an err argument is given, that error is returned.
func valueAsNeo4jDate(v oc.Value, err ...error) (neo4j.Date, error) {
	if len(err) != 0 {
		return neo4j.Date{}, err[0]
	}
	fmt.Println(reflect.TypeOf(v.Get()))
	s, ok := v.Get().(neo4j.Date)
	if !ok {
		return neo4j.Date{}, oc.ErrStringValueRequired
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
		return neo4j.LocalDateTime{}, oc.ErrStringValueRequired
	}
	return s, nil
}
