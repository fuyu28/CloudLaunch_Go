# screencap-cli.exe を GitHub Releases から取得して build/windows/ と build/bin/ に配置するスクリプト。
#
# 用途:
#   Windows インストーラに同梱するスクリーンショット用CLI（screencap-cli.exe）を
#   ダウンロードし、build/windows/screencap-cli.exe としてリネーム配置する。
#   加えて build/bin/screencap-cli.exe にもコピーする（wails dev / wails build の
#   バイナリは build/bin に置かれ、実行ファイル隣接のCLIとして解決されるため）。
#
# 使い方:
#   wails.json の preBuildHooks（windows/*）から wails dev / wails build 実行時に
#   自動実行される。手動で実行することも可能:
#     powershell -ExecutionPolicy Bypass -File scripts/fetch-screencap-cli.ps1
#   配置後に wails build を行うと NSIS インストーラに screencap-cli.exe が同梱される。
#
# バージョン更新:
#   下記 $ScreencapVersion と $ScreencapSha256 を両方更新して再実行する。

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# PS5.1 の Invoke-WebRequest はプログレスバー描画でダウンロードが激遅になるため無効化する
$ProgressPreference = 'SilentlyContinue'

# 取得する screencap-cli のバージョン（タグでピン留め）
$ScreencapVersion = 'v0.4.0'
# 期待する SHA256（改ざん・破損検知およびスキップ判定に使う）
$ScreencapSha256 = '71249aeaf7bbd8f2c9523c256e6cbd3cdee79b68d15f5020b7cc22d7cb736ceb'

# リポジトリルートは $PSScriptRoot（このスクリプトのある scripts/）の親として解決する。
# フックの作業ディレクトリ（build/bin）に依存しないようにするため。
$repoRoot = Split-Path -Parent $PSScriptRoot
# パス区切りはスラッシュで統一する（Windows でも有効で、macOS の pwsh から手動実行しても壊れない）
$destDir = Join-Path $repoRoot 'build/windows'
$destFile = Join-Path $destDir 'screencap-cli.exe'
$binDir = Join-Path $repoRoot 'build/bin'
$binFile = Join-Path $binDir 'screencap-cli.exe'

$downloadUrl = "https://github.com/fuyu28/screencap-rs/releases/download/${ScreencapVersion}/screencap-cli-${ScreencapVersion}-windows-x86_64.exe"

New-Item -ItemType Directory -Force -Path $destDir | Out-Null
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

# 指定ファイルの SHA256 を大文字文字列で返す。
function Get-Sha256Hex {
    param([string]$Path)
    $stream = [System.IO.File]::OpenRead($Path)
    $sha256 = [System.Security.Cryptography.SHA256]::Create()
    try {
        $hash = $sha256.ComputeHash($stream)
        return [System.BitConverter]::ToString($hash).Replace('-', '')
    }
    finally {
        $sha256.Dispose()
        $stream.Dispose()
    }
}

# 既存 exe の sha256 が期待値と一致すればスキップ（build/bin へのコピーは保証する）。
if (Test-Path -LiteralPath $destFile) {
    $actual = Get-Sha256Hex -Path $destFile
    if ($actual -ieq $ScreencapSha256) {
        Write-Host "==> screencap-cli ${ScreencapVersion} は既に配置済み（sha256 一致）のためダウンロードをスキップします"
        Copy-Item -LiteralPath $destFile -Destination $binFile -Force
        Write-Host "==> build/bin へコピー: $binFile"
        exit 0
    }
    Write-Host "==> 既存の screencap-cli.exe の sha256 が一致しないため再取得します"
}

Write-Host "==> screencap-cli ${ScreencapVersion} をダウンロードします"
Write-Host "    from: $downloadUrl"

# 失敗時は最大3回リトライ（各失敗後 2 秒待ち）。-UseBasicParsing は PS5.1 互換のため付与。
$maxAttempts = 3
$downloaded = $false
for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
    try {
        Invoke-WebRequest -UseBasicParsing -Uri $downloadUrl -OutFile $destFile
        $downloaded = $true
        break
    }
    catch {
        Write-Host "    ダウンロード失敗（試行 $attempt/$maxAttempts）: $($_.Exception.Message)"
        if ($attempt -lt $maxAttempts) {
            Start-Sleep -Seconds 2
        }
    }
}

if (-not $downloaded) {
    # $ErrorActionPreference=Stop 下の Write-Error は即 throw して exit に到達しないため -ErrorAction Continue を付ける
    Write-Error "screencap-cli のダウンロードに失敗しました: $downloadUrl" -ErrorAction Continue
    exit 1
}

# ダウンロード物の sha256 を検証する。不一致なら削除して非0終了。
$actual = Get-Sha256Hex -Path $destFile
if ($actual -ine $ScreencapSha256) {
    Remove-Item -LiteralPath $destFile -Force
    Write-Error "sha256 が一致しません（期待: $ScreencapSha256 / 実際: $actual）" -ErrorAction Continue
    exit 1
}

# wails dev / wails build のバイナリは build/bin に置かれるため、そこにもコピーする。
Copy-Item -LiteralPath $destFile -Destination $binFile -Force

Write-Host "==> 配置完了: $destFile"
Write-Host "==> 配置完了: $binFile"
