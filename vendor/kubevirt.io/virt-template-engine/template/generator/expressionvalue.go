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
 * Copyright The KubeVirt Authors.
 *
 */

package generator

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	letterCharacters       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitCharacters        = "0123456789"
	alphanumericCharacters = letterCharacters + digitCharacters
	symbolCharacters       = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

	minLength = 1
	maxLength = 255
)

var (
	exprPattern  = regexp.MustCompile(`\[([a-zA-Z0-9\-\\]+)\](\{(\w+)\})`)
	rangePattern = regexp.MustCompile(`(\\w)|(\\d)|(\\a)|(\\A)|(.-.)|([a-zA-Z0-9]+)`)
)

// ExpressionValue implements the Generator interface. It generates
// a random string based on the input expression. The input expression is
// a string with a regex-like syntax, which should follow the form of "[a-zA-Z0-9]{length}".
// The expression defines the range and length of the resulting random characters.
//
// The following character classes are supported in the range:
//
// range | characters
// -------------------------------------------------------------
// "\w"  | abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_
// "\d"  | 0123456789
// "\a"  | abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
// "\A"  | !"#$%&'()*+,-./:;<=>?@[\]^_`{|}~
//
// Generated examples:
//
// expression       | generated value
// ----------------------------------
// "test[0-9]{1}x"  | "test7x"
// "[0-1]{8}"       | "01001100"
// "0x[A-F0-9]{4}"  | "0xB3AF"
// "[a-zA-Z0-9]{8}" | "hW4yQU5i"
type ExpressionValue struct{}

// GenerateValue generates random string based on the input expression.
// The input expression is a pseudo-regex formatted string. See
// ExpressionValue for more details.
func (g ExpressionValue) GenerateValue(s string) (string, error) {
	for {
		idx := exprPattern.FindStringIndex(s)
		if idx == nil {
			break
		}

		expr := s[idx[0]:idx[1]]
		ranges, length, err := extractRangesAndLength(expr)
		if err != nil {
			return "", err
		}

		alphabet, err := getAlphabetFromRanges(ranges)
		if err != nil {
			return "", err
		}

		if s, err = replaceWithGeneratedValue(s, expr, alphabet, length); err != nil {
			return "", err
		}
	}

	return s, nil
}

// extractRangesAndLength extracts the expression's ranges (e.g. [A-Z0-9]) and length
// (eg. {3}). This helper function also validates the expression syntax and
// its length (must be within 1..255).
func extractRangesAndLength(expr string) (ranges string, length int, err error) {
	lengthStart := strings.LastIndex(expr, "{")

	ranges = expr[1 : lengthStart-1]
	if !rangePattern.MatchString(ranges) {
		return "", 0, fmt.Errorf("malformed ranges syntax: %s", ranges)
	}

	length, err = strconv.Atoi(expr[lengthStart+1 : len(expr)-1])
	if err != nil {
		return "", 0, fmt.Errorf("malformed length syntax: %w", err)
	}
	if length < minLength || length > maxLength {
		return "", 0, fmt.Errorf("range must be within [%d-%d] characters: %d", minLength, maxLength, length)
	}

	return ranges, length, nil
}

// getAlphabetFromRanges constructs an alphabet with all characters from the specified ranges.
// Characters in the returned alphabet are deduplicated and sorted.
func getAlphabetFromRanges(ranges string) (string, error) {
	matches := rangePattern.FindAllStringSubmatch(ranges, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("malformed ranges syntax: %s", ranges)
	}

	b := strings.Builder{}
	for _, match := range rangePattern.FindAllStringSubmatch(ranges, -1) {
		var err error
		rangeStr := match[0]
		switch rangeStr {
		case `\w`:
			_, err = b.WriteString(letterCharacters + digitCharacters + "_")
		case `\d`:
			_, err = b.WriteString(digitCharacters)
		case `\a`:
			_, err = b.WriteString(letterCharacters + digitCharacters)
		case `\A`:
			_, err = b.WriteString(symbolCharacters)
		default:
			if len(rangeStr) == 3 && rangeStr[1] == '-' {
				rangeStr, err = subAlphabet(rangeStr[0], rangeStr[2])
				if err != nil {
					return "", err
				}
			}
			_, err = b.WriteString(rangeStr)
		}
		if err != nil {
			return "", err
		}
	}

	return removeDuplicates(b.String()), nil
}

// subAlphabet produces a string that contains all alphanumeric characters within a specified range.
func subAlphabet(from, to byte) (string, error) {
	left := strings.Index(alphanumericCharacters, string(from))
	right := strings.Index(alphanumericCharacters, string(to))
	if left == -1 {
		return "", fmt.Errorf("invalid start character in range: %s", string(from))
	}
	if right == -1 {
		return "", fmt.Errorf("invalid end character in range: %s", string(to))
	}
	if left > right {
		return "", fmt.Errorf("invalid range specified: %s-%s", string(from), string(to))
	}

	return alphanumericCharacters[left : right+1], nil
}

// removeDuplicates removes duplicated characters from the input string
func removeDuplicates(in string) string {
	out := []rune(in)
	slices.Sort(out)
	return string(slices.Compact(out))
}

// replaceWithGeneratedValue replaces the first occurrence of the given expression
// in the input string with random characters of the specified alphabet and length.
func replaceWithGeneratedValue(in, expr, alphabet string, length int) (string, error) {
	var val []byte

	alphabetLen := big.NewInt(int64(len(alphabet)))
	if alphabetLen.Int64() < 1 {
		return "", fmt.Errorf("alphabet cannot be empty: %s", expr)
	}

	for range length {
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		val = append(val, alphabet[idx.Int64()])
	}

	return strings.Replace(in, expr, string(val), 1), nil
}
