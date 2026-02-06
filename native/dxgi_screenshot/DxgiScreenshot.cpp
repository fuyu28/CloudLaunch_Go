#include <d3d11.h>
#include <dxgi1_2.h>
#include <windows.h>
#include <wincodec.h>
#include <wrl/client.h>

#include <algorithm>
#include <cstdint>
#include <cstring>
#include <iostream>
#include <string>
#include <vector>

using Microsoft::WRL::ComPtr;

namespace {

struct Options {
  std::wstring outPath;
  int cropX = 0;
  int cropY = 0;
  int cropW = 0;
  int cropH = 0;
  int monitorIndex = -1;
  std::wstring format = L"png";
};

void PrintUsage() {
  std::wcerr << L"Usage: dxgi_screenshot.exe --out <file> --crop <x> <y> <w> <h> "
                L"[--monitor <index>] [--format png]\n";
}

std::wstring ToWide(const std::string& value) {
  if (value.empty()) {
    return L"";
  }
  int needed = MultiByteToWideChar(CP_UTF8, 0, value.c_str(),
                                   static_cast<int>(value.size()), nullptr, 0);
  if (needed <= 0) {
    return L"";
  }
  std::wstring out(needed, L'\0');
  MultiByteToWideChar(CP_UTF8, 0, value.c_str(),
                      static_cast<int>(value.size()), out.data(), needed);
  return out;
}

bool ParseInt(const char* value, int* out) {
  if (!value || !out) {
    return false;
  }
  char* end = nullptr;
  long result = std::strtol(value, &end, 10);
  if (end == value) {
    return false;
  }
  *out = static_cast<int>(result);
  return true;
}

bool ParseArgs(int argc, char** argv, Options* options) {
  if (!options) {
    return false;
  }
  for (int i = 1; i < argc; i++) {
    std::string arg = argv[i];
    if (arg == "--out" && i + 1 < argc) {
      options->outPath = ToWide(argv[++i]);
      continue;
    }
    if (arg == "--crop" && i + 4 < argc) {
      if (!ParseInt(argv[++i], &options->cropX) ||
          !ParseInt(argv[++i], &options->cropY) ||
          !ParseInt(argv[++i], &options->cropW) ||
          !ParseInt(argv[++i], &options->cropH)) {
        return false;
      }
      continue;
    }
    if (arg == "--monitor" && i + 1 < argc) {
      if (!ParseInt(argv[++i], &options->monitorIndex)) {
        return false;
      }
      continue;
    }
    if (arg == "--format" && i + 1 < argc) {
      options->format = ToWide(argv[++i]);
      continue;
    }
    std::wcerr << L"Unknown argument: " << ToWide(arg) << L"\n";
    return false;
  }
  if (options->outPath.empty() || options->cropW <= 0 || options->cropH <= 0) {
    return false;
  }
  if (!options->format.empty()) {
    std::wstring fmt = options->format;
    std::transform(fmt.begin(), fmt.end(), fmt.begin(), ::towlower);
    if (fmt != L"png") {
      std::wcerr << L"Unsupported format: " << fmt << L"\n";
      return false;
    }
  }
  return true;
}

bool SetDpiAwareness() {
  auto user32 = LoadLibraryW(L"user32.dll");
  if (!user32) {
    return false;
  }
  using SetDpiAwarenessFn = BOOL(WINAPI*)(HANDLE);
  auto setDpi = reinterpret_cast<SetDpiAwarenessFn>(
      GetProcAddress(user32, "SetProcessDpiAwarenessContext"));
  if (setDpi) {
    setDpi(reinterpret_cast<HANDLE>(-4));  // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
  }
  FreeLibrary(user32);
  return true;
}

struct OutputItem {
  ComPtr<IDXGIOutput1> output;
  DXGI_OUTPUT_DESC desc{};
};

bool EnumerateOutputs(const ComPtr<IDXGIAdapter>& adapter,
                      std::vector<OutputItem>* outputs) {
  if (!adapter || !outputs) {
    return false;
  }
  for (UINT i = 0;; i++) {
    ComPtr<IDXGIOutput> output;
    if (adapter->EnumOutputs(i, &output) == DXGI_ERROR_NOT_FOUND) {
      break;
    }
    ComPtr<IDXGIOutput1> output1;
    if (FAILED(output.As(&output1))) {
      continue;
    }
    DXGI_OUTPUT_DESC desc{};
    if (FAILED(output1->GetDesc(&desc))) {
      continue;
    }
    outputs->push_back({output1, desc});
  }
  return !outputs->empty();
}

int FindOutputIndex(const std::vector<OutputItem>& outputs, int x, int y, int w, int h) {
  if (outputs.empty()) {
    return -1;
  }
  int centerX = x + w / 2;
  int centerY = y + h / 2;
  for (size_t i = 0; i < outputs.size(); i++) {
    const RECT& rect = outputs[i].desc.DesktopCoordinates;
    if (centerX >= rect.left && centerX < rect.right && centerY >= rect.top &&
        centerY < rect.bottom) {
      return static_cast<int>(i);
    }
  }
  return 0;
}

bool CropRectForOutput(const RECT& desktop, int x, int y, int w, int h, RECT* outCrop) {
  if (!outCrop) {
    return false;
  }
  RECT crop{};
  crop.left = x - desktop.left;
  crop.top = y - desktop.top;
  crop.right = crop.left + w;
  crop.bottom = crop.top + h;
  RECT bounds{};
  bounds.left = 0;
  bounds.top = 0;
  bounds.right = desktop.right - desktop.left;
  bounds.bottom = desktop.bottom - desktop.top;
  crop.left = std::max(crop.left, bounds.left);
  crop.top = std::max(crop.top, bounds.top);
  crop.right = std::min(crop.right, bounds.right);
  crop.bottom = std::min(crop.bottom, bounds.bottom);
  if (crop.right <= crop.left || crop.bottom <= crop.top) {
    return false;
  }
  *outCrop = crop;
  return true;
}

bool SavePng(const std::wstring& path, const std::vector<uint8_t>& pixels, int width, int height) {
  ComPtr<IWICImagingFactory> factory;
  if (FAILED(CoCreateInstance(CLSID_WICImagingFactory, nullptr, CLSCTX_INPROC_SERVER,
                              IID_PPV_ARGS(&factory)))) {
    return false;
  }
  ComPtr<IWICStream> stream;
  if (FAILED(factory->CreateStream(&stream))) {
    return false;
  }
  if (FAILED(stream->InitializeFromFilename(path.c_str(), GENERIC_WRITE))) {
    return false;
  }
  ComPtr<IWICBitmapEncoder> encoder;
  if (FAILED(factory->CreateEncoder(GUID_ContainerFormatPng, nullptr, &encoder))) {
    return false;
  }
  if (FAILED(encoder->Initialize(stream.Get(), WICBitmapEncoderNoCache))) {
    return false;
  }
  ComPtr<IWICBitmapFrameEncode> frame;
  if (FAILED(encoder->CreateNewFrame(&frame, nullptr))) {
    return false;
  }
  if (FAILED(frame->Initialize(nullptr))) {
    return false;
  }
  if (FAILED(frame->SetSize(static_cast<UINT>(width), static_cast<UINT>(height)))) {
    return false;
  }
  WICPixelFormatGUID format = GUID_WICPixelFormat32bppBGRA;
  if (FAILED(frame->SetPixelFormat(&format))) {
    return false;
  }
  UINT stride = static_cast<UINT>(width * 4);
  UINT bufferSize = static_cast<UINT>(pixels.size());
  if (FAILED(frame->WritePixels(static_cast<UINT>(height), stride, bufferSize,
                                const_cast<BYTE*>(pixels.data())))) {
    return false;
  }
  if (FAILED(frame->Commit())) {
    return false;
  }
  if (FAILED(encoder->Commit())) {
    return false;
  }
  return true;
}

}  // namespace

