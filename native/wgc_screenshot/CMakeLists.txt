cmake_minimum_required(VERSION 3.20)
project(wgc_screenshot LANGUAGES CXX)

add_library(wgc_screenshot SHARED WgcScreenshot.cpp)

target_compile_features(wgc_screenshot PRIVATE cxx_std_17)

if(MSVC)
  target_compile_options(wgc_screenshot PRIVATE /permissive- /await)
endif()

target_compile_definitions(wgc_screenshot PRIVATE WIN32_LEAN_AND_MEAN NOMINMAX)

target_link_libraries(wgc_screenshot PRIVATE d3d11 dxgi windowsapp)

set_target_properties(wgc_screenshot PROPERTIES
  OUTPUT_NAME "wgc_screenshot"
)
