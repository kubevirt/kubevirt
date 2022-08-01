package main

// This tool maintains the deploy/images.csv and the deploy/images.env files, to be used to generate the HCO CSV
// the csv (comma separated) file structure is:
// - environment variable name, to be place in the env file
// - name - the image name with no tag or digest
// - tag - the environment variable name, that holds the image tag
// - digest - the latest known digest
//
// The application will loop over the csv lines and will read the digest for name:${tag}, and will write back the file
// if there is a change
//
// It is also possible to query a digest for a single image, and get the result in the standard output. use the --image
// flag with the image name in repo/name:tag format; for example:
// $ digester --image hello-world
// hello-world@sha256:31b9c7d48790f0d8c50ab433d9c3b7e17666d6993084c002c2ff1ca09b96391d
//
// to get the image digest only, without the full image name, use the -d flag in addition to the --image flag
// $ tools/digester/digester -d --image hello-world
// 31b9c7d48790f0d8c50ab433d9c3b7e17666d6993084c002c2ff1ca09b96391d
//
import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

const (
	csvFile      = "deploy/images.csv"
	envFile      = "deploy/images.env"
	digestPrefix = "@sha256:"
)

type Image struct {
	EnvVar string
	Name   string
	Tag    string
	Digest string
}

func (i Image) getArr() []string {
	return []string{
		i.EnvVar,
		i.Name,
		i.Tag,
		i.Digest,
	}
}

func NewImage(fields []string) *Image {
	image := &Image{
		EnvVar: fields[0],
		Name:   fields[1],
		Tag:    fields[2],
	}

	if len(fields) > 3 {
		image.Digest = fields[3]
	}

	return image
}

type message struct {
	index    int
	digest   string
	fullName string
}

func (i *Image) setDigest(digest string) {
	i.Digest = digest
}

var (
	imageToDigest   string
	digestOnly      = false
	singleImageMode = false
)

func init() {
	flag.StringVar(&imageToDigest, "image", "", "single image in name:tag format; if exists, returns only one digest fo this image, instead of processing the CSV file.")
	flag.BoolVar(&digestOnly, "d", false, "when using --image, digester will only print the digest hex itself, without the image name")

	flag.Parse()

	if len(imageToDigest) > 0 {
		singleImageMode = true
	}
}
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	// Single image use-case
	if singleImageMode {
		querySingleImage(ctx, imageToDigest, digestOnly)
	} else {
		updateImages(ctx)
	}
}

func updateImages(ctx context.Context) {
	fmt.Println("Checking image digests")
	err, images := getCurrentImageList()

	wg := &sync.WaitGroup{}
	wg.Add(len(images) - 1) // the first "image" is the CSV title

	ch := make(chan message, len(images)-1)

	go func() {
		wg.Wait()
		close(ch)
	}()

	start := time.Now()
	for i, image := range images[1:] {
		go readOneDigest(ctx, image, i+1, wg, ch)
	}

	changed := checkForChanges(ch, images)

	howLong := time.Now().Sub(start)
	fmt.Println("took", howLong)

	if changed {
		fmt.Printf("Found new digests. Updating the %s file\n", csvFile)
		err = writeCsv(images)
		exitOnError(err, "failed to update the %s files", csvFile)
	} else {
		fmt.Println("The images file is up to date")
	}

	err = writeEnvFile(images)
	exitOnError(err, "failed to update the %s files", envFile)
}

func checkForChanges(ch chan message, images []*Image) bool {
	changed := false
	for msg := range ch {
		if images[msg.index].Digest != msg.digest {
			changed = true
			fmt.Printf("New digest for %s - %s\n", msg.fullName, msg.digest)
			images[msg.index].setDigest(msg.digest)
		}
	}
	return changed
}

func getCurrentImageList() (error, []*Image) {
	f, err := os.Open(csvFile)
	exitOnError(err, "can't open %s for reading", csvFile)

	reader := csv.NewReader(f)
	lines, err := reader.ReadAll()
	exitOnError(err, "can't read %s", csvFile)

	err = f.Close()
	exitOnError(err, "error while closing %s", csvFile)

	images := make([]*Image, 0, len(lines))
	for _, line := range lines {
		images = append(images, NewImage(line))
	}
	return err, images
}

