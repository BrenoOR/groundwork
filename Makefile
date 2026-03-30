MCP_BINARY := groundwork-mcp
MCP_CMD    := ./cmd/groundwork-mcp
MCP_OUT    := ./bin/$(MCP_BINARY)

.PHONY: build-mcp test lint run-mcp clean

build-mcp:
	go build -o $(MCP_OUT) $(MCP_CMD)

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

run-mcp:
	go run $(MCP_CMD) $(ARGS)

clean:
	rm -rf ./bin
