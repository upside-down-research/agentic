ASSET_FILES = $(shell find . -type f -name '*.go')
.PHONY: clean run all test

all: output/agentic

output/agentic: $(ASSET_FILES)
	@mkdir -p output
	@rm -f  output/agentic
	@go build -o output/agentic cmd/agentic/main.go

# Keep legacy binary for backwards compatibility
output/agentic-legacy: $(ASSET_FILES)
	@mkdir -p output
	@rm -f  output/agentic-legacy
	@go build -o output/agentic-legacy cmd/main.go

# Build reasoning agent
output/reasoning-agent: $(ASSET_FILES)
	@mkdir -p output
	@rm -f  output/reasoning-agent
	@go build -o output/reasoning-agent cmd/reasoning-agent/main.go

run: output/agentic
	@./output/agentic

test:
	@go test -v ./...

clean:
	rm -rf output
