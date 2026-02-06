# DXGI Screenshot EXE

Desktop Duplication (DXGI) を使ってスクリーンショットを取得するヘルパーEXEです。

## ビルド

```bat
mkdir build
cd build
cmake .. -G "Visual Studio 17 2022" -A x64
cmake --build . --config Release
```

ビルド後、`Release/dxgi_screenshot.exe` を `build/bin` に配置してください。
`wails build` 実行時に `./scripts/copy-dxgi-exe.sh` が自動でコピーします。

## 使い方

```
dxgi_screenshot.exe --out <file> --crop <x> <y> <w> <h> [--monitor <index>] [--format png]
```
