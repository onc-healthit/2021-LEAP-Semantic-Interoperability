package postgresql

import (
	"database/sql"
)

type Database struct {
	Driver       *sql.DB
	Adapter      string     `json:"adapter" yaml:"adapter"`
	DatabaseName string     `json:"db" yaml:"db"`
	User         string     `json:"user" yaml:"user"`
	Pwd          string     `json:"pwd" yaml:"pwd"`
	URI          string     `json:"uri" yaml:"uri"`
	Valuesets    []Valueset `json:"valuesets" yaml:"valuesets"`
}

type Valueset struct {
	TableId string  `yaml:"tableId"`
	Queries []Query `yaml:"queries"`
}

type Query struct {
	Query string `yaml:"query"`
}

func (db *Database) GetConnection() error {
	conn, err := openDB(*db)
	if err != nil {
		return err
	}
	db.Driver = conn
	return nil
}

// openDB returns a connection pool
func openDB(db Database) (*sql.DB, error) {
	conn, err := sql.Open(db.Adapter, db.URI)
	// db.SetMaxOpenConns(1)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
