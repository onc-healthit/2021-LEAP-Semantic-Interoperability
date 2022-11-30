package valueset

import (
	"database/sql"
	"net/url"
)

type Config struct {
	Databases []struct {
		Database `json:"database" yaml:"database"`
	} `json:"databases" yaml:"databases"`
}

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

func (cfg Config) Close() {
	for _, db := range cfg.Databases {
		db.Database.Driver.Close()
	}
}

func (cfg Config) GetConnection(vs Valueset, datab *Database) error {
	db, err := openDB(cfg)
	if err != nil {
		return err
	}
	datab.Driver = db
	return nil
}

// openDB returns a connection pool
func openDB(cfg Config) (*sql.DB, error) {
	db, err := sql.Open(cfg.Databases[0].Adapter, cfg.Databases[0].URI)
	// db.SetMaxOpenConns(1)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Process(addr string) (map[string]string, error) {
	urlQuery, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	urlParams := make(map[string]string)
	for k, v := range urlQuery.Query() {
		urlParams[k] = v[0]
	}
	return urlParams, nil
}
