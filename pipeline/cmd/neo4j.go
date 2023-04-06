package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-metrics"

	"github.com/cloudprivacylabs/lpg/v2"
	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/lsa/pkg/csv"
	"github.com/cloudprivacylabs/lsa/pkg/ls"
	"github.com/cloudprivacylabs/opencypher"
	"github.com/drone/envsubst"
	_ "github.com/joho/godotenv/autoload"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
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
	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		panic(err)
	}
	db, err := envsubst.EvalEnv(step.Database)
	if err != nil {
		panic(err)
	}
	drv := neo.NewDriver(driver, db)
	return drv.NewSession(context.Background())
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

// type SaveGraphStep struct {
// 	session   *neo.Session
// 	cfg       neo.Config
// 	Neo4jStep `yaml:",inline"`
// }

// func (SaveGraphStep) Name() string { return "neo4j/save" }

// func (s *SaveGraphStep) Run(pctx *pipeline.PipelineContext) error {
// 	if s.session == nil {
// 		s.session = s.getSession()
// 		s.cfg = s.getConfig()
// 	}
// 	// begin new transaction
// 	tx, err := s.session.BeginTransaction(pctx.Context)
// 	if err != nil {
// 		pctx.ErrorLogger(pctx, err)
// 		return err
// 	}
// 	start := time.Now()
// 	_, err = neo.SaveGraph(pctx.Context, s.session, tx, pctx.Graph, func(*lpg.Node) bool { return true }, s.cfg, s.BatchSize)
// 	if err != nil {
// 		tx.Rollback()
// 		pctx.ErrorLogger(pctx, err)
// 		return err
// 	}
// 	pctx.Context.GetLogger().Info(map[string]interface{}{"time elapsed for graph creation": time.Since(start)})
// 	if err := tx.Commit(); err != nil {
// 		pctx.ErrorLogger(pctx, err)
// 		return err
// 	}
// 	return nil
// }

type MergeGraphStep struct {
	session       *neo.Session
	CacheByID     string `json:"cacheBy" yaml:"cacheBy"`
	cfg           neo.Config
	Neo4jStep     `yaml:",inline"`
	cachedDBGraph map[string]*neo.DBGraph

	mu sync.Mutex
}

func (MergeGraphStep) Name() string { return "neo4j/merge" }

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

