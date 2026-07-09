.PHONY: test fmt test-integration

test:
	go vet ./...
	go test ./...

fmt:
	gofmt -w .

test-integration:
	go test -tags integration ./...
