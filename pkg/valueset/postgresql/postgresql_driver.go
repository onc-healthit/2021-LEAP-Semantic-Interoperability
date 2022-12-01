package postgresql

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// match (n:OMOPConcept) return n
func GetResults(db Database, queryParams map[string]string) (map[string]string, error) {
	ret := make(map[string]string)
	seenTableIds := make(map[string]struct{})
	for _, vs := range db.Valuesets {
		if _, seen := seenTableIds[vs.TableId]; seen {
			continue
		}
		if err := db.GetConnection(); err != nil {
			return nil, err
		}
		seenTableIds[vs.TableId] = struct{}{}
		for _, query := range vs.Queries {
			for key, qPval := range queryParams {
				rows, err := db.Driver.Query(query.Query, pgx.NamedArgs{key: qPval})
				if err != nil {
					return nil, err
				}
				if !rows.Next() {
					continue
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
					if rows.Next() {
						return nil, errors.New("more than 1 row returned")
					}
					// dereference values
					// m := make(map[string]interface{})
					for i, colName := range cols {
						val := columnPointers[i].(*interface{})
						v := *val
						ret[colName] = v.(string)
					}
					// ret = append(ret, m)
					// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
					fmt.Println(ret)
				}
			}

		}
	}
	return ret, nil
}
