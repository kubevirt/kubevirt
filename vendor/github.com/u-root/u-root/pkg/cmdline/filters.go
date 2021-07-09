// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmdline

import (
	"fmt"
	"strings"
)

// RemoveFilter filters out variable for a given space-separated kernel commandline
func removeFilter(input string, variables []string) string {
	var newCl []string

	// kernel variables must allow '-' and '_' to be equivalent in variable
	// names. We will replace dashes with underscores for processing as
	// `doParse` is doing.
	for i, v := range variables {
		variables[i] = strings.Replace(v, "-", "_", -1)
	}

	doParse(input, func(flag, key, canonicalKey, value, trimmedValue string) {
		skip := false
		for _, v := range variables {
			if canonicalKey == v {
				skip = true
				break
			}
		}
		if skip {
			return
		}
		newCl = append(newCl, flag)
	})
	return strings.Join(newCl, " ")
}

// Filter represents and kernel commandline filter
type Filter interface {
	// Update filters a given space-separated kernel commandline
	Update(cmdline string) string
}

type updater struct {
	appendCmd string
	removeVar []string
	reuseVar  []string
}

// NewUpdateFilter return a kernel command line Filter that:
// removes variables listed in 'removeVar',
// append extra parameters from the 'appendCmd' and
// append variables listed in 'reuseVar' using the value from the running kernel
func NewUpdateFilter(appendCmd string, removeVar, reuseVar []string) Filter {
	return &updater{
		appendCmd: appendCmd,
		removeVar: removeVar,
		reuseVar:  reuseVar,
	}
}

func (u *updater) Update(cmdline string) string {
	acl := ""
	if len(u.appendCmd) > 0 {
		acl = " " + u.appendCmd
	}
	for _, f := range u.reuseVar {
		value, present := Flag(f)
		if present {
			acl = fmt.Sprintf("%s %s=%s", acl, f, value)
		}
	}

	return removeFilter(cmdline, u.removeVar) + acl
}
