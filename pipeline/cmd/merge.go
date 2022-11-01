package cmd

import (
	"time"

	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/lsa/pkg/ls"
)

type MergeStep struct {
	s *Step
}

func (ms *MergeStep) Run(pctx *pipeline.PipelineContext) error {
	if ms.s.pls == nil {
		initPlSession(pctx, ms.s)
	}
	defer ms.s.pls.session.Close()
	// begin new transaction
	tx, err := ms.s.pls.session.BeginTransaction()
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	start := time.Now()
	dbGraph, ids, edgeIds, err := ms.s.pls.session.LoadDBGraph(tx, pctx.Graph, ms.s.cfg)
	if err != nil {
		return err
	}

	_, ops, err := neo.Merge(pctx.Graph, dbGraph, ids, edgeIds, ms.s.cfg)
	if err != nil {
		return err
	}
	for ix := 0; ix < len(ops.Ops); ix += ms.s.BatchSize {
		qs := neo.OperationQueue{Ops: ops.Ops[ix:ms.s.BatchSize]}
		if err = neo.RunOperations(ls.DefaultContext(), ms.s.pls.session, tx, qs); err != nil {
			return err
		}
	}
	// _, err = neo.SaveGraph(ls.DefaultContext(), ms.s.pls.session, tx, pctx.Graph, func(*lpg.Node) bool { return true }, ms.s.cfg, ms.s.BatchSize)
	// if err != nil {
	// 	tx.Rollback()
	// 	pctx.ErrorLogger(pctx, err)
	// 	return err
	// }
	pctx.Context.GetLogger().Info(map[string]interface{}{"time elapsed for graph merge": time.Since(start)})
	if err := tx.Commit(); err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	return nil
}

func init() {
	pipeline.RegisterPipelineStep("neo4j/mergeGraph", func() pipeline.Step { return &MergeStep{} })
}
