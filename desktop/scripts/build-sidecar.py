#!/usr/bin/env python3
"""Build the Go aichange engine as a Tauri sidecar binary.

Writes src-tauri/binaries/aichange-<target-triple>[.exe]

Does not replace the repo-root aichange / aichange.exe CLI artifacts.
"""
from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

desktop = Path(__file__).resolve().parents[1]
go_mod = desktop.parent
bin_dir = desktop / "src-tauri" / "binaries"
bin_dir.mkdir(parents=True, exist_ok=True)

triple = os.environ.get("TAURI_ENV_TARGET_TRIPLE") or os.environ.get("TARGET")
if not triple:
    proc = subprocess.run(
        ["rustc", "--print", "host-tuple"],
        capture_output=True,
        text=True,
    )
    if proc.returncode == 0 and proc.stdout.strip():
        triple = proc.stdout.strip()
    else:
        proc = subprocess.run(["rustc", "-vV"], capture_output=True, text=True, check=True)
        for line in proc.stdout.splitlines():
            if line.startswith("host:"):
                triple = line.split()[1]
                break
if not triple:
    sys.exit("could not determine rustc target triple")

goos, goarch = None, None
if "windows" in triple:
    goos = "windows"
elif "linux" in triple:
    goos = "linux"
elif "darwin" in triple or "apple" in triple:
    goos = "darwin"
if "aarch64" in triple or triple.startswith("arm64"):
    goarch = "arm64"
elif "x86_64" in triple or "amd64" in triple:
    goarch = "amd64"
elif "i686" in triple or "i586" in triple:
    goarch = "386"

ext = ".exe" if "windows" in triple else ""
out = bin_dir / f"aichange-{triple}{ext}"

env = os.environ.copy()
env["CGO_ENABLED"] = "0"
if goos:
    env["GOOS"] = goos
if goarch:
    env["GOARCH"] = goarch

cmd = ["go", "build", "-o", str(out), "."]
print(" ".join(cmd), "->", out, file=sys.stderr)
subprocess.check_call(cmd, cwd=go_mod, env=env)
print(out)
