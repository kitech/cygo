
project(cxrt)
cmake_minimum_required(VERSION 3.0)

if("${CMAKE_VERBOSE_MAKEFILE}" STREQUAL "FALSE")
  set(CMAKE_VERBOSE_MAKEFILE on)
endif()

include(../cxrt.cmake)

add_executable(genoe opkgs/foo.c)
target_link_libraries(genoe -L. cxrt crn ${cxrt_ldflags})






