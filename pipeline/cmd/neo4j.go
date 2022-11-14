package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/cloudprivacylabs/lpg"
	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/lsa/pkg/ls"
	"github.com/cloudprivacylabs/opencypher"
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
	session       *neo.Session
	cfg           neo.Config
	cachedDBGraph map[string]*neo.DBGraph
	Neo4jStep
	// CacheByID keeps the ID that uniquely identifies a graph
	CacheByID string `json:"cacheBy" yaml:"cacheBy"`

	mu sync.Mutex
}

func (s *MergeGraphStep) getIDToCache(g *lpg.Graph) string {
	if len(s.CacheByID) == 0 {
		return ""
	}
	idq := fmt.Sprintf("match (n {`%s`:$id}) return n.`%s` as id", ls.SchemaNodeIDTerm, ls.EntityIDTerm)
	ectx := opencypher.NewEvalContext(g)
	ectx.SetParameter("$id", opencypher.ValueOf(s.CacheByID))
	r, err := opencypher.ParseAndEvaluate(idq, ectx)
	if err != nil {
		panic(err)
	}
	rs := r.Get().(opencypher.ResultSet)
	if len(rs.Rows) == 1 {
		v := rs.Rows[0]["id"].Get()
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
	return ""
}

func (s *MergeGraphStep) createNodesEdges(ctx *ls.Context, tx neo4j.Transaction, dbGraph *neo.DBGraph, delta []neo.Delta) error {
	// Create nodes
	createNodeDeltas := neo.SelectDelta(delta, func(d neo.Delta) bool {
		_, ok := d.(neo.CreateNodeDelta)
		return ok
	})
	nodes := make([]*lpg.Node, 0, len(createNodeDeltas))
	for _, x := range createNodeDeltas {
		nodes = append(nodes, x.(neo.CreateNodeDelta).DBNode)
	}
	if len(nodes) > 0 {
		if err := neo.CreateNodesUnwind(ctx, nodes, dbGraph.NodeIds, s.cfg)(tx); err != nil {
			return err
		}
	}

	createEdgeDeltas := neo.SelectDelta(delta, func(d neo.Delta) bool {
		_, ok := d.(neo.CreateEdgeDelta)
		return ok
	})
	edges := make([]*lpg.Edge, 0, len(createEdgeDeltas))
	for _, c := range createEdgeDeltas {
		edges = append(edges, c.(neo.CreateEdgeDelta).DBEdge)
	}
	if len(edges) > 0 {
		if err := neo.CreateEdgesUnwind(ctx, edges, dbGraph.NodeIds, s.cfg)(tx); err != nil {
			return err
		}
	}

	updateDeltas := neo.SelectDelta(delta, func(d neo.Delta) bool {
		if _, ok := d.(neo.CreateNodeDelta); ok {
			return false
		}
		if _, ok := d.(neo.CreateEdgeDelta); ok {
			return false
		}
		return true
	})

	for _, c := range updateDeltas {
		if err := c.Run(tx, dbGraph.NodeIds, dbGraph.EdgeIds, s.cfg); err != nil {
			return err
		}
	}

	return nil
}

func (s *MergeGraphStep) Run(pctx *pipeline.PipelineContext) error {
	if s.session == nil {
		s.session = s.getSession()
		s.cfg = s.getConfig()
		s.cachedDBGraph = make(map[string]*neo.DBGraph)
	}

	merge := func(tx neo4j.Transaction, graph *lpg.Graph, useCache bool) (*neo.DBGraph, []neo.Delta, error) {
		var dbGraph *neo.DBGraph
		if useCache {
			idc := s.getIDToCache(graph)
			if len(idc) > 0 {
				var ok bool
				dbGraph, ok = s.cachedDBGraph[idc]
				// Remember only one graph
				if !ok {
					s.cachedDBGraph = make(map[string]*neo.DBGraph)
				}
			}
		}

		if dbGraph == nil {
			var err error
			dbGraph, err = s.session.LoadDBGraph(tx, graph, s.cfg)
			if err != nil {
				return nil, nil, err
			}
		}
		pctx.Context.GetLogger().Debug(map[string]interface{}{"Loaded db graph with nodes": dbGraph.G.GetNodes().MaxSize()})
		delta, err := neo.Merge(graph, dbGraph, s.cfg)
		if err != nil {
			return nil, nil, err
		}
		return dbGraph, delta, nil
	}

	doTx := func(tx neo4j.Transaction, delta []neo.Delta, dbGraph *neo.DBGraph, useCache bool) error {
		pctx.Context.GetLogger().Debug(map[string]interface{}{"Merge delta": len(delta)})
		if err := s.createNodesEdges(pctx.Context, tx, dbGraph, delta); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			pctx.ErrorLogger(pctx, err)
			return err
		}
		if useCache {
			idc := s.getIDToCache(dbGraph.G)
			if len(idc) > 0 {
				s.cachedDBGraph = map[string]*neo.DBGraph{idc: dbGraph}
			}
		}
		return nil
	}
	transaction, err := s.session.BeginTransaction()
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	dbGraph, delta, err := merge(transaction, pctx.Graph, true)
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	if err := doTx(transaction, delta, dbGraph, true); err != nil {
		transaction.Rollback()
		pctx.ErrorLogger(pctx, err)
		return err
	}
	return pctx.Next()
}

func init() {
	pipeline.RegisterPipelineStep("neo4j/save", func() pipeline.Step { return &SaveGraphStep{} })
	pipeline.RegisterPipelineStep("neo4j/merge", func() pipeline.Step { return &MergeGraphStep{} })
}
