.PHONY: test fmt test-integration test-integration-openai

test:
	go vet ./...
	go test ./...

fmt:
	gofmt -w .

test-integration:
	go test -tags integration ./...

test-integration-openai:
	set -a; if [ -f .env ]; then . ./.env; fi; set +a; \
	GOCACHE="$${GOCACHE:-/tmp/second-opinion-go-cache}" go test -tags integration -run '^TestOpenAICompatibleConformance$$' ./internal/provider/openai_compatible
