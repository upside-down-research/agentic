ASSET_FILES = $(shell find . -type f -name '*.go')
.PHONY: clean run all

all: output/agentic

output/agentic: $(ASSET_FILES)
	@mkdir -p output
	@go build -o output/agentic cmd/main.go

run: output/agentic
	@./output/agentic

clean:
	rm -rf output
