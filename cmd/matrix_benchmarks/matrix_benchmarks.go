package matrix_benchmarks

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
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

	saveInfo := func(dest_dir, fname string, content []byte) error {
		dest_fname := dest_dir + "/" + fname
		err = ioutil.WriteFile(dest_fname, content, 0644)
		if err != nil {
			return fmt.Errorf("Failed to write into output file at %s: %v", dest_fname, err)
		}

		return nil
	}

	saveInfoInt := func(dest_dir, fname string, value int) error {
		return saveInfo(dest_dir, fname, []byte(fmt.Sprintf("%d\n", value)))
	}

	saveSettings := func(benchmark_name string, test_result *v1.TestResult, exit_code int) (string, error) {
		dest_dir := fmt.Sprintf("%s/%s/%s/%s/", f.OutputDir, test_result.TestSpec.ProwName, test_result.FinishDate, benchmark_name)

		if err := os.MkdirAll(dest_dir, os.ModePerm); err != nil {
			return "", fmt.Errorf("Failed to create output directory %s: %v", f.OutputDir, err)
		}

		if err := saveInfoInt(dest_dir, "exit_code", exit_code); err != nil {
			return "", err
		}

		settings := "expe=nightly\n"
		settings += "benchmark=" + benchmark_name + "\n"
		settings += "operator-version=" + test_result.TestSpec.OperatorVersion + "\n"
		settings += "openshift-version=" + test_result.TestSpec.Branch + "\n"
		settings += "instance-type=g4dn.xlarge\n"
		settings += "@finish-date=" + test_result.FinishDate + "\n"

		if err := saveInfo(dest_dir, "settings", []byte(settings)); err != nil {
			return "", err
		}

		return dest_dir, nil
	}

	var processGPUBurnLogs = func(test_matrix *v1.MatrixSpec, test_result *v1.TestResult) error {
		gpu_burn_logs, err := FetchGPUBurnLogs(test_matrix, test_result, test_result.BuildId)

		if err != nil {
			log.Warningf("Failed to fetch the GPU burn logs of the test %s/%s: %v", test_result.TestSpec.ProwName, test_result.BuildId, err)
			return nil
		}

		if gpu_burn_logs == ""{
			log.Warningf("Could not find GPU burn logs for the test %s/%s", test_result.TestSpec.ProwName, test_result.BuildId)
			return nil
		}

		exit_code := 0
		dest_dir, err := saveSettings("gpu-burn", test_result, exit_code)
		if err != nil {
			return err
		}

		if err = saveInfo(dest_dir, "pod.log", []byte(gpu_burn_logs)); err != nil {
			return err
		}

		return nil
	}

	var processSteps = func(test_matrix *v1.MatrixSpec, test_result *v1.TestResult) error {

		exit_code := 0
		dest_dir, err := saveSettings("test-properties", test_result, exit_code)
		if err != nil {
			return err
		}

		if err = saveInfoInt(dest_dir, "step_count", len(test_result.ToolboxStepsResults)); err != nil {
			return err
		}

		test_passed := 0
		if test_result.Passed {
			test_passed = 1
		}

		if err = saveInfoInt(dest_dir, "test_passed", test_passed); err != nil {
			return err
		}

		if err = saveInfoInt(dest_dir, "ansible_tasks_ok", test_result.Ok); err != nil {
			return err
		}

		return nil
	}

	populate.PopulateTestStepLogs(matrices_spec)

	err = populate.TraverseAllTestResults(matrices_spec, func(test_matrix *v1.MatrixSpec, test_result *v1.TestResult) error {
		if err := processGPUBurnLogs(test_matrix, test_result); err != nil {
			return err
		}

		if err := processSteps(test_matrix, test_result); err != nil {
			return err
		}

		return nil
	})
	return err
}
