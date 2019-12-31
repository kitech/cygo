

set(mydir ${CMAKE_CURRENT_LIST_DIR})
set(party3dir ${CMAKE_CURRENT_LIST_DIR}/3rdparty)

#  ${mydir}/bdwgc/include
include_directories(${mydir}/src ${mydir}/include)
add_library(cxrt STATIC  ${mydir}/src/cxrtbase.c
  ${mydir}/src/cxmemory.c ${mydir}/src/cxstring.c
  ${mydir}/src/cxhashtable.c ${mydir}/src/cxarray.c
  ${mydir}/src/cxqueue.c
  ${mydir}/src/cxiface.c
#  ${mydir}/src/cppminrt.cpp
  )

set(cltcdir ${party3dir}/cltc/src) # need ln cltc/src/include cltc/src/collectc
include_directories(${mydir}/corona-c ${party3dir}/picoev ${cltcdir}/include ${cltcdir})
set(corona_c_srcs
  ${mydir}/corona-c/coro.c
	${mydir}/corona-c/corowp.c
	${mydir}/corona-c/hook.c
	${mydir}/corona-c/hookcb.c
	${mydir}/corona-c/hook2.c
#	${mydir}/corona-c/hookbyplt.c
	${mydir}/corona-c/futex.c
	${mydir}/corona-c/corona_util.c
	${mydir}/corona-c/rxilog.c
	${mydir}/corona-c/atomic.c
	${mydir}/corona-c/datstu.c
	${mydir}/corona-c/szqueue.c
	${mydir}/corona-c/chan.c
	${mydir}/corona-c/hchan.c
	${mydir}/corona-c/hselect.c
#	${mydir}/corona-c/netpoller_ev.c
  ${mydir}/corona-c/netpoller_event.c
  # ${mydir}/corona-c/netpoller_epoll.c
	${mydir}/corona-c/coronagc.c
	${mydir}/corona-c/corona.c
	${mydir}/corona-c/functrace.c
  ${party3dir}/picoev/picoev_epoll.c
  )

# include_directories(${party3dir}/plthook)
# set(plthook_c_srcs
#    ${party3dir}/plthook/plthook_elf.c
#    )

set(cltc_c_srcs
  ${cltcdir}/array.c ${cltcdir}/hashtable.c
  ${cltcdir}/queue.c ${cltcdir}/deque.c ${cltcdir}/common.c
)

add_library(crn STATIC ${corona_c_srcs}
  ${cltc_c_srcs}
  # ${plthook_c_srcs}
  )

#add_executable(corona ${corona_c_srcs} corona-c/main.c)
set(CMAKE_C_FLAGS "-g -O0 -std=c11 -D_GNU_SOURCE ")
#set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -pedantic") # non ISO C warning
#set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fsanitize=address") # stack corrupt
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -ggdb3 -fno-omit-frame-pointer") # stack corrupt
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fstack-protector -fstack-protector-all")
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -DGC_THREADS")
#set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -DGC_DEBUG")
set(corona_c_flags "-DNRDEBUG -DCORO_STACKALLOC -DCORO_UCONTEXT -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK -fstack-usage")
set(CMAKE_CXX_FLAGS "-nostdlib -fno-rtti -fno-rtti -fno-exceptions")
#set(CMAKE_CXX_COMPILER "cc")

set_target_properties(crn PROPERTIES COMPILE_FLAGS ${corona_c_flags})
#set_target_properties(corona PROPERTIES COMPILE_FLAGS ${corona_c_flags})
#target_link_libraries(corona -L./bdwgc/.libs -L./cltc/lib gc collectc event event_pthreads pthread dl)
#set(gclib "${mydir}/bdwgc/.libs/libgc.a") # -L${mydir}/bdwgc/.libs
set(gclib "-lgc")
set(libevents_ldflags "-levent -levent_pthreads")
set(cxrt_ldflags "-levent -levent_pthreads ${gclib} -lpthread -ldl -lc")
# note: all libraries which maybe create threads, must put before -lgc


