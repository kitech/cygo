
project(cxrt)
cmake_minimum_required(VERSION 3.0)

if("${CMAKE_VERBOSE_MAKEFILE}" STREQUAL "FALSE")
  set(CMAKE_VERBOSE_MAKEFILE on)
endif()

include(../cxrt.cmake)

add_executable(genoe opkgs/foo.c)
include_directories("/usr/lib/libffi-3.2.1/include")
target_link_libraries(genoe -L. cxrt crn -lcurl -ldwarf -lelf ${cxrt_ldflags})

# add_executable(co1 co1.c ../corona-c/coro.c)
# target_link_libraries(co1 -L../bdwgc/.libs gc pthread)
# set_target_properties(co1 PROPERTIES COMPILE_FLAGS ${corona_c_flags})





