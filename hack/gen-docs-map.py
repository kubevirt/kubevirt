#!/usr/bin/env python3
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Generates docs_map.md, an index of all documentation files under docs/
# grouped by directory, and AGENTS.md, a thin pointer to that map. The two
# files are the entry points to KubeVirt's documentation for humans and AI
# tooling alike.
#
# AGENTS.md deliberately carries no content of its own; project knowledge
# belongs in documentation files under docs/, linked from docs_map.md.
#
# Usage:
#   hack/gen-docs-map.py           regenerate docs_map.md and AGENTS.md
#   hack/gen-docs-map.py --verify  exit non-zero if either file is stale

import difflib
import re
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
DOCS_DIR = ROOT_DIR / "docs"
MAP_FILE = ROOT_DIR / "docs_map.md"
AGENTS_FILE = ROOT_DIR / "AGENTS.md"

HEADER = """\
<!-- GENERATED FILE - DO NOT EDIT. Regenerate with hack/gen-docs-map.py (or make generate). -->

# KubeVirt Documentation Map

Index of all documentation files under docs/, grouped by directory.
"""

AGENTS_CONTENT = """\
<!-- GENERATED FILE - DO NOT EDIT. This file intentionally stays a thin
pointer to the documentation map; put actual content under docs/ and link it
from docs_map.md instead. Regenerate with hack/gen-docs-map.py. -->

Documentation about this repository can be found in [docs_map.md](docs_map.md).
"""

# Hint printed when a generated file does not match its committed version.
STALE_HINTS = {
    MAP_FILE: "Run hack/gen-docs-map.py (or make generate) and commit the result.",
    AGENTS_FILE: "AGENTS.md must remain a thin pointer to the documentation map; add content under docs/ and link it from docs_map.md instead.",
}

# Documentation file names must stay within this set of characters so that
# every map entry is a well-formed markdown link.
SAFE_NAME = re.compile(r"^[A-Za-z0-9._/-]+$")


def fail(message):
    """Print an error message to stderr and exit with status 1."""
    print(f"ERROR: {message}", file=sys.stderr)
    sys.exit(1)


def map_directories():
    """Return docs/ itself followed by its subdirectories, in sorted order.

    Hidden directories (any path component starting with a dot) are skipped.
    """
    subdirs = sorted(
        (
            path
            for path in DOCS_DIR.rglob("*")
            if path.is_dir()
            and not path.is_symlink()
            and not any(part.startswith(".") for part in path.relative_to(DOCS_DIR).parts)
        ),
        key=lambda p: p.as_posix(),
    )

    return [DOCS_DIR] + subdirs


def generate_map():
    """Return the full contents of docs_map.md as a string."""
    lines = HEADER.splitlines()

    for directory in map_directories():
        docs = sorted(
            (
                path
                for path in directory.iterdir()
                if path.is_file() and path.suffix == ".md" and not path.name.startswith(".")
            ),
            key=lambda p: p.as_posix(),
        )

        if not docs:
            continue

        lines.append("")
        lines.append(f"## {directory.relative_to(ROOT_DIR).as_posix()}")
        lines.append("")

        for doc in docs:
            rel = doc.relative_to(ROOT_DIR).as_posix()

            if not SAFE_NAME.match(rel):
                fail(f"{rel}: unsupported characters in documentation file name")

            lines.append(f"- [{rel.removeprefix('docs/')}]({rel})")

    return "\n".join(lines) + "\n"


def verify_files(files):
    """Compare every committed file against its fresh generation."""
    for path, generated in files.items():
        if not path.is_file():
            fail(f"{path} does not exist; run hack/gen-docs-map.py to generate it")

        try:
            committed = path.read_text(encoding="utf-8")

        except UnicodeDecodeError:
            fail(f"{path} is not valid UTF-8; regenerate it with hack/gen-docs-map.py")

        if committed == generated:
            print(f"{path} is up to date")
            continue

        sys.stdout.writelines(
            difflib.unified_diff(
                committed.splitlines(keepends=True),
                generated.splitlines(keepends=True),
                fromfile=str(path),
                tofile="regenerated",
            )
        )

        fail(f"{path} is out of date. {STALE_HINTS[path]}")


def write_files(files):
    """Write every generated file to disk."""
    for path, content in files.items():
        with path.open("w", encoding="utf-8", newline="\n") as f:
            f.write(content)


def main():
    args = sys.argv[1:]

    if len(args) > 1 or (args and args[0] != "--verify"):
        fail("usage: hack/gen-docs-map.py [--verify]")

    if not DOCS_DIR.is_dir():
        fail(f"{DOCS_DIR} not found")

    files = {
        MAP_FILE: generate_map(),
        AGENTS_FILE: AGENTS_CONTENT,
    }

    if args:
        verify_files(files)
    else:
        write_files(files)

    return 0


if __name__ == "__main__":
    sys.exit(main())
