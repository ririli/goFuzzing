#!/usr/bin/env python3
"""Render a race-baseline result dir into a human-readable REPORT.md.

Reads package_logs/*.json (go test -json), deduplicates data races by the pair
of project-code access sites, separates real panics from the `test timed out`
abortions caused by the per-package gate, and lists packages that did not
finish all --count rounds.

Usage:
  python3 race_digest.py RESULT_DIR PROJECT_MARKER
"""
import argparse
import os
import re
import sys
from collections import Counter

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from decode_test_json import blocks  # noqa: E402

ACCESS = re.compile(r"^\s*(Read|Write|Previous read|Previous write) at (0x[0-9a-f]+) by goroutine (\d+)")
FRAME = re.compile(r"^\s+(\S+\.go:\d+)")
TIMEOUT_MARK = "panic: test timed out after"
NOISE = ("/runtime/", "testing.go", "toolchain", "/reflect/")


def signature(lines, marker):
    """(site_a, site_b) of the two racing accesses, in project code."""
    sides = []
    want = False
    for ln in lines:
        m = ACCESS.match(ln)
        if m:
            want = True
            continue
        if not want:
            continue
        f = FRAME.match(ln)
        if not f:
            continue
        p = f.group(1)
        if any(n in p for n in NOISE):
            continue
        if marker in p:
            sides.append(p)
            want = False
    if len(sides) < 2:
        sides.append("<second-site-outside-project>")
    return tuple(sorted(sides[:2]))


def summary_head(res_dir):
    rows, stats = [], []
    with open(os.path.join(res_dir, "summary.txt"), errors="replace") as fh:
        for ln in fh:
            if ln.startswith("  ") and ":" in ln:
                k, v = ln.split(":", 1)
                stats.append("%-18s %s" % (k.strip(), v.strip()))
            elif ln.lstrip().startswith("Package"):
                continue
            elif ln.strip() and not ln.startswith(" ") and set(ln.strip()) != {"="}:
                p = ln.split()
                if len(p) == 5 and p[1] in ("PASS", "FAIL"):
                    rows.append(p)
    return stats, rows


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("res_dir")
    ap.add_argument("marker", help="substring identifying project frames, e.g. fabio")
    args = ap.parse_args()

    logs = os.path.join(args.res_dir, "package_logs")
    groups = {}
    real_panics = []
    fake_timeout = Counter()
    for name in sorted(os.listdir(logs)):
        if not name.endswith(".json"):
            continue
        pkg = name.split("__", 1)[-1][:-5] if "__" in name else name[:-5]
        path = os.path.join(logs, name)
        for test, lines in blocks(path, "race"):
            key = signature(lines, args.marker)
            g = groups.setdefault(key, {"n": 0, "pkg": pkg, "test": test, "lines": lines})
            g["n"] += 1
        for test, lines in blocks(path, "panic"):
            if any(TIMEOUT_MARK in l for l in lines):
                fake_timeout[pkg] += 1
                continue
            real_panics.append({"head": "%s  [%s]" % (test or "-", pkg), "body": lines[:14]})

    stats, rows = summary_head(args.res_dir)
    out = ["# race 基线可读报告 — %s" % os.path.basename(args.res_dir), ""]
    out += ["```"] + stats + ["```", ""]
    out += ["- 去重后数据竞争对数: **%d**（原始 WARNING 块 %d）"
            % (len(groups), sum(g["n"] for g in groups.values()))]
    out += ["- 真实 panic 条目: **%d**；`test timed out` 假 panic（25min 闸截断所致）: **%d**"
            % (len(real_panics), sum(fake_timeout.values()))]
    if fake_timeout:
        out += ["  - 未跑满 30 轮的包（检出数欠采样）: %s" % ", ".join(sorted(fake_timeout))]
    out += [""]

    def render(title, items, limit=12):
        out.append("## %s (%d)\n" % (title, len(items)))
        for i, it in enumerate(items[:limit]):
            out.append("### %d. %s" % (i + 1, it["head"]))
            out.extend(["```"] + it["body"] + ["```", ""])
        if len(items) > limit:
            out.append("_另有 %d 条未展开_\n" % (len(items) - limit))

    race_items = [{"head": "%s <-> %s  ×%d  [%s]  test=%s"
                          % (k[0], k[1], v["n"], v["pkg"], v["test"] or "-"),
                   "body": v["lines"][:26]}
                  for k, v in sorted(groups.items(), key=lambda x: -x[1]["n"])]
    render("数据竞争（按访问点对去重）", race_items)
    render("真实 panic", real_panics)

    dirty = [r for r in rows if r[1] == "FAIL" or r[2] != "0" or r[3] != "0"]
    out += ["## 非干净包 (%d)\n" % len(dirty),
            "| Package | Result | Races | Panics | Seconds |", "|---|---|---|---|---|"]
    out += ["| %s | %s | %s | %s | %s |" % tuple(r) for r in dirty]

    dst = os.path.join(args.res_dir, "REPORT.md")
    with open(dst, "w") as fh:
        fh.write("\n".join(out) + "\n")
    print("wrote %s (%d bytes)" % (dst, os.path.getsize(dst)))


if __name__ == "__main__":
    main()
