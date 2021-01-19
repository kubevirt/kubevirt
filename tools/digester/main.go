package main

// This tool maintain the deploy/images.csv and the deploy/images.env files, to be used to generate the HCO CSV
// the csv (comma separated) file structure is:
// - environment variable name, to be place in the env file
// - name - the image name with no tag or digest
// - tag - the environment variable name, that holds the image tag
// - digest - the latest known digest
//
// The application will loop over the csv lines and will read the digest for name:${tag}, and will write back the file
// if there is a change

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	docker "docker.io/go-docker"
)

const (
	csvFile = "deploy/images.csv"
	envFile = "deploy/images.env"
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

func (i *Image) setDigest(digest string) {
	i.Digest = digest
}

func main() {

	digestOnly := false
	imageToDigest := flag.String("image", "", "single image in name:tag format; if exists, returns only one digest fo this image, instead of processing the CSV file.")
	flag.BoolVar(&digestOnly, "d", false, "when using --image, digester will only print the digest hex itself, without the image name")

	flag.Parse()

	cli, err := docker.NewEnvClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Single image use-case
	if imageToDigest != nil && *imageToDigest != "" {
		if strings.Contains(*imageToDigest, "@sha256:") {
			fmt.Printf("%s is already in a digest format\n", *imageToDigest)
			os.Exit(1)
		}
		inspect, err := cli.DistributionInspect(ctx, *imageToDigest, "")
		if err != nil {
			fmt.Printf("Error while trying to get digest for %s; %s\n", *imageToDigest, err)
			os.Exit(1)
		}

		digest := inspect.Descriptor.Digest.Hex()
		loc := strings.LastIndex(*imageToDigest, ":")
		if digestOnly {
			fmt.Println(digest)
		} else {
			imageName := (*imageToDigest)[:loc] + "@sha256:" + digest
			fmt.Println(imageName)
		}
		os.Exit(0)
	}

	fmt.Println("Checking image digests")
	f, err := os.Open(csvFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	reader := csv.NewReader(f)
	lines, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = f.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	images := make([]*Image, 0, len(lines))
	for _, line := range lines {
		images = append(images, NewImage(line))
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(images) - 1) // the first "image" is the CSV title

	type message struct {
		index    int
		digest   string
		fullName string
	}

	ch := make(chan message, len(images)-1)

	go func() {
		wg.Wait()
		close(ch)
	}()

	start := time.Now()
	for i, image := range images[1:] {
		go func(image *Image, index int) {
			fullName := fmt.Sprintf("%s:%s", image.Name, os.Getenv(image.Tag))
			fmt.Printf("Reading digest for %s\n", fullName)
			inspect, err := cli.DistributionInspect(ctx, fullName, "")
			if err != nil {
				fmt.Printf("Error while trying to get digest for %s; %s\n", fullName, err)
				os.Exit(1)
			}

			digest := inspect.Descriptor.Digest.Hex()
			ch <- message{index: index, digest: digest, fullName: fullName}
			wg.Done()
		}(image, i+1)
	}

	changed := false
	for msg := range ch {
		if images[msg.index].Digest != msg.digest {
			changed = true
			fmt.Printf("New digest for %s - %s\n", msg.fullName, msg.digest)
			images[msg.index].setDigest(msg.digest)
		}
	}

	howLong := time.Now().Sub(start)
	fmt.Println("took", howLong)
	if changed {
		fmt.Println("Found new digests. Updating the file")
		if err = writeCsv(images); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		fmt.Println("The images file is up to date")
	}

	if err = writeEnvFile(images); err != nil {
		fmt.Println(err)
		os.Exit(1)
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
	f, err := os.OpenFile(envFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)

	imageList := make([]string, len(images)-1, len(images)-1)
	for i, image := range images[1:] {
		imageDigest := fmt.Sprintf("%s@sha256:%s", image.Name, image.Digest)
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
