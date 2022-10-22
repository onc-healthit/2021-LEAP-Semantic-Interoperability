package cmd

import (
	"time"

	"github.com/cloudprivacylabs/lpg"
	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/lsa/pkg/ls"
	_ "github.com/joho/godotenv/autoload"
)

type SaveStep struct {
	s *Step
}

func (s SaveStep) Run(pctx *pipeline.PipelineContext) error {
	if s.s.pls == nil {
		initPlSession(pctx, s.s)
	}
	defer s.s.pls.session.Close()
	// begin new transaction
	tx, err := s.s.pls.session.BeginTransaction()
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	start := time.Now()
	_, err = neo.SaveGraph(ls.DefaultContext(), s.s.pls.session, tx, pctx.Graph, func(*lpg.Node) bool { return true }, s.s.cfg, s.s.BatchSize)
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

func init() {
	pipeline.RegisterPipelineStep("neo4j/saveGraph", func() pipeline.Step { return &SaveStep{} })
}
