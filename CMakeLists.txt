
project(cxrt)
cmake_minimum_required(VERSION 3.5)

if("${CMAKE_VERBOSE_MAKEFILE}" STREQUAL "FALSE")
	set(CMAKE_VERBOSE_MAKEFILE on)
endif()
# set(CMAKE_C_COMPILER "clang")
# set(CMAKE_CXX_COMPILER "/path/to/clang++")

option(USE_LIBEVENT "Build with libevent" OFF)
option(USE_LIBEV "Build with libev" OFF)
option(USE_LIBUV "Build with libuv" OFF)
option(USE_FCONTEXT "Build with boost fcontext" OFF)
option(USE_LIBSIGSEGV "Build with libsigsegv" OFF)
include(cxrt.cmake)

add_executable(simlnk tests/simlink.c)
target_link_libraries(simlnk cxrt)

add_executable(corona corona-c/main.c)
set_target_properties(corona PROPERTIES COMPILE_FLAGS ${corona_c_flags})
target_link_libraries(corona -L. crn cxrt -L./bdwgc/.libs -L./cltc/lib gc event event_pthreads pthread dl)
# for mac, on linux just /usr/include
set(vcpkg_dir "/opt/vcpkg/root/installed/x64-osx-dynamic")
include_directories(${vcpkg_dir}/include)
TARGET_LINK_LIBRARIES(corona -L${vcpkg_dir}/lib -Wl,-rpath,${vcpkg_dir}/lib)

# target_compile_options(corona PRIVATE )
