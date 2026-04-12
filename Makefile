.PHONY: format
format:
	gofmt -w -s .

.PHONY: test
test:
	go test -v -race -timeout 3m -coverpkg=./... -coverprofile=coverage.out ./...

# Install golangci-lint tool to run lint locally
# https://golangci-lint.run/usage/install
.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix

.PHONY: go-tidy
go-tidy:
	go mod tidy
	cd jsonschema && go mod tidy
	cd example && go mod tidy
	cd benchmark && go mod tidy

