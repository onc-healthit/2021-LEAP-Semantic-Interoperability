package pkg

import "github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"

type Config struct {
	Valuesets []Valueset `yaml:"valuesets"`
}

type Valueset struct {
	TableId string  `yaml:"tableId"`
	Queries []Query `yaml:"queries"`
}

type Query struct {
	Query   string   `yaml:"query"`
	Columns []string `yaml:"columns"`
}

func parseYAML() (map[string][]map[string][]map[string][]string, error) {
	var cfg Config
	if err := cmdutil.ReadJSONOrYAML("cfg/queries.yaml", &cfg); err != nil {
		return nil, err
	}
	queries := make([]string, 0)
	columns := make([]string, 0)
	vs_list := cfg.valuesets
	for _, doc := range vs_list {
		for _, qry := range doc.queries {
			queries = append(queries, qry.qry)
			columns = append(columns, qry.columns...)
		}
	}
}
