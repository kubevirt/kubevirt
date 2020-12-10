// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cmdline is parser for kernel command-line args from /proc/cmdline.
//
// It's conformant with
// https://www.kernel.org/doc/html/v4.14/admin-guide/kernel-parameters.html,
// though making 'var_name' and 'var-name' equivalent may need to be done
// separately.
package cmdline

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/u-root/u-root/pkg/shlex"
)

// CmdLine lets people view the raw & parsed /proc/cmdline in one place
type CmdLine struct {
	Raw   string
	AsMap map[string]string
	Err   error
}

var (
	// procCmdLine package level static variable initialized once
	once        sync.Once
	procCmdLine CmdLine
)

func cmdLineOpener() {
	cmdlineReader, err := os.Open("/proc/cmdline")
	if err != nil {
		errorMsg := fmt.Sprintf("Can't open /proc/cmdline: %v", err)
		log.Print(errorMsg)
		procCmdLine = CmdLine{Err: fmt.Errorf(errorMsg)}
		return
	}

	procCmdLine = parse(cmdlineReader)
	cmdlineReader.Close()
}

// NewCmdLine returns a populated CmdLine struct
func NewCmdLine() CmdLine {
	// We use cmdLineReader so tests can inject here
	once.Do(cmdLineOpener)
	return procCmdLine
}

// FullCmdLine returns the full, raw cmdline string
func FullCmdLine() string {
	once.Do(cmdLineOpener)
	return procCmdLine.Raw
}

// parse returns the current command line, trimmed
func parse(cmdlineReader io.Reader) CmdLine {
	raw, err := ioutil.ReadAll(cmdlineReader)
	line := CmdLine{}
	if err != nil {
		log.Printf("Can't read command line: %v", err)
		line.Err = err
		line.Raw = ""
	} else {
		line.Raw = strings.TrimRight(string(raw), "\n")
		line.AsMap = parseToMap(line.Raw)
	}
	return line
}

//
func doParse(input string, handler func(flag, key, canonicalKey, value, trimmedValue string)) {

	lastQuote := rune(0)
	quotedFieldsCheck := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}

	for _, flag := range strings.FieldsFunc(string(input), quotedFieldsCheck) {
		// kernel variables must allow '-' and '_' to be equivalent in variable
		// names. We will replace dashes with underscores for processing.

		// Split the flag into a key and value, setting value="1" if none
		split := strings.Index(flag, "=")

		if len(flag) == 0 {
			continue
		}
		var key, value string
		if split == -1 {
			key = flag
			value = "1"
		} else {
			key = flag[:split]
			value = flag[split+1:]
		}
		canonicalKey := strings.Replace(key, "-", "_", -1)
		trimmedValue := strings.Trim(value, "\"'")

		// Call the user handler
		handler(flag, key, canonicalKey, value, trimmedValue)
	}

}

// parseToMap turns a space-separated kernel commandline into a map
func parseToMap(input string) map[string]string {

	flagMap := make(map[string]string)
	doParse(input, func(flag, key, canonicalKey, value, trimmedValue string) {
		// We store the value twice, once with dash, once with underscores
		// Just in case people check with the wrong method
		flagMap[canonicalKey] = trimmedValue
		flagMap[key] = trimmedValue
	})

	return flagMap
}

// ContainsFlag verifies that the kernel cmdline has a flag set
func ContainsFlag(flag string) bool {
	once.Do(cmdLineOpener)
	_, present := Flag(flag)
	return present
}

// Flag returns the a flag, and whether it was set
func Flag(flag string) (string, bool) {
	once.Do(cmdLineOpener)
	canonicalFlag := strings.Replace(flag, "-", "_", -1)
	value, present := procCmdLine.AsMap[canonicalFlag]
	return value, present
}

// getFlagMap gets specified flags as a map
func getFlagMap(flagName string) map[string]string {
	return parseToMap(flagName)
}

// GetInitFlagMap gets the init flags as a map
func GetInitFlagMap() map[string]string {
	initflags, _ := Flag("uroot.initflags")
	return getFlagMap(initflags)
}

// GetUinitArgs gets the uinit argvs.
func GetUinitArgs() []string {
	uinitargs, _ := Flag("uroot.uinitargs")
	return shlex.Argv(uinitargs)
}

// FlagsForModule gets all flags for a designated module
// and returns them as a space-seperated string designed to be passed to insmod
// Note that similarly to flags, module names with - and _ are treated the same.
func FlagsForModule(name string) string {
	once.Do(cmdLineOpener)
	var ret string
	flagsAdded := make(map[string]bool) // Ensures duplicate flags aren't both added
	// Module flags come as moduleName.flag in /proc/cmdline
	prefix := strings.Replace(name, "-", "_", -1) + "."
	for flag, val := range procCmdLine.AsMap {
		canonicalFlag := strings.Replace(flag, "-", "_", -1)
		if !flagsAdded[canonicalFlag] && strings.HasPrefix(canonicalFlag, prefix) {
			flagsAdded[canonicalFlag] = true
			// They are passed to insmod space seperated as flag=val
			ret += strings.TrimPrefix(canonicalFlag, prefix) + "=" + val + " "
		}
	}
	return ret
}
