
project(cxrt)
cmake_minimum_required(VERSION 3.5)

if("${CMAKE_VERBOSE_MAKEFILE}" STREQUAL "FALSE")
	set(CMAKE_VERBOSE_MAKEFILE on)
endif()

include(cxrt.cmake)

add_executable(simlnk tests/simlink.c)
target_link_libraries(simlnk cxrt)

add_executable(corona corona-c/main.c)
set_target_properties(corona PROPERTIES COMPILE_FLAGS ${corona_c_flags})
target_link_libraries(corona -L. crn -L./bdwgc/.libs -L./cltc/lib gc event event_pthreads pthread dl)

# target_compile_options(corona PRIVATE )
