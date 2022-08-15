package cmd

import (
	"log"

	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/opencypher/graph"
	_ "github.com/joho/godotenv/autoload"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type plSession struct {
	session *neo.Session
}

func initPlSession(step *Step) {
	user := getEnvVariable("user")
	password := getEnvVariable("pwd")
	uri := getEnvVariable("uri")
	var auth neo4j.AuthToken
	if len(user) > 0 {
		auth = neo4j.BasicAuth(user, password, step.Realm)
	} else {
		auth = neo4j.NoAuth()
	}
	driver, err := neo4j.NewDriver(uri, auth)
	if err != nil {
		log.Fatal(err)
	}
	drv := neo.NewDriver(driver, step.Database)
	step.pls = &plSession{session: drv.NewSession()}
}

type Step struct {
	pls       *plSession
	cfg       neo.Config
	CfgFile   string `json:"cfg" yaml:"cfg"`
	Realm     string `json:"realm" yaml:"realm"`
	Database  string `json:"db" yaml:"db"`
	BatchSize int    `json:"batchSize" yaml:"batchSize"`
}

func (s *Step) Run(pl *pipeline.PipelineContext) error {
	if s.pls == nil {
		initPlSession(s)
	}
	defer s.pls.session.Close()
	// begin new transaction
	tx, err := s.pls.session.BeginTransaction()
	if err != nil {
		stdLogger.Println(err)
		pl.Context.GetLogger().Error(map[string]interface{}{"error starting transaction": err})
		return err
	}
	err = cmdutil.ReadJSONOrYAML(s.CfgFile, &s.cfg)
	if err != nil {
		return err
	}
	neo.InitNamespaceTrie(&s.cfg)
	_, err = neo.SaveGraph(s.pls.session, tx, pl.GetGraphRW(), func(graph.Node) bool { return true }, s.cfg, s.BatchSize)
	if err != nil {
		tx.Rollback()
		stdLogger.Println(err)
		return err
	}
	tx.Commit()
	_, err = pl.NextInput()
	if err != nil {
		stdLogger.Println(err)
		return err
	}
	return nil
}

func init() {
	pipeline.RegisterPipelineStep("saveGraph/neo4j", func() pipeline.Step { return &Step{} })
}
