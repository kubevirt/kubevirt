package utils

// cdi-file-host pod/service relative values
const (
	// FileHostName provides a deployment and service name for tests
	FileHostName = "cdi-file-host"
	// FileHostNs provides a deployment ans service namespace for tests
	FileHostNs = "kube-system"
	// FileHostS3Bucket provides an S3 bucket name for tests (e.g. http://<serviceIP:port>/FileHostS3Bucket/image)
	FileHostS3Bucket = "images"
	// AccessKeyValue provides a username to use for http and S3 (see hack/build/docker/cdi-func-test-file-host-http/htpasswd)
	AccessKeyValue = "admin"
	// SecretKeyValue provides a password to use for http and S3 (see hack/build/docker/cdi-func-test-file-host-http/htpasswd)
	SecretKeyValue = "password"
	// HttpAuthPort provides a cdi-file-host service auth port for tests
	HTTPAuthPort = 81
	// HttpNoAuthPort provides a cdi-file-host service no-auth port for tests, requires AccessKeyValue and SecretKeyValue
	HTTPNoAuthPort = 80
	// S3Port provides a cdi-file-host service S3 port, requires AccessKey and SecretKeyValue
	S3Port = 9000
)
