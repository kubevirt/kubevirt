// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// Package editorconfig allows parsing and using EditorConfig files, as defined
// in https://editorconfig.org/.
package editorconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const DefaultName = ".editorconfig"

// File is an EditorConfig file with a number of sections.
type File struct {
	Root     bool
	Sections []Section
}

// Section is a single EditorConfig section, which applies a number of
// properties to the filenames matching it.
type Section struct {
	// Name is the section's name. Usually, this will be a valid pattern
	// matching string, such as "[*.go]" without the square brackets.
	//
	// It may also describe a language such as "[[shell]]",
	// although this is an out-of-spec feature that may be changed at any time.
	Name string

	// Properties is the list of name-value properties contained by a
	// section. It is kept in increasing order, to allow binary searches.
	Properties []Property

	// TODO: properties are not actually kept in increasing order
}

// Property is a single property with a name and a value, which can be
// represented as a single line like "indent_size=8".
type Property struct {
	// Name is always lowercase and allows identifying a property.
	Name string
	// Value holds data for a property.
	Value string
}

// String turns a property into its INI format.
func (p Property) String() string { return fmt.Sprintf("%s=%s", p.Name, p.Value) }

// String turns a file into its INI format.
func (f *File) String() string {
	var b strings.Builder
	if f.Root {
		fmt.Fprintf(&b, "root=true\n\n")
	}
	for i, section := range f.Sections {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		fmt.Fprintf(&b, "[%s]\n", section.Name)
		for _, prop := range section.Properties {
			fmt.Fprintf(&b, "%s=%s\n", prop.Name, prop.Value)
		}
	}
	return b.String()
}

// Lookup finds a property by its name within a section and returns a pointer to
// it, or nil if no such property exists.
//
// Note that most of the time, Get should be used instead.
func (s Section) Lookup(name string) *Property {
	// TODO: binary search
	for i, prop := range s.Properties {
		if prop.Name == name {
			return &s.Properties[i]
		}
	}
	return nil
}

// Get returns the value of a property found by its name. If no such property
// exists, an empty string is returned.
func (s Section) Get(name string) string {
	if prop := s.Lookup(name); prop != nil {
		return prop.Value
	}
	return ""
}

// IndentSize is a shortcut for Get("indent_size") as an int.
func (s Section) IndentSize() int {
	n, _ := strconv.Atoi(s.Get("indent_size"))
	return n
}

// IndentSize is a shortcut for Get("trim_trailing_whitespace") as a bool.
func (s Section) TrimTrailingWhitespace() bool {
	return s.Get("trim_trailing_whitespace") == "true"
}

// IndentSize is a shortcut for Get("insert_final_newline") as a bool.
func (s Section) InsertFinalNewline() bool {
	return s.Get("insert_final_newline") == "true"
}

// IndentSize is similar to Get("indent_size"), but it handles the "tab" default
// and returns an int. When unset, it returns 0.
func (s Section) TabWidth() int {
	value := s.Get("indent_size")
	if value == "tab" {
		value = s.Get("tab_width")
	}
	n, _ := strconv.Atoi(value)
	return n
}

// Add introduces a number of properties to the section. Properties that were
// already part of the section are ignored.
func (s *Section) Add(properties ...Property) {
	for _, prop := range properties {
		if s.Lookup(prop.Name) == nil {
			s.Properties = append(s.Properties, prop)
		}
	}
}

// String turns a section into its INI format.
func (s Section) String() string {
	var b strings.Builder
	if s.Name != "" {
		fmt.Fprintf(&b, "[%s]\n", s.Name)
	}
	for _, prop := range s.Properties {
		fmt.Fprintf(&b, "%s=%s\n", prop.Name, prop.Value)
	}
	return b.String()
}

// Filter returns the set of properties in f which apply to a file
// given its name and optional languages.
// Properties from later sections take precedence. The name should be a path
// relative to the directory holding the EditorConfig.
//
// If cache is non-nil, the map will be used to reuse patterns translated and
// compiled to regular expressions.
//
// Note that this function doesn't apply defaults; for that, see Find.
//
// Note that, since the EditorConfig spec doesn't allow backslashes as path
// separators, backslashes in name are converted to forward slashes.
func (f *File) Filter(name string, languages []string, cache map[string]*regexp.Regexp) Section {
	name = filepath.ToSlash(name)
	result := Section{}
	for i := len(f.Sections) - 1; i >= 0; i-- {
		section := f.Sections[i]

		if len(section.Name) > 2 && section.Name[0] == '[' && section.Name[len(section.Name)-1] == ']' {
			sectionLang := section.Name[1 : len(section.Name)-1]
			for _, language := range languages {
				if language == sectionLang {
					result.Add(section.Properties...)
					break
				}
			}
			continue
		}

		rx := cache[section.Name]
		if rx == nil {
			rx = toRegexp(section.Name)
			if cache != nil {
				cache[section.Name] = rx
			}
		}
		if rx.MatchString(name) {
			result.Add(section.Properties...)
		}
	}
	return result
}

// Find figures out the properties that apply to a file name on disk, and
// returns them as a section. The name doesn't need to be an absolute path.
//
// It is equivalent to Query{}.Find; please note that no caching at all takes
// place in this mode.
func Find(name string, languages []string) (Section, error) {
	return Query{}.Find(name, languages)
}

