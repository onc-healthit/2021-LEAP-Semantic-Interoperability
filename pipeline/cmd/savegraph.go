package cmd

import (
	"errors"
	"log"
	"os"

	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/cmdutil"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/lsa/pkg/ls"
	"github.com/cloudprivacylabs/opencypher/graph"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/spf13/cobra"
)

type plSession struct {
	session *neo.Session
}

func initPlSession(step *Step) {
	var auth neo4j.AuthToken
	if len(step.User) > 0 {
		auth = neo4j.BasicAuth(step.User, step.Password, step.Realm)
	} else {
		auth = neo4j.NoAuth()
	}
	driver, err := neo4j.NewDriver(step.URI, auth)
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
	User      string `json:"user" yaml:"user"`
	Password  string `json:"pwd" yaml:"pwd"`
	URI       string `json:"uri" yaml:"uri"`
	Realm     string `json:"realm" yaml:"realm"`
	Database  string `json:"db" yaml:"db"`
	BatchSize int    `json:"batchSize" yaml:"batchSize"`
}

// lsaneo create --user neo4j --pwd password  --uri "neo4j://34.213.163.7:7687" --db neo4j ../examples/test.json --batch 0
func (s *Step) Run(pl *pipeline.PipelineContext) error {
	if s.pls.session == nil {
		initPlSession(s)
	}
	defer s.pls.session.Close()
	// begin new transaction
	tx, err := s.pls.session.BeginTransaction()
	if err != nil {
		return err
	}
	_, err = neo.SaveGraph(s.pls.session, tx, pl.GetGraphRW(), func(graph.Node) bool { return true }, s.cfg, s.BatchSize)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	_, err = pl.NextInput()
	if err != nil {
		return err
	}
	return nil
}

var saveGraphCmd = &cobra.Command{
	Use:   "saveGraph",
	Short: "Commit a graph to the neo4j database",
	Args:  cobra.MaximumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		step := &Step{}
		var cfg neo.Config
		var err error
		if cfgfile, _ := cmd.Flags().GetString("cfg"); len(cfgfile) == 0 {
			err := cmdutil.ReadJSONOrYAML("lsaneo.config.yaml", &cfg)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		} else {
			if err := cmdutil.ReadJSONOrYAML(cfgfile, &cfg); err != nil {
				return err
			}
		}
		neo.InitNamespaceTrie(&cfg)
		step.cfg = cfg
		step.Database, err = cmd.Flags().GetString("db")
		if err != nil {
			return err
		}
		step.User, err = cmd.Flags().GetString("user")
		if err != nil {
			return err
		}
		step.URI, err = cmd.Flags().GetString("uri")
		if err != nil {
			return err
		}
		step.Password, err = cmd.Flags().GetString("pwd")
		if err != nil {
			return err
		}
		step.Realm, err = cmd.Flags().GetString("realm")
		if err != nil {
			return err
		}
		step.BatchSize, err = cmd.Flags().GetInt("batch")
		if err != nil {
			return err
		}
		p := []pipeline.Step{
			step,
		}
		initPlSession(step)
		pctx, err := pipeline.Run(ls.DefaultContext(), p, nil, pipeline.InputsFromFiles([]string{"testdata/testsuitegraph.json"}))
		if err != nil {
			return err
		}
		g, err := cmdutil.ReadGraph([]string{"testdata/testsuitegraph.json"}, nil, "json")
		pctx.Graph = g
		err = step.Run(pctx)
		return err
	},
}

func init() {
	rootCmd.AddCommand(saveGraphCmd)

	saveGraphCmd.PersistentFlags().String("uri", "neo4j://localhost:7687", "DB URI")
	saveGraphCmd.MarkFlagRequired("uri")
	saveGraphCmd.PersistentFlags().String("user", "", "Username")
	saveGraphCmd.PersistentFlags().String("pwd", "", "Password")
	saveGraphCmd.PersistentFlags().String("realm", "", "Realm")
	saveGraphCmd.PersistentFlags().String("db", "", "Database name")
	saveGraphCmd.Flags().String("input", "json", "Input graph format (json, jsonld)")
	saveGraphCmd.Flags().String("cfg", "", "configuration spec for node properties and labels (default: lsaneo.config.yaml)")
	saveGraphCmd.Flags().Int("batch", 0, "batching size for creation of nodes and edges")

	pipeline.RegisterPipelineStep("saveGraph/neo4j", func() pipeline.Step { return &Step{} })
}
