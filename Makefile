BINARY     := groundwork
MCP_BINARY := groundwork-mcp
CMD        := ./cmd/groundwork
MCP_CMD    := ./cmd/groundwork-mcp
OUT        := ./bin/$(BINARY)
MCP_OUT    := ./bin/$(MCP_BINARY)

.PHONY: build build-mcp build-all test lint run run-mcp clean

build:
	go build -o $(OUT) $(CMD)

build-mcp:
	go build -o $(MCP_OUT) $(MCP_CMD)

build-all: build build-mcp

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

run:
	go run $(CMD) $(ARGS)

run-mcp:
	go run $(MCP_CMD) $(ARGS)

clean:
	rm -rf ./bin