package valueset

import (
	"fmt"
	"testing"
)

func TestPsql(t *testing.T) {
	cfg, err := parseYAML("../testdata/cfg/queries.yaml")
	if err != nil {
		fmt.Println(err)
	}
	_, err = process("http://example.com/?concept_id=45756798&concept_name=Microbiology")
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Println(getResults(queries, columns, qP))
	fmt.Println(cfg)
}
