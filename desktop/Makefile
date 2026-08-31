# Desktop shell around the existing Go engine. Does not replace aichange / aichange.exe.
.PHONY: sidecar dev build icons clean

PYTHON ?= python3
TAURI ?= cargo tauri

sidecar:
	$(PYTHON) scripts/build-sidecar.py

icons:
	$(PYTHON) scripts/generate-icons.py

dev: sidecar
	$(TAURI) dev

build: sidecar
	$(TAURI) build

clean:
	rm -rf src-tauri/target src-tauri/gen src-tauri/binaries/aichange-*