func (s *MergeGraphStep) createNodesEdges(ctx *ls.Context, tx neo4j.ExplicitTransaction, session *neo.Session, dbGraph *neo.DBGraph, delta []neo.Delta) error {
	// Create nodes
	createNodeDeltas := neo.SelectDelta(delta, func(d neo.Delta) bool {
		_, ok := d.(neo.CreateNodeDelta)
		return ok
	})
	nodes := make([]*lpg.Node, 0, len(createNodeDeltas))
	for _, x := range createNodeDeltas {
		node := x.(neo.CreateNodeDelta).DBNode
		nodes = append(nodes, node)
	}
	if len(nodes) > 0 {
		start := time.Now()
		if err := neo.CreateNodesUnwind(ctx, nodes, dbGraph, s.cfg)(tx); err != nil {
			return err
		}
		metrics.MeasureSince([]string{"neo4j.createNodes"}, start)
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
		start := time.Now()
		if err := neo.CreateEdgesUnwind(ctx, session, edges, dbGraph, s.cfg)(tx); err != nil {
			return err
		}
		metrics.MeasureSince([]string{"neo4j.createEdges"}, start)
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

	start := time.Now()
	for _, c := range updateDeltas {
		if err := c.Run(ctx, tx, session, dbGraph.NodeIds, dbGraph.EdgeIds, s.cfg); err != nil {
			return err
		}
	}
	metrics.MeasureSince([]string{"neo4j.update"}, start)
	return nil
}

func (s *MergeGraphStep) Run(pctx *pipeline.PipelineContext) error {
	if s.session == nil {
		s.session = s.getSession()
		s.cfg = s.getConfig()
		s.cachedDBGraph = make(map[string]*neo.DBGraph)
	}

	cache := neo.Neo4jCache{}
	var idToCache string

	merge := func(tx neo4j.ExplicitTransaction, graph *lpg.Graph, useCache bool) (*neo.DBGraph, []neo.Delta, error) {
		var dbGraph *neo.DBGraph
		if useCache {
			idToCache = s.getIDToCache(graph)
			if len(idToCache) > 0 {
				var ok bool
				dbGraph, ok = s.cachedDBGraph[idToCache]
				// Remember only one graph
				if !ok {
					s.cachedDBGraph = make(map[string]*neo.DBGraph)
				}
			}
		}
		if dbGraph == nil {
			var err error
			dbGraph, err = s.session.LoadDBGraph(pctx.Context, tx, graph, s.cfg, &cache)
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

	doTx := func(tx neo4j.ExplicitTransaction, delta []neo.Delta, dbGraph *neo.DBGraph, useCache bool) error {
		pctx.Context.GetLogger().Debug(map[string]interface{}{"Merge delta": len(delta)})
		if err := s.createNodesEdges(pctx.Context, tx, s.session, dbGraph, delta); err != nil {
			return err
		}
		if err := neo.LinkMergedEntities(pctx.Context, tx, s.cfg, delta, dbGraph.NodeIds); err != nil {
			return err
		}
		start := time.Now()
		if err := tx.Commit(pctx.Context); err != nil {
			pctx.ErrorLogger(pctx, err)
			return err
		}
		metrics.MeasureSince([]string{"neo4j.merge.Commit"}, start)

		return nil
	}
	transaction, err := s.session.BeginTransaction(pctx.Context)
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}

	compactGraph, err := neo.Compact(pctx.Graph)
	if err != nil {
		return err
	}

	dbGraph, delta, err := merge(transaction, compactGraph, false) // Caching is broken
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	if err := doTx(transaction, delta, dbGraph, true); err != nil {
		s.cachedDBGraph = make(map[string]*neo.DBGraph)
		transaction.Rollback(pctx.Context)
		pctx.ErrorLogger(pctx, err)
		return err
	}
	if len(idToCache) > 0 {
		s.cachedDBGraph[idToCache] = dbGraph
	}

	return pctx.Next()
}

type LoadGraphStep struct {
	Neo4jStep `yaml:",inline"`
	Query     string   `json:"query" yaml:"query"`
	Entities  []string `json:"entities" yaml:"entities"`

	session *neo.Session
	cfg     neo.Config
}

func (LoadGraphStep) Name() string { return "neo4j/load" }

func (s *LoadGraphStep) Run(pctx *pipeline.PipelineContext) error {
	if s.session == nil {
		s.session = s.getSession()
		s.cfg = s.getConfig()
	}
	transaction, err := s.session.BeginTransaction(pctx.Context)
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	defer transaction.Commit(pctx.Context)
	result, err := transaction.Run(pctx.Context, s.Query, nil)
	if err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	for result.Next(pctx.Context) {
		rec := result.Record()
		// Result must be nodes
		if len(rec.Values) != 1 {
			return fmt.Errorf("Expecting a list of nodes from the query, but got %d outputs", len(rec.Values))
		}
		node, ok := rec.Values[0].(neo4j.Node)
		if !ok {
			return fmt.Errorf("Expecting a node from the query, but got %T", rec.Values[0])
		}
		g := ls.NewDocumentGraph()
		err := s.session.LoadEntityNodes(pctx.Context, transaction, g, []string{node.ElementId}, s.cfg, func(node *lpg.Node) bool {
			if len(s.Entities) == 0 {
				return true
			}
			if _, ok := node.GetProperty(ls.EntitySchemaTerm); !ok {
				return true
			}
			for _, x := range s.Entities {
				if node.HasLabel(x) {
					return true
				}
			}
			return false
		})
		if err != nil {
			pctx.ErrorLogger(pctx, err)
			return err
		}
		pctx.SetGraph(g)
		if err := pctx.Next(); err != nil {
			return err
		}
	}
	if err := result.Err(); err != nil {
		pctx.ErrorLogger(pctx, err)
		return err
	}
	return nil
}

type RunStatementStep struct {
	Neo4jStep  `yaml:",inline"`
	InputData  string   `json:"inputData" yaml:"inputData"`
	Sheet      string   `json:"sheet" yaml:"sheet"`
	HeaderRow  int      `json:"headerRow" yaml:"headerRow"`
	Separator  string   `json:"separator" yaml:"separator"`
	Statements []string `json:"statements" yaml:"statements"`

	session *neo.Session
}

func (s *RunStatementStep) Name() string { return "neo4j/run" }

func (s *RunStatementStep) runStatements(ctx context.Context, env map[string]string) error {
	transaction, err := s.session.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	for _, stmt := range s.Statements {
		stmt := os.Expand(stmt, func(k string) string {
			return env[k]
		})
		fmt.Println(stmt)
		if _, err := transaction.Run(ctx, stmt, map[string]any{}); err != nil {
			transaction.Rollback(ctx)
			return fmt.Errorf("Statement: %s, data: %v, error: %w", stmt, env, err)
		}
	}
	return transaction.Commit(ctx)
}

func (s *RunStatementStep) Run(pctx *pipeline.PipelineContext) error {
	if s.session == nil {
		s.session = s.getSession()
	}
	// If there is input data, open it
	var stream <-chan csv.Row
	if len(s.InputData) > 0 {
		input, err := os.Open(s.InputData)
		if err != nil {
			return err
		}
		defer input.Close()

		if strings.ToLower(filepath.Ext(s.InputData)) == ".csv" {
			stream, err = csv.StreamCSVRows(input, s.Separator, s.HeaderRow)
			if err != nil {
				return err
			}
		} else {
			stream, err = csv.StreamExcelSheetRows(input, s.Sheet, s.HeaderRow)
			if err != nil {
				return err
			}
		}
	}

	if stream != nil {
		for row := range stream {
			if row.Err != nil {
				return row.Err
			}
			env := make(map[string]string)
			n := len(row.Headers)
			if len(row.Row) < n {
				n = len(row.Row)
			}
			for i := 0; i < n; i++ {
				env[row.Headers[i]] = row.Row[i]
			}
			if err := s.runStatements(pctx.Context, env); err != nil {
				return err
			}
		}
	} else {
		if err := s.runStatements(pctx.Context, map[string]string{}); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	//	pipeline.RegisterPipelineStep("neo4j/save", func() pipeline.Step { return &SaveGraphStep{} })
	pipeline.RegisterPipelineStep("neo4j/merge", func() pipeline.Step { return &MergeGraphStep{} })
	pipeline.RegisterPipelineStep("neo4j/load", func() pipeline.Step { return &LoadGraphStep{} })
	pipeline.RegisterPipelineStep("neo4j/run", func() pipeline.Step { return &RunStatementStep{} })
}
