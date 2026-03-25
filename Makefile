BINARY   := groundwork
CMD      := ./cmd/groundwork
OUT      := ./bin/$(BINARY)

.PHONY: build test lint run clean

build:
	go build -o $(OUT) $(CMD)

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

run:
	go run $(CMD) $(ARGS)

clean:
	rm -rf ./bin