

set(mydir ${CMAKE_CURRENT_LIST_DIR})

include_directories(${mydir}/src ${mydir}/include ${mydir}/bdwgc/include)

add_library(cxrt STATIC  ${mydir}/src/cxrtbase.c
  ${mydir}/src/cxmemory.c ${mydir}/src/cxstring.c
  ${mydir}/src/cxhashtable.c ${mydir}/src/cxarray.c)

include_directories(${mydir}/corona-c ${mydir}/cltc/include)
set(corona_c_srcs
  ${mydir}/corona-c/coro.c
	${mydir}/corona-c/corowp.c
	${mydir}/corona-c/hook.c
	${mydir}/corona-c/hookcb.c
	${mydir}/corona-c/futex.c
	${mydir}/corona-c/corona_util.c
	${mydir}/corona-c/rxilog.c
	${mydir}/corona-c/atomic.c
	${mydir}/corona-c/szqueue.c
	${mydir}/corona-c/chan.c
	${mydir}/corona-c/hchan.c
	${mydir}/corona-c/hselect.c
#	${mydir}/corona-c/netpoller_ev.c
	${mydir}/corona-c/netpoller_event.c
	${mydir}/corona-c/coronagc.c
	${mydir}/corona-c/corona.c
	${mydir}/corona-c/functrace.c
)
add_library(crn STATIC ${corona_c_srcs})
#add_executable(corona ${corona_c_srcs} corona-c/main.c)
set(CMAKE_C_FLAGS "-g -O0 -std=c11 -D_GNU_SOURCE")
set(corona_c_flags "-DNRDEBUG -DCORO_STACKALLOC -DCORO_UCONTEXT -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK -DGC_THREADS -fstack-usage")
set_target_properties(crn PROPERTIES COMPILE_FLAGS ${corona_c_flags})
#set_target_properties(corona PROPERTIES COMPILE_FLAGS ${corona_c_flags})
#target_link_libraries(corona -L./bdwgc/.libs -L./cltc/lib gc collectc event event_pthreads pthread dl)
set(cxrt_ldflags "-L${mydir}/bdwgc/.libs -L${mydir}/cltc/lib -lgc -lcollectc -levent -levent_pthreads -lpthread -ldl")

