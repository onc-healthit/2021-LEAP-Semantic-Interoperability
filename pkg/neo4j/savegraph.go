package neo4j

import (
	"log"

	neo "github.com/cloudprivacylabs/lsa-neo4j"
	"github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/opencypher/graph"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
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

	pipeline.RegisterPipelineStep("saveGraph/neo4j", func() pipeline.Step {
		return Step{
			pls:       step.pls,
			cfg:       step.cfg,
			CfgFile:   step.CfgFile,
			User:      step.User,
			Password:  step.Password,
			URI:       step.URI,
			Database:  step.Database,
			BatchSize: step.BatchSize,
			Realm:     step.Realm,
		}
	})
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

func (s Step) Run(pl *pipeline.PipelineContext) error {
	if s.pls.session == nil {
		initPlSession(&s)
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
