#include <windows.h>

#include <d3d11.h>
#include <dxgi1_2.h>
#include <wincodec.h>

#include <winrt/Windows.Foundation.h>
#include <winrt/Windows.Graphics.Capture.h>
#include <winrt/Windows.Graphics.DirectX.Direct3D11.h>
#include <winrt/Windows.Graphics.DirectX.h>
#include <winrt/base.h>

#include <windows.graphics.capture.interop.h>
#include <windows.graphics.directx.direct3d11.interop.h>

using namespace winrt;

static HRESULT CreateD3DDevice(com_ptr<ID3D11Device> &device,
                               com_ptr<ID3D11DeviceContext> &context) {
  UINT flags = D3D11_CREATE_DEVICE_BGRA_SUPPORT;
  D3D_FEATURE_LEVEL levels[] = {D3D_FEATURE_LEVEL_11_1, D3D_FEATURE_LEVEL_11_0};
  D3D_FEATURE_LEVEL level;
  return D3D11CreateDevice(nullptr, D3D_DRIVER_TYPE_HARDWARE, nullptr, flags,
                           levels, ARRAYSIZE(levels), D3D11_SDK_VERSION,
                           device.put(), &level, context.put());
}

static HRESULT CreateDirect3DDeviceFromDXGI(
    const com_ptr<ID3D11Device> &device,
    winrt::Windows::Graphics::DirectX::Direct3D11::IDirect3DDevice &outDevice) {
  com_ptr<IDXGIDevice> dxgiDevice;
  try {
    dxgiDevice = device.as<IDXGIDevice>();
  } catch (...) {
    return E_NOINTERFACE;
  }
  com_ptr<IInspectable> inspectable;
  HRESULT hr =
      CreateDirect3D11DeviceFromDXGIDevice(dxgiDevice.get(), inspectable.put());
  if (FAILED(hr)) {
    return hr;
  }
  outDevice =
      inspectable
          .as<winrt::Windows::Graphics::DirectX::Direct3D11::IDirect3DDevice>();
  return S_OK;
}

static HRESULT SavePngFromTexture(const com_ptr<ID3D11Device> &device,
                                  const com_ptr<ID3D11DeviceContext> &context,
                                  const com_ptr<ID3D11Texture2D> &texture,
                                  const wchar_t *filePath) {
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

  com_ptr<IWICImagingFactory> factory;
  hr = CoCreateInstance(CLSID_WICImagingFactory, nullptr, CLSCTX_INPROC_SERVER,
                        IID_PPV_ARGS(factory.put()));
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  com_ptr<IWICBitmapEncoder> encoder;
  hr = factory->CreateEncoder(GUID_ContainerFormatPng, nullptr, encoder.put());
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  com_ptr<IWICStream> stream;
  hr = factory->CreateStream(stream.put());
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  hr = stream->InitializeFromFilename(filePath, GENERIC_WRITE);
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  hr = encoder->Initialize(stream.get(), WICBitmapEncoderNoCache);
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  com_ptr<IWICBitmapFrameEncode> frame;
  hr = encoder->CreateNewFrame(frame.put(), nullptr);
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  hr = frame->Initialize(nullptr);
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  hr = frame->SetSize(desc.Width, desc.Height);
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  WICPixelFormatGUID format = GUID_WICPixelFormat32bppBGRA;
  hr = frame->SetPixelFormat(&format);
  if (FAILED(hr)) {
    context->Unmap(staging.get(), 0);
    return hr;
  }

  hr = frame->WritePixels(desc.Height, mapped.RowPitch,
                          mapped.RowPitch * desc.Height,
                          reinterpret_cast<BYTE *>(mapped.pData));

  context->Unmap(staging.get(), 0);
  if (FAILED(hr)) {
    return hr;
  }

  hr = frame->Commit();
  if (FAILED(hr)) {
    return hr;
  }

  return encoder->Commit();
}

extern "C" __declspec(dllexport) HRESULT
CaptureWindowToPngFile(HWND hwnd, const wchar_t *path) {
  if (!hwnd || !path) {
    return E_INVALIDARG;
  }

  winrt::init_apartment(winrt::apartment_type::multi_threaded);

  com_ptr<ID3D11Device> d3dDevice;
  com_ptr<ID3D11DeviceContext> d3dContext;
  HRESULT hr = CreateD3DDevice(d3dDevice, d3dContext);
  if (FAILED(hr)) {
    return hr;
  }

  winrt::Windows::Graphics::DirectX::Direct3D11::IDirect3DDevice winrtDevice{
      nullptr};
  hr = CreateDirect3DDeviceFromDXGI(d3dDevice, winrtDevice);
  if (FAILED(hr)) {
    return hr;
  }

  com_ptr<IGraphicsCaptureItemInterop> interop =
      winrt::get_activation_factory<
          winrt::Windows::Graphics::Capture::GraphicsCaptureItem>()
          .as<IGraphicsCaptureItemInterop>();

  winrt::Windows::Graphics::Capture::GraphicsCaptureItem item{nullptr};
  hr = interop->CreateForWindow(
      hwnd,
      winrt::guid_of<winrt::Windows::Graphics::Capture::GraphicsCaptureItem>(),
      reinterpret_cast<void **>(winrt::put_abi(item)));
  if (FAILED(hr)) {
    return hr;
  }

  auto size = item.Size();
  if (size.Width <= 0 || size.Height <= 0) {
    return E_FAIL;
  }

  auto framePool =
      winrt::Windows::Graphics::Capture::Direct3D11CaptureFramePool::
          CreateFreeThreaded(winrtDevice,
                             winrt::Windows::Graphics::DirectX::
                                 DirectXPixelFormat::B8G8R8A8UIntNormalized,
                             1, size);

  auto session = framePool.CreateCaptureSession(item);

  handle frameEvent{CreateEvent(nullptr, FALSE, FALSE, nullptr)};
  if (!frameEvent) {
    return HRESULT_FROM_WIN32(GetLastError());
  }

  winrt::Windows::Graphics::Capture::Direct3D11CaptureFrame captured{nullptr};
  auto revoker = framePool.FrameArrived(auto_revoke, [&](auto &, auto &) {
    auto frame = framePool.TryGetNextFrame();
    if (frame) {
      captured = frame;
      SetEvent(frameEvent.get());
    }
  });

  session.StartCapture();

  DWORD waitResult = WaitForSingleObject(frameEvent.get(), 2000);
  if (waitResult != WAIT_OBJECT_0) {
    return HRESULT_FROM_WIN32(WAIT_TIMEOUT);
  }

  if (!captured) {
    return E_FAIL;
  }

  auto surface = captured.Surface();
  com_ptr<ID3D11Texture2D> texture;
  com_ptr<
      ::Windows::Graphics::DirectX::Direct3D11::IDirect3DDxgiInterfaceAccess>
      access;
  hr = winrt::get_unknown(surface)->QueryInterface(IID_PPV_ARGS(access.put()));
  if (FAILED(hr)) {
    return hr;
  }
  hr = access->GetInterface(IID_PPV_ARGS(texture.put()));
  if (FAILED(hr)) {
    return hr;
  }

  return SavePngFromTexture(d3dDevice, d3dContext, texture, path);
}

BOOL APIENTRY DllMain(HMODULE module, DWORD reason, LPVOID) {
  if (reason == DLL_PROCESS_DETACH) {
    winrt::uninit_apartment();
  }
  return TRUE;
}
