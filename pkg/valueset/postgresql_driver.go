package valueset

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// match (n:OMOPConcept) return n
func GetResults(cfg Config, queryParams map[string]string) (interface{}, error) {
	ret := make([]map[string]interface{}, 0)
DATABASE:
	for _, dbs := range cfg.Databases {
		for _, vs := range dbs.Valuesets {
			if err := cfg.GetConnection(vs, &dbs.Database); err != nil {
				return nil, err
			}
			if dbs.Driver == nil {
				continue DATABASE
			}
			for _, query := range vs.Queries {
				for _, val := range queryParams {
					rows, err := dbs.Driver.Query(query.Query, pgx.NamedArgs{"concept_id": val})
					if err != nil {
						return nil, err
					}
					cols, err := rows.Columns()
					if err != nil {
						return nil, err
					}

					for rows.Next() {
						columns := make([]interface{}, len(cols))
						columnPointers := make([]interface{}, len(cols))
						for i := range columns {
							columnPointers[i] = &columns[i]
						}
						// Scan the result into the column pointers...
						if err := rows.Scan(columnPointers...); err != nil {
							panic(err)
						}
						// dereference values
						m := make(map[string]interface{})
						for i, colName := range cols {
							val := columnPointers[i].(*interface{})
							m[colName] = *val
						}
						ret = append(ret, m)
						// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
						fmt.Println(m)
					}
				}
			}
		}
	}
	return ret, nil
}
