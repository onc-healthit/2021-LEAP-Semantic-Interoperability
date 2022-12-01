package valueset

import (
	"fmt"
	"testing"
)

func TestL(t *testing.T) {
	x, err := parseYAML("../testdata/cfg/valueset-databases.yaml")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(x)
}
