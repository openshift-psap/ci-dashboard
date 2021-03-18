package gen_matrix

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	"github.com/openshift-psap/ci-dashboard/pkg/artifacts"
	"github.com/openshift-psap/ci-dashboard/pkg/config"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
	matrix_tpl "github.com/openshift-psap/ci-dashboard/pkg/template/matrix"
)

const (
	DefaultConfigFile  = "examples/gpu-operator.yml"
	DefaultOutputFile = "output/matrix.gen.html"
)

var log = logrus.New()

func GetLogger() *logrus.Logger {
	return log
}

type Flags struct {
	ConfigFile string
	OutputFile string
}

type Context struct {
	*cli.Context
	Flags *Flags
}

func BuildCommand() *cli.Command {
	// Create a flags struct to hold our flags
	gen_matrixFlags := Flags{}

	// Create the 'gen_matrix' command
	gen_matrix := cli.Command{}
	gen_matrix.Name = "gen_matrix"
	gen_matrix.Usage = "Generate the test matrix from Prow results"
	gen_matrix.Action = func(c *cli.Context) error {
		return gen_matrixWrapper(c, &gen_matrixFlags)
	}

	// Setup the flags for this command
	gen_matrix.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config-file",
			Aliases:     []string{"c"},
			Usage:       "Configuration file to use for fetching the Prow results",
			Destination: &gen_matrixFlags.ConfigFile,
			Value:       DefaultConfigFile,
			EnvVars:     []string{"CI_DASHBOARD_GENMATRIX_CONFIG_FILE"},
		},
		&cli.StringFlag{
			Name:        "output-file",
			Aliases:     []string{"o"},
			Usage:       "Output file where the generated matrix will be stored",
			Destination: &gen_matrixFlags.OutputFile,
			Value:       DefaultOutputFile,
			EnvVars:     []string{"CI_DASHBOARD_GENMATRIX_OUTPUT_FILE"},
		},
	}

	return &gen_matrix
}

func saveGeneratedHtml(generated_html []byte, f *Flags) error {
	output_dir, err := filepath.Abs(filepath.Dir(f.OutputFile))
    if err != nil {
		return fmt.Errorf("Failed to get cache directory for %s: %v", f.OutputFile, err)
    }

	err = os.MkdirAll(output_dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Failed to create cache directory %s: %v", output_dir, err)
    }

	err = ioutil.WriteFile(f.OutputFile, generated_html, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write into cache file at %s: %v", f.OutputFile, err)
	}

	return nil
}

func populateTestMatrices(matricesSpec *v1.MatricesSpec) error {
	for matrix_name, test_matrix := range matricesSpec.Matrices {
		log.Printf("* %s: %s\n", matrix_name, test_matrix.Description)
		for test_group, tests := range test_matrix.Tests {
			for test_idx := range tests {
				test := &tests[test_idx]
				fmt.Printf(" - %s\n", test.ProwName)
				test_build_id, test_finished, err := artifacts.FetchLastTestResult(test_matrix, matrix_name, *test,
					"finished.json", artifacts.TypeJson)
				if err != nil {
					return err
				}
				test.TestGroup = test_group
				test.BuildId = test_build_id
				test.Passed = test_finished.Json["passed"].(bool)
				test.Result = test_finished.Json["result"].(string)
				ts := test_finished.Json["timestamp"].(float64)
				test.FinishDate = time.Unix(int64(ts), 0).Format("2006-01-02 15:04")
			}
		}
	}

	return nil
}

func gen_matrixWrapper(c *cli.Context, f *Flags) error {
	matricesSpec, err := config.ParseMatricesConfigFile(f.ConfigFile)
	if err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	if err = populateTestMatrices(matricesSpec); err != nil {
		return fmt.Errorf("error fetching the matrix results: %v", err)
	}

	generated_html, err := matrix_tpl.Generate(matricesSpec)
	if err != nil {
		return fmt.Errorf("error generating the matrix page from the template: %v", err)
	}

	if err = saveGeneratedHtml(generated_html, f); err != nil {
		return fmt.Errorf("error saving the generated matrix page: %v", err)
	}

	log.Infof("Test matrix saved into '%s'", f.OutputFile)


	return nil
}
