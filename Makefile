SWEEP?=global
BINARY=terraform-provider-activedirectory
TEST?=./...
TEST_COUNT?=1

default: build

build: lint
	go build

install: lint
	go install

lint:
	@golangci-lint run ./...
	@tfproviderlint ./...

tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install github.com/bflad/tfproviderlint/cmd/tfproviderlint

test: lint
	go test $(TEST) $(TESTARGS) -timeout=120s -parallel=4

testacc:
	TF_ACC=1 go test $(TEST) -v -count $(TEST_COUNT) -parallel 20 $(TESTARGS) -timeout 120m

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test $(TEST) -v -sweep=$(SWEEP) $(SWEEPARGS)
