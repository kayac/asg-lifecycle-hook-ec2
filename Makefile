VERSION := $(shell git describe --tags)
.PHONY: build build.lambda image test clean

build: *.go go.*
	go build

build.lambda: *.go go.*
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bootstrap.amd64
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -o bootstrap.arm64

test:
	go clean -testcache
	go test -v -race ./...

clean:
	rm -f asg-lifecycle-hook-ec2 bootstrap.* dist/

image: build.lambda Dockerfile
	docker buildx build \
		--load \
		--tag ghcr.io/kayac/asg-lifecycle-hook-ec2:$(VERSION) \
		--platform linux/amd64,linux/arm64 \
		.

push: image
	docker push ghcr.io/kayac/asg-lifecycle-hook-ec2:$(VERSION)
