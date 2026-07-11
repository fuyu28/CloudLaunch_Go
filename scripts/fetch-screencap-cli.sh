#!/usr/bin/env bash
# screencap-cli.exe を GitHub Releases から取得して build/windows/ と build/bin/ に配置するスクリプト。
#
# 用途:
#   Windows インストーラに同梱するスクリーンショット用CLI（screencap-cli.exe）を
#   ダウンロードし、build/windows/screencap-cli.exe としてリネーム配置する。
#   加えて build/bin/screencap-cli.exe にもコピーする（wails dev / wails build の
#   バイナリは build/bin に置かれ、実行ファイル隣接のCLIとして解決されるため）。
#
# 使い方:
#   wails build / wails dev を実行する前にこのスクリプトを一度実行しておく。
#     ./scripts/fetch-screencap-cli.sh
#   配置後に wails build を行うと NSIS インストーラに screencap-cli.exe が同梱される。
#
# バージョン更新:
#   下記 SCREENCAP_VERSION と SCREENCAP_SHA256 を両方更新して再実行する。
set -euo pipefail

# 取得する screencap-cli のバージョン（タグでピン留め）
SCREENCAP_VERSION="v0.4.0"
# 期待する SHA256（改ざん・破損検知およびスキップ判定に使う）
SCREENCAP_SHA256="71249aeaf7bbd8f2c9523c256e6cbd3cdee79b68d15f5020b7cc22d7cb736ceb"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dest_dir="$repo_root/build/windows"
dest_file="$dest_dir/screencap-cli.exe"
bin_dir="$repo_root/build/bin"
bin_file="$bin_dir/screencap-cli.exe"

download_url="https://github.com/fuyu28/screencap-rs/releases/download/${SCREENCAP_VERSION}/screencap-cli-${SCREENCAP_VERSION}-windows-x86_64.exe"

mkdir -p "$dest_dir"
mkdir -p "$bin_dir"

# sha256sum を計算する（shasum が無ければ sha256sum を試す）。
compute_sha256() {
    local file="$1"
    if command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" | awk '{print $1}'
    elif command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | awk '{print $1}'
    else
        echo "ERROR: shasum も sha256sum も見つかりません" >&2
        return 1
    fi
}

# 既存 exe の sha256 が期待値と一致すればスキップ（build/bin へのコピーは保証する）。
if [[ -f "$dest_file" ]]; then
    actual="$(compute_sha256 "$dest_file")"
    if [[ "$actual" == "$SCREENCAP_SHA256" ]]; then
        echo "==> screencap-cli ${SCREENCAP_VERSION} は既に配置済み（sha256 一致）のためダウンロードをスキップします"
        cp -f "$dest_file" "$bin_file"
        echo "==> build/bin へコピー: $bin_file"
        exit 0
    fi
    echo "==> 既存の screencap-cli.exe の sha256 が一致しないため再取得します"
fi

echo "==> screencap-cli ${SCREENCAP_VERSION} をダウンロードします"
echo "    from: $download_url"

# 失敗時は非0で終了（curl -f）、リダイレクト追従（-L）、リトライ付き
if ! curl -fL --retry 3 --retry-delay 2 -o "$dest_file" "$download_url"; then
    echo "ERROR: screencap-cli のダウンロードに失敗しました: $download_url" >&2
    exit 1
fi

# ダウンロード物の sha256 を検証する。不一致なら削除して非0終了。
actual="$(compute_sha256 "$dest_file")"
if [[ "$actual" != "$SCREENCAP_SHA256" ]]; then
    echo "ERROR: sha256 が一致しません（期待: $SCREENCAP_SHA256 / 実際: $actual）" >&2
    rm -f "$dest_file"
    exit 1
fi

# wails dev / wails build のバイナリは build/bin に置かれるため、そこにもコピーする。
cp -f "$dest_file" "$bin_file"

echo "==> 配置完了: $dest_file"
echo "==> 配置完了: $bin_file"
