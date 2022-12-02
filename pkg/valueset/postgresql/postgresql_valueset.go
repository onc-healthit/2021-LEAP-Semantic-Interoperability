package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cloudprivacylabs/leap/pkg/valueset"
	"github.com/cloudprivacylabs/leap/pkg/valueset/utils"
	"github.com/jackc/pgx/v5"
	"github.com/mitchellh/mapstructure"
)

func init() {
	valueset.RegisterDB("pgx", NewPostgresqlDataStore)
}

type PostgesqlDataStore struct {
	Database `mapstructure:",squash"`
}

func (pgx *PostgesqlDataStore) ValueSetLookup(tableId string, queryParams map[string]string) (map[string]string, error) {
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
	psqlDs.URI = utils.ExpandVariables("${uri}", env)
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
}

type Valueset struct {
	TableId string  `yaml:"tableId"`
	Queries []Query `yaml:"queries"`
}

type Query struct {
	Query string `yaml:"query"`
}

// setConnection sets the database connection at most once
func (db *Database) setConnection() error {
	conn, err := openDB(db)
	if err != nil {
		return err
	}
	// set the db connection only once
	if db.DB == nil {
		db.DB = conn
	}
	return nil
}

// openDB returns a connection pool
func openDB(db *Database) (*sql.DB, error) {
	conn, err := sql.Open("pgx", db.URI)
	// conn.SetMaxOpenConns(db.maxOpenConns)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// close closes the database connection pool
func (db *Database) close() error {
	return db.DB.Close()
}

func (db *Database) getResults(ctx context.Context, queryParams map[string]string, tableID string) (map[string]string, error) {
	ret := make(map[string]string)
	for _, vs := range db.Valuesets {
		if vs.TableId != tableID {
			continue
		}
		if err := db.setConnection(); err != nil {
			return nil, err
		}
		for _, query := range vs.Queries {
			for key, qPval := range queryParams {
				rows, err := db.DB.Query(query.Query, pgx.NamedArgs{key: qPval})
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
						return nil, err
					}
					if rows.Next() {
						return nil, valueset.ErrMultipleValues(valueset.ErrMultipleValues{
							TableId: tableID,
							Query:   query.Query,
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
				}
			}

		}
	}
	return ret, nil
}
