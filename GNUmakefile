default:
	go test ./...
	go vet ./...
	golint ./...
.PHONY: default

package:
	GOOS=linux GOARCH=amd64 go build .
	mkdeb build mkdeb.json
.PHONY: package
