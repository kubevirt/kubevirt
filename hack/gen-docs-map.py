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
# grouped by directory. It serves as the entry point to KubeVirt's
# documentation for humans and AI tooling alike.
#
# Usage:
#   hack/gen-docs-map.py           regenerate docs_map.md
#   hack/gen-docs-map.py --verify  exit non-zero if docs_map.md is stale

import difflib
import re
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parent.parent
DOCS_DIR = ROOT_DIR / "docs"
MAP_FILE = ROOT_DIR / "docs_map.md"

HEADER = """\
<!-- GENERATED FILE - DO NOT EDIT. Regenerate with hack/gen-docs-map.py (or make generate). -->

# KubeVirt Documentation Map

Index of all documentation files under docs/, grouped by directory.
"""

# Hint printed when docs_map.md does not match its committed version.
STALE_HINT = "Run hack/gen-docs-map.py (or make generate) and commit the result."

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


def verify_map():
    """Compare the committed docs_map.md against its fresh generation."""
    generated = generate_map()

    if not MAP_FILE.is_file():
        fail(f"{MAP_FILE} does not exist; run hack/gen-docs-map.py to generate it")

    try:
        committed = MAP_FILE.read_text(encoding="utf-8")

    except UnicodeDecodeError:
        fail(f"{MAP_FILE} is not valid UTF-8; regenerate it with hack/gen-docs-map.py")

    if committed == generated:
        print(f"{MAP_FILE} is up to date")
        return

    sys.stdout.writelines(
        difflib.unified_diff(
            committed.splitlines(keepends=True),
            generated.splitlines(keepends=True),
            fromfile=str(MAP_FILE),
            tofile="regenerated",
        )
    )

    fail(f"{MAP_FILE} is out of date. {STALE_HINT}")


def write_map():
    """Write the generated map to docs_map.md."""
    with MAP_FILE.open("w", encoding="utf-8", newline="\n") as f:
        f.write(generate_map())


def main():
    args = sys.argv[1:]

    if len(args) > 1 or (args and args[0] != "--verify"):
        fail("usage: hack/gen-docs-map.py [--verify]")

    if not DOCS_DIR.is_dir():
        fail(f"{DOCS_DIR} not found")

    if args:
        verify_map()
    else:
        write_map()

    return 0


if __name__ == "__main__":
    sys.exit(main())
