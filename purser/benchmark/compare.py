#!/usr/bin/env python3
"""Diff two purser benchmark JSON snapshots from PURSER_BENCH_SNAPSHOT captures.

Snapshots are written by TestBenchmarkSnapshot in snapshot_test.go. Each file lists
size=256 hot-path benchmarks with ns_per_op, bytes_per_op, and allocs_per_op.

Output modes
------------
Default (always printed):
    One row per benchmark. Three columns show only the *change* since the prior
    capture. Unchanged metrics print "=" so noise stays low.

Verbose (-v / --verbose, printed after the compact table):
    One row per metric with absolute old and new values plus the same change
    string used in compact mode. Benchmark name appears only on the first metric
    row for that benchmark.

File selection
--------------
    python3 compare.py
        Compare results/previous.json (rotated before each capture) against the
        newest *.json in results/ (excluding previous.json).

    python3 compare.py path/to/new.json
        Compare previous.json against the given new snapshot.

    python3 compare.py path/to/old.json path/to/new.json
        Compare two explicit snapshot files.

Flags -v and --verbose are ignored when resolving file paths (they do not start
with a hyphen in a way that would be mistaken for a path; only "-v" is filtered).
"""

from __future__ import annotations

import glob
import json
import os
import sys

# -----------------------------------------------------------------------------
# Paths and column labels
# -----------------------------------------------------------------------------

# Directory next to this script where snapshot_test.go writes JSON (gitignored).
RESULTS = os.path.join(os.path.dirname(__file__), "results")

# Keys in each benchmark object inside the JSON; order matches snapshot_test.go.
METRICS = ("ns_per_op", "bytes_per_op", "allocs_per_op")

# Compact-table column headers (deltas only).
METRIC_COL = {
    "ns_per_op": "Δ ns/op",
    "bytes_per_op": "Δ B/op",
    "allocs_per_op": "Δ allocs",
}

# Verbose-table metric column (absolute old/new values).
METRIC_LABEL = {
    "ns_per_op": "ns/op",
    "bytes_per_op": "B/op",
    "allocs_per_op": "allocs/op",
}

# Only show percent change when |pct| is at least this (avoids "+0.0%" clutter).
PCT_MIN = 0.1


# -----------------------------------------------------------------------------
# Loading snapshots and resolving CLI paths
# -----------------------------------------------------------------------------


def load_snapshot(path: str) -> tuple[dict, dict]:
    """Load one snapshot file into a bench map and a small metadata dict.

    Args:
        path: Filesystem path to a JSON file from PURSER_BENCH_SNAPSHOT.

    Returns:
        benches: Maps full benchmark name (e.g. "Locker/Seal/size=256") to its
            entry dict with ns_per_op, bytes_per_op, allocs_per_op.
        meta: captured_at, go_version, goos, goarch, commit from the file root.
    """
    with open(path) as f:
        doc = json.load(f)
    # Index by name so we can join old/new rows without caring about list order.
    benches = {b["name"]: b for b in doc["benchmarks"]}
    meta = {
        k: doc.get(k, "")
        for k in ("captured_at", "go_version", "goos", "goarch", "commit")
    }
    return benches, meta


def resolve_paths(args: list[str]) -> tuple[str, str]:
    """Determine which two snapshot files to compare.

    Strips flag-like tokens (arguments starting with "-") so -v does not become
    a path. Supports zero, one, or two positional JSON paths; see module docstring.

    Args:
        args: Typically sys.argv[1:] from main.

    Returns:
        (old_path, new_path) absolute or relative paths as provided.

    Exits:
        With a message if no snapshot exists when using default discovery.
    """
    paths = [a for a in args if not a.startswith("-")]
    if len(paths) >= 2:
        # Explicit old and new files.
        return paths[0], paths[1]
    if len(paths) == 1:
        # New file given; old is always the rotated backup from the last capture.
        return os.path.join(RESULTS, "previous.json"), paths[0]
    # Default: previous.json vs the most recently modified snapshot in results/.
    old_p = os.path.join(RESULTS, "previous.json")
    cur = [
        f
        for f in glob.glob(os.path.join(RESULTS, "*.json"))
        if not f.endswith("previous.json")
    ]
    if not cur:
        sys.exit("no snapshot in results/")
    return old_p, max(cur, key=os.path.getmtime)


