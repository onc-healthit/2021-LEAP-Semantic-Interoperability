package pkg

import (
	"fmt"
	"testing"
)

func TestPsql(t *testing.T) {
	queries, columns, err := parseYAML()
	if err != nil {
		fmt.Println(err)
	}
	qP, err := process("http://example.com/?concept_id=45756798&concept_name=Microbiology")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(getResults(queries, columns, qP))
}
