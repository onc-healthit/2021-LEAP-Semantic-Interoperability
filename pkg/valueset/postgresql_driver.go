package pkg

import (
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// match (n:OMOPConcept) return n
func getResults(queries, columns []string, queryParams map[string]string) (interface{}, error) {
	db, err := sql.Open("pgx", "postgres://postgres@0.0.0.0:5433")
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to database: %v\n", err))
	}
	defer db.Close()
	ret := make([]map[string]interface{}, 0)
	for _, query := range queries {
		for _, val := range queryParams {
			rows, err := db.Query(query, pgx.NamedArgs{"concept_id": val})
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
	return ret, nil
}