# -----------------------------------------------------------------------------
# Formatting helpers (units and deltas)
# -----------------------------------------------------------------------------


def short_name(name: str) -> str:
    """Shorten a full benchmark name for narrow table columns.

    All current purser benchmarks use a /size=256 suffix; drop it. Keep the last
    two path segments so "Purser/Store" and "Nulllocker/Store" stay distinct.

    Args:
        name: Full name from JSON, e.g. "Locker/Seal/size=256".

    Returns:
        A shorter label, e.g. "Locker/Seal".
    """
    if name.endswith("/size=256"):
        name = name[: -len("/size=256")]
    parts = name.split("/")
    if len(parts) >= 2:
        return "/".join(parts[-2:])
    return parts[-1]


def fmt_bytes(n: int) -> str:
    """Format a non-negative byte count with a human-readable unit.

    Args:
        n: Byte count (sign ignored; used for magnitudes and absolute values).

    Returns:
        String like "360 B", "3.6 KiB", or "1.2 MiB".
    """
    n = abs(n)
    if n < 1024:
        return f"{n} B"
    if n < 1024 * 1024:
        return f"{n / 1024:.1f} KiB"
    return f"{n / (1024 * 1024):.1f} MiB"


def fmt_value(metric: str, n: int) -> str:
    """Format one absolute metric for verbose old/new columns.

    Args:
        metric: One of METRICS keys.
        n: Raw integer from JSON.

    Returns:
        Value with unit suffix, e.g. "71420 ns" or "31 allocs".
    """
    if metric == "ns_per_op":
        return f"{n} ns"
    if metric == "bytes_per_op":
        return fmt_bytes(n)
    return f"{n} allocs"


def fmt_change(metric: str, old: int, new: int) -> str:
    """Format the change from old to new with units; "=" when equal.

    Used for both compact delta columns and verbose "change" column. Percent is
    appended only when old is non-zero, delta is non-zero, and |percent| >= PCT_MIN.

    Args:
        metric: One of METRICS keys (selects unit).
        old: Previous snapshot value.
        new: Current snapshot value.

    Returns:
        "=" if unchanged, else a signed delta like "-500 ns (-1.4%)" or "+1 B".
    """
    if old == new:
        return "="
    delta = new - old
    # Pick unit and scale for the absolute delta magnitude.
    if metric == "ns_per_op":
        mag = f"{delta:+d} ns"
    elif metric == "bytes_per_op":
        if abs(delta) < 1024:
            mag = f"{delta:+d} B"
        else:
            mag = f"{delta / 1024:+.1f} KiB"
    else:
        # allocs_per_op: plain integer delta (column header carries "allocs").
        mag = f"{delta:+d}"
    if old and abs(delta) > 0:
        pct = 100.0 * delta / old
        if abs(pct) >= PCT_MIN:
            return f"{mag} ({pct:+.1f}%)"
    return mag


# -----------------------------------------------------------------------------
# Printing tables
# -----------------------------------------------------------------------------


def print_meta(old_p: str, new_p: str, old_meta: dict, new_meta: dict) -> None:
    """Print a single header line: which files and capture context.

    Uses basename only so long results/ paths do not wrap the table. Commit and
    Go version are taken from old first, then new, if missing on one side.

    Args:
        old_p: Path to the baseline (previous) snapshot.
        new_p: Path to the current snapshot.
        old_meta: Metadata dict from load_snapshot for old_p.
        new_meta: Metadata dict from load_snapshot for new_p.
    """
    commit = old_meta.get("commit") or new_meta.get("commit") or "?"
    platform = f"{old_meta.get('goos', '?')}/{old_meta.get('goarch', '?')}"
    go = old_meta.get("go_version", "?")
    print(
        f"{os.path.basename(old_p)} → {os.path.basename(new_p)}"
        f"  ·  {commit}  ·  {go}  ·  {platform}"
    )


