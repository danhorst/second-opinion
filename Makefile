.PHONY: test fmt test-integration

test:
	python3 -m unittest discover -s scripts/tests -p 'test_*.py'
	go vet ./...
	go test ./...

fmt:
	gofmt -w .

test-integration:
	go test -tags integration ./...
