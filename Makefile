default: build

.PHONY: build install fmt vet test testacc

build:
	go build -o terraform-provider-opteryx

install: build
	go install .

fmt:
	gofmt -s -w .

vet:
	go vet ./...

test:
	go test ./...

# Requires OPTERYX_ENDPOINT/OPTERYX_TOKEN (or provider defaults) pointed at a
# real policy.opteryx instance -- these create and destroy actual policies.
testacc:
	TF_ACC=1 go test ./... -v -timeout 120m
