.PHONY: clean run

output/agentic:
	@mkdir -p output
	@go build -o output/agentic cmd/main.go

run: output/agentic
	@./output/agentic

clean:
	rm -rf output
