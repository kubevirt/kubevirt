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

import "log/slog"

func SlogErr(err error) slog.Attr {
	return slog.Any("err", err)
}

func BodyDiff(diff string) slog.Attr {
	return slog.String(BodyDiffKey, diff)
}

func BodyDump(dump string) slog.Attr {
	return slog.String(BodyDumpKey, dump)
}
