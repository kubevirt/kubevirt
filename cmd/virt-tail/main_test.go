/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2025 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProcessDataCompleteLine(t *testing.T) {
	var lineBuf strings.Builder
	var output []string

	oldPrintln := printLine
	printLine = func(s string) { output = append(output, s) }
	defer func() { printLine = oldPrintln }()

	processData(&lineBuf, []byte("hello world\n"))

	if len(output) != 1 || output[0] != "hello world" {
		t.Fatalf("expected [hello world], got %v", output)
	}
	if lineBuf.Len() != 0 {
		t.Fatalf("expected empty lineBuf, got %q", lineBuf.String())
	}
}

func TestProcessDataPartialLine(t *testing.T) {
	var lineBuf strings.Builder
	var output []string

	oldPrintln := printLine
	printLine = func(s string) { output = append(output, s) }
	defer func() { printLine = oldPrintln }()

	processData(&lineBuf, []byte("hello"))

	if len(output) != 0 {
		t.Fatalf("expected no output, got %v", output)
	}
	if lineBuf.String() != "hello" {
		t.Fatalf("expected lineBuf to be 'hello', got %q", lineBuf.String())
	}

	processData(&lineBuf, []byte(" world\n"))

	if len(output) != 1 || output[0] != "hello world" {
		t.Fatalf("expected [hello world], got %v", output)
	}
}

func TestProcessDataMultipleLines(t *testing.T) {
	var lineBuf strings.Builder
	var output []string

	oldPrintln := printLine
	printLine = func(s string) { output = append(output, s) }
	defer func() { printLine = oldPrintln }()

	processData(&lineBuf, []byte("line1\nline2\nline3\n"))

	if len(output) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(output), output)
	}
	if output[0] != "line1" || output[1] != "line2" || output[2] != "line3" {
		t.Fatalf("unexpected output: %v", output)
	}
}

func TestProcessDataMultipleLinesWithPartial(t *testing.T) {
	var lineBuf strings.Builder
	var output []string

	oldPrintln := printLine
	printLine = func(s string) { output = append(output, s) }
	defer func() { printLine = oldPrintln }()

	processData(&lineBuf, []byte("line1\nline2\npartial"))

	if len(output) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(output), output)
	}
	if output[0] != "line1" || output[1] != "line2" {
		t.Fatalf("unexpected output: %v", output)
	}
	if lineBuf.String() != "partial" {
		t.Fatalf("expected lineBuf 'partial', got %q", lineBuf.String())
	}
}

func TestProcessDataSequentialSmallReads(t *testing.T) {
	var lineBuf strings.Builder
	var output []string

	oldPrintln := printLine
	printLine = func(s string) { output = append(output, s) }
	defer func() { printLine = oldPrintln }()

	// Simulate bytes arriving one at a time
	data := "logline 0000001\nlogline 0000002\nlogline 0000003\n"
	for i := range data {
		processData(&lineBuf, []byte{data[i]})
	}

	if len(output) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(output), output)
	}
	if output[0] != "logline 0000001" || output[1] != "logline 0000002" || output[2] != "logline 0000003" {
		t.Fatalf("unexpected output: %v", output)
	}
}

func TestProcessDataLargeSequentialBatch(t *testing.T) {
	var lineBuf strings.Builder
	var output []string

	oldPrintln := printLine
	printLine = func(s string) { output = append(output, s) }
	defer func() { printLine = oldPrintln }()

	// Simulate the actual test pattern: 8192 sequential log lines
	const numLines = 8192
	var allData strings.Builder
	for i := 1; i <= numLines; i++ {
		line := strings.Repeat("x", 120)
		allData.WriteString(line)
		allData.WriteByte('\n')
	}

	// Feed in random-sized chunks to simulate real I/O
	raw := []byte(allData.String())
	chunkSizes := []int{1, 7, 13, 127, 128, 129, 255, 256, 4096, 32768}
	offset := 0
	chunkIdx := 0
	for offset < len(raw) {
		size := chunkSizes[chunkIdx%len(chunkSizes)]
		chunkIdx++
		end := offset + size
		if end > len(raw) {
			end = len(raw)
		}
		processData(&lineBuf, raw[offset:end])
		offset = end
	}

	if len(output) != numLines {
		t.Fatalf("expected %d lines, got %d", numLines, len(output))
	}
}

func TestTailFileIntegration(t *testing.T) {
	dir := t.TempDir()
	logFile := dir + "/test.log"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	v := &VirtTail{ctx: ctx, logFile: logFile}

	var mu sync.Mutex
	var output []string
	oldPrintln := printLine
	printLine = func(s string) {
		mu.Lock()
		output = append(output, s)
		mu.Unlock()
	}
	defer func() { printLine = oldPrintln }()

	done := make(chan error, 1)
	go func() { done <- v.run() }()

	// Write lines to the file with small delays
	const numLines = 100
	f, err := os.Create(logFile)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= numLines; i++ {
		fmt.Fprintf(f, "logline %07d\n", i)
		if i%10 == 0 {
			f.Sync()
			time.Sleep(10 * time.Millisecond)
		}
	}
	f.Sync()
	f.Close()

	// Wait for all lines to be read
	deadline := time.After(10 * time.Second)
	for {
		mu.Lock()
		count := len(output)
		mu.Unlock()
		if count >= numLines {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timeout: expected %d lines, got %d", numLines, len(output))
			mu.Unlock()
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(output) != numLines {
		t.Fatalf("expected %d lines, got %d", numLines, len(output))
	}
	for i, line := range output {
		expected := fmt.Sprintf("logline %07d", i+1)
		if line != expected {
			t.Fatalf("line %d: expected %q, got %q", i, expected, line)
		}
	}
}
