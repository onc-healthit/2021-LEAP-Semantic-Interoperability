package pkg

import (
	"time"

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
	goDate := time.Time{}
	// check if argument a neo4j.Date or neo4j.LocalDateTime
	if date, ok := args[0].Get().(neo4j.Date); ok {
		goDate = date.Time()
	} else if dateTime, ok := args[0].Get().(neo4j.LocalDateTime); ok {
		goDate = dateTime.Time()
	}
	tm := time.Now()
	// check for second argument type, if null default to time.Now()
	if len(args) > 1 && args[1].Get() != nil {
		if date, ok := args[1].Get().(neo4j.Date); ok {
			tm = date.Time()
		} else if dateTime, ok := args[1].Get().(neo4j.LocalDateTime); ok {
			tm = dateTime.Time()
		}
	}
	// determine age by total month difference, days will be tiebreak
	deltaMonths := (tm.Year()*12 + int(tm.Month()) - 1) - (goDate.Year()*12 + int(goDate.Month()) - 1)
	if goDate.Day() >= tm.Day() {
		age := deltaMonths / 12
		return oc.RValue{Value: age}, nil
	}
	deltaMonths--
	age := deltaMonths / 12
	return oc.RValue{Value: age}, nil
}
