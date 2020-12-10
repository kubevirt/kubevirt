// Copyright 2017-2020 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package shlex is a simplified command-line shell-like argument parser.
//
// shlex will parse for example
//
//     start --append="foobar foobaz" --nogood 'food'
//
// into the appropriate argvs to start the command.
package shlex

func isWhitespace(b byte) bool {
	return b == '\t' || b == '\n' || b == '\v' ||
		b == '\f' || b == '\r' || b == ' '
}

type quote uint8

const (
	unquoted quote = iota
	escape
	singleQuote
	doubleQuote
	doubleQuoteEscape
	comment
)

// Argv splits a command line according to usual simple shell rules.
//
// Argv was written from the spec of Grub quoting at
// https://www.gnu.org/software/grub/manual/grub/grub.html#Quoting
// except that the escaping of newline is not supported
func Argv(s string) []string {
	var ret []string
	var token []byte

	var context quote
	lastWhiteSpace := true
	for i := range []byte(s) {
		quotes := context != unquoted
		switch context {
		case unquoted:
			switch s[i] {
			case '\\':
				context = escape
				// strip out the quote
				continue
			case '\'':
				context = singleQuote
				// strip out the quote
				continue
			case '"':
				context = doubleQuote
				// strip out the quote
				continue
			case '#':
				if lastWhiteSpace {
					context = comment
					// strip out the rest
					continue
				}
			}

		case escape:
			context = unquoted

		case singleQuote:
			if s[i] == '\'' {
				context = unquoted
				// strip out the quote
				continue
			}

		case doubleQuote:
			switch s[i] {
			case '\\':
				context = doubleQuoteEscape
				// strip out the quote
				continue
			case '"':
				context = unquoted
				// strip out the quote
				continue
			}

		case doubleQuoteEscape:
			switch s[i] {
			case '$', '"', '\\', '\n': // or newline
			default:
				token = append(token, '\\')
			}

			context = doubleQuote

		case comment:
			// should end on newline

			// strip out the rest
			continue

		}

		lastWhiteSpace = isWhitespace(s[i])

		if !isWhitespace(s[i]) || quotes {
			token = append(token, s[i])
		} else if len(token) > 0 {
			ret = append(ret, string(token))
			token = token[:0]
		}
	}

	if len(token) > 0 {
		ret = append(ret, string(token))
	}
	return ret
}
