package matrix

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"html/template"
	"strings"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
)

type TemplateBase struct {
	Spec *v1.MatricesSpec
	Description string
	Date string
}

func Generate(matrixTemplate string, matrices *v1.MatricesSpec, date string) ([]byte, error) {
	matrix_template, err := ioutil.ReadFile(matrixTemplate)
	if err != nil {
		return []byte{}, fmt.Errorf("Matrix template file %s cannot be read: %v", matrixTemplate, err)
	}

	tmpl_data := TemplateBase{
		Spec: matrices,
		Date: date,
	}

	fmap := template.FuncMap{
        "nb_last_test": func() string {
			return fmt.Sprintf("%d", matrices.NbTestHistory)
		},
        "group_name": func(txt string) string {
			pipe_pos := strings.Index(txt, "|")
			if pipe_pos == -1 {
				return txt
			} else {
				return txt[pipe_pos+1:]
			}
		},
		"artifacts_url": func(matrix v1.MatrixSpec, test v1.TestResult) string {
			return fmt.Sprintf("%s/%s/%s/artifacts/%s/%s",
				matrix.ArtifactsURL, test.TestSpec.ProwName, test.BuildId, test.TestSpec.TestName, matrix.ProwStep)
		},
		"spyglass_url": func(matrix v1.MatrixSpec, prowName string, test v1.TestResult) string {
			return fmt.Sprintf("%s/%s/%s", matrix.ViewerURL, prowName, test.BuildId)
		},
		"test_status_descr": func(test v1.TestResult, status string) string {
			if status == "success" {
				return "Test passed"
			} else if status == "step_success" {
				return "Test failed but the operator step passed"
			} else if status == "step_failed" {
				return "Test failed because the operator step failed"
			} else if status == "step_missing" {
				return "Test failed but operator step wasn't executed"
			} else {
				return fmt.Sprintf("Test: %t, Step: %t (status: %s)",
					test.Passed, test.StepPassed, status)
			}
		},
		"test_status": func(test v1.TestResult) string {
			if test.Passed {
				return "success"
			} else if !test.StepExecuted {
				return "step_missing"
			} else if test.StepPassed {
				return "step_success"
			} else if !test.StepPassed {
				return "step_failed"
			} else {
				return "parsin_error"
			}
		},
    }

	tmpl := template.Must(template.New("runtime").Funcs(fmap).Parse(string(matrix_template)))

	var buff bytes.Buffer
	if err = tmpl.Execute(&buff, tmpl_data); err != nil {
		return []byte{}, fmt.Errorf("Matrix template file %s could not applied: %v", matrixTemplate, err)
	}

	generated_html := buff.Bytes()

	return generated_html, nil
}
