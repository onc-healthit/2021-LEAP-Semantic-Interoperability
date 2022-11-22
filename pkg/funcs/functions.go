package pkg

import (
	"time"

	"github.com/cloudprivacylabs/lsa/pkg/ls"
	oc "github.com/cloudprivacylabs/opencypher"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

func init() {
	oc.RegisterGlobalFunc(oc.Function{
		Name:      "age",
		MinArgs:   1,
		MaxArgs:   2,
		ValueFunc: ageFunc,
	})
	oc.RegisterGlobalFunc(oc.Function{
		Name:      "lookupValueset",
		MinArgs:   1,
		MaxArgs:   1,
		ValueFunc: lookupValuesetFunc,
	})
}

var GetNow = time.Now

func ageFunc(ctx *oc.EvalContext, args []oc.Value) (oc.Value, error) {
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
	tm := GetNow()
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
	if int(goDate.Month()) == int(tm.Month()) {
		age := deltaMonths / 12
		if goDate.Day() == tm.Day() {
			return oc.RValue{Value: age + 1}, nil
		}
		return oc.RValue{Value: age}, nil
	}
	deltaMonths--
	age := deltaMonths / 12
	return oc.RValue{Value: age}, nil
}

var ValuesetLookupFunc func(*ls.Context, ls.ValuesetLookupRequest) (ls.ValuesetLookupResponse, error)

func lookupValuesetFunc(ctx *oc.EvalContext, args []oc.Value) (oc.Value, error) {
	if args[0] == nil {
		return oc.RValue{}, nil
	}
	vs, ok := args[0].Get().(map[string]oc.Value)
	if !ok {
		return oc.RValue{}, nil
	}
	req := ls.ValuesetLookupRequest{
		TableIDs:  []string{vs["tableId"].Get().(string)},
		KeyValues: make(map[string]string),
	}
	for k, v := range vs {
		req.KeyValues[k] = v.Get().(string)
	}

	vals := make(map[string]oc.Value)
	resp, err := ValuesetLookupFunc(ls.DefaultContext(), req)
	if err != nil {
		return oc.RValue{}, nil
	}
	for k, v := range resp.KeyValues {
		vals[k] = oc.ValueOf(v)
	}

	return oc.RValue{Value: vals}, nil
}
