#!/usr/bin/env python3
"""Generate a dark Change Monitor icon set (PNG, ICO, ICNS) with stdlib only."""
from __future__ import annotations

import struct
import zlib
from pathlib import Path

OUT = Path(__file__).resolve().parents[1] / "src-tauri" / "icons"
BG = (0x0D, 0x11, 0x17, 0xFF)
FG = (0x58, 0xA6, 0xFF, 0xFF)
MARK = (0xE6, 0xED, 0xF3, 0xFF)


def pixel(x: int, y: int, n: int) -> bytes:
    # Dark field, rounded-ish blue tile, inner pale bar (a tiny "C" mark).
    m = n - 1 if n > 1 else 1
    fx, fy = x / m, y / m
    r = 0.16
    inset = 0.18
    inside = inset <= fx <= 1 - inset and inset <= fy <= 1 - inset
    # crude rounded rect by cutting corners
    if inside:
        cx = min(fx - inset, 1 - inset - fx, fy - inset, 1 - inset - fy)
        if cx < r * (1 - 2 * inset) * 0.35 and (
            (fx < inset + 0.2 and fy < inset + 0.2)
            or (fx > 1 - inset - 0.2 and fy < inset + 0.2)
            or (fx < inset + 0.2 and fy > 1 - inset - 0.2)
            or (fx > 1 - inset - 0.2 and fy > 1 - inset - 0.2)
        ):
            # keep corner as bg if far from center of corner — skip, fill FG
            pass
        # inner C-ish cut
        inner = 0.34 <= fx <= 0.78 and 0.34 <= fy <= 0.66
        bar = 0.34 <= fx <= 0.52 and 0.34 <= fy <= 0.66
        if inner and not bar:
            return bytes(BG)
        return bytes(FG)
    return bytes(BG)


def png(size: int) -> bytes:
    raw = b"".join(b"\x00" + b"".join(pixel(x, y, size) for x in range(size)) for y in range(size))
    ihdr = struct.pack(">IIBBBBB", size, size, 8, 6, 0, 0, 0)

    def chunk(tag: bytes, data: bytes) -> bytes:
        crc = zlib.crc32(tag + data) & 0xFFFFFFFF
        return struct.pack(">I", len(data)) + tag + data + struct.pack(">I", crc)

    return (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", ihdr)
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b"")
    )


def ico(images: list[bytes]) -> bytes:
    # PNG-in-ICO (Vista+). Fine for Tauri/Windows.
    count = len(images)
    header = struct.pack("<HHH", 0, 1, count)
    offset = 6 + 16 * count
    entries = b""
    blobs = b""
    for data in images:
        # IHDR is 8+8+13? PNG: sig 8, then IHDR chunk: 4 len + 4 tag + 13 data
        w = struct.unpack(">I", data[16:20])[0]
        h = struct.unpack(">I", data[20:24])[0]
        # sizes 256 stored as 0
        entry_w = 0 if w >= 256 else w
        entry_h = 0 if h >= 256 else h
        entries += struct.pack("<BBBBHHII", entry_w, entry_h, 0, 0, 1, 32, len(data), offset)
        blobs += data
        offset += len(data)
    return header + entries + blobs


def icns(png_256: bytes) -> bytes:
    # ic09 = 256x256 PNG. Good enough for macOS bundling.
    payload = b"ic09" + struct.pack(">I", 8 + len(png_256)) + png_256
    return b"icns" + struct.pack(">I", 8 + len(payload)) + payload


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    p32 = png(32)
    p128 = png(128)
    p256 = png(256)
    (OUT / "32x32.png").write_bytes(p32)
    (OUT / "128x128.png").write_bytes(p128)
    (OUT / "128x128@2x.png").write_bytes(p256)
    (OUT / "icon.png").write_bytes(p256)
    (OUT / "icon.ico").write_bytes(ico([p32, p128, p256]))
    (OUT / "icon.icns").write_bytes(icns(p256))
    print(OUT)


if __name__ == "__main__":
    main()
