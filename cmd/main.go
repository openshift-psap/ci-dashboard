package main

import (
	"os"

	"github.com/openshift-psap/ci-dashboard/cmd/daily_matrix"
	"github.com/openshift-psap/ci-dashboard/pkg/artifacts"
	"github.com/openshift-psap/ci-dashboard/pkg/config"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

type Flags struct {
	Debug bool
}

func main() {
	// Create a flags struct to hold our flags
	flags := Flags{}

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.UseShortOptionHandling = true
	app.EnableBashCompletion = true
	app.Usage = "Manage the GPU Operator Prow CI dashboard"
	app.Version = "0.0.0"

	// Setup the flags for this command
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "debug",
			Aliases:     []string{"d"},
			Usage:       "Enable debug-level logging",
			Destination: &flags.Debug,
			EnvVars:     []string{"CI_DASHBOARD_DEBUG"},
		},
	}

	app.Commands = []*cli.Command{
		daily_matrix.BuildCommand(),
	}

	// Set log-level for all subcommands
	app.Before = func(app *cli.Context) error {
		logLevel := log.InfoLevel
		if flags.Debug {
			logLevel = log.DebugLevel
		}
		daily_matrixLog := daily_matrix.GetLogger()
		daily_matrixLog.SetLevel(logLevel)

		configLog := config.GetLogger()
		configLog.SetLevel(logLevel)

		artifactsLog := artifacts.GetLogger()
		artifactsLog.SetLevel(logLevel)
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
