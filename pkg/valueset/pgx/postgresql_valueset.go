package pgx

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/cloudprivacylabs/leap/pkg/utils"
	"github.com/cloudprivacylabs/lsa/layers/cmd/valueset"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mitchellh/mapstructure"
)

func init() {
	valueset.RegisterDB("pgx", NewPostgresqlDataStore)
}

type PostgesqlDataStore struct {
	Database `mapstructure:",squash"`
}

func (pgx *PostgesqlDataStore) ValueSetLookup(ctx context.Context, tableId string, queryParams map[string]string) (map[string]string, error) {
	return pgx.getResults(context.Background(), queryParams, tableId)
}

func (pgx *PostgesqlDataStore) GetTableIds() map[string]struct{} {
	return pgx.tableIds
}

func (pgx *PostgesqlDataStore) Close() error {
	return pgx.close()
}

func NewPostgresqlDataStore(value interface{}, env map[string]string) (valueset.ValuesetDB, error) {
	psqlDs := &PostgesqlDataStore{}
	if err := mapstructure.Decode(value, psqlDs); err != nil {
		return psqlDs, err
	}
	psqlDs.tableIds = make(map[string]struct{})
	for _, vs := range psqlDs.Valuesets {
		psqlDs.tableIds[vs.TableId] = struct{}{}
	}
	// psqlDs.URI = utils.ExpandVariables("${uri}", env)
	psqlDs.URI = "postgres://postgres@0.0.0.0:5433"
	psqlDs.User = utils.ExpandVariables("${pgx_user}", env)
	psqlDs.Pwd = utils.ExpandVariables("${pwd}", env)
	return psqlDs, nil
}

type Database struct {
	DB           *sql.DB
	DatabaseName string     `json:"db" yaml:"db"`
	User         string     `json:"user" yaml:"user"`
	Pwd          string     `json:"pwd" yaml:"pwd"`
	URI          string     `json:"uri" yaml:"uri"`
	Valuesets    []Valueset `json:"valuesets" yaml:"valuesets"`
	tableIds     map[string]struct{}
	once         sync.Once
}

type Valueset struct {
	TableId string  `yaml:"tableId"`
	Queries []Query `yaml:"queries"`
}

type Query struct {
	Query string `yaml:"query"`
}

// getConnection sets the database connection at most once and returns the DB connection pool
func (db *Database) getConnection() *sql.DB {
	db.once.Do(func() {
		conn, err := sql.Open("pgx", db.URI)
		if err != nil {
			panic(fmt.Sprintf("cannot open database with adapter pgx, URI %s", db.URI))
		}
		db.DB = conn
	})
	if db.DB == nil {
		panic("missing database connection")
	}
	return db.DB
}

// close closes the database connection pool
func (db *Database) close() error {
	return db.DB.Close()
}

var runQuery = func(db *sql.DB, q string, args ...interface{}) (*sql.Rows, error) { return db.Query(q, args...) }

// // Returns s quoted as a string literal, in single-quotes. Any
// // single-quotes are escaped with \', and \ are escaped with \\
// func quoteStringLiteral(s string) string {
// 	s = strings.ReplaceAll(s, `(`, `'`)
// 	s = strings.ReplaceAll(s, `)`, `''`)
// 	return s
// }

func (db *Database) getResults(ctx context.Context, queryParams map[string]string, tableID string) (map[string]string, error) {
	db.DB = db.getConnection()
	ret := make(map[string]string)
	for _, vs := range db.Valuesets {
		if vs.TableId != tableID {
			continue
		}
		var ix int
	NextQuery:
		for _, query := range vs.Queries {
			for key, qPval := range queryParams {
				fmt.Println(ix)
				ix++
				rows, err := runQuery(db.DB, query.Query, pgx.NamedArgs{key: qPval})
				if err != nil {
					return nil, err
				}
				// if !rows.Next() {
				// 	continue
				// }
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
						return nil, err
					}
					if rows.Next() {
						return nil, valueset.ErrMultipleValues(valueset.ErrMultipleValues{
							TableId: tableID,
							Query:   map[string]string{"query": query.Query},
						})
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
					delete(queryParams, key)
					continue NextQuery
				}
			}

		}
	}
	return ret, nil
}
