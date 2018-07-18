# KubeVirt's OpenAPI Specification

This folder contains an [OpenAPI specification](https://github.com/OAI/OpenAPI-Specification) for KubeVirt API.
To modify the API (of v1 for example), please edit one or more of these files from the pkg/api/v1/ directory: schema.go, types.go, defaults.go. Then execute `make generate`. This will generate swagger.json and other files.
