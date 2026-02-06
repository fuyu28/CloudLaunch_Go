cmake_minimum_required(VERSION 3.20)
project(dxgi_screenshot LANGUAGES CXX)

add_executable(dxgi_screenshot DxgiScreenshot.cpp)
target_compile_features(dxgi_screenshot PRIVATE cxx_std_17)
target_compile_definitions(dxgi_screenshot PRIVATE WIN32_LEAN_AND_MEAN NOMINMAX)
target_link_libraries(dxgi_screenshot PRIVATE d3d11 dxgi windowscodecs user32)

set_target_properties(dxgi_screenshot PROPERTIES
  OUTPUT_NAME "dxgi_screenshot"
)
