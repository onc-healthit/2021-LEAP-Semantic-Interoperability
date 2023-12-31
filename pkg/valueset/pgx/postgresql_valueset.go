package pgx

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"cloudprivacylabs/leap/pkg/utils"
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
	return pgx.getResults(ctx, queryParams, tableId)
}

func (pgx *PostgesqlDataStore) GetTableIds() map[string]struct{} {
	return pgx.tableIds
}

func (pgx *PostgesqlDataStore) Close() error {
	return pgx.close()
}

func NewPostgresqlDataStore(value interface{}, env map[string]string) (valueset.ValuesetDB, error) {
	psqlDs := &PostgesqlDataStore{}
	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           &psqlDs.Database,
		TagName:          "json",
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		panic(err)
	}
	if err := decoder.Decode(value); err != nil {
		return psqlDs, err
	}
	psqlDs.tableIds = make(map[string]struct{})
	for _, vs := range psqlDs.Valuesets {
		psqlDs.tableIds[vs.TableId] = struct{}{}
	}
	psqlDs.Params.URI = utils.ExpandVariables(psqlDs.Params.URI, env)
	return psqlDs, nil
}

type Database struct {
	DB     *sql.DB
	Params Params `json:"params" yaml:"params"`

	Valuesets []Valueset `json:"valuesets" yaml:"valuesets"`
	tableIds  map[string]struct{}
	once      sync.Once
}

type Params struct {
	URI string `json:"uri" yaml:"uri"`
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
		conn, err := sql.Open("pgx", db.Params.URI)
		if err != nil {
			panic(fmt.Sprintf("cannot open database with adapter pgx, URI %s", db.Params.URI))
		}
		db.DB = conn
		_, err = conn.Query("select 1")
		if err != nil {
			panic(err)
		}
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

var runQuery = func(db *sql.DB, q string, args ...any) (*sql.Rows, error) {
	return db.Query(q, args...)
}

func (db *Database) getResults(ctx context.Context, queryParams map[string]string, tableID string) (map[string]string, error) {
	db.DB = db.getConnection()
	ret := make(map[string]string)
	for _, vs := range db.Valuesets {
		if vs.TableId != tableID {
			continue
		}
		for _, query := range vs.Queries {
			queryArgs := pgx.NamedArgs{}
			for key, qPVal := range queryParams {
				queryArgs[key] = qPVal
			}
			rows, err := runQuery(db.DB, query.Query, queryArgs)
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
					return nil, err
				}
				if rows.Next() {
					return nil, valueset.ErrMultipleValues(valueset.ErrMultipleValues{
						TableId: tableID,
						Query:   map[string]string{"query": query.Query},
					})
				}
				// dereference values
				for i, colName := range cols {
					val := columnPointers[i].(*interface{})
					v := *val
					if v != nil {
						ret[colName] = fmt.Sprint(v)
					}
				}
				// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
				return ret, nil
			}
		}
	}
	return ret, nil
}
