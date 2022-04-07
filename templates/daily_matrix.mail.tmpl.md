
{{ .Spec.Description }}
{{ .Spec.Description | md_section }}

{{ range $matrix_name, $matrix := .Spec.Matrices -}}
{{ range $test_group, $tests := .Tests -}}

{{ $test_group | group_name}}
{{ $test_group | group_name | md_subsection}}

{{ range $test := $tests -}}
{{ if $test.OldTests }}
{{ $last_test := (index $test.OldTests 0) }}
{{$test_status := test_status $last_test -}}

* {{ $matrix.OperatorName }} {{ $test.OperatorVersion }}: {{ $last_test.Result }}
  - {{ test_status_descr $last_test $test_status | unescape_html }}, finished at {{ $last_test.FinishDate }}
{{ range $message_type := test_message_types -}}
{{ range $flake, $message := test_messages $message_type $last_test -}}
{{ $message_type}}: {{ $message }}.
{{ end }}
{{ end }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}
---
Document generated on {{ .Date }}.
