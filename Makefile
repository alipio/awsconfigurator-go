.DEFAULT_GOAL = test
.PHONY: FORCE

BINARY = awsconfigurator

# Build
build: $(BINARY)
.PHONY: build

clean:
	rm -f $(BINARY)
.PHONY: clean

# Test
test:
	@set -a && . ./.env && set +a && go test ./... -p 1
.PHONY: test

# Non-PHONY targets (real files)
$(BINARY): FORCE
	go build -o $@ ./cmd/awsconfigurator

go.mod: FORCE
	go mod tidy
	go mod verify
go.sum: go.mod