func querySingleImage(ctx context.Context, imageToDigest string, digestOnly bool) {
	if strings.Contains(imageToDigest, digestPrefix) {
		fmt.Printf("%s is already in a digest format\n", imageToDigest)
		os.Exit(1)
	}

	imgRef, err := docker.ParseReference("//" + imageToDigest)
	exitOnError(err, "failed to parse container reference")

	digest, err := docker.GetDigest(ctx, nil, imgRef)
	exitOnError(err, "failed to get digest from image")

	if err != nil {
		fmt.Printf("Error while trying to get digest for %s; %s\n", imageToDigest, err)
		os.Exit(1)
	}

	if digestOnly {
		fmt.Println(digest)
	} else {
		loc := strings.LastIndex(imageToDigest, ":")
		var imageName string
		if loc == -1 {
			imageName = buildImageDigestName(imageToDigest, digest.Hex())
		} else {
			imageName = buildImageDigestName(imageToDigest[:loc], digest.Hex())
		}

		fmt.Println(imageName)
	}
}

func writeCsv(images []*Image) error {
	f, err := os.OpenFile(csvFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)

	lines := make([][]string, 0, len(images))
	for _, image := range images {
		lines = append(lines, image.getArr())
	}

	err = writer.WriteAll(lines)
	if err != nil {
		return err
	}
	return nil
}

func writeEnvFile(images []*Image) error {
	f, err := os.OpenFile(envFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)

	imageList := make([]string, len(images)-1, len(images)-1)
	for i, image := range images[1:] {
		imageDigest := buildImageDigestName(image.Name, image.Digest)
		_, err = writer.WriteString(fmt.Sprintf("%s=%s\n", image.EnvVar, imageDigest))
		if err != nil {
			return err
		}
		imageList[i] = imageDigest
	}

	if len(imageList) > 0 {
		_, err = writer.WriteString(fmt.Sprintf("DIGEST_LIST=\"%s\"\n", imageList[0]))
		if err != nil {
			return err
		}
		for _, image := range imageList[1:] {
			_, err = writer.WriteString(fmt.Sprintf("DIGEST_LIST=\"${DIGEST_LIST},%s\"\n", image))
			if err != nil {
				return err
			}
		}
	}

	return writer.Flush()
}

func exitOnError(err error, msg string, fmtParams ...interface{}) {
	if err != nil {
		fmt.Printf("%s; %v\n", fmt.Sprintf(msg, fmtParams...), err)
		os.Exit(1)
	}
}

func readOneDigest(ctx context.Context, image *Image, index int, wg *sync.WaitGroup, ch chan message) {
	fullName := fmt.Sprintf("//%s:%s", image.Name, os.Getenv(image.Tag))
	fmt.Println("Reading digest for", fullName)

	imgRef, err := docker.ParseReference(fullName)
	exitOnError(err, "failed to parse container reference")

	digest, err := retryGetDigest(ctx, nil, imgRef, 5, 1*time.Second)

	exitOnError(err, "failed to get digest from image")

	ch <- message{index: index, digest: digest.Hex(), fullName: fullName}
	wg.Done()
}

func retryGetDigest(ctx context.Context, sys *types.SystemContext, imgRef types.ImageReference, attempts int, sleep time.Duration) (digest digest.Digest, err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			fmt.Println("retrying after error:", err)
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			time.Sleep(sleep + jitter/2)
			sleep *= 3
		}
		digest, err = docker.GetDigest(ctx, sys, imgRef)
		if err == nil {
			return digest, nil
		}
	}
	return "", fmt.Errorf("aborting after %d attempts, last error: %w", attempts, err)
}

func buildImageDigestName(name, digest string) string {
	return fmt.Sprintf("%s%s%s", name, digestPrefix, digest)
}
