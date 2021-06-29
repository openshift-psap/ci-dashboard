package matrix_benchmarks

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	"github.com/openshift-psap/ci-dashboard/pkg/config"
	"github.com/openshift-psap/ci-dashboard/pkg/populate"
)

const (
	DefaultConfigFile  = "examples/gpu-operator.yml"
	DefaultOutputDir = "output/matrix_benchmarking/"
	DefaultTestHistory = -1
)

var log = logrus.New()

func GetLogger() *logrus.Logger {
	return log
}

type Flags struct {
	ConfigFile string
	OutputDir string
	TestHistory int
}

type Context struct {
	*cli.Context
	Flags *Flags
}

func BuildCommand() *cli.Command {
	// Create a flags struct to hold our flags
	matrix_benchFlags := Flags{}

	// Create the 'matrix_bench' command
	matrix_bench := cli.Command{}
	matrix_bench.Name = "matrix_benchmarks"
	matrix_bench.Usage = "Generate MatrixBenchmarking results from Prow test artifacts"
	matrix_bench.Action = func(c *cli.Context) error {
		return matrix_benchWrapper(c, &matrix_benchFlags)
	}

	// Setup the flags for this command
	matrix_bench.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config-file",
			Aliases:     []string{"c"},
			Usage:       "Configuration file to use for fetching the Prow results",
			Destination: &matrix_benchFlags.ConfigFile,
			Value:       DefaultConfigFile,
			EnvVars:     []string{"CI_DASHBOARD_MATRIX_BENCH_CONFIG_FILE"},
		},
		&cli.StringFlag{
			Name:        "output-dir",
			Aliases:     []string{"o"},
			Usage:       "Output directory where the generated MatrixBenchmarking results will be stored",
			Destination: &matrix_benchFlags.OutputDir,
			Value:       DefaultOutputDir,
			EnvVars:     []string{"CI_DASHBOARD_MATRIX_BENCH_OUTPUT_FILE"},
		},
		&cli.IntFlag{
			Name:        "test-history",
			Aliases:     []string{"th"},
			Usage:       "Number of tests to fetch",
			Destination: &matrix_benchFlags.TestHistory,
			Value:       DefaultTestHistory,
			EnvVars:     []string{"CI_DASHBOARD_MATRIX_BENCH_TEST_HISTORY"},
		},
	}

	return &matrix_bench
}


func matrix_benchWrapper(c *cli.Context, f *Flags) error {
	matrices_spec, err := config.ParseMatricesConfigFile(f.ConfigFile)
	if err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	if err = populate.PopulateTestMatrices(matrices_spec, f.TestHistory); err != nil {
		return fmt.Errorf("error fetching the matrix results: %v", err)
	}

	//current_time := time.Now()
	//generation_date := current_time.Format("2006-01-02 15h04")

	for _, test_matrix := range matrices_spec.Matrices {
		for _, tests := range test_matrix.Tests {
			for test_idx := range tests {
				for _, test_result := range tests[test_idx].OldTests {
					FetchGPUBurnLogs(&test_matrix, test_result, test_result.BuildId)

					gpu_burn_logs, err := FetchGPUBurnLogs(&test_matrix, test_result, test_result.BuildId)

					if err != nil {
						log.Warningf("Failed to fetch the GPU burn logs of the test %s/%s: %v", test_result.TestSpec.ProwName, test_result.BuildId, err)
					}

					if gpu_burn_logs == ""{
						log.Warningf("Could not find GPU burn logs for the test %s/%s", test_result.TestSpec.ProwName, test_result.BuildId)
						continue
					}
					dest_dir := fmt.Sprintf("%s/%s/%s/gpu-burn/", f.OutputDir, test_result.TestSpec.ProwName, test_result.BuildId)
					err = os.MkdirAll(dest_dir, os.ModePerm)
					if err != nil {
						return fmt.Errorf("Failed to create output directory %s: %v", f.OutputDir, err)
					}

					dest_fname := dest_dir + "/pod.log"
					err = ioutil.WriteFile(dest_fname, []byte(gpu_burn_logs), 0644)
					if err != nil {
						return fmt.Errorf("Failed to write into output file at %s: %v", dest_fname, err)
					}
				}
			}
		}
	}

	return nil
}
