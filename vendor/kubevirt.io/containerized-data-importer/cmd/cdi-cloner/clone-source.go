package main

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog"
	"kubevirt.io/containerized-data-importer/pkg/common"
	prometheusutil "kubevirt.io/containerized-data-importer/pkg/util/prometheus"
)

var (
	contentType string
	uploadBytes uint64
)

func init() {
	flag.StringVar(&contentType, "content_type", "", "archive|kubevirt")
	flag.Uint64Var(&uploadBytes, "upload_bytes", 0, "approx number of bytes in input")
	klog.InitFlags(nil)
}

func getEnvVarOrDie(name string) string {
	value := os.Getenv(name)
	if value == "" {
		klog.Fatalf("Error geting env var %s", name)
	}
	return value
}

func createHTTPClient(clientKey, clientCert, serverCert []byte) *http.Client {
	clientKeyPair, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		klog.Fatalf("Error %s creating client keypair", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(serverCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientKeyPair},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	return client
}

func startPrometheus() {
	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		klog.Fatalf("Error %s creating temp dir", err)
	}

	prometheusutil.StartPrometheusEndpoint(certsDirectory)
}

func createProgressReader(readCloser io.ReadCloser, ownerUID string, totalBytes uint64) io.ReadCloser {
	progress := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "clone_progress",
			Help: "The clone progress in percentage",
		},
		[]string{"ownerUID"},
	)
	prometheus.MustRegister(progress)

	promReader := prometheusutil.NewProgressReader(readCloser, totalBytes, progress, ownerUID)
	promReader.StartTimedUpdate()

	return promReader
}

func pipeToGzip(reader io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()
	gzw := gzip.NewWriter(pw)

	go func() {
		n, err := io.Copy(gzw, reader)
		if err != nil {
			klog.Fatalf("Error %s piping to gzip", err)
		}
		gzw.Close()
		pw.Close()
		klog.Infof("Wrote %d bytes\n", n)
	}()

	return pr
}

func main() {
	flag.Parse()
	defer klog.Flush()

	klog.Infof("content_type is %q\n", contentType)
	klog.Infof("upload_bytes is %d", uploadBytes)

	ownerUID := getEnvVarOrDie(common.OwnerUID)

	clientKey := []byte(getEnvVarOrDie("CLIENT_KEY"))
	clientCert := []byte(getEnvVarOrDie("CLIENT_CERT"))
	serverCert := []byte(getEnvVarOrDie("SERVER_CA_CERT"))

	url := getEnvVarOrDie("UPLOAD_URL")

	klog.V(1).Infoln("Starting cloner target")

	reader := pipeToGzip(createProgressReader(os.Stdin, ownerUID, uploadBytes))

	startPrometheus()

	client := createHTTPClient(clientKey, clientCert, serverCert)

	req, _ := http.NewRequest("POST", url, reader)

	if contentType != "" {
		req.Header.Set("x-cdi-content-type", contentType)
		klog.Infof("Set header to %s", contentType)
	}

	response, err := client.Do(req)
	if err != nil {
		klog.Fatalf("Error %s POSTing to %s", err, url)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		klog.Fatalf("Unexpected status code %d", response.StatusCode)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, response.Body)
	if err != nil {
		klog.Fatalf("Error %s copying response body", err)
	}

	klog.V(1).Infof("Response body:\n%s", buf.String())

	klog.V(1).Infoln("clone complete")
}
