.PHONY: all format build run
all: format build

format:
	gofmt -w .

build: 
	go build .

run:
	@[ "${GITHUB_TOKEN_FILE}" ] || ( echo "GITHUB_TOKEN_FILE is not set"; exit 1 )
	@[ "${RELEASE}" ] || ( echo "RELEASE is not set"; exit 1 )

	go run . \
		--github-token-file $(GITHUB_TOKEN_FILE) \
		--release $(RELEASE)
