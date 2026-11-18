

set(mydir ${CMAKE_CURRENT_LIST_DIR})
set(party3dir ${CMAKE_CURRENT_LIST_DIR}/3rdparty)

#  ${mydir}/bdwgc/include
include_directories(${mydir}/src ${mydir}/include)
add_library(cxrt STATIC  ${mydir}/src/cxrtbase.c
		${mydir}/src/cxmemory.c ${mydir}/src/cxstring.c
		${mydir}/src/cxhashtable.c ${mydir}/src/cxarray.c
		${mydir}/src/cxqueue.c
		${mydir}/src/cxiface.c
		${mydir}/src/creflect.c
#  ${mydir}/src/cppminrt.cpp
		)

set(cltcdir ${party3dir}/cltc/src) # need ln cltc/src/include cltc/src/collectc
set(vaitdir ${party3dir}/)
include_directories(${mydir}/corona-c ${cltcdir}/include ${cltcdir} ${vaitdir})
set(corona_c_srcs
		${mydir}/corona-c/coro.c
	${mydir}/corona-c/corowp.c
	${mydir}/corona-c/iohook.c
	${mydir}/corona-c/iohookcb.c
	${mydir}/corona-c/iohook2.c
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
	# ${mydir}/corona-c/netpoller_ev.c
		${mydir}/corona-c/netpoller_event.c
		# ${mydir}/corona-c/netpoller_epoll.c
	${mydir}/corona-c/coronagc.c
	${mydir}/corona-c/corona.c
	${mydir}/corona-c/functrace.c
	#	${party3dir}/picoev/picoev_epoll.c
		)

# include_directories(${party3dir}/plthook)
# set(plthook_c_srcs
#    ${party3dir}/plthook/plthook_elf.c
#    )

set(cltc_c_srcs
		${cltcdir}/array.c ${cltcdir}/hashtable.c
		${cltcdir}/array.c ${cltcdir}/treetable.c
		${cltcdir}/queue.c ${cltcdir}/deque.c
		${cltcdir}/queue.c ${cltcdir}/pqueue.c
		${cltcdir}/common.c
)

# ---- Zig sources with C ABI (compiled into libcrn) ----
#   corona-c/zigsrc/corona_zig.zig exports crn_zig_* functions (see
#   corona-c/zigsrc/corona_zig.h). Built as a single PIC object so the same
#   object serves libcrn.a and libcrn.so. If zig is unavailable, C-only build
#   still works (warning only).
find_program(ZIG_EXECUTABLE zig)
set(ZIG_DEMO_SRC ${mydir}/corona-c/zigsrc/corona_zig.zig)
set(ZIG_DEMO_OBJ ${CMAKE_CURRENT_BINARY_DIR}/crn_zig_demo.o)
set(ZIG_OBJS "")
if(ZIG_EXECUTABLE)
  add_custom_command(
    OUTPUT  ${ZIG_DEMO_OBJ}
    COMMAND ${ZIG_EXECUTABLE} build-obj -fPIC -O Debug -lc
            -femit-bin=${ZIG_DEMO_OBJ}
            ${ZIG_DEMO_SRC}
    DEPENDS ${ZIG_DEMO_SRC}
    COMMENT "zig build-obj corona-c/zigsrc/corona_zig.zig"
    VERBATIM)
  set_source_files_properties(${ZIG_DEMO_OBJ} PROPERTIES
      GENERATED TRUE
      EXTERNAL_OBJECT TRUE)
  set(ZIG_OBJS ${ZIG_DEMO_OBJ})
else()
  message(WARNING "zig not found: zigsrc demos are omitted from libcrn")
endif()

# this is the "object library" target: compiles the sources only once
add_library(crn_objlib OBJECT  ${corona_c_srcs}
		${cltc_c_srcs}
		${ZIG_OBJS}
		# ${plthook_c_srcs}
		)
# shared libraries need PIC
set_property(TARGET crn_objlib PROPERTY POSITION_INDEPENDENT_CODE 1)

add_library(crn_st STATIC $<TARGET_OBJECTS:crn_objlib> ${ZIG_OBJS})
add_library(crn_sh SHARED $<TARGET_OBJECTS:crn_objlib> ${ZIG_OBJS})
set_target_properties(crn_st PROPERTIES OUTPUT_NAME crn)
set_target_properties(crn_sh PROPERTIES OUTPUT_NAME crn)
target_link_libraries(crn_sh cxrt)

