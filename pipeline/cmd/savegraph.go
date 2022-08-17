package cmd

import (
	"log"
	"time"

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

func initPlSession(pl *pipeline.PipelineContext, step *Step) {
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
	err = cmdutil.ReadJSONOrYAML(step.CfgFile, &step.cfg)
	if err != nil {
		pl.ErrorLogger(pl, err)
		panic(err)
	}
	neo.InitNamespaceTrie(&step.cfg)
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
		initPlSession(pl, s)
	}
	defer s.pls.session.Close()
	// begin new transaction
	tx, err := s.pls.session.BeginTransaction()
	if err != nil {
		stdLogger.Println(err)
		pl.ErrorLogger(pl, err)
		return err
	}
	start := time.Now()
	_, err = neo.SaveGraph(s.pls.session, tx, pl.GetGraphRW(), func(graph.Node) bool { return true }, s.cfg, s.BatchSize)
	if err != nil {
		tx.Rollback()
		stdLogger.Println(err)
		pl.ErrorLogger(pl, err)
		return err
	}
	pl.Context.GetLogger().Info(map[string]interface{}{"time elapsed for graph creation": time.Since(start)})
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func init() {
	pipeline.RegisterPipelineStep("saveGraph/neo4j", func() pipeline.Step { return &Step{} })
}
