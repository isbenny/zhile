#!/bin/bash
# Build and install the local Mac application without publishing a release.
set -euo pipefail
MODE="${1:---verify}"
case "$MODE" in run|--verify|--build) ;; *) echo "Usage: $0 [run|--verify|--build]" >&2; exit 2;; esac
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="$ROOT_DIR/build/macos"
APP_BUNDLE="$BUILD_DIR/知乐.app"
INSTALL_BUNDLE="$HOME/Applications/知乐.app"
DESKTOP_LINK="$HOME/Desktop/知乐.app"
mkdir -p "$APP_BUNDLE/Contents/MacOS" "$APP_BUNDLE/Contents/Resources" "$BUILD_DIR/Zhile.iconset"
cd "$ROOT_DIR"
CGO_ENABLED=1 go build -trimpath -ldflags='-s -w' -o "$APP_BUNDLE/Contents/MacOS/Zhile" ./desktop_go
for size in 16 32 128 256 512; do
    sips -z "$size" "$size" website/dist/assets/icon.png --out "$BUILD_DIR/Zhile.iconset/icon_${size}x${size}.png" >/dev/null
    doubled=$((size * 2))
    sips -z "$doubled" "$doubled" website/dist/assets/icon.png --out "$BUILD_DIR/Zhile.iconset/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$BUILD_DIR/Zhile.iconset" -o "$APP_BUNDLE/Contents/Resources/Zhile.icns"
cp LICENSE NOTICE "$APP_BUNDLE/Contents/Resources/"
python3 - "$ROOT_DIR" "$APP_BUNDLE" <<'PY'
import pathlib, plistlib, re, sys
root, bundle = map(pathlib.Path, sys.argv[1:])
version = re.search(r'AppVersion = "([^"]+)"', (root/'core/version.go').read_text()).group(1)
info = dict(CFBundleExecutable='Zhile', CFBundleIdentifier='io.github.isbenny.zhile.mac',
    CFBundleName='知乐', CFBundleDisplayName='知乐', CFBundleIconFile='Zhile.icns',
    CFBundlePackageType='APPL', CFBundleShortVersionString=version, CFBundleVersion='3',
    LSMinimumSystemVersion='12.0', NSPrincipalClass='NSApplication',
    NSHighResolutionCapable=True, NSAppTransportSecurity={'NSAllowsLocalNetworking':True})
(bundle/'Contents/Info.plist').write_bytes(plistlib.dumps(info))
PY
codesign --force --sign - "$APP_BUNDLE"
codesign --verify --strict "$APP_BUNDLE"
if [ "$MODE" = '--build' ]; then exit 0; fi
# Stop only this branded app; leave previews and the previous Go Music DL alone.
pkill -x Zhile >/dev/null 2>&1 || true
mkdir -p "$HOME/Applications"
if [ -e "$INSTALL_BUNDLE" ]; then
    mv "$INSTALL_BUNDLE" "$BUILD_DIR/知乐-backup-$(date +%Y%m%d%H%M%S).app"
fi
ditto "$APP_BUNDLE" "$INSTALL_BUNDLE"
if [ ! -e "$DESKTOP_LINK" ] && [ ! -L "$DESKTOP_LINK" ]; then
    ln -s "$INSTALL_BUNDLE" "$DESKTOP_LINK"
elif [ "$(readlink "$DESKTOP_LINK" || true)" != "$INSTALL_BUNDLE" ]; then
    echo "Existing desktop item was preserved: $DESKTOP_LINK" >&2
fi
open "$INSTALL_BUNDLE"
if [ "$MODE" = '--verify' ]; then
    for attempt in {1..20}; do
        if pgrep -x Zhile >/dev/null && curl -fsS http://127.0.0.1:37777/music/healthz >/dev/null; then
            echo "Installed and running: $INSTALL_BUNDLE"
            exit 0
        fi
        sleep 1
    done
    echo 'Application did not become ready; inspect the local app window.' >&2
    exit 1
fi
