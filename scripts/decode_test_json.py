#!/usr/bin/env python3
"""Decode `go test -json` output into plain text blocks.

go test -json puts every output line into an {"Output": "..."} record, which is
unusable as a human-facing report. This module turns those records back into
text and splits out the two kinds of findings the race baseline cares about.

CLI:
  python3 decode_test_json.py [--kind race|panic|all] FILE [FILE...]

Importable:
  blocks(path, kind) -> [(test, [text_line, ...]), ...]
"""
import argparse
import json
import re
import sys

RACE_START = "WARNING: DATA RACE"
RACE_END = "=================="
PANIC_START = re.compile(r"^(panic: |fatal error: )")
TIMEOUT_MARK = "panic: test timed out after"
PANIC_BLOCK_LINES = 40


def records(path):
    """Yield (test, text) for each output line; non-json lines pass through."""
    with open(path, errors="replace") as fh:
        for line in fh:
            try:
                rec = json.loads(line)
            except Exception:
                yield "", line.rstrip("\n")
                continue
            out = rec.get("Output")
            if out is not None:
                yield rec.get("Test") or "", out.rstrip("\n")


def blocks(path, kind="all"):
    """Return [(test, lines)] for race and/or panic findings in one log."""
    found = {"race": [], "panic": []}
    cur = None
    for test, text in records(path):
        if RACE_START in text:
            cur = ("race", test, [text])
            found["race"].append(cur)
            continue
        if PANIC_START.match(text):
            cur = ("panic", test, [text])
            found["panic"].append(cur)
            continue
        if cur is None:
            continue
        cur[2].append(text)
        if cur[0] == "race" and text.strip() == RACE_END:
            cur = None
        elif cur[0] == "panic" and len(cur[2]) >= PANIC_BLOCK_LINES:
            cur = None
    picked = found[kind] if kind != "all" else found["race"] + found["panic"]
    return [(test, lines) for _kind, test, lines in picked]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--kind", choices=("race", "panic", "all"), default="all")
    ap.add_argument("files", nargs="+")
    args = ap.parse_args()
    for p in args.files:
        for _test, lines in blocks(p, args.kind):
            sys.stdout.write("\n".join(lines) + "\n")


if __name__ == "__main__":
    main()
