package populate

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/openshift-psap/ci-dashboard/pkg/artifacts"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
)

var log = logrus.New()

func PopulateTestFromFinished(test *v1.TestResult, test_finished artifacts.ArtifactResult) error {
	if test_finished.Json["passed"] != nil {
		test.Passed = test_finished.Json["passed"].(bool)
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

func PopulateTestFromStepFinished(test_result *v1.TestResult, step_test_finished artifacts.ArtifactResult) error {
	if step_test_finished.Json["passed"] != nil {
		test_result.StepPassed = step_test_finished.Json["passed"].(bool)
		test_result.StepExecuted = true
	}
	if step_test_finished.Json["result"] != nil {
		test_result.StepResult = step_test_finished.Json["result"].(string)
		test_result.StepExecuted = true
	} else {
		test_result.StepResult = "N/A"
	}

	return nil
}

func PopulateTestWarnings(test_result *v1.TestResult) error {
	warnings, err := artifacts.FetchTestWarnings(test_result)
	if err != nil {
		log.Warningf("Failed to get the warnings of the test %s/%s: %v",
			test_result.TestSpec.ProwName, test_result.BuildId, err)
		return nil
	}
	if len(warnings) == 0 {
		return nil
	}
	test_result.Warnings = make(map[string]string)
	for warning_name, warning_value := range warnings {
		test_result.Warnings[warning_name] = warning_value
		log.Debugf("Test warning: %s: %s", warning_name, warning_value)
	}

	return nil
}

func PopulateTestFromToolboxLogs(test_result *v1.TestResult, toolbox_logs map[string]artifacts.JsonArray) error {
	test_result.Ok = 0
	test_result.Failures = 0
	test_result.Ignored = 0

	for toolbox_step_name, toolbox_step_json := range toolbox_logs {
		fmt.Println(toolbox_step_name)

		stats := toolbox_step_json[len(toolbox_step_json)-1].(map[string]interface{})["stats"].(map[string]interface{})["localhost"].(map[string]interface{})
		ok := int(stats["ok"].(float64))
		failures := int(stats["failures"].(float64))
		ignored := int(stats["ignored"].(float64))
		log.Debugf("Step %s: ok %d, failures %d, ignored %d", toolbox_step_name, ok, failures, ignored)

		test_result.ToolboxStepsResults = append(test_result.ToolboxStepsResults, v1.ToolboxStepResult{Name: toolbox_step_name, Ok: ok, Failures: failures, Ignored: ignored})
		stepResults := &test_result.ToolboxStepsResults[len(test_result.ToolboxStepsResults)-1]

		test_result.Ok += ok
		test_result.Failures += failures
		test_result.Ignored += ignored

		path := "artifacts/"+ toolbox_step_name + "/EXPECTED_FAIL"
		expectedFailBytes, err := artifacts.FetchTestStepResult(test_result, path, artifacts.TypeBytes)
		if err == nil {
			expectedFail := string(expectedFailBytes.Bytes)
			stepResults.ExpectedFailure = expectedFail
			log.Debugf("Expected failure: %s", expectedFail)
			test_result.Failures -= 1
		} else {
			// not an expected fail, ignore
		}
		fmt.Println("--------------------------");
	}

	log.Debugf("Test: ok %d, failures %d, ignored %d, expected fail: %d",
		test_result.Ok, test_result.Failures, test_result.Ignored)

	return nil
}

func PopulateTestStepLogs(matrices_spec *v1.MatricesSpec) {
	var populateTestStepLogs = func(test_result *v1.TestResult) error {
		test_toolbox_logs, err := artifacts.FetchTestToolboxLogs(test_result)
		if err != nil {
			log.Warningf("Failed to get the toolbox steps of the test %s/%s: %v",
				test_result.TestSpec.ProwName, test_result.BuildId, err)
			return nil
		}
		if err = PopulateTestFromToolboxLogs(test_result, test_toolbox_logs); err != nil {
			log.Warningf("Failed to get the toolbox step logs of the test %s/%s: %v",
				test_result.TestSpec.ProwName, test_result.BuildId, err)
			return nil
		}

		return nil
	}

	TraverseAllTestResults(matrices_spec, populateTestStepLogs)
}

func TraverseAllTestResults(matrices_spec *v1.MatricesSpec, cb func(test_result *v1.TestResult) error) error {
	for _, test_matrix := range matrices_spec.Matrices {
		for _, tests := range test_matrix.Tests {
			for _, test := range tests {
				for _, test_result := range test.OldTests {
					if err := cb(test_result); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func populateTestResult(test *v1.TestSpec, build_id string, finished_file artifacts.ArtifactResult) *v1.TestResult {
	test_result := &v1.TestResult{
		TestSpec: test,
		BuildId: build_id,
	}
	var err error

	if err = PopulateTestFromFinished(test_result, finished_file); err != nil {
		log.Warningf("Failed to store the last results of test %s/%s: %v",
			test.ProwName, test_result.BuildId, err)
		return test_result
	}

	test_result.ToolboxSteps, err = artifacts.FetchTestToolboxSteps(test_result)
	if err != nil {
		log.Warningf("Failed to parse the steps of test %s/%s: %v",
			test.ProwName, test_result.BuildId, err)
	}

	step_test_result_finished, err := artifacts.FetchTestStepResult(test_result, "finished.json", artifacts.TypeJson)
	if err != nil {
		// if finished.json can be parsed as an HTML file, the file certainly does'nt exist --> do not warn about it
		_, err_json_as_html := artifacts.FetchTestStepResult(test_result, "finished.json", artifacts.TypeHtml)
		if err_json_as_html != nil {
			log.Warningf("Failed to fetch the results of test step %s/%s: %v",
				test.ProwName, test_result.BuildId, err)
		}
	}

	if err = PopulateTestFromStepFinished(test_result, step_test_result_finished); err != nil {
		log.Warningf("Failed to store the results of test step %s/%s: %v", test.ProwName, test_result.BuildId, err)
	}

	if !test_result.StepPassed {
		contentBytes, err := artifacts.FetchTestStepResult(test_result, "artifacts/FLAKE", artifacts.TypeBytes)

		if err == nil {
			content := string(contentBytes.Bytes)
			if !strings.Contains(content, "doctype html") {
				test_result.KnownFlake = strings.TrimSuffix(content, "\n")
			}
		} else {
			log.Warningf("Failed to check if %s/%s is a flake: %v", test.ProwName, test_result.BuildId, err)
		}
	}

	if err = PopulateTestWarnings(test_result); err != nil {
		log.Warningf("Failed to fetch the warnings of test step %s/%s: %v", test.ProwName, test_result.BuildId, err)
	}

	/* --- */

	ocpVersion_content, err := artifacts.FetchTestStepResult(test_result, "artifacts/ocp.version", artifacts.TypeBytes)
	if err == nil {
		test_result.OpenShiftVersion = strings.TrimSuffix(string(ocpVersion_content.Bytes), "\n")
	} else {
		log.Warningf("Failed to read the OpenShift version (%s/%s): %v", test.ProwName, test_result.BuildId, err)
	}
	if strings.Contains(test_result.OpenShiftVersion, "doctype") {
		// 404 page not recognized
		test_result.OpenShiftVersion = "[PARSING ERROR]"
	} else if test_result.OpenShiftVersion == "MISSING" {
		test_result.OpenShiftVersion = ""
	}

	operatorVersion_content, err := artifacts.FetchTestStepResult(test_result, "artifacts/operator.version", artifacts.TypeBytes)
	if err == nil {
		test_result.OperatorVersion = strings.TrimSuffix(string(operatorVersion_content.Bytes), "\n")
	} else {
		log.Warningf("Failed to read the Operator version (%s/%s): %v", test.ProwName, test_result.BuildId, err)
	}
	if strings.Contains(test_result.OperatorVersion, "doctype") {
		// 404 page not recognized
		test_result.OperatorVersion = "[PARSING ERROR] " + test_result.TestSpec.OperatorVersion
	} else if test_result.OperatorVersion == "MISSING" {
		test_result.OperatorVersion = ""
	}

	ciartifactsVersion_content, err := artifacts.FetchTestStepResult(test_result, "artifacts/ci_artifact.git_version", artifacts.TypeBytes)
	if err == nil {
		test_result.CiArtifactsVersion = strings.TrimSuffix(string(ciartifactsVersion_content.Bytes), "\n")
	} else {
		log.Warningf("Failed to read the CI-Artifacts version (%s/%s): %v", test.ProwName, test_result.BuildId, err)
	}
	if strings.Contains(test_result.CiArtifactsVersion, "doctype") {
		// 404 page not recognized
		test_result.CiArtifactsVersion = "PARSING ERROR"
	} else if test_result.CiArtifactsVersion == "MISSING" {
		test_result.CiArtifactsVersion = ""
	}
	return test_result
}

func populateTest(test_matrix *v1.MatrixSpec, test_group string, test *v1.TestSpec, test_history int) error {
	test.TestGroup = test_group
	test.Matrix = test_matrix

	var branch string
	if test.Variant != "" {
		branch = fmt.Sprintf("%s-%s", test.Branch, test.Variant)
	} else {
		branch = test.Branch
	}

	test.ProwName = fmt.Sprintf("%s-%s-%s", test_matrix.ProwConfig, branch, test.TestName)

	test_build_ids, finished_files, err := artifacts.FetchLastNTestResults(test_matrix, test.ProwName, test_history,
		"finished.json", artifacts.TypeJson)
	if err != nil {
		return fmt.Errorf("Failed to fetch the last %d test results for %s: %v", test_history, test.ProwName, err)
	}
	for _, build_id := range test_build_ids {
		test_result := populateTestResult(test, build_id, finished_files[build_id])

		test.OldTests = append(test.OldTests, test_result)
	}

	return nil
}

func PopulateTestMatrices(matricesSpec *v1.MatricesSpec, test_history int) error {
	// override matricesSpec.TestHistory if we received a flag value
	if test_history >= 0 {
		matricesSpec.TestHistory = test_history
	} else {
		test_history = matricesSpec.TestHistory
	}

	for matrix_name := range matricesSpec.Matrices {
		test_matrix := matricesSpec.Matrices[matrix_name]
		test_matrix.Name = matrix_name

		log.Printf("* %s: %s\n", test_matrix.Name, test_matrix.Description)
		for test_group, tests := range test_matrix.Tests {
			for test_idx := range tests {
				if err := populateTest(&test_matrix, test_group, &tests[test_idx], test_history); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
