package pkg

import (
	"database/sql"
	"fmt"
	"os"
)

func getResults(queries []string, queryParams map[string]string) {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	// executes commands from input

}
