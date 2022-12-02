package valueset

import (
	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
)

type Config struct {
	ValuesetDBs []ValuesetDB
}

type ValuesetDB interface {
	ValueSetLookup(string, map[string]string) (map[string]string, error)
	GetTableIds() map[string]struct{}
	Close() error
}

var valuesetFactory = make(map[string]func(interface{}, map[string]string) (ValuesetDB, error))

func RegisterDB(name string, fn func(interface{}, map[string]string) (ValuesetDB, error)) {
	valuesetFactory[name] = fn
}

type unmarshalConfig struct {
	Databases map[string]interface{} `json:"databases" yaml:"databases"`
}

func UnmarshalConfig(yamlMap map[string][]interface{}, env map[string]string) (Config, error) {
	vsDbs := make([]ValuesetDB, 0)
	for _, rec := range yamlMap {
		for _, mp := range rec {
			for _, vals := range mp.(map[string]interface{}) {
				for key := range vals.(map[string]interface{}) {
					if fn, ok := valuesetFactory[key]; ok {
						vsdb, err := fn(vals, env)
						if err != nil {
							return Config{}, err
						}
						vsDbs = append(vsDbs, vsdb)
					}
				}
			}

		}
	}
	return Config{ValuesetDBs: vsDbs}, nil
}

func LoadConfig(filename string, env map[string]string) (Config, error) {
	var uc unmarshalConfig
	if err := cmdutil.ReadJSONOrYAML(filename, &uc.Databases); err != nil {
		return Config{}, err
	}
	ymlMap := make(map[string][]interface{}, 0)
	for key, rec := range uc.Databases {
		m := cmdutil.YAMLToMap(rec)
		ymlMap[key] = m.([]interface{})
	}
	return UnmarshalConfig(ymlMap, env)
}
