generate_daily_matrix: \
	gpu

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

# static

static:
	cp -rv static/* output/

.PHONY: static

#

matrix_benchmarks:
	go run cmd/main.go --debug matrix_benchmarks \
           --config-file examples/gpu-operator.yml \
           --output-dir output/matrix_benchmarks

.PHONY: matrix_benchmarking

#

build:
	go build -o ci-dashboard cmd/main.go
