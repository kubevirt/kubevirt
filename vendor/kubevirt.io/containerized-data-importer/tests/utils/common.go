package utils

const (
	// cdi-file-host pod/service relative values
	FileHostName     = "cdi-file-host" // deployment and service name
	FileHostNs       = "kube-system"   // deployment and service namespace
	FileHostS3Bucket = "images"        // s3 bucket name (e.g. http://<serviceIP:port>/FileHostS3Bucket/image)
	AccessKeyValue   = "admin"         // http && s3 username, see hack/build/docker/cdi-func-test-file-host-http/htpasswd
	SecretKeyValue   = "password"      // http && s3 password,  ditto
	HttpAuthPort     = 81              // cdi-file-host service auth port
	HttpNoAuthPort   = 80              // cdi-file-host service no-auth port, requires AccessKeyValue and SecretKeyValue
	S3Port           = 9000            // cdi-file-host service S3 port, requires AccessKey and SecretKeyValue
)
