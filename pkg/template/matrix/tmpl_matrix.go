package matrix

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"html/template"
	"strings"

	v1 "github.com/openshift-psap/ci-dashboard/api/matrix/v1"
)

const (
	MatrixTemplate  = "templates/matrix.tmpl.html"
)

func Generate(matrixSpec *v1.MatricesSpec) ([]byte, error) {
	matrix_template, err := ioutil.ReadFile(MatrixTemplate)
	if err != nil {
		return []byte{}, fmt.Errorf("Matrix template file %s cannot be read: %v", MatrixTemplate, err)
	}

	tmpl_data := matrixSpec

	fmap := template.FuncMap{
        "indent": func(len int, txt string) string {
			return strings.ReplaceAll(txt, "\n", "\n"+strings.Repeat(" ", len))
		},
        "group_name": func(txt string) string {
			pipe_pos := strings.Index(txt, "|")
			if pipe_pos == -1 {
				return txt
			} else {
				return txt[pipe_pos+1:]
			}
		},
    }

	tmpl := template.Must(template.New("runtime").Funcs(fmap).Parse(string(matrix_template)))

	var buff bytes.Buffer
	if err = tmpl.Execute(&buff, tmpl_data); err != nil {
		return []byte{}, fmt.Errorf("Matrix template file %s could not applied: %v", MatrixTemplate, err)
	}

	generated_html := buff.Bytes()

	return generated_html, nil
}
