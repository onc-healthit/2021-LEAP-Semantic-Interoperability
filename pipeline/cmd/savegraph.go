package cmd

import (
	"log"

	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/opencypher/graph"
	"github.com/drone/envsubst"
	_ "github.com/joho/godotenv/autoload"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type plSession struct {
	session *neo.Session
}

func initPlSession(step *Step) {
	user, err := envsubst.EvalEnv(step.User)
	if err != nil {
		panic(err)
	}
	password, err := envsubst.EvalEnv(step.Pwd)
	if err != nil {
		panic(err)
	}
	uri, err := envsubst.EvalEnv(step.URI)
	if err != nil {
		panic(err)
	}
	realm, err := envsubst.EvalEnv(step.Realm)
	if err != nil {
		panic(err)
	}
	var auth neo4j.AuthToken
	if len(user) > 0 {
		auth = neo4j.BasicAuth(user, password, realm)
	} else {
		auth = neo4j.NoAuth()
	}
	driver, err := neo4j.NewDriver(uri, auth)
	if err != nil {
		log.Fatal(err)
	}
	db, err := envsubst.EvalEnv(step.Database)
	if err != nil {
		panic(err)
	}
	drv := neo.NewDriver(driver, db)
	step.pls = &plSession{session: drv.NewSession()}
}

type Step struct {
	pls       *plSession
	cfg       neo.Config
	CfgFile   string `json:"cfg" yaml:"cfg"`
	Realm     string `json:"realm" yaml:"realm"`
	Database  string `json:"db" yaml:"db"`
	BatchSize int    `json:"batchSize" yaml:"batchSize"`
	User      string `json:"user" yaml:"user"`
	Pwd       string `json:"pwd" yaml:"pwd"`
	URI       string `json:"uri" yaml:"uri"`
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
