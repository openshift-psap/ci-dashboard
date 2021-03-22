package daily_matrix

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
	DefaultOutputFile = "output/gpu-operator_daily-matrix.html"
	DefaultTemplateFile = "templates/daily_matrix.tmpl.html"
)

var log = logrus.New()

func GetLogger() *logrus.Logger {
	return log
}

type Flags struct {
	ConfigFile string
	OutputFile string
	TemplateFile string
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
	}

	return &daily_matrix
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

func populateTestFromFinished(test *v1.TestResult, test_finished artifacts.ArtifactResult) error {
	test.Passed = test_finished.Json["passed"].(bool)
	test.Result = test_finished.Json["result"].(string)
	ts := test_finished.Json["timestamp"].(float64)
	test.FinishDate = time.Unix(int64(ts), 0).Format("2006-01-02 15:04")

	return nil
}

func populateTestFromStepFinished(test *v1.TestResult, step_test_finished artifacts.ArtifactResult) error {
	test.StepPassed = step_test_finished.Json["passed"].(bool)
	test.StepResult = step_test_finished.Json["result"].(string)
	return nil
}

func populateTestMatrices(matricesSpec *v1.MatricesSpec) error {
	for matrix_name, test_matrix := range matricesSpec.Matrices {
		log.Printf("* %s: %s\n", matrix_name, test_matrix.Description)
		for test_group, tests := range test_matrix.Tests {
			for test_idx := range tests {
				test := &tests[test_idx]

				test.ProwName = fmt.Sprintf("%s-%s-%s", test_matrix.ProwConfig, test.Branch, test.TestName)

				fmt.Printf(" - %s\n", test.ProwName)
				test_build_id, test_finished, err := artifacts.FetchLastTestResult(test_matrix, matrix_name, *test,
					"finished.json", artifacts.TypeJson)
				if err != nil {
					return err
				}
				test.TestSpec = test

				test.TestGroup = test_group
				test.BuildId = test_build_id
				if err = populateTestFromFinished(&test.TestResult, test_finished); err != nil {
					log.Warningf("Failed to get the last results of test %s/%s: %v", test.ProwName, test_build_id, err)
				}
				old_test_build_ids, old_tests, err := artifacts.FetchLastNTestResults(test_matrix, matrix_name, test.ProwName, matricesSpec.NbTestHistory,
					"finished.json", artifacts.TypeJson)
				if err != nil {
					return err
				}
				for _, old_test_build_id := range old_test_build_ids {
					old_test_finished := old_tests[old_test_build_id]
					old_test := v1.TestResult{TestSpec: test}
					old_test.BuildId = old_test_build_id
					test.OldTests = append(test.OldTests, &old_test)

					if err = populateTestFromFinished(&old_test, old_test_finished); err != nil {
						log.Warningf("Failed to store the last results of test %s/%s: %v", test.ProwName, old_test_build_id, err)
						continue
					}
					if old_test.Passed {
						continue
					}
					step_test_finished, err := artifacts.FetchTestStepResult(test_matrix, old_test, "finished.json", artifacts.TypeJson)
					if (err != nil) {
						log.Warningf("Failed to fetch the results of test step %s/%s: %v", test.ProwName, old_test_build_id, err)
						continue
					}
					if err = populateTestFromStepFinished(&old_test, step_test_finished); err != nil {
						log.Warningf("Failed to store the results of test step %s/%s: %v", test.ProwName, old_test_build_id, err)
					}
				}
			}
		}
	}

	return nil
}

func daily_matrixWrapper(c *cli.Context, f *Flags) error {
	matricesSpec, err := config.ParseMatricesConfigFile(f.ConfigFile)
	if err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	if err = populateTestMatrices(matricesSpec); err != nil {
		return fmt.Errorf("error fetching the matrix results: %v", err)
	}

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
