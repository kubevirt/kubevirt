package imageupload

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	uploadcdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/upload/v1beta1"
)

func (c *command) uploadData(token string, file *os.File) error {
	uploadURL, err := ConstructUploadProxyPathAsync(c.uploadProxyURL, token, c.insecure)
	if err != nil {
		return err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	bar := pb.Full.Start64(fileInfo.Size())
	bar.SetWriter(os.Stdout)
	bar.Set(pb.Bytes, true)
	reader := bar.NewProxyReader(file)

	client := GetHTTPClientFn(c.insecure)
	req, _ := http.NewRequest("POST", uploadURL, io.NopCloser(reader))

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/octet-stream")
	req.ContentLength = fileInfo.Size()

	clientDo := func() error {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("unexpected return value %d, %s", resp.StatusCode, string(body))
		}
		return nil
	}

	c.cmd.Println()
	bar.Start()

	retry := uint(0)
	for retry < c.uploadRetries {
		if err = clientDo(); err == nil {
			break
		}
		retry++
		if retry < c.uploadRetries {
			time.Sleep(time.Duration(retry*rand.UintN(50)) * time.Millisecond)
		}
	}

	bar.Finish()
	c.cmd.Println()

	if err != nil && retry == c.uploadRetries {
		return fmt.Errorf("error uploading image after %d retries: %w", c.uploadRetries, err)
	}

	return nil
}

func (c *command) getUploadToken() (string, error) {
	request := &uploadcdiv1.UploadTokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "token-for-virtctl",
		},
		Spec: uploadcdiv1.UploadTokenRequestSpec{
			PvcName: c.name,
		},
	}

	response, err := c.client.CdiClient().UploadV1beta1().UploadTokenRequests(c.namespace).Create(context.Background(), request, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return response.Status.Token, nil
}

// ConstructUploadProxyPath - receives uploadproxy address and concatenates to it URI
func ConstructUploadProxyPath(uploadProxyURL string) (string, error) {
	u, err := url.Parse(uploadProxyURL)

	if err != nil {
		return "", err
	}

	if !strings.Contains(uploadProxyURL, uploadProxyURI) {
		u.Path = path.Join(u.Path, uploadProxyURI)
	}
	return u.String(), nil
}

// ConstructUploadProxyPathAsync - receives uploadproxy address and concatenates to it URI
func ConstructUploadProxyPathAsync(uploadProxyURL, token string, insecure bool) (string, error) {
	u, err := url.Parse(uploadProxyURL)
	if err != nil {
		return "", err
	}

	if !strings.Contains(uploadProxyURL, uploadProxyURIAsync) {
		u.Path = path.Join(u.Path, uploadProxyURIAsync)
	}

	// Attempt to discover async URL
	client := GetHTTPClientFn(insecure)
	req, err := http.NewRequest("HEAD", u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		// Async not available, use regular upload URL.
		return ConstructUploadProxyPath(uploadProxyURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Async not available, use regular upload URL.
		return ConstructUploadProxyPath(uploadProxyURL)
	}

	return u.String(), nil
}
