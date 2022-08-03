package neo4j

import (
	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/opencypher/graph"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type PlSession struct {
	S         *neo.Session
	Tx        neo4j.Transaction
	Cfg       neo.Config
	BatchSize int
}

// pipeline
func (ps PlSession) Run(pl *pipeline.PipelineContext) error {
	_, err := neo.SaveGraph(ps.S, ps.Tx, pl.GetGraphRW(), func(graph.Node) bool { return true }, ps.Cfg, ps.BatchSize)
	if err != nil {
		return err
	}
	_, err = pl.NextInput()
	if err != nil {
		return err
	}
	return nil
}