// Query allows fine-grained control of how EditorConfig files are found and
// used. It also attempts to cache and reuse work, which makes its Find method
// significantly faster when used on many files.
type Query struct {
	// ConfigName specifies what EditorConfig file name to use when
	// searching for files on disk. If empty, it defaults to DefaultName.
	ConfigName string

	// FileCache keeps track of which directories are known to contain an
	// EditorConfig. Existing entries which are nil mean that the directory
	// is known to not contain an EditorConfig.
	//
	// If nil, no caching takes place.
	FileCache map[string]*File

	// RegexpCache keeps track of patterns which have already been
	// translated to a regular expression and compiled, to save repeating
	// the work.
	//
	// If nil, no caching takes place.
	RegexpCache map[string]*regexp.Regexp

	// Version specifies an EditorConfig version to use when applying its
	// spec. When empty, it defaults to the latest version. This field
	// should generally be left untouched.
	Version string
}

// Find figures out the properties that apply to a file on disk
// given its name and languages, returns them as a section.
// The name doesn't need to be an absolute path.
//
// Any relevant EditorConfig files are parsed and used as necessary. Parsing the
// files can be cached in Query.
//
// The defaults for supported properties are applied before returning.
func (q Query) Find(name string, languages []string) (Section, error) {
	name, err := filepath.Abs(name)
	if err != nil {
		return Section{}, err
	}
	configName := q.ConfigName
	if configName == "" {
		configName = DefaultName
	}

	result := Section{}
	dir := name
	for {
		if d := filepath.Dir(dir); d != dir {
			dir = d
		} else {
			break
		}
		file, e := q.FileCache[dir]
		if !e {
			// TODO: replace with io/fs
			f, err := os.Open(filepath.Join(dir, configName))
			if os.IsNotExist(err) {
				// continue below, caching the nil file
			} else if err != nil {
				return Section{}, err
			} else {
				var err error
				file, err = Parse(f)
				f.Close()
				if err != nil {
					return Section{}, err
				}
			}
			if q.FileCache != nil {
				q.FileCache[dir] = file
			}
		}
		if file == nil {
			continue
		}
		relative := name[len(dir)+1:]
		result.Add(file.Filter(relative, languages, q.RegexpCache).Properties...)
		if file.Root {
			break
		}
	}

	if result.Get("indent_style") == "tab" {
		if value := result.Get("tab_width"); value != "" {
			// When indent_style is "tab" and tab_width is set,
			// indent_size should default to tab_width.
			result.Add(Property{Name: "indent_size", Value: value})
		}
		if q.Version != "" && q.Version < "0.9.0" { // TODO: semver comparison?
		} else if result.Get("indent_size") == "" {
			// When indent_style is "tab", indent_size defaults to
			// "tab". Only on 0.9.0 and later.
			result.Add(Property{Name: "indent_size", Value: "tab"})
		}
	} else if result.Get("tab_width") == "" {
		if value := result.Get("indent_size"); value != "" && value != "tab" {
			// tab_width defaults to the value of indent_size.
			result.Add(Property{Name: "tab_width", Value: value})
		}
	}
	return result, nil
}

// Bundle mvdan.cc/sh/v3/pattern into pattern_bundle.go,
// since mvdan.cc/sh/v3/cmd/shfmt depends on this module
// and we don't want to end up with circular module dependencies.
// This should be fine, as the package is small, and the toolchain can omit what is unused.
// Note that we can't use @version on the sh/v3 module, so we automatically pull @latest via go.mod.

func toRegexp(pat string) *regexp.Regexp {
	if i := strings.IndexByte(pat, '/'); i == 0 {
		pat = pat[1:]
	} else if i < 0 {
		pat = "**/" + pat
	}
	rxStr, err := patternRegexp(pat, patternFilenames|patternBraces|patternEntireString)
	if err != nil {
		panic(err)
	}
	return regexp.MustCompile(rxStr)
}

func Parse(r io.Reader) (*File, error) {
	f := &File{}
	scanner := bufio.NewScanner(r)
	var section *Section
	for scanner.Scan() {
		line := scanner.Text()
		if i := strings.Index(line, " #"); i >= 0 {
			line = line[:i]
		} else if i := strings.Index(line, " ;"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)

		if len(line) > 2 && line[0] == '[' && line[len(line)-1] == ']' {
			name := line[1 : len(line)-1]
			if len(name) > 4096 {
				section = &Section{} // ignore
				continue
			}
			f.Sections = append(f.Sections, Section{Name: name})
			section = &f.Sections[len(f.Sections)-1]
			continue
		}
		i := strings.IndexAny(line, "=:")
		if i < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		value := strings.TrimSpace(line[i+1:])
		switch key {
		case "root", "indent_style", "indent_size", "tab_width", "end_of_line",
			"charset", "trim_trailing_whitespace", "insert_final_newline":
			value = strings.ToLower(value)
		}
		// The spec tests require supporting at least these lengths.
		// Larger lengths rarely make sense,
		// and they could mean holding onto lots of memory,
		// so use them as limits.
		if len(key) > 1024 || len(value) > 4096 {
			continue
		}
		if section != nil {
			section.Add(Property{Name: key, Value: value})
		} else if key == "root" {
			f.Root = value == "true"
		}
	}
	return f, nil
}
