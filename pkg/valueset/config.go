package valueset

import (
	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
)

type Config struct {
	ValuesetDBs []ValuesetDB
}

type ValuesetDB interface {
	ValueSetLookup(string, map[string]string) (map[string]string, error)
	GetTableIds() []string
	Close() error
}

var valuesetFactory = make(map[string]func(interface{}, map[string]string) (ValuesetDB, error))

func RegisterDB(name string, fn func(interface{}, map[string]string) (ValuesetDB, error)) {
	valuesetFactory[name] = fn
}

type unmarshalConfig struct {
	Databases map[string]interface{} `json:"databases" yaml:"databases"`
}

func (uc *unmarshalConfig) UnmarshalConfig(filename string, env map[string]string) (Config, error) {
	vsDbs := make([]ValuesetDB, 0)
	for _, rec := range uc.Databases {
		m := cmdutil.YAMLToMap(rec)
		x := m.([]interface{})
		for ix := range x {
			for _, vals := range x[ix].(map[string]interface{}) {
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
	return uc.UnmarshalConfig(filename, env)
}
