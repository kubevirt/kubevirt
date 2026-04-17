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

package proxy

import (
	"context"
	"log/slog"

	"github.com/deckhouse/kube-api-rewriter/pkg/labels"
)

func LoggerWithCommonAttrs(ctx context.Context, attrs ...any) *slog.Logger {
	logger := slog.Default()
	logger = logger.With(
		slog.String("proxy.name", labels.NameFromContext(ctx)),
		slog.String("resource", labels.ResourceFromContext(ctx)),
		slog.String("method", labels.MethodFromContext(ctx)),
		slog.String("watch", labels.WatchFromContext(ctx)),
	)
	return logger.With(attrs...)
}
