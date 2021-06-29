package matrix_benchmarks

import (
	"fmt"
	"strings"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
	"github.com/openshift-psap/ci-dashboard/pkg/artifacts"
)

func FetchGPUBurnLogs(test_matrix *v1.MatrixSpec, test_result *v1.TestResult, build_id string) (string, error) {
	test_spec := test_result.TestSpec
	for _, step_name := range test_result.ToolboxSteps {
		if ! strings.Contains(step_name, "_run_gpu_burn") {
			continue
		}
		html_gpu_burn_files, err := artifacts.FetchTestStepResult(test_matrix, test_spec, build_id, fmt.Sprintf("artifacts/%s/", step_name), artifacts.TypeHtml)
		if err != nil {
			return "", err
		}

		gpu_burn_files, err := artifacts.ListFilesInDirectory(html_gpu_burn_files.Html, false, true)
		if err != nil {
			return "", err
		}
		log.Debugf("==> %s", gpu_burn_files)
		for _, filename := range gpu_burn_files {
			if !(strings.HasPrefix(filename, "gpu_burn.") && strings.HasSuffix(filename, ".log")) {
				continue
			}
			gpu_burn_logs, err := artifacts.FetchTestStepResult(test_matrix, test_spec, build_id, fmt.Sprintf("artifacts/%s/%s", step_name, filename), artifacts.TypeBytes)
			if err != nil {
				return "", err
			}
			return string(gpu_burn_logs.Bytes), nil
		}
	}

	return "", nil
}
