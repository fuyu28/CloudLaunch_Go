# WGC Screenshot DLL

Windows Graphics Capture (WGC) を使ってウィンドウのスクリーンショットを取得する DLL です。

## ビルド

```bat
mkdir build
cd build
cmake .. -G "Visual Studio 17 2022" -A x64
cmake --build . --config Release
```

ビルド後、`Release/wgc_screenshot.exe` を `build/bin` に配置してください。
`wails build` 実行時に `./scripts/copy-wgc-dll.sh` が自動でコピーします。

## エクスポート

- `HRESULT CaptureWindowToPngFile(HWND hwnd, const wchar_t* path)`
