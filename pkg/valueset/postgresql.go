package pkg

import "github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"

type config struct {
	valuesets []valueset `yaml:"valuesets"`
}

type valueset struct {
	tableId string
	queries []query
}

type query struct {
	qry     string
	columns []string
}

func parseYAML() (map[string][]map[string][]map[string][]string, error) {
	var cfg config
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
