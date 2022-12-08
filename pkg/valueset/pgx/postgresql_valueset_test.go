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
	db.once.Do(func() {})
	return func(t *testing.T) {
		db.close()
	}
}

func TestGetResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	database := &Database{
		DB: db,
		Valuesets: []Valueset{
			{
				TableId: "gender",
				Queries: []Query{
					{
						Query: "select concept_id,concept_name from concepts where vocabulary_id='ABMS' and concept_code=@concept_code;",
					},
					{
						Query: "select concept_id,concept_name from concepts where vocabulary_id='ABMS' and concept_name=@concept_code;",
					},
				},
			},
		},
	}
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
		// {
		// 	params:  map[string]string{"concept_code": "OMOP4821942"},
		// 	tableId: "gender",
		// },
	}
	mockedRow := sqlmock.NewRows([]string{"concept_id", "concept_name"}).AddRow("45756805", "Pediatric Cardiology")
	// mockedRow2 := sqlmock.NewRows([]string{"concept_id", "concept_name"}).AddRow("45756801", "Pathology - Molecular Genetic")

	q1 := "select concept_id,concept_name from concepts where vocabulary_id='ABMS' and concept_code=@concept_code;"
	q2 := "select concept_id,concept_name from concepts where vocabulary_id='ABMS' and concept_name=@concept_code;"

	argMap := map[string]string{
		q1: "OMOP4821938",
		q2: "Pathology - Molecular Genetic",
	}

	runQuery = func(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
		return db.Query(query, argMap[query])
	}
	mock.ExpectQuery(q1).WithArgs("OMOP4821938").WillReturnRows(mockedRow)
	// mock.ExpectQuery(q2).WithArgs("Pathology - Molecular Genetic").WillReturnRows(mockedRow2)

	// execute func
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
