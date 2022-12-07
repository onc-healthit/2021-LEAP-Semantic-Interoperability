package pgx

import (
	"context"
	"database/sql"

	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type getResultsCases struct {
	params  map[string]string
	tableId string
}

func initAndCloseDB(t *testing.T, db *Database) func(t *testing.T) {
	t.Log("initializing db")
	db.URI = "postgres://postgres@0.0.0.0:5433"
	db.getConnection()
	return func(t *testing.T) {
		db.close()
	}
}

func TestGetResults(t *testing.T) {
	// if err := godotenv.Load(); err != nil {
	// 	t.Error(err)
	// }
	// env := map[string]string{
	// 	"${uri}": os.Getenv("uri"),
	// }
	// cfg, err := valueset.LoadConfig("../../testdata/cfg/valueset-databases.yaml", env)
	// if err != nil {
	// 	t.Error(err)
	// }

	db, mock, err := sqlmock.New()
	database := &Database{
		DB: db,
		Valuesets: []Valueset{
			{
				TableId: "gender",
				Queries: []Query{
					{
						Query: "select concept_id,concept_name from concepts where vocabulary_id='ABMS' and concept_code=$1;",
					},
					{
						Query: "select concept_id,concept_name from concepts where vocabulary_id='ABMS' and concept_name=@concept_code;",
					},
				},
			},
		},
	}
	database.once.Do(func() {})
	defer initAndCloseDB(t, database)(t)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cases := []getResultsCases{
		{
			params: map[string]string{
				"concept_code": "OMOP4821938",
				"concept_name": "Pathology - Molecular Genetic",
			},
			tableId: "gender",
		},
		{
			params:  map[string]string{"concept_code": "OMOP4821942"},
			tableId: "gender",
		},
	}
	// mockedRow := sqlmock.NewRows([]string{"concept_id", "concept_name"})
	// mockedRow2 := sqlmock.NewRows([]string{"concept_id", "concept_name"})

	// mock.ExpectBegin()
	q1 := "select concept_id,concept_name from concepts where vocabulary_id='ABMS' and concept_code=$1;"
	q2 := "select concept_id,concept_name from concepts where vocabulary_id='ABMS' and concept_name=@concept_name;"

	argMap := map[string][]interface{}{
		q1: []interface{}{"OMOP4821938"},
		q2: []interface{}{"Pathology - Molecular Genetic"},
	}

	runQuery = func(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
		return db.Query(query, argMap[query]...)
	}
	mock.ExpectQuery(
		q1,
	).WithArgs("OMOP4821938")
	mock.ExpectQuery(
		q2,
	).WithArgs("Pathology - Molecular Genetic")
	// mock.ExpectCommit()

	// execute func
	// for ix, d := range cfg.ValuesetDBs {
	// 	_, err := d.ValueSetLookup(context.Background(), cases[ix].tableId, cases[ix].params)
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	// }
	for _, tt := range cases {
		_, err := database.getResults(context.Background(), tt.params, tt.tableId)
		if err != nil {
			t.Error(err)
		}
		// results = append(results, res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
