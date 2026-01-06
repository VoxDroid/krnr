#!/usr/bin/env python3
"""Build script for local cross-platform builds.

Features:
- builds a set of targets (os/arch) into `dist/`
- embeds version (via -ldflags) when provided or detected from git
- creates SHA256 sums in `dist/SHA256SUMS.txt`
- simple CLI with --version, --targets, --jobs, --clean

Usage:
  python scripts/build_local.py --version v0.1.0
  python scripts/build_local.py --targets linux/amd64,windows/amd64
"""

from __future__ import annotations

import argparse
import concurrent.futures
import hashlib
import os
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Iterable, List, Tuple

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "dist"

DEFAULT_TARGETS = [
    "linux/amd64",
    "linux/arm64",
    "darwin/amd64",
    "darwin/arm64",
    "windows/amd64",
    "windows/arm64",
]


def run(cmd: List[str], env=None) -> Tuple[int, str, str]:
    proc = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
        text=True,
    )
    out, err = proc.communicate()
    return proc.returncode, out, err


def detect_go() -> str:
    go = shutil.which("go")
    if not go:
        raise SystemExit("Go toolchain not found in PATH. Install Go and retry.")
    return go


def detect_version(provided: str | None) -> str:
    if provided:
        return provided

    # 1) Try to read from internal/version/version.go
    vfile = ROOT / "internal" / "version" / "version.go"
    if vfile.exists():
        try:
            txt = vfile.read_text(encoding="utf-8")
            import re

            m = re.search(r"var\s+Version\s*=\s*\"([^\"]+)\"", txt)
            if m:
                return m.group(1)
        except Exception:
            pass

    # 2) try git describe --tags
    try:
        out = subprocess.check_output(["git", "describe", "--tags", "--abbrev=0"], stderr=subprocess.DEVNULL, text=True).strip()
        return out
    except Exception:
        # fallback to HEAD short sha
        try:
            out = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"], text=True).strip()
            return f"dev-{out}"
        except Exception:
            return "dev"


def build_target(target: str, version: str) -> Tuple[str, bool, str]:
    """Build a single target. Returns (target, succeeded, message)"""
    go = detect_go()
    osname, arch = target.split("/")
    ext = ".exe" if osname == "windows" else ""
    # Normalize version for filename (strip leading 'v' if present)
    version_no_v = version[1:] if version.startswith('v') else version
    out_name = f"krnr-{version_no_v}-{osname}-{arch}{ext}"
    out_path = DIST / out_name
    env = os.environ.copy()
    env["GOOS"] = osname
    env["GOARCH"] = arch
    env["CGO_ENABLED"] = "0"

    # Embed normalized version into binary
    ldflag = f"-X github.com/VoxDroid/krnr/internal/version.Version={version_no_v}"
    cmd = [go, "build", "-ldflags", ldflag, "-o", str(out_path), "."]

    rc, out, err = run(cmd, env=env)
    if rc != 0:
        return target, False, err.strip() or out.strip()
    return target, True, str(out_path)


def sha256_of_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            h.update(chunk)
    return h.hexdigest()


def write_sums(files: Iterable[Path]) -> None:
    sums_path = DIST / "SHA256SUMS.txt"
    with sums_path.open("w", encoding="utf-8") as fh:
        for p in sorted(files):
            fh.write(f"{sha256_of_file(p)}  {p.name}\n")


def parse_targets(s: str | None) -> List[str]:
    if not s:
        return DEFAULT_TARGETS
    items = [x.strip() for x in s.split(",") if x.strip()]
    return items


def main(argv: List[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Local cross-platform build script for krnr")
    p.add_argument("--version", "-v", help="Version to embed (e.g., v0.1.0). If omitted, detect from git")
    p.add_argument("--targets", "-t", help="Comma-separated list of targets (os/arch). E.g. linux/amd64,windows/amd64")
    p.add_argument("--jobs", "-j", type=int, default=4, help="Parallel build jobs")
    p.add_argument("--clean", action="store_true", help="Clean dist/ before building")
    args = p.parse_args(argv)

    try:
        detect_go()
    except SystemExit as e:
        print(str(e), file=sys.stderr)
        return 1

    version = detect_version(args.version)
    print(f"Building version: {version}")

    targets = parse_targets(args.targets)
    print(f"Targets: {targets}")

    if args.clean and DIST.exists():
        print("Cleaning dist/")
        shutil.rmtree(DIST)

    DIST.mkdir(parents=True, exist_ok=True)

    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as ex:
        futures = {ex.submit(build_target, t, version): t for t in targets}
        for fut in concurrent.futures.as_completed(futures):
            t = futures[fut]
            try:
                targ, ok, msg = fut.result()
                results.append((targ, ok, msg))
                if ok:
                    print(f"[OK] {targ} -> {msg}")
                else:
                    print(f"[FAIL] {targ}: {msg}")
            except Exception as exc:
                print(f"[ERROR] {t}: {exc}")
                results.append((t, False, str(exc)))

    built = [Path(msg) for (_, ok, msg) in results if ok]
    if built:
        write_sums(built)
        print(f"Wrote {DIST / 'SHA256SUMS.txt'}")
    else:
        print("No artifacts built successfully.")

    failed = [r for r in results if not r[1]]
    if failed:
        print("Some builds failed:")
        for t, ok, msg in failed:
            print(f" - {t}: {msg}")
        return 2

    print("All builds succeeded.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
