
{{ .Spec.Description }}
{{ .Spec.Description | md_section }}

{{ range $matrix_name, $matrix := .Spec.Matrices -}}
{{ range $test_group, $tests := .Tests -}}

{{ $test_group | group_name}}
{{ $test_group | group_name | md_subsection}}

{{ range $test := $tests -}}
{{ $last_test := (index $test.OldTests 0) }}
{{$test_status := test_status $last_test -}}

* {{ $matrix.OperatorName }} {{ $test.OperatorVersion }}: {{ $last_test.Result }}
  - {{ test_status_descr $last_test $test_status | unescape_html }}
  - Test finished at {{ $last_test.FinishDate }}

{{ end }}{{ end }}{{ end }}
---
Document generated on {{ .Date }}.
