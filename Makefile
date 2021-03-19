run:
	go run cmd/main.go --debug daily_matrix

build:
	go build -o ci-dashboard cmd/main.go
