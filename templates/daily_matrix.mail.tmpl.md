
{{ .Spec.Description }}
{{ .Spec.Description | md_section }}

{{ range $matrix_name, $matrix := .Spec.Matrices -}}
{{ range $test_group, $tests := .Tests -}}

{{ $test_group | group_name}}
{{ $test_group | group_name | md_subsection}}

{{ range $test := $tests -}}
{{$test_status := test_status $test.TestResult -}}

* {{ $matrix.OperatorName }} {{ $test.OperatorVersion }}: {{ $test.Result }}
  - {{ test_status_descr $test.TestResult $test_status | unescape_html }}
  - Test finished at {{ $test.FinishDate }}

{{ end }}{{ end }}{{ end }}
---
Document generated on {{ .Date }}.
