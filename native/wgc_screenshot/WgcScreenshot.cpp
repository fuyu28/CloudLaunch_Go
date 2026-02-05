#include <windows.h>

#include <d3d11.h>
#include <dwmapi.h>
#include <dxgi1_2.h>
#include <wincodec.h>

#include <cstring>
#include <string>
#include <vector>

#include <winrt/Windows.Foundation.h>
#include <winrt/Windows.Graphics.Capture.h>
#include <winrt/Windows.Graphics.DirectX.Direct3D11.h>
#include <winrt/Windows.Graphics.DirectX.h>
#include <winrt/base.h>

#include <windows.graphics.capture.interop.h>
#include <windows.graphics.directx.direct3d11.interop.h>

using namespace winrt;

struct CropRect {
  UINT x;
  UINT y;
  UINT width;
  UINT height;
};

static void LogHresult(const wchar_t* stage, HRESULT hr) {
  fwprintf(stderr, L"%ls failed: 0x%08X\n", stage, static_cast<unsigned int>(hr));
}

class TempFileGuard {
public:
  explicit TempFileGuard(std::wstring path) : path_(std::move(path)) {}
  const wchar_t* Path() const {
    return path_.c_str();
  }
  void CommitTo(const wchar_t* target) {
    if (committed_) {
      return;
    }
    if (!MoveFileExW(path_.c_str(), target, MOVEFILE_REPLACE_EXISTING)) {
      throw HRESULT_FROM_WIN32(GetLastError());
    }
    committed_ = true;
  }
  ~TempFileGuard() {
    if (!committed_) {
      DeleteFileW(path_.c_str());
    }
  }

private:
  std::wstring path_;
  bool committed_ = false;
};

static HRESULT CreateD3DDevice(com_ptr<ID3D11Device>& device,
                               com_ptr<ID3D11DeviceContext>& context) {
  UINT flags = D3D11_CREATE_DEVICE_BGRA_SUPPORT;
  D3D_FEATURE_LEVEL levels[] = {D3D_FEATURE_LEVEL_11_1, D3D_FEATURE_LEVEL_11_0};
  D3D_FEATURE_LEVEL level;
  return D3D11CreateDevice(nullptr, D3D_DRIVER_TYPE_HARDWARE, nullptr, flags, levels,
                           ARRAYSIZE(levels), D3D11_SDK_VERSION, device.put(), &level,
                           context.put());
}

static HRESULT CreateDirect3DDeviceFromDXGI(
    const com_ptr<ID3D11Device>& device,
    winrt::Windows::Graphics::DirectX::Direct3D11::IDirect3DDevice& outDevice) {
  com_ptr<IDXGIDevice> dxgiDevice;
  try {
    dxgiDevice = device.as<IDXGIDevice>();
  } catch (...) {
    return E_NOINTERFACE;
  }
  com_ptr<IInspectable> inspectable;
  HRESULT hr = CreateDirect3D11DeviceFromDXGIDevice(dxgiDevice.get(), inspectable.put());
  if (FAILED(hr)) {
    return hr;
  }
  outDevice = inspectable.as<winrt::Windows::Graphics::DirectX::Direct3D11::IDirect3DDevice>();
  return S_OK;
}

