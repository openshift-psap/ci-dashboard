package populate

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/openshift-psap/ci-dashboard/pkg/artifacts"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
)

var log = logrus.New()

func PopulateTestFromFinished(test *v1.TestResult, test_finished artifacts.ArtifactResult) error {
	if test_finished.Json["passed"] != nil {
		test.Passed = test_finished.Json["passed"].(bool)
	} else {
		test.Passed = false
	}
	if test_finished.Json["result"] != nil {
		test.Result = test_finished.Json["result"].(string)
	} else {
		test.Result = "N/A"
	}
	if test_finished.Json["timestamp"] != nil {
		ts := test_finished.Json["timestamp"].(float64)
		test.FinishDate = time.Unix(int64(ts), 0).Format("2006-01-02 15:04")
	} else {
		test.FinishDate = "N/A"
	}
	return nil
}

func PopulateTestFromStepFinished(test *v1.TestResult, step_test_finished artifacts.ArtifactResult) error {
	test.StepExecuted = true
	if step_test_finished.Json["passed"] != nil {
		test.StepPassed = step_test_finished.Json["passed"].(bool)
	} else {
		test.StepPassed = false
	}
	if step_test_finished.Json["result"] != nil {
		test.StepResult = step_test_finished.Json["result"].(string)
	} else {
		test.StepResult = "N/A"
	}

	return nil
}

func PopulateTestFromToolboxLogs(test *v1.TestResult, toolbox_logs map[string]artifacts.JsonArray) error {
	test.Ok = 0
	test.Failures = 0
	test.Ignored = 0

	for toolbox_step_name, toolbox_step_json := range toolbox_logs {

		stats := toolbox_step_json[len(toolbox_step_json)-1].(map[string]interface{})["stats"].(map[string]interface{})["localhost"].(map[string]interface{})
		ok := int(stats["ok"].(float64))
		failures := int(stats["failures"].(float64))
		ignored := int(stats["ignored"].(float64))
		log.Debugf("Step %s: ok %d, failures %d, ignored %d", toolbox_step_name, ok, failures, ignored)

		test.ToolboxStepsResults = append(test.ToolboxStepsResults,
			v1.ToolboxStepResult{Name: toolbox_step_name, Ok: ok, Failures: failures, Ignored: ignored}, )

		test.Ok += ok
		test.Failures += failures
		test.Ignored += ignored
	}
	log.Debugf("Test: ok %d, failures %d, ignored %d", test.Ok, test.Failures, test.Ignored)

	return nil
}

func PopulateTestMatrices(matricesSpec *v1.MatricesSpec, test_history int) error {
	for matrix_name, test_matrix := range matricesSpec.Matrices {
		log.Printf("* %s: %s\n", matrix_name, test_matrix.Description)
		for test_group, tests := range test_matrix.Tests {
			for test_idx := range tests {
				test := &tests[test_idx]
				var branch string
				if test.Variant != "" {
					branch = fmt.Sprintf("%s-%s", test.Branch, test.Variant)
				} else {
					branch = test.Branch
				}

				test.ProwName = fmt.Sprintf("%s-%s-%s", test_matrix.ProwConfig, branch, test.TestName)
				test.TestGroup = test_group

				// override matricesSpec.TestHistory if we received a flag value
				if test_history >= 0 {
					matricesSpec.TestHistory = test_history
				}

				test_build_ids, old_tests, err := artifacts.FetchLastNTestResults(&test_matrix, matrix_name, test.ProwName, matricesSpec.TestHistory,
					"finished.json", artifacts.TypeJson)
				if err != nil {
					return fmt.Errorf("Failed to fetch the last %d test results for %s: %v", matricesSpec.TestHistory, test.ProwName, err)
				}
				for _, old_test_build_id := range test_build_ids {
					old_test_finished := old_tests[old_test_build_id]
					old_test := v1.TestResult{TestSpec: test}
					old_test.BuildId = old_test_build_id
					test.OldTests = append(test.OldTests, &old_test)

					if err = PopulateTestFromFinished(&old_test, old_test_finished); err != nil {
						log.Warningf("Failed to store the last results of test %s/%s: %v",
							test.ProwName, old_test_build_id, err)
						continue
					}

					old_test_toolbox_logs, err := artifacts.FetchTestToolboxLogs(&test_matrix, test, old_test_build_id)

					if err = PopulateTestFromToolboxLogs(&old_test, old_test_toolbox_logs); err != nil {
						log.Warningf("Failed to get the toolbox step logs of the test %s/%s: %v", test.ProwName, old_test_build_id, err)
					}

					if old_test.Passed {
						continue
					}
					step_old_test_finished, err := artifacts.FetchTestStepResult(&test_matrix, test, old_test_build_id, "finished.json", artifacts.TypeJson)
					if err != nil {
						// if finished.json can be parsed as an HTML file, the file certainly does'nt exist --> do not warn about it
						_, err_json_as_html := artifacts.FetchTestStepResult(&test_matrix, test, old_test_build_id, "finished.json", artifacts.TypeHtml)
						if err_json_as_html != nil {
							log.Warningf("Failed to fetch the results of test step %s/%s: %v",
								test.ProwName, old_test_build_id, err)
						}
					}
					if err = PopulateTestFromStepFinished(&old_test, step_old_test_finished); err != nil {
						log.Warningf("Failed to store the results of test step %s/%s: %v", test.ProwName, old_test_build_id, err)
					}
				}
			}
		}
	}

	return nil
}
