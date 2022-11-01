package cmd

import (
	"time"

	"github.com/cloudprivacylabs/lpg"
	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/drone/envsubst"
	_ "github.com/joho/godotenv/autoload"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Neo4jStep struct {
	CfgFile   string `json:"cfg" yaml:"cfg"`
	Realm     string `json:"realm" yaml:"realm"`
	Database  string `json:"db" yaml:"db"`
	BatchSize int    `json:"batchSize" yaml:"batchSize"`
	User      string `json:"user" yaml:"user"`
	Pwd       string `json:"pwd" yaml:"pwd"`
	URI       string `json:"uri" yaml:"uri"`
}

func (step Neo4jStep) getSession() *neo.Session {
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
		panic(err)
	}
	db, err := envsubst.EvalEnv(step.Database)
	if err != nil {
		panic(err)
	}
	drv := neo.NewDriver(driver, db)
	return drv.NewSession()
}

func (step Neo4jStep) getConfig() neo.Config {
	var cfg neo.Config
	err := cmdutil.ReadJSONOrYAML(step.CfgFile, &cfg)
	if err != nil {
		panic(err)
	}
	neo.InitNamespaceTrie(&cfg)
	return cfg
}

type SaveGraphStep struct {
	session *neo.Session
	cfg     neo.Config
	Neo4jStep
}

func (s *SaveGraphStep) Run(pctx *pipeline.PipelineContext) error {
	if s.session == nil {
		s.session = s.getSession()
		s.cfg = s.getConfig()
	}
	// begin new transaction
	tx, err := s.session.BeginTransaction()
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	start := time.Now()
	_, err = neo.SaveGraph(pctx.Context, s.session, tx, pctx.Graph, func(*lpg.Node) bool { return true }, s.cfg, s.BatchSize)
	if err != nil {
		tx.Rollback()
		pctx.ErrorLogger(pctx, err)
		return err
	}
	pctx.Context.GetLogger().Info(map[string]interface{}{"time elapsed for graph creation": time.Since(start)})
	if err := tx.Commit(); err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	return nil
}

type MergeGraphStep struct {
	session *neo.Session
	cfg     neo.Config
	Neo4jStep
}

func (s *MergeGraphStep) Run(pctx *pipeline.PipelineContext) error {
	if s.session == nil {
		s.session = s.getSession()
		s.cfg = s.getConfig()
	}
	// begin new transaction
	tx, err := s.session.BeginTransaction()
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}

	start := time.Now()
	dbGraph, nodeIds, edgeIds, err := s.session.LoadDBGraph(tx, pctx.Graph, s.cfg)
	if err != nil {
		tx.Rollback()
		pctx.ErrorLogger(pctx, err)
		return err
	}
	pctx.Context.GetLogger().Debug(map[string]interface{}{"Loaded db graph with nodes": dbGraph.GetNodes().MaxSize()})
	delta, err := neo.Merge(pctx.Graph, dbGraph, nodeIds, edgeIds, s.cfg)
	if err != nil {
		tx.Rollback()
		pctx.ErrorLogger(pctx, err)
		return err
	}
	pctx.Context.GetLogger().Debug(map[string]interface{}{"Merge delta": len(delta)})

	createNodeDeltas := neo.SelectDelta(delta, func(d neo.Delta) bool {
		_, ok := d.(neo.CreateNodeDelta)
		return ok
	})
	for _, c := range createNodeDeltas {
		if err := c.Run(tx, nodeIds, edgeIds, s.cfg); err != nil {
			tx.Rollback()
			pctx.ErrorLogger(pctx, err)
			return err
		}
	}
	updateDeltas := neo.SelectDelta(delta, func(d neo.Delta) bool {
		_, ok := d.(neo.CreateNodeDelta)
		return !ok
	})
	for _, c := range updateDeltas {
		if err := c.Run(tx, nodeIds, edgeIds, s.cfg); err != nil {
			tx.Rollback()
			pctx.ErrorLogger(pctx, err)
			return err
		}
	}
	pctx.Context.GetLogger().Info(map[string]interface{}{"time elapsed for graph merge": time.Since(start)})
	if err := tx.Commit(); err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	return nil
}

func init() {
	pipeline.RegisterPipelineStep("neo4j/save", func() pipeline.Step { return &SaveGraphStep{} })
	pipeline.RegisterPipelineStep("neo4j/merge", func() pipeline.Step { return &MergeGraphStep{} })
}
