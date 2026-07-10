.PHONY: test vet doc

test:
	go test ./...

vet:
	go vet ./...

doc:
	go doc --all

tidy:
	go mod tidy