static HRESULT SavePngFromTexture(const com_ptr<ID3D11Device>& device,
                                  const com_ptr<ID3D11DeviceContext>& context,
                                  const com_ptr<ID3D11Texture2D>& texture, const wchar_t* filePath,
                                  const CropRect* cropRect) {
  if (!filePath || !*filePath) {
    return E_INVALIDARG;
  }

  D3D11_TEXTURE2D_DESC desc = {};
  texture->GetDesc(&desc);

  D3D11_TEXTURE2D_DESC stagingDesc = desc;
  stagingDesc.BindFlags = 0;
  stagingDesc.MiscFlags = 0;
  stagingDesc.CPUAccessFlags = D3D11_CPU_ACCESS_READ;
  stagingDesc.Usage = D3D11_USAGE_STAGING;

  com_ptr<ID3D11Texture2D> staging;
  HRESULT hr = device->CreateTexture2D(&stagingDesc, nullptr, staging.put());
  if (FAILED(hr)) {
    return hr;
  }

  context->CopyResource(staging.get(), texture.get());

  D3D11_MAPPED_SUBRESOURCE mapped = {};
  hr = context->Map(staging.get(), 0, D3D11_MAP_READ, 0, &mapped);
  if (FAILED(hr)) {
    return hr;
  }
  struct MapScope {
    com_ptr<ID3D11DeviceContext> context;
    com_ptr<ID3D11Texture2D> texture;
    ~MapScope() {
      if (context && texture) {
        context->Unmap(texture.get(), 0);
      }
    }
  } mapScope{context, staging};

  com_ptr<IWICImagingFactory> factory;
  hr = CoCreateInstance(CLSID_WICImagingFactory, nullptr, CLSCTX_INPROC_SERVER,
                        IID_PPV_ARGS(factory.put()));
  if (FAILED(hr)) {
    return hr;
  }

  com_ptr<IWICBitmapEncoder> encoder;
  hr = factory->CreateEncoder(GUID_ContainerFormatPng, nullptr, encoder.put());
  if (FAILED(hr)) {
    return hr;
  }

  com_ptr<IWICStream> stream;
  hr = factory->CreateStream(stream.put());
  if (FAILED(hr)) {
    return hr;
  }

  TempFileGuard tempFile(std::wstring(filePath) + L".tmp");
  hr = stream->InitializeFromFilename(tempFile.Path(), GENERIC_WRITE);
  if (FAILED(hr)) {
    return hr;
  }

  hr = encoder->Initialize(stream.get(), WICBitmapEncoderNoCache);
  if (FAILED(hr)) {
    return hr;
  }

  com_ptr<IWICBitmapFrameEncode> frame;
  hr = encoder->CreateNewFrame(frame.put(), nullptr);
  if (FAILED(hr)) {
    return hr;
  }

  hr = frame->Initialize(nullptr);
  if (FAILED(hr)) {
    return hr;
  }

  UINT outputWidth = desc.Width;
  UINT outputHeight = desc.Height;
  UINT outputStride = mapped.RowPitch;
  const BYTE* outputBytes = reinterpret_cast<BYTE*>(mapped.pData);
  std::vector<BYTE> cropped;

  if (cropRect && cropRect->width > 0 && cropRect->height > 0) {
    outputWidth = cropRect->width;
    outputHeight = cropRect->height;
    outputStride = outputWidth * 4;
    size_t bufferSize = static_cast<size_t>(outputStride) * outputHeight;
    cropped.resize(bufferSize);

    for (UINT y = 0; y < outputHeight; ++y) {
      const BYTE* source = reinterpret_cast<BYTE*>(mapped.pData) +
                           (static_cast<size_t>(cropRect->y) + y) * mapped.RowPitch +
                           static_cast<size_t>(cropRect->x) * 4;
      BYTE* dest = cropped.data() + static_cast<size_t>(y) * outputStride;
      std::memcpy(dest, source, outputStride);
    }
    outputBytes = cropped.data();
  }

  hr = frame->SetSize(outputWidth, outputHeight);
  if (FAILED(hr)) {
    return hr;
  }

  WICPixelFormatGUID format = GUID_WICPixelFormat32bppBGRA;
  hr = frame->SetPixelFormat(&format);
  if (FAILED(hr)) {
    return hr;
  }

  hr = frame->WritePixels(outputHeight, outputStride, outputStride * outputHeight,
                          const_cast<BYTE*>(outputBytes));
  if (FAILED(hr)) {
    return hr;
  }

  hr = frame->Commit();
  if (FAILED(hr)) {
    return hr;
  }

  hr = encoder->Commit();
  if (FAILED(hr)) {
    return hr;
  }

  try {
    tempFile.CommitTo(filePath);
  } catch (HRESULT moveErr) {
    return moveErr;
  }
  return S_OK;
}

