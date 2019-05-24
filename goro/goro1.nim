{.passc:"-g -O0"}

import os
import asyncdispatch
import asyncfutures
import threadpool
import tables
import deques
import locks

include "nimlog.nim"
include "nimplus.nim"
include "coro.nim"
include "hook.nim"
include "otherc.nim"

{.compile:"../noro/noro.c".}
{.compile:"../noro/norogc.c".}
{.compile:"../noro/noro_util.c".}
{.passc:"-I . -I ../noro -I ../noro/include -DGC_THREADS".}
{.passl:"-L ../bdwgc/.libs -lgc -lpthread".}


var noroh : pointer
proc noro_init_and_wait_done():pointer {.importc.}
proc noro_post(fnptr:pointer, args:pointer) {.importc.}
proc noro_set_thread_createcb(fnptr:pointer, args:pointer) {.importc.}
proc noro_malloc(size:csize) : pointer {.importc.}

proc noro_thread_createcbfn(args:pointer) =
    linfo("noro thread created", args)
    #setupForeignThreadGc()
    return

linfo "wait proc0 ..."
noro_set_thread_createcb(noro_thread_createcbfn, nil)
noroh = noro_init_and_wait_done()
linfo "goro inited done"

proc hellofn(args:pointer) =
    linfo 123, args
    var p : pointer
    p = noro_malloc(1234)
    p = noro_malloc(2345)
    p = noro_malloc(3456)
    p = noro_malloc(456)
    p = noro_malloc(567)
    return

proc umain() =

    noro_post(hellofn, nil)
    linfo "posted"
    return

proc atrivaltofn(fd:AsyncFD):bool = return false
addTimer(16000, false, atrivaltofn)

proc timedoutfn0(fd:AsyncFD):bool =
    #umain()
    return false
addTimer(21000, false, timedoutfn0)

include "tests/tcpcon0.nim"
include "tests/usleep0.nim"

if isMainModule:
    # umain()
    var cnter = 0
    while true:
        cnter += 1
        if cnter mod 6 == 1:
            #runtest_tcpcon0()
            runtest_usleep((cnter/6).int + 1)
            discard
        poll(500)

