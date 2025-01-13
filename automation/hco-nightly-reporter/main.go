package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/slack-go/slack"
)

const (
	basicPrawURL        = "https://storage.googleapis.com/kubevirt-prow/logs/periodic-hco-push-nightly-build-main"
	latestBuildURL      = basicPrawURL + "/latest-build.txt"
	finishedURLTemplate = basicPrawURL + "/%s/finished.json"
	jobURLTemplate      = basicPrawURL + "/%s/prowjob.json"
)

type finished struct {
	Timestamp int64  `json:"timestamp"`
	Passed    bool   `json:"passed"`
	Result    string `json:"result"`
	Revision  string `json:"revision"`
}

func (f finished) getBuildTime() time.Time {
	return time.Unix(f.Timestamp, 0).UTC()
}

var (
	token     string
	channelId string
	groupId   string
)

func init() {
	var ok bool
	token, ok = os.LookupEnv("HCO_REPORTER_SLACK_TOKEN")
	if !ok {
		fmt.Fprintln(os.Stderr, "HCO_REPORTER_SLACK_TOKEN environment variable not set")
		os.Exit(1)
	}

	channelId, ok = os.LookupEnv("HCO_CHANNEL_ID")
	if !ok {
		fmt.Fprintln(os.Stderr, "HCO_CHANNEL_ID environment variable not set")
		os.Exit(1)
	}

	groupId, ok = os.LookupEnv("HCO_GROUP_ID")
	if !ok {
		fmt.Fprintln(os.Stderr, "HCO_GROUP_ID environment variable not set")
		os.Exit(1)
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	blocks, jobURL, err := generateMessage(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = sendMessageToSlackChannel(blocks)

	if err != nil {
		writeSendError(err, jobURL)
		os.Exit(1)
	}

	fmt.Println("Successfully sent message to the channel")
}

func writeSendError(err error, jobURL string) {
	fmt.Fprintln(os.Stderr, "failed to send the message to the channel; ", err.Error())
	if serr, ok := err.(slack.SlackErrorResponse); ok {
		for _, msg := range serr.ResponseMetadata.Messages {
			fmt.Fprintln(os.Stderr, msg)
		}
	}

	if len(jobURL) > 0 {
		fmt.Fprintln(os.Stderr, "job URL: ", jobURL)
	}
}

func generateMessage(ctx context.Context) ([]slack.Block, string, error) {
	client := http.DefaultClient
	client.Timeout = time.Second * 3

	latestBuild, err := getLatestBuild(ctx, client)
	if err != nil {
		return nil, "", fmt.Errorf("failed to latest job ID; %s", err.Error())
	}

	buildStatus, err := getBuildStatus(ctx, latestBuild)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch the build status; %s", err.Error())
	}

	buildTime := time.Unix(buildStatus.Timestamp, 0).UTC()
	if time.Since(buildTime).Hours() > 24 {
		return generateNoBuildMessage(buildTime), "", nil
	}

	jobURL, err := getJob(ctx, latestBuild)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch the job info; %s", err.Error())
	}

	return generateStatusMessage(buildStatus, buildTime, jobURL), jobURL, nil
}

func sendMessageToSlackChannel(blocks []slack.Block) error {
	s := slack.New(token)
	_, _, err := s.PostMessage(channelId, slack.MsgOptionBlocks(blocks...))
	return err
}

func generateMentionBlock() slack.Block {
	return slack.NewRichTextBlock("mention", slack.NewRichTextSection(
		slack.NewRichTextSectionUserGroupElement(groupId),
	))
}

func generateNoBuildMessage(buildTime time.Time) []slack.Block {
	return []slack.Block{
		slack.NewRichTextBlock("status", slack.NewRichTextSection(
			slack.NewRichTextSectionEmojiElement("failed", 3, nil),
			slack.NewRichTextSectionTextElement(
				" Nightly build wasn't run today", nil,
			),
		)),
		slack.NewRichTextBlock("last-build-time", slack.NewRichTextSection(
			slack.NewRichTextSectionTextElement("Last build was at ", nil),
			slack.NewRichTextSectionDateElement(buildTime.UTC().Unix(), "{date_long_full} at {time}, {ago}", nil, nil),
		)),
		generateMentionBlock(),
	}
}

func generateStatusMessage(buildStatus *finished, buildTime time.Time, jobURL string) []slack.Block {
	var status, emoji string
	if buildStatus.Passed {
		status = "passed"
		emoji = "solid-success"
	} else {
		status = "failed"
		emoji = "failed"
	}

	blocks := []slack.Block{
		slack.NewRichTextBlock("status", slack.NewRichTextSection(
			slack.NewRichTextSectionEmojiElement(emoji, 3, nil),
			slack.NewRichTextSectionTextElement(
				" Nightly build ", nil,
			),
			slack.NewRichTextSectionTextElement(" ", nil),
			slack.NewRichTextSectionLinkElement(jobURL, status, &slack.RichTextSectionTextStyle{Bold: true}),
			slack.NewRichTextSectionTextElement(", at ", nil),
			slack.NewRichTextSectionDateElement(buildTime.UTC().Unix(), "{date_long} at {time}", nil, nil),
		)),
	}

	if !buildStatus.Passed {
		blocks = append(blocks, generateMentionBlock())
	}
	return blocks
}

func getLatestBuild(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequest(http.MethodGet, latestBuildURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	latestBuildBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(latestBuildBytes), nil
}

func getBuildStatus(ctx context.Context, latestBuild string) (*finished, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(finishedURLTemplate, latestBuild), nil)
	if err != nil {
		return nil, err
	}

	finishedResp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer finishedResp.Body.Close()

	f := &finished{}
	dec := json.NewDecoder(finishedResp.Body)
	if err = dec.Decode(&f); err != nil {
		return nil, err
	}
	return f, nil
}

func getJob(ctx context.Context, latestBuild string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(jobURLTemplate, latestBuild), nil)
	if err != nil {
		return "", err
	}

	jobResp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}

	defer jobResp.Body.Close()

	job := struct {
		Status struct {
			URL string `json:"url,omitempty"`
		} `json:"status"`
	}{}
	dec := json.NewDecoder(jobResp.Body)
	err = dec.Decode(&job)
	if err != nil {
		return "", err
	}
	return job.Status.URL, nil
}
