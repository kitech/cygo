
project(cxrt)
cmake_minimum_required(VERSION 3.0)

set(CMAKE_VERBOSE_MAKEFILE on)
set(CMAKE_C_FLAGS "-g -O0 -std=c11 -D_GNU_SOURCE")

include_directories(./src ./include ./bdwgc/include)

add_library(cxrt STATIC src/cxmemory.c src/cxstring.c src/cxrtbase.c)

add_executable(simlnk tests/simlink.c)
target_link_libraries(simlnk cxrt)

#include_directories(./corona-c ./cltc/include)
#add_library(corona STATIC corona-c/corona.c)
