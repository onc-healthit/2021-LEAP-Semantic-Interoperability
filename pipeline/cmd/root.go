// Copyright 2021 Cloud Privacy Labs, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"

	"github.com/spf13/cobra"

	"github.com/cloudprivacylabs/leap/pkg/input"
	pipeline "github.com/cloudprivacylabs/lsa/layers/cmd/pipeline"
	"github.com/cloudprivacylabs/lsa/pkg/ls"
)

var logger = ls.NewDefaultLogger()

var (
	rootCmd = &cobra.Command{
		Use:   "pipeline pipeline-file files...",
		Short: "LSA data pipeline CLI",
		Args:  cobra.MinimumNArgs(1),
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if f, _ := cmd.Flags().GetString("cpuprofile"); len(f) > 0 {
				file, err := os.Create(f)
				if err != nil {
					panic(err)
				}
				pprof.StartCPUProfile(file)
			}
			if b, _ := cmd.Flags().GetBool("log"); b {
				logger.Level = ls.LogLevelDebug
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, _ []string) {
			if f, _ := cmd.Flags().GetString("cpuprofile"); len(f) > 0 {
				pprof.StopCPUProfile()
			}
		},
		RunE: func(ccmd *cobra.Command, args []string) error {
			pl, err := pipeline.ReadPipeline(args[0])
			if err != nil {
				return err
			}
			ctx := getContext(ccmd)
			inputs, err := input.ReadPaths(ctx, args[1:]...)
			if err != nil {
				return err
			}

			for x := range inputs {
				fmt.Println(x)
			}

			// returns *pipeline.PipelineContext
			_, err = pipeline.Run(ctx, pl, nil, func() (io.ReadCloser, error) {
				ctx.GetLogger().Debug(map[string]interface{}{
					"pipeline": "fetching next input",
				})
				select {
				case entry, ok := <-inputs:
					ctx.GetLogger().Debug(map[string]interface{}{
						"pipeline": entry.Name,
						"ok":       ok,
					})
					if ok {
						return io.NopCloser(entry.Stream), entry.Err
					}
				}
				return nil, nil
			})
			return nil
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().String("cpuprofile", "", "Write cpu profile to file")
	rootCmd.PersistentFlags().Bool("log", false, "Enable logging")
	rootCmd.PersistentFlags().Bool("log.debug", false, "Enable logging at debug level")
	rootCmd.PersistentFlags().Bool("log.info", false, "Enable logging at info level")
}

func getContext(cmd *cobra.Command) *ls.Context {
	l1, _ := cmd.Flags().GetBool("log")
	l2, _ := cmd.Flags().GetBool("log.debug")
	if l1 || l2 {
		logger.Level = ls.LogLevelDebug
	}
	l1, _ = cmd.Flags().GetBool("log.info")
	if l1 {
		logger.Level = ls.LogLevelInfo
	}

	ctx := ls.DefaultContext().SetLogger(logger)
	return ctx
}

func failErr(err error) {
	log.Fatalf(err.Error())
}

func fail(msg string) {
	log.Fatalf(msg)
}