static bool TryGetClientCropRect(HWND hwnd, const D3D11_TEXTURE2D_DESC& desc, CropRect& crop) {
  RECT frame = {};
  if (!GetWindowRect(hwnd, &frame)) {
    HRESULT hr = DwmGetWindowAttribute(hwnd, DWMWA_EXTENDED_FRAME_BOUNDS, &frame, sizeof(frame));
    if (FAILED(hr)) {
      return false;
    }
  }

  RECT client = {};
  if (!GetClientRect(hwnd, &client)) {
    return false;
  }

  POINT topLeft{client.left, client.top};
  if (!ClientToScreen(hwnd, &topLeft)) {
    return false;
  }

  int cropX = topLeft.x - frame.left;
  int cropY = topLeft.y - frame.top;
  int cropW = client.right - client.left;
  int cropH = client.bottom - client.top;
  if (cropW <= 0 || cropH <= 0) {
    return false;
  }

  int maxW = static_cast<int>(desc.Width);
  int maxH = static_cast<int>(desc.Height);
  if (cropX < 0) {
    cropW += cropX;
    cropX = 0;
  }
  if (cropY < 0) {
    cropH += cropY;
    cropY = 0;
  }
  if (cropX + cropW > maxW) {
    cropW = maxW - cropX;
  }
  if (cropY + cropH > maxH) {
    cropH = maxH - cropY;
  }
  if (cropW <= 0 || cropH <= 0) {
    return false;
  }

  crop = CropRect{static_cast<UINT>(cropX), static_cast<UINT>(cropY), static_cast<UINT>(cropW),
                  static_cast<UINT>(cropH)};
  return true;
}

