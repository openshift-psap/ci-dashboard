generate_daily_matrix: \
	gpu \
	sro \
	nto \
	nfd

# GPU Operator

gpu: output/gpu-operator_daily-matrix.html output/gpu-operator_daily-matrix.html

output/gpu-operator_daily-matrix.md: templates/daily_matrix.mail.tmpl.md
	go run cmd/main.go --debug daily_matrix \
           --config-file examples/gpu-operator.yml \
           --template $< \
           --output-file $@

output/gpu-operator_daily-matrix.html: templates/daily_matrix.tmpl.html
	go run cmd/main.go --debug daily_matrix \
           --config-file examples/gpu-operator.yml \
           --template $< \
           --output-file $@

# SRO

sro: output/sro_daily-matrix.html output/sro_daily-matrix.md

output/sro_daily-matrix.html: templates/daily_matrix.tmpl.html
	go run cmd/main.go --debug daily_matrix \
           --config-file examples/sro.yml \
           --template $< \
           --output-file $@

output/sro_daily-matrix.md: templates/daily_matrix.mail.tmpl.md
	go run cmd/main.go --debug daily_matrix \
           --config-file examples/sro.yml \
           --template $< \
           --output-file $@

# NTO

nto: output/nto_daily-matrix.html output/nto_daily-matrix.md

output/nto_daily-matrix.html: templates/daily_matrix.tmpl.html
	go run cmd/main.go --debug daily_matrix \
           --config-file examples/nto.yml \
           --template $< \
           --output-file $@

output/nto_daily-matrix.md: templates/daily_matrix.mail.tmpl.md
	go run cmd/main.go --debug daily_matrix \
           --config-file examples/nto.yml \
           --template  $< \
           --output-file $@

# NFD-Operator

nfd: output/nfd_daily-matrix.html output/nfd_daily-matrix.md

output/nfd_daily-matrix.html: templates/daily_matrix.tmpl.html
	go run cmd/main.go --debug daily_matrix \
           --config-file examples/nto.yml \
           --template $< \
           --output-file $@

output/nfd_daily-matrix.md: templates/daily_matrix.mail.tmpl.md
	go run cmd/main.go --debug daily_matrix \
           --config-file examples/nto.yml \
           --template  $< \
           --output-file $@

#

build:
	go build -o ci-dashboard cmd/main.go
