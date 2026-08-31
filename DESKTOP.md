# Desktop shell (Tauri 2)

See desktop/ for the Tauri 2 window around the existing Go engine.

The inbox is still the Go engine. `desktop/` is a thin native window around `aichange serve`. It does not rewrite the UI and does not replace `aichange` / `aichange.exe`.

```
desktop/                 Tauri 2 project
  ui/index.html          splash (dark #0d1117) until the engine is up
  src-tauri/             Rust window, menu, folder dialog
  scripts/build-sidecar  Go binary named aichange-<triple> for bundling
aichange.exe             unchanged CLI / server; browser still works
```

## Why Tauri, not Electron

Tauri uses the OS webview (WebView2 on Windows, WKWebView on macOS, WebKitGTK on Linux) instead of shipping Chromium. The inbox already lives in the Go server on localhost, so the desktop app only needs a window, a child process, and a native folder picker.

## Run (dev)

Needs Go 1.22+, Rust (rustup), the Tauri CLI, and a webview.

From desktop/, build the Go sidecar then start Tauri:

    python3 scripts/build-sidecar.py
    cargo tauri dev

Or run make dev. Node users can install the Tauri CLI package and run the tauri dev script.

On Windows PowerShell, from desktop/, run the sidecar build script, then the Tauri CLI (dev or build).

First launch: File then Open repo, pick a git checkout. Later launches reuse that path. Browser-based aichange serve remains supported.

## Windows release

Build on a Windows machine, not by cross-compiling from Linux. Install Rust with the MSVC toolchain, Go 1.22+, Visual Studio Build Tools (MSVC and Windows SDK), and WebView2. Current Windows 10/11 already include WebView2; if missing, install the Evergreen runtime from Microsoft Edge documentation.

From desktop/, build the Go sidecar for the host triple, then run the Tauri build. Artifacts land in src-tauri/target/release/bundle/: an NSIS setup executable, a WiX msi, and Change Monitor.exe with the engine binary beside it.

tauri.conf.json uses downloadBootstrapper so an installer can fetch WebView2 when needed. Keep the repo-root aichange.exe CLI; the bundled engine is a separate copy under src-tauri/binaries/ named with the Rust target triple.

## Linux and macOS

Debian/Ubuntu needs libwebkit2gtk-4.1-dev, libgtk-3-dev, librsvg2-dev, and patchelf. Then from desktop/, build the sidecar and run the Tauri build. Linux produces deb and AppImage; macOS produces app and dmg.

## Cross-compilation

The Go engine cross-compiles with GOOS and GOARCH (see the root Makefile dist target). The Tauri shell does not cross well: produce the Windows msi/exe on Windows, the macOS dmg on macOS, and the Linux deb on Linux. Do not mix a Linux-built shell with a Windows engine binary.

## How it wires up

1. Splash window, title Change Monitor, background #0d1117.
2. Rust picks a free 127.0.0.1 port in 7380-7479.
3. Starts the bundled engine: serve --addr 127.0.0.1:<port> --repo <path>.
4. When stdout contains "aichange listening on", the webview loads that URL (Go-embedded inbox).
5. File, Open repo (Ctrl/Cmd+O) or a header-path click opens a native folder dialog, then POST /api/repo with the chosen path.
6. Quit stops the child process.

No React or Vite rewrite of the inbox.
