
project(cxrt)
cmake_minimum_required(VERSION 3.0)

set(CMAKE_VERBOSE_MAKEFILE on)
set(CMAKE_C_FLAGS "-g -O0 -std=c11 -D_GNU_SOURCE")

include_directories(./src ./include ./bdwgc/include)

add_library(cxrt STATIC src/cxmemory.c src/cxstring.c src/cxrtbase.c)

add_executable(simlnk tests/simlink.c)
target_link_libraries(simlnk cxrt)

include_directories(./corona-c ./cltc/include)
set(corona_c_srcs
  corona-c/coro.c
	corona-c/corowp.c
	corona-c/hook.c
	corona-c/hookcb.c
	corona-c/futex.c
	corona-c/corona_util.c
	corona-c/rxilog.c
	corona-c/atomic.c
	corona-c/szqueue.c
	corona-c/chan.c
	corona-c/hchan.c
	corona-c/hselect.c
#	corona-c/netpoller_ev.c
	corona-c/netpoller_event.c
	corona-c/coronagc.c
	corona-c/corona.c
	corona-c/functrace.c
)
add_library(crn STATIC ${corona_c_srcs})
add_executable(corona ${corona_c_srcs} corona-c/main.c)
set(corona_c_flags "-DNRDEBUG -DCORO_STACKALLOC -DCORO_UCONTEXT -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK -DGC_THREADS -fstack-usage")
set_target_properties(crn PROPERTIES COMPILE_FLAGS ${corona_c_flags})
set_target_properties(corona PROPERTIES COMPILE_FLAGS ${corona_c_flags})
target_link_libraries(corona -L./bdwgc/.libs -L./cltc/lib gc collectc event event_pthreads pthread dl)
# target_compile_options(corona PRIVATE )