static HRESULT CaptureWindowToPngFileEx(HWND hwnd, const wchar_t* path, int clientOnly) {
  if (!hwnd || !path) {
    return E_INVALIDARG;
  }

  HRESULT coInit = CoInitializeEx(nullptr, COINIT_MULTITHREADED);
  bool coInitialized = (coInit == S_OK || coInit == S_FALSE);
  struct CoScope {
    bool initialized = false;
    ~CoScope() {
      if (initialized) {
        CoUninitialize();
      }
    }
  } coScope{coInitialized};

  winrt::init_apartment(winrt::apartment_type::multi_threaded);
  struct ApartmentScope {
    ~ApartmentScope() {
      winrt::uninit_apartment();
    }
  } apartmentScope;

  com_ptr<ID3D11Device> d3dDevice;
  com_ptr<ID3D11DeviceContext> d3dContext;
  HRESULT hr = CreateD3DDevice(d3dDevice, d3dContext);
  if (FAILED(hr)) {
    LogHresult(L"CreateD3DDevice", hr);
    return hr;
  }

  winrt::Windows::Graphics::DirectX::Direct3D11::IDirect3DDevice winrtDevice{nullptr};
  hr = CreateDirect3DDeviceFromDXGI(d3dDevice, winrtDevice);
  if (FAILED(hr)) {
    LogHresult(L"CreateDirect3DDeviceFromDXGI", hr);
    return hr;
  }

  com_ptr<IGraphicsCaptureItemInterop> interop =
      winrt::get_activation_factory<winrt::Windows::Graphics::Capture::GraphicsCaptureItem>()
          .as<IGraphicsCaptureItemInterop>();

  winrt::Windows::Graphics::Capture::GraphicsCaptureItem item{nullptr};
  hr = interop->CreateForWindow(
      hwnd, winrt::guid_of<winrt::Windows::Graphics::Capture::GraphicsCaptureItem>(),
      reinterpret_cast<void**>(winrt::put_abi(item)));
  if (FAILED(hr)) {
    LogHresult(L"CreateForWindow", hr);
    return hr;
  }

  auto size = item.Size();
  if (size.Width <= 0 || size.Height <= 0) {
    fwprintf(stderr, L"Capture item size invalid: %dx%d\n", size.Width, size.Height);
    return E_FAIL;
  }

  auto framePool =
      winrt::Windows::Graphics::Capture::Direct3D11CaptureFramePool::CreateFreeThreaded(
          winrtDevice,
          winrt::Windows::Graphics::DirectX::DirectXPixelFormat::B8G8R8A8UIntNormalized, 1, size);

  auto session = framePool.CreateCaptureSession(item);
  struct CaptureScope {
    winrt::Windows::Graphics::Capture::GraphicsCaptureSession session;
    winrt::Windows::Graphics::Capture::Direct3D11CaptureFramePool framePool;
    ~CaptureScope() {
      if (session) {
        session.Close();
      }
      if (framePool) {
        framePool.Close();
      }
    }
  } captureScope{session, framePool};

  handle frameEvent{CreateEvent(nullptr, FALSE, FALSE, nullptr)};
  if (!frameEvent) {
    return HRESULT_FROM_WIN32(GetLastError());
  }

  winrt::Windows::Graphics::Capture::Direct3D11CaptureFrame captured{nullptr};
  auto revoker = framePool.FrameArrived(auto_revoke, [&](auto&, auto&) {
    auto frame = framePool.TryGetNextFrame();
    if (frame) {
      captured = frame;
      SetEvent(frameEvent.get());
    }
  });

  session.StartCapture();

  DWORD waitResult = WaitForSingleObject(frameEvent.get(), 2000);
  if (waitResult != WAIT_OBJECT_0) {
    fwprintf(stderr, L"Frame wait timeout\n");
    return HRESULT_FROM_WIN32(WAIT_TIMEOUT);
  }

  if (!captured) {
    fwprintf(stderr, L"Captured frame is empty\n");
    return E_FAIL;
  }

  auto surface = captured.Surface();
  com_ptr<ID3D11Texture2D> texture;
  com_ptr<::Windows::Graphics::DirectX::Direct3D11::IDirect3DDxgiInterfaceAccess> access;
  hr = winrt::get_unknown(surface)->QueryInterface(IID_PPV_ARGS(access.put()));
  if (FAILED(hr)) {
    LogHresult(L"QueryInterface(IDirect3DDxgiInterfaceAccess)", hr);
    return hr;
  }
  hr = access->GetInterface(IID_PPV_ARGS(texture.put()));
  if (FAILED(hr)) {
    LogHresult(L"GetInterface(ID3D11Texture2D)", hr);
    return hr;
  }

  CropRect crop = {};
  CropRect* cropPtr = nullptr;
  if (clientOnly != 0) {
    D3D11_TEXTURE2D_DESC desc = {};
    texture->GetDesc(&desc);
    if (TryGetClientCropRect(hwnd, desc, crop)) {
      cropPtr = &crop;
    }
  }

  return SavePngFromTexture(d3dDevice, d3dContext, texture, path, cropPtr);
}

static int PrintUsage() {
  const wchar_t* message =
      L"Usage: wgc_screenshot.exe --hwnd <value> --out <path> [--client-only]\n";
  DWORD written = 0;
  HANDLE errHandle = GetStdHandle(STD_ERROR_HANDLE);
  if (errHandle != INVALID_HANDLE_VALUE) {
    WriteConsoleW(errHandle, message, lstrlenW(message), &written, nullptr);
  }
  return 2;
}

int wmain(int argc, wchar_t** argv) {
  HWND hwnd = nullptr;
  const wchar_t* outPath = nullptr;
  int clientOnly = 0;

  for (int i = 1; i < argc; ++i) {
    if (wcscmp(argv[i], L"--hwnd") == 0 && i + 1 < argc) {
      hwnd = reinterpret_cast<HWND>(static_cast<uintptr_t>(_wcstoui64(argv[++i], nullptr, 10)));
      continue;
    }
    if (wcscmp(argv[i], L"--out") == 0 && i + 1 < argc) {
      outPath = argv[++i];
      continue;
    }
    if (wcscmp(argv[i], L"--client-only") == 0) {
      clientOnly = 1;
      continue;
    }
  }

  if (!hwnd || !outPath) {
    return PrintUsage();
  }

  HRESULT hr = CaptureWindowToPngFileEx(hwnd, outPath, clientOnly);
  return FAILED(hr) ? static_cast<int>(hr) : 0;
}
