#!/usr/bin/env bash
set -euo pipefail

if [[ "${TARGET:-}" != "windows_amd64" ]]; then
  exit 0
fi

binary_path="${1:?binary path required}"
version="${2:?version required}"
output="${3:-dist/winget-manifest.yaml}"

checksum=$(sha256sum "$binary_path" | awk '{print $1}')

mkdir -p "$(dirname "$output")"

cat > "$output" <<EOF
PackageIdentifier: Pepabo.xpoint-cli
PackageVersion: ${version}
PackageLocale: ja-JP
Publisher: GMO Pepabo Inc,
PackageName: xpoint-cli
License: MIT
ShortDescription: cli tool for X-point
Installers:
  - Architecture: x64
    InstallerType: portable
    InstallerUrl: https://github.com/pepabo/xpoint-cli/releases/download/v${version}/xp_${version}_windows_amd64.exe
    InstallerSha256: ${checksum}
    Commands: [ xp ]
ManifestType: singleton
ManifestVersion: 1.12.0
EOF