# eg. cmake -DCMAKE_PREFIX_PATH="path1;path2;path3" ..
# crox: $ cmake -DANDROID_PLATFORM=android-28 -DCMAKE_TOOLCHAIN_FILE=/opt/android-ndk/build/cmake/android.toolchain.cmake -DANDROID_ABI=arm64-v8a -DCMAKE_FIND_ROOT_PATH="/opt/devsys-arm64;/opt/vcpkg/installed/arm64-android-release"  ..
#
# set(CMAKE_FIND_ROOT_PATH "${FIND_DIRS}")
# set(CMAKE_PREFIX_PATH "${FIND_DIRS}")
# set(CMAKE_LIBRARY_PATH "${FIND_DIRS}")
find_library(LIBEVENT1 event_pthreads)
find_library(LIBEVENT2 event)
find_library(LIBGC gc)
find_file(GC_H gc.h)
find_file(EVENT_H event.h)
find_library(LIBSIGSEGV sigsegv)

cmake_path(GET GC_H PARENT_PATH GC_INCDIR)
cmake_path(GET EVENT_H PARENT_PATH EVENT_INCDIR)

message("libgc info ${LIBGC}")
message("libgc info ${GC_H} ${GC_INC_DIR}")
message("libgc info ${LIBEVENT1}")
message("libgc info ${LIBSIGSEGV}")

include_directories(${GC_INCDIR} ${EVENT_INCDIR})

#add_executable(corona ${corona_c_srcs} corona-c/main.c)
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -g -O0 -fPIC -std=c11 -D_GNU_SOURCE ")
#set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -pedantic") # non ISO C warning
# set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fsanitize=address,undefined") # stack corrupt
# set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fsanitize-recover=address -fno-common") # stack corrupt
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -ggdb3 -fno-omit-frame-pointer") # stack corrupt
# set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fno-stack-protector")
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fstack-protector -fstack-protector-all")
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fstack-protector-strong")
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -DGC_THREADS")
# set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fstack-usage") # no macos
#set(CMAKE_CXX_FLAGS "${CMAKE_C_FLAGS} -fstack-usage")
#set(CMAKE_CXX_FLAGS "${CMAKE_C_FLAGS} -nostdlib -fno-rtti -fno-exceptions")
# set(CMAKE_CXX_FLAGS "${CMAKE_C_FLAGS} -fno-rtti -fno-exceptions")
#set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -DGC_DEBUG")
#set(CMAKE_CXX_COMPILER "cc")

set(corona_c_flags "-DNRDEBUG -DCORO_STACKALLOC -DCORO_UCONTEXT -DHAVE_UCONTEXT_H -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK")
set(corona_c_flags "${corona_c_flags}  -D_XOPEN_SOURCE=600") # macos: explicit level, fixes tiny sizeof(ucontext_t)
set(corona_c_flags "${corona_c_flags}  -DLOG_USE_COLOR")
# set(corona_c_flags "${corona_c_flags} -fstack-usage")

set_target_properties(crn_objlib PROPERTIES COMPILE_FLAGS ${corona_c_flags})
set_target_properties(crn_st PROPERTIES COMPILE_FLAGS ${corona_c_flags})
set_target_properties(crn_sh PROPERTIES COMPILE_FLAGS ${corona_c_flags})
#set_target_properties(corona PROPERTIES COMPILE_FLAGS ${corona_c_flags})
#target_link_libraries(corona -L./bdwgc/.libs -L./cltc/lib gc collectc event event_pthreads pthread dl)
#set(gclib "${mydir}/bdwgc/.libs/libgc.a") # -L${mydir}/bdwgc/.libs
# set(gclib "-lgc -lsigsegv -L/opt/vcpkg/installed/x86-linux/lib -L/opt/devsys32/lib")
# set(libevents_ldflags "-levent -levent_pthreads")
# set(cxrt_ldflags "${gclib} -lpthread -ldl -lc")

set(gclib ${LIBGC} ${LIBSIGSEGV})
set(libevents_ldflags ${LIBEVENT2} ${LIBEVENT1})
set(cxrt_ldflags ${gclib}  -lpthread -ldl -lc)
target_link_libraries(crn_sh ${libevents_ldflags} ${cxrt_ldflags})

# target_link_libraries(crn_sh ${LIBGC} ${LIBEVENT1} ${LIBEVENT2} ${LIBSIGSEGV})
# target_link_libraries(crn_sh "${libevents_ldflags} ${cxrt_ldflags}")
# note: all libraries which maybe create threads, must put before -lgc
