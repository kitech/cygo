

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
include_directories(${mydir}/corona-c ${cltcdir}/../ ${cltcdir}/include ${vaitdir})
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

# ---- target detection: NDK / Termux / plain Linux|macOS ----
# No files are created; this only reads CMake variables, environment
# variables, and compiler predefined macros.
set(CRN_IS_ANDROID FALSE)
if(CMAKE_SYSTEM_NAME STREQUAL "Android")
  set(CRN_IS_ANDROID TRUE)
elseif(DEFINED ENV{TERMUX_VERSION})
  set(CRN_IS_ANDROID TRUE)
elseif("$ENV{PREFIX}" MATCHES "com\\.termux")
  set(CRN_IS_ANDROID TRUE)
elseif(CMAKE_C_COMPILER MATCHES "android|termux")
  set(CRN_IS_ANDROID TRUE)
endif()

# Fallback: ask the compiler for its predefined macros (no file written).
if(NOT CRN_IS_ANDROID AND NOT WIN32)
  execute_process(
    COMMAND ${CMAKE_C_COMPILER} -dM -E -x c /dev/null
    OUTPUT_VARIABLE _crn_macros
    ERROR_QUIET)
  if(_crn_macros MATCHES "__ANDROID__|__BIONIC__")
    set(CRN_IS_ANDROID TRUE)
  endif()
endif()

if(CRN_IS_ANDROID)
  # bionic has no ucontext; use vendored higan-emu/libco.
  # repo:   https://github.com/higan-emu/libco
  # commit: e18e09d634d612a01781168ad4d76be10a7e3bad (2024-09-09, v20, ISC)
  list(APPEND corona_c_srcs ${mydir}/3rdparty/libco/libco.c)
endif()

# include_directories(${party3dir}/plthook)
# set(plthook_c_srcs
#    ${party3dir}/plthook/plthook_elf.c
#    )

set(cltc_c_srcs
		${cltcdir}/../collect_cds.c
		#${cltcdir}/array.c ${cltcdir}/hashtable.c
		#${cltcdir}/array.c ${cltcdir}/treetable.c
		#${cltcdir}/queue.c ${cltcdir}/deque.c
		#${cltcdir}/queue.c ${cltcdir}/pqueue.c
		#${cltcdir}/common.c
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

# macOS: the raw object is linked by clang/ld which does not provide Zig's
# compiler-rt, so `___zig_probe_stack` would be undefined (ziglang/zig#6817).
#   -fno-stack-check   removes that reference (object has only small frames)
#   -target ...macos.N also silences "built for newer macOS version" warnings
# Alternative if stack probing must be kept: build a static lib with
#   zig build-lib -static -fcompiler-rt ...  and link it instead of the .o.
set(ZIG_TARGET_FLAGS "")
if(APPLE)
  if(CMAKE_SYSTEM_PROCESSOR MATCHES "arm64|aarch64")
    set(ZIG_ARCH aarch64)
  else()
    set(ZIG_ARCH x86_64)
  endif()
  if(NOT CMAKE_OSX_DEPLOYMENT_TARGET)
    set(CMAKE_OSX_DEPLOYMENT_TARGET "11.0")
  endif()
  set(ZIG_TARGET_FLAGS "-target;${ZIG_ARCH}-macos.${CMAKE_OSX_DEPLOYMENT_TARGET}")
endif()

if(ZIG_EXECUTABLE)
  add_custom_command(
    OUTPUT  ${ZIG_DEMO_OBJ}
    COMMAND ${ZIG_EXECUTABLE} build-obj -fPIC -O Debug -lc -fno-stack-check
            ${ZIG_TARGET_FLAGS}
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

set(corona_c_flags "-DNRDEBUG -DCORO_STACKALLOC -DHAVE_SETJMP_H -DHAVE_SIGALTSTACK")
if(CRN_IS_ANDROID)
  # bionic has no ucontext; use vendored higan-emu/libco
  set(corona_c_flags "${corona_c_flags} -DCORO_LIBCO89 -DLIBCO_MP")
else()
  set(corona_c_flags "${corona_c_flags} -DCORO_UCONTEXT -DHAVE_UCONTEXT_H")
endif()
set(corona_c_flags "${corona_c_flags}  -D_XOPEN_SOURCE=600") # macos: explicit level, fixes tiny sizeof(ucontext_t)
if(APPLE)
  # _XOPEN_SOURCE hides BSD types (u_char/u_int in nameser.h via resolv.h);
  # _DARWIN_C_SOURCE re-enables them while keeping the XOPEN visibility level
  set(corona_c_flags "${corona_c_flags}  -D_DARWIN_C_SOURCE")
  set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -D_DARWIN_C_SOURCE")
endif()
if(CRN_IS_ANDROID)
  # bionic: _XOPEN_SOURCE alone hides some declarations; enable default APIs
  set(corona_c_flags "${corona_c_flags}  -D_DEFAULT_SOURCE")
endif()
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
