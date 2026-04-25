//go:build vmlogchecker

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"kubevirt.io/kubevirt/tests/vmlogchecker"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorPurple = "\033[35m"

	exitCodeRuntimeError    = 1
	exitCodeUnexpectedError = 2
)

var (
	logger    = log.New(os.Stdout, "", 0)
	errLogger = log.New(os.Stderr, "", 0)
)

func main() {
	logFile := flag.String("log", "", "Log file to analyze (required)")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	allLevels := flag.Bool("all-levels", false, "Check all log levels (default: only ERROR level, matching test reporter)")
	errorsOnly := flag.Bool("errors-only", false, "Print only unexpected errors (lines that need attention)")
	flag.Parse()

	if *logFile == "" {
		printUsage()
		os.Exit(exitCodeRuntimeError)
	}

	file, err := os.Open(*logFile)
	if err != nil {
		errLogger.Fatalf("Error opening log file: %v", err)
	}
	defer file.Close()

	totalLines, allowlistedCount, unexpectedCount, err := processLog(file, *allLevels, *errorsOnly, *noColor)
	if err != nil {
		errLogger.Fatalf("Error reading log file: %v", err)
	}

	printSummary(totalLines, allowlistedCount, unexpectedCount)

	if unexpectedCount > 0 {
		os.Exit(exitCodeUnexpectedError)
	}
}

func processLog(file *os.File, allLevels, errorsOnly, noColor bool) (totalLines, allowlistedCount, unexpectedCount int, err error) {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if !errorsOnly {
				fmt.Println()
			}
			continue
		}

		totalLines++

		if !allLevels && !vmlogchecker.IsErrorLevel(line) {
			if !errorsOnly {
				fmt.Println(line)
			}
			continue
		}

		switch vmlogchecker.ClassifyLogLine(line) {
		case vmlogchecker.AllowlistedError:
			allowlistedCount++
			if !errorsOnly {
				printColored(line, colorPurple, noColor)
			}
		case vmlogchecker.UnexpectedError:
			unexpectedCount++
			printColored(line, colorRed, noColor)
		default:
			if !errorsOnly {
				fmt.Println(line)
			}
		}
	}
	return totalLines, allowlistedCount, unexpectedCount, scanner.Err()
}

func printColored(line, color string, noColor bool) {
	if noColor {
		fmt.Println(line)
	} else {
		fmt.Printf("%s%s%s\n", color, line, colorReset)
	}
}

func printUsage() {
	errLogger.Println("Error: --log flag is required")
	errLogger.Printf("Usage: %s --log <logfile> [--no-color] [--all-levels] [--errors-only]", os.Args[0])
	errLogger.Println("")
	errLogger.Println("Options:")
	errLogger.Println("  --log          Log file to analyze (required)")
	errLogger.Println("  --no-color     Disable colored output")
	errLogger.Println("  --all-levels   Check all log levels (default: only ERROR level)")
	errLogger.Println("  --errors-only  Print only unexpected errors (lines that need attention)")
	errLogger.Println("")
	errLogger.Println("Output:")
	errLogger.Println("  - Normal lines: default color")
	errLogger.Printf("  - Allowlisted errors: %spurple%s (expected/known)", colorPurple, colorReset)
	errLogger.Printf("  - Unexpected errors: %sred%s (NEED ATTENTION)", colorRed, colorReset)
}

func printSummary(totalLines, allowlistedCount, unexpectedCount int) {
	logger.Println("")
	logger.Println("═══════════════════════════════════════════════════════════")
	logger.Println("  VM Log Analysis Summary")
	logger.Println("═══════════════════════════════════════════════════════════")
	logger.Printf("  Total lines:          %d", totalLines)
	logger.Printf("  Allowlisted errors:   %d (expected)", allowlistedCount)

	if unexpectedCount > 0 {
		logger.Printf("  Unexpected errors:    %d NEEDS ATTENTION", unexpectedCount)
	} else {
		logger.Println("  Unexpected errors:    0")
	}

	logger.Println("═══════════════════════════════════════════════════════════")
}
