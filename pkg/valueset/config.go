package valueset

import (
	"database/sql"
	"fmt"
	"io/ioutil"

	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ValuesetDBs []ValuesetDB
}

/*
	Databases []struct {
		postgresql.Database `json:"database" yaml:"database"`
	} `json:"databases" yaml:"databases"`
*/

type ValuesetDB interface {
	ValueSetLookup(string, map[string]string) (map[string]string, error)
	GetTableIds() []string
}

var DatabaseFactory = map[string]func(interface{}) (ValuesetDB, error){
	"pgx": NewPostgresqlDataStore,
}

func RegisterDB(name string, fn func(interface{}) (ValuesetDB, error)) {
	DatabaseFactory[name] = fn
}

func init() {
	RegisterDB("pgx", NewPostgresqlDataStore)
}

type PostgesqlDataStore struct {
	dsn string
	db  *sql.DB
}

func (pgx PostgesqlDataStore) ValueSetLookup(s string, m map[string]string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (pgx PostgesqlDataStore) GetTableIds() []string {
	return []string{}
}
func NewPostgresqlDataStore(value interface{}) (ValuesetDB, error) {
	return PostgesqlDataStore{}, nil
}

type unmarshalConfig struct {
	Databases map[string]interface{} `json:"databases" yaml:"databases"`
}

// var m = map[string]func(...interface{}){"foo": foo, "bar": bar}
// var c = map[string]int{"foo": 1, "bar": 2}

// func (cfg Config) Close() {
// 	for _, db := range cfg.Databases {
// 		db.Database.Driver.Close()
// 	}
// }

func parseYAML(filename string) (Config, error) {
	var cfg unmarshalConfig
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	if err := yaml.Unmarshal(data, &cfg.Databases); err != nil {
		return Config{}, err
	}

	// use factory to unmarshal config
	vsDbs := make([]ValuesetDB, 0)
	for _, rec := range cfg.Databases {
		m := cmdutil.YAMLToMap(rec)
		// mp, _ := m.(map[string]interface{})
		fmt.Println(m)
		x := m.([]interface{})
		for ix := range x {
			for _, v := range x[ix].(map[string]interface{}) {
				for _, val := range v.(map[string]interface{}) {
					drvTyp, ok := val.(string)
					if ok {
						if fn, ok := DatabaseFactory[drvTyp]; ok {
							vsdb, err := fn(val)
							if err != nil {
								return Config{}, err
							}
							vsDbs = append(vsDbs, vsdb)
						}
					}
				}
			}

		}
	}
	loadEnv(cfg)
	return Config{}, nil
}

func loadEnv(cfg unmarshalConfig) {
	// for ix, db := range cfg.Databases {
	// 	switch db.Adapter {
	// 	case "pgx":
	// 		if err := godotenv.Load("valueset/pgx.env"); err != nil {
	// 			panic(err)
	// 		}
	// 		cfg.Databases[ix].Database.DatabaseName = os.Getenv("driver")
	// 		cfg.Databases[ix].URI = os.Getenv("uri")
	// 		return
	// 	}
	// }
}