int main(int argc, char** argv) {
  Options options;
  if (!ParseArgs(argc, argv, &options)) {
    PrintUsage();
    return 2;
  }

  SetDpiAwareness();

  if (FAILED(CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED))) {
    std::wcerr << L"Failed to initialize COM\n";
    return 1;
  }

  ComPtr<IDXGIFactory1> factory;
  if (FAILED(CreateDXGIFactory1(IID_PPV_ARGS(&factory)))) {
    std::wcerr << L"Failed to create DXGI factory\n";
    CoUninitialize();
    return 1;
  }

  std::vector<OutputItem> outputs;
  for (UINT adapterIndex = 0;; adapterIndex++) {
    ComPtr<IDXGIAdapter> adapter;
    if (factory->EnumAdapters(adapterIndex, &adapter) == DXGI_ERROR_NOT_FOUND) {
      break;
    }
    EnumerateOutputs(adapter, &outputs);
  }
  if (outputs.empty()) {
    std::wcerr << L"No DXGI outputs found\n";
    CoUninitialize();
    return 1;
  }

  int outputIndex = options.monitorIndex;
  if (outputIndex < 0 || outputIndex >= static_cast<int>(outputs.size())) {
    outputIndex = FindOutputIndex(outputs, options.cropX, options.cropY, options.cropW,
                                  options.cropH);
  }
  if (outputIndex < 0 || outputIndex >= static_cast<int>(outputs.size())) {
    std::wcerr << L"Monitor index is out of range\n";
    CoUninitialize();
    return 1;
  }

  const OutputItem& selected = outputs[outputIndex];
  RECT cropRect{};
  if (!CropRectForOutput(selected.desc.DesktopCoordinates, options.cropX, options.cropY,
                         options.cropW, options.cropH, &cropRect)) {
    std::wcerr << L"Crop rect is invalid for selected monitor\n";
    CoUninitialize();
    return 1;
  }

  ComPtr<ID3D11Device> device;
  ComPtr<ID3D11DeviceContext> context;
  D3D_FEATURE_LEVEL featureLevel;
  HRESULT hr = D3D11CreateDevice(nullptr, D3D_DRIVER_TYPE_HARDWARE, nullptr, 0, nullptr, 0,
                                 D3D11_SDK_VERSION, &device, &featureLevel, &context);
  if (FAILED(hr)) {
    hr = D3D11CreateDevice(nullptr, D3D_DRIVER_TYPE_WARP, nullptr, 0, nullptr, 0,
                           D3D11_SDK_VERSION, &device, &featureLevel, &context);
  }
  if (FAILED(hr)) {
    std::wcerr << L"Failed to create D3D11 device\n";
    CoUninitialize();
    return 1;
  }

  ComPtr<IDXGIOutputDuplication> duplication;
  hr = selected.output->DuplicateOutput(device.Get(), &duplication);
  if (FAILED(hr)) {
    std::wcerr << L"Failed to duplicate output\n";
    CoUninitialize();
    return 1;
  }

  DXGI_OUTDUPL_FRAME_INFO frameInfo{};
  ComPtr<IDXGIResource> frameResource;
  hr = duplication->AcquireNextFrame(500, &frameInfo, &frameResource);
  if (hr == DXGI_ERROR_WAIT_TIMEOUT) {
    std::wcerr << L"Timeout waiting for frame\n";
    CoUninitialize();
    return 1;
  }
  if (FAILED(hr)) {
    std::wcerr << L"Failed to acquire frame\n";
    CoUninitialize();
    return 1;
  }

  ComPtr<ID3D11Texture2D> frame;
  if (FAILED(frameResource.As(&frame))) {
    duplication->ReleaseFrame();
    std::wcerr << L"Failed to access frame texture\n";
    CoUninitialize();
    return 1;
  }

  D3D11_TEXTURE2D_DESC desc{};
  frame->GetDesc(&desc);
  D3D11_TEXTURE2D_DESC stagingDesc = desc;
  stagingDesc.Usage = D3D11_USAGE_STAGING;
  stagingDesc.BindFlags = 0;
  stagingDesc.CPUAccessFlags = D3D11_CPU_ACCESS_READ;
  stagingDesc.MiscFlags = 0;

  ComPtr<ID3D11Texture2D> staging;
  if (FAILED(device->CreateTexture2D(&stagingDesc, nullptr, &staging))) {
    duplication->ReleaseFrame();
    std::wcerr << L"Failed to create staging texture\n";
    CoUninitialize();
    return 1;
  }

  context->CopyResource(staging.Get(), frame.Get());
  duplication->ReleaseFrame();

  D3D11_MAPPED_SUBRESOURCE mapped{};
  if (FAILED(context->Map(staging.Get(), 0, D3D11_MAP_READ, 0, &mapped))) {
    std::wcerr << L"Failed to map staging texture\n";
    CoUninitialize();
    return 1;
  }

  int cropWidth = cropRect.right - cropRect.left;
  int cropHeight = cropRect.bottom - cropRect.top;
  std::vector<uint8_t> pixels(static_cast<size_t>(cropWidth) * cropHeight * 4);
  const uint8_t* src = static_cast<const uint8_t*>(mapped.pData);
  for (int y = 0; y < cropHeight; y++) {
    const uint8_t* row = src + (cropRect.top + y) * mapped.RowPitch + cropRect.left * 4;
    uint8_t* dest = pixels.data() + static_cast<size_t>(y) * cropWidth * 4;
    std::memcpy(dest, row, static_cast<size_t>(cropWidth) * 4);
  }

  context->Unmap(staging.Get(), 0);

  if (!SavePng(options.outPath, pixels, cropWidth, cropHeight)) {
    std::wcerr << L"Failed to write PNG\n";
    CoUninitialize();
    return 1;
  }

  std::wcout << L"monitor=" << outputIndex << L"\n";
  CoUninitialize();
  return 0;
}