def print_compact(old: dict, new: dict) -> None:
    """Print the default summary: one row per benchmark, three delta columns.

    Rows are sorted by full benchmark name. Missing benchmarks (only in old or
    only in new) get a single row with em-dashes in metric columns.

    Args:
        old: Bench map from the older snapshot.
        new: Bench map from the newer snapshot.
    """
    headers = (
        "benchmark",
        METRIC_COL["ns_per_op"],
        METRIC_COL["bytes_per_op"],
        METRIC_COL["allocs_per_op"],
    )
    rows: list[tuple[str, ...]] = []

    for name in sorted(set(old) | set(new)):
        label = short_name(name)
        o, n = old.get(name), new.get(name)
        if not o or not n:
            # Snapshot lists differ (unusual unless a benchmark was added/removed).
            where = "old" if not o else "new"
            rows.append((f"{label} (missing in {where})", "—", "—", "—"))
            continue
        rows.append(
            (
                label,
                fmt_change("ns_per_op", o["ns_per_op"], n["ns_per_op"]),
                fmt_change("bytes_per_op", o["bytes_per_op"], n["bytes_per_op"]),
                fmt_change("allocs_per_op", o["allocs_per_op"], n["allocs_per_op"]),
            )
        )

    _print_aligned(headers, rows)


def print_verbose(old: dict, new: dict) -> None:
    """Print the -v detail table: old, new, and change per metric.

    Same fmt_change rules as compact (including "="). Benchmark column shows
    short_name only on the first metric row; following rows leave it blank so
    the block groups visually under one benchmark.

    Args:
        old: Bench map from the older snapshot.
        new: Bench map from the newer snapshot.
    """
    headers = ("benchmark", "metric", "old", "new", "change")
    rows: list[tuple[str, ...]] = []

    for name in sorted(set(old) | set(new)):
        o, n = old.get(name), new.get(name)
        if not o or not n:
            where = "old" if not o else "new"
            rows.append((short_name(name), "—", "—", "—", f"missing in {where}"))
            continue
        first = True
        for key in METRICS:
            # Repeat benchmark label only once per block of three metric rows.
            label = short_name(name) if first else ""
            first = False
            a, b = o[key], n[key]
            rows.append(
                (
                    label,
                    METRIC_LABEL[key],
                    fmt_value(key, a),
                    fmt_value(key, b),
                    fmt_change(key, a, b),
                )
            )

    _print_aligned(headers, rows)


def _print_aligned(headers: tuple[str, ...], rows: list[tuple[str, ...]]) -> None:
    """Render a simple ASCII table with space-padded columns.

    Column width is the max of header and cell lengths in that column. No external
    dependencies (tabulate, etc.).

    Args:
        headers: Column titles.
        rows: One tuple per row; length must match len(headers).
    """
    widths = [len(h) for h in headers]
    for row in rows:
        for i, cell in enumerate(row):
            widths[i] = max(widths[i], len(cell))

    def line(cells: tuple[str, ...]) -> str:
        return "  ".join(c.ljust(widths[i]) for i, c in enumerate(cells))

    print(line(headers))
    print(line(tuple("-" * w for w in widths)))
    for row in rows:
        print(line(row))


# -----------------------------------------------------------------------------
# Entry point
# -----------------------------------------------------------------------------


def main() -> None:
    """Load snapshots, print compact summary, optionally print verbose detail.

    Workflow:
        1. Parse -v / --verbose from sys.argv.
        2. Resolve old/new JSON paths (see resolve_paths).
        3. Load both files.
        4. Print metadata line and compact delta table.
        5. If verbose, print a blank line and the detail table.
    """
    verbose = "--verbose" in sys.argv or "-v" in sys.argv
    old_p, new_p = resolve_paths(sys.argv[1:])
    old, old_meta = load_snapshot(old_p)
    new, new_meta = load_snapshot(new_p)
    print_meta(old_p, new_p, old_meta, new_meta)
    print()
    print_compact(old, new)
    if verbose:
        print()
        print("detail (old / new / change):")
        print_verbose(old, new)


if __name__ == "__main__":
    main()
