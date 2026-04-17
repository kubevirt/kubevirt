/*
Copyright 2024 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sort"
	"sync"

	"github.com/kr/text"
	"sigs.k8s.io/yaml"
)

// PrettyHandler is a custom handler to print pretty debug logs:
// - Print attributes unquoted
// - Print body.dump and body.diff as sections
//
// Notes on implementation: record in the Handle method contains only attrs from Info/Debug calls,
// other Attrs are stored inside parent Handlers. There is no way to access those attributes
// in a simple manner, e.g. via slog exposed methods.
// Internal slog logic around Attrs includes grouping, preformatting, replacing. It is not simple
// to reimplement it, so lazy JsonHandler workaround is used to re-use this internal machinery
// in exchange to performance. This handler is meant to use for debugging purposes, so it is OK.
//
// For one who brave enough to optimize this Handler, please, please, read these sources thoroughly:
// - https://dusted.codes/creating-a-pretty-console-logger-using-gos-slog-package
// - https://betterstack.com/community/guides/logging/logging-in-go/
// - https://github.com/golang/example/tree/master/slog-handler-guide

const BodyDiffKey = "body.diff"
const BodyDumpKey = "body.dump"

const dateTimeWithSecondsFrac = "2006-01-02 15:04:05.000"

// PrettyHandler is a pretty print handler close to default slog handler.
type PrettyHandler struct {
	jh   slog.Handler
	jhb  *bytes.Buffer
	jhmu *sync.Mutex
	w    io.Writer
	wmu  *sync.Mutex
	opts *slog.HandlerOptions
}

func NewPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	b := &bytes.Buffer{}
	return &PrettyHandler{
		jh: slog.NewJSONHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaultAttrs(opts.ReplaceAttr),
		}),
		jhb:  b,
		jhmu: &sync.Mutex{},
		w:    w,
		wmu:  &sync.Mutex{},
		opts: opts,
	}
}

// Enabled returns if level is enabled for this handler.
func (h *PrettyHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.jh.Enabled(ctx, l)
}

func (h *PrettyHandler) WithAttrs(as []slog.Attr) slog.Handler {
	return &PrettyHandler{
		jh:   h.jh.WithAttrs(as),
		jhb:  h.jhb,
		jhmu: h.jhmu,
		w:    h.w,
		wmu:  h.wmu,
		opts: h.opts,
	}
}

// WithGroup adds group
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{
		jh:   h.jh.WithGroup(name),
		jhb:  h.jhb,
		jhmu: h.jhmu,
		w:    h.w,
		wmu:  h.wmu,
		opts: h.opts,
	}
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	// Get all attributes set by parent Handlers via JsonHandler.
	allAttrs, err := h.gatherAttrs(ctx, r)
	if err != nil {
		return err
	}

	// Separate dumps and other attributes.
	dumps := make(map[string]string)
	groups := make(map[string]any)
	attrs := make([]slog.Attr, 0)
	for attrKey, attr := range allAttrs {
		switch v := attr.(type) {
		case map[string]any, []any:
			groups[attrKey] = v
		case string:
			switch attrKey {
			case BodyDumpKey, BodyDiffKey:
				dumps[attrKey] = v
			default:
				attrs = append(attrs, slog.String(attrKey, v))
			}
		default:
			attrs = append(attrs, slog.Any(attrKey, attr))
		}
	}

	var b bytes.Buffer
	// Write main line: time, level, message and attributes.
	b.WriteString(r.Time.Format(dateTimeWithSecondsFrac))
	b.WriteString(" ")

	b.WriteString(r.Level.String())
	b.WriteString(" ")

	b.WriteString(r.Message)
	b.WriteString(" ")

	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Key < attrs[j].Key
	})
	for i, attr := range attrs {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(attr.Key)
		b.WriteString("=\"")
		b.WriteString(attr.Value.String())
		b.WriteString("\"")
	}
	ensureEndingNewLine(&b)

	if h.opts != nil && h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		b.WriteString(fmt.Sprintf("  source=%s:%d %s\n", f.File, f.Line, f.Function))
	}

	// Add sectioned info: grouped attributes, a body diff and a body dump.
	if len(groups) > 0 {
		groupsBytes, err := yaml.Marshal(groups)
		if err != nil {
			return fmt.Errorf("error marshaling grouped attrs: %w", err)
		}
		//b.WriteString("Grouped attrs:\n")
		b.Write(text.IndentBytes(groupsBytes, []byte("  ")))
		ensureEndingNewLine(&b)
	}

	for _, dumpName := range []string{BodyDumpKey, BodyDiffKey} {
		if diff, ok := dumps[dumpName]; ok {
			b.WriteString(fmt.Sprintf("  %s:\n", dumpName))
			b.WriteString(text.Indent(diff, "    "))
			ensureEndingNewLine(&b)
		}
	}

	//if diff, ok := dumps[BodyDiffKey]; ok {
	//	b.WriteString("  body.diff:\n")
	//	b.WriteString(text.Indent(diff, "    "))
	//	ensureEndingNewLine(&b)
	//}
	//
	//if dump, ok := dumps[BodyDumpKey]; ok {
	//	b.WriteString("  body.dump:\n")
	//	b.WriteString(text.Indent(dump, "    "))
	//	ensureEndingNewLine(&b)
	//}

	// Use Mutex to sync access to the shared Writer.
	h.wmu.Lock()
	defer h.wmu.Unlock()
	_, err = b.WriteTo(h.w)
	return err
}

func ensureEndingNewLine(buf *bytes.Buffer) {
	last := string(buf.Bytes()[buf.Len()-1:])
	if last != "\n" {
		buf.WriteString("\n")
	}
}

func (h *PrettyHandler) gatherAttrs(ctx context.Context, r slog.Record) (map[string]any, error) {
	h.jhmu.Lock()
	defer func() {
		h.jhb.Reset()
		h.jhmu.Unlock()
	}()
	if err := h.jh.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(h.jhb.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func suppressDefaultAttrs(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey ||
			a.Key == slog.SourceKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}
