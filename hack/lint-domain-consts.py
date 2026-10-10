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
"""Reject exported constants declared outside a domain but used only within it."""

import argparse
import re
import subprocess
import sys
from pathlib import Path

CONST_LINE = re.compile(r"^\s*([A-Z][A-Za-z0-9_]*)\s*=")
NOLINT = re.compile(r"//nolint:(?:[^\n]*,)?domain-consts(?:[\s,]|$)")
EXPORTED_IDENTIFIER = re.compile(r"\b[A-Z][A-Za-z0-9_]*\b")


def go_files(paths: list[str]) -> list[str]:
    files = []
    for raw in paths:
        path = Path(raw)
        if path.is_file() and path.suffix == ".go":
            files.append(path.as_posix())
        elif path.is_dir():
            files.extend(go.as_posix() for go in path.rglob("*.go") if "vendor" not in go.parts)
    return files


def exported_consts(path: Path) -> list[str]:
    names = []
    in_block = False
    for line in path.read_text(errors="replace").splitlines():
        stripped = line.strip()
        if stripped.startswith("const ("):
            in_block = True
            continue
        if in_block and stripped.startswith(")"):
            in_block = False
            continue
        candidate = stripped.removeprefix("const ") if stripped.startswith("const ") and "(" not in stripped else line
        if (in_block or stripped.startswith("const ")) and not NOLINT.search(line):
            match = CONST_LINE.match(candidate)
            if match:
                names.append(match.group(1))
    return names


def in_domain(path: str, domain: str) -> bool:
    return domain in Path(path).parts


def all_go_files() -> list[str]:
    proc = subprocess.run(
        ["git", "ls-files", "-z", "--", "*.go", ":(exclude)vendor/**"],
        text=True,
        capture_output=True,
    )
    if proc.returncode != 0:
        sys.stderr.write(proc.stderr)
        sys.exit(proc.returncode)
    return [path for path in proc.stdout.split("\0") if path]


def usage_index(names: set[str]) -> dict[str, list[str]]:
    usages = {name: [] for name in names}
    for filename in all_go_files():
        tokens = set(EXPORTED_IDENTIFIER.findall(Path(filename).read_text(errors="replace")))
        for name in names & tokens:
            usages[name].append(filename)
    return usages


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("paths", nargs="+", help="Go files or directories to scan")
    parser.add_argument("--domain", required=True, help="Path component owning the constants")
    args = parser.parse_args()

    definitions = []
    for rel in go_files(args.paths):
        if in_domain(rel, args.domain):
            continue
        for name in exported_consts(Path(rel)):
            definitions.append((rel, name))

    usages = usage_index({name for _, name in definitions})
    findings = []
    for rel, name in definitions:
        hits = [hit for hit in usages[name] if hit != rel]
        if hits and all(in_domain(hit, args.domain) for hit in hits):
            findings.append((rel, name, hits))

    if not findings:
        print(f"domain-consts: no {args.domain}-only constants outside {args.domain}/")
        return 0
    print(f"domain-consts: {len(findings)} constant(s) should move into {args.domain}/")
    for defined_in, name, hits in findings:
        print(f"\n{defined_in}\n  {name}\n  used only in {args.domain}; move it into {args.domain} or add //nolint:domain-consts")
        for hit in hits:
            print(f"    {hit}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
