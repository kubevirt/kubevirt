//go:build vmlogchecker

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"kubevirt.io/kubevirt/tests/vmlogchecker"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorPurple = "\033[35m"
)

func main() {
	logFile := flag.String("log", "", "Log file to analyze (required)")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	allLevels := flag.Bool("all-levels", false, "Check all log levels (default: only ERROR level, matching test reporter)")
	errorsOnly := flag.Bool("errors-only", false, "Print only unexpected errors (lines that need attention)")
	flag.Parse()

	if *logFile == "" {
		fmt.Fprintf(os.Stderr, "Error: --log flag is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s --log <logfile> [--no-color] [--all-levels] [--errors-only]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  --log          Log file to analyze (required)\n")
		fmt.Fprintf(os.Stderr, "  --no-color     Disable colored output\n")
		fmt.Fprintf(os.Stderr, "  --all-levels   Check all log levels (default: only ERROR level)\n")
		fmt.Fprintf(os.Stderr, "  --errors-only  Print only unexpected errors (lines that need attention)\n")
		fmt.Fprintf(os.Stderr, "\nOutput:\n")
		fmt.Fprintf(os.Stderr, "  - Normal lines: default color\n")
		fmt.Fprintf(os.Stderr, "  - Allowlisted errors: %spurple%s (expected/known)\n", colorPurple, colorReset)
		fmt.Fprintf(os.Stderr, "  - Unexpected errors: %sred%s (NEED ATTENTION)\n", colorRed, colorReset)
		os.Exit(1)
	}

	file, err := os.Open(*logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	var totalLines, allowlistedCount, unexpectedCount int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if !*errorsOnly {
				fmt.Println()
			}
			continue
		}

		totalLines++

		if !*allLevels && !vmlogchecker.IsErrorLevel(line) {
			if !*errorsOnly {
				fmt.Println(line)
			}
			continue
		}

		classification := vmlogchecker.ClassifyLogLine(line)

		switch classification {
		case vmlogchecker.AllowlistedError:
			allowlistedCount++
			if !*errorsOnly {
				if *noColor {
					fmt.Println(line)
				} else {
					fmt.Printf("%s%s%s\n", colorPurple, line, colorReset)
				}
			}
		case vmlogchecker.UnexpectedError:
			unexpectedCount++
			if *noColor {
				fmt.Println(line)
			} else {
				fmt.Printf("%s%s%s\n", colorRed, line, colorReset)
			}
		default:
			if !*errorsOnly {
				fmt.Println(line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "\nError reading log file: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  VM Log Analysis Summary\n")
	fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  Total lines:          %d\n", totalLines)
	fmt.Fprintf(os.Stderr, "  Allowlisted errors:   %d (expected)\n", allowlistedCount)

	if unexpectedCount > 0 {
		fmt.Fprintf(os.Stderr, "  Unexpected errors:    %d NEEDS ATTENTION\n", unexpectedCount)
	} else {
		fmt.Fprintf(os.Stderr, "  Unexpected errors:    0\n")
	}
	fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════\n")

	if unexpectedCount > 0 {
		os.Exit(1)
	}
}
