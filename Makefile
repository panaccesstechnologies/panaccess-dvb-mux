.PHONY: test fixture

test:
	go test ./...

fixture:
	go run ./cmd/tsfixture
