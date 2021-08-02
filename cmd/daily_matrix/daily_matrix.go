package daily_matrix

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/openshift-psap/ci-dashboard/pkg/config"
	"github.com/openshift-psap/ci-dashboard/pkg/populate"
	matrix_tpl "github.com/openshift-psap/ci-dashboard/pkg/template/matrix"

)

const (
	DefaultConfigFile  = "examples/gpu-operator.yml"
	DefaultOutputFile = "output/gpu-operator_daily-matrix.html"
	DefaultTemplateFile = "templates/daily_matrix.tmpl.html"
	DefaultTestHistory = -1
)

var log = logrus.New()

func GetLogger() *logrus.Logger {
	return log
}

type Flags struct {
	ConfigFile string
	OutputFile string
	TemplateFile string
	TestHistory int
}

type Context struct {
	*cli.Context
	Flags *Flags
}

func BuildCommand() *cli.Command {
	// Create a flags struct to hold our flags
	daily_matrixFlags := Flags{}

	// Create the 'daily_matrix' command
	daily_matrix := cli.Command{}
	daily_matrix.Name = "daily_matrix"
	daily_matrix.Usage = "Generate a daily test matrix from Prow results"
	daily_matrix.Action = func(c *cli.Context) error {
		return daily_matrixWrapper(c, &daily_matrixFlags)
	}

	// Setup the flags for this command
	daily_matrix.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config-file",
			Aliases:     []string{"c"},
			Usage:       "Configuration file to use for fetching the Prow results",
			Destination: &daily_matrixFlags.ConfigFile,
			Value:       DefaultConfigFile,
			EnvVars:     []string{"CI_DASHBOARD_DAILYMATRIX_CONFIG_FILE"},
		},
		&cli.StringFlag{
			Name:        "output-file",
			Aliases:     []string{"o"},
			Usage:       "Output file where the generated matrix will be stored",
			Destination: &daily_matrixFlags.OutputFile,
			Value:       DefaultOutputFile,
			EnvVars:     []string{"CI_DASHBOARD_DAILYMATRIX_OUTPUT_FILE"},
		},
		&cli.StringFlag{
			Name:        "template",
			Aliases:     []string{"t"},
			Usage:       "Template file from which the matrix will be generated",
			Destination: &daily_matrixFlags.TemplateFile,
			Value:       DefaultTemplateFile,
			EnvVars:     []string{"CI_DASHBOARD_DAILYMATRIX_TEMPLATE_FILE"},
		},
		&cli.IntFlag{
			Name:        "test-history",
			Aliases:     []string{"th"},
			Usage:       "Number of tests to fetch",
			Destination: &daily_matrixFlags.TestHistory,
			Value:       DefaultTestHistory,
			EnvVars:     []string{"CI_DASHBOARD_DAILYMATRIX_TEST_HISTORY"},
		},
	}

	return &daily_matrix
}

func saveGeneratedHtml(generated_html []byte, f *Flags) error {
	output_dir, err := filepath.Abs(filepath.Dir(f.OutputFile))
    if err != nil {
		return fmt.Errorf("Failed to get output directory for %s: %v", f.OutputFile, err)
    }

	err = os.MkdirAll(output_dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Failed to create output directory %s: %v", output_dir, err)
    }

	err = ioutil.WriteFile(f.OutputFile, generated_html, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write into output file at %s: %v", f.OutputFile, err)
	}

	return nil
}

func daily_matrixWrapper(c *cli.Context, f *Flags) error {
	matricesSpec, err := config.ParseMatricesConfigFile(f.ConfigFile)
	if err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	if err = populate.PopulateTestMatrices(matricesSpec, f.TestHistory); err != nil {
		return fmt.Errorf("error fetching the matrix results: %v", err)
	}

	populate.PopulateTestStepLogs(matricesSpec)

	currentTime := time.Now()
	generation_date := currentTime.Format("2006-01-02 15h04")

	generated_html, err := matrix_tpl.Generate(f.TemplateFile, matricesSpec, generation_date)
	if err != nil {
		return fmt.Errorf("error generating the matrix page from the template: %v", err)
	}

	if err = saveGeneratedHtml(generated_html, f); err != nil {
		return fmt.Errorf("error saving the generated matrix page: %v", err)
	}

	log.Infof("Daily test matrix saved into '%s'", f.OutputFile)

	return nil
}
