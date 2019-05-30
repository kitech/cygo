# note: Must use switch: --gc:boehm --threads:on
# Sometimes cannot automatically use corona.nim.cfg

{.passc:"-g -O0"} #  -fsanitize=address

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
include "ascproj.nim"


var noroh : pointer
proc noro_set_thread_createcb(fnptr:pointer, args:pointer) {.importc.}
proc noro_set_frame_funcs(getter, setter : pointer) {.importc.}
proc noro_init_and_wait_done():pointer {.importc.}
proc noro_post(fnptr:pointer, args:pointer) {.importc.}
proc noro_malloc(size:csize) : pointer {.importc.}
proc hchan_new(cap:int) : pointer {.importc.}
proc hchan_close(hc:pointer) : bool {.importc.}
proc hchan_is_closed(hc:pointer) : bool {.importc.}
proc hchan_send(hc:pointer, data:pointer) : bool {.importc.}
proc hchan_recv(hc:pointer, data:ptr pointer) : bool {.importc.}
proc hchan_len(hc:pointer) : int {.importc.}
proc hchan_cap(hc:pointer) : int {.importc.}

proc noro_thread_createcbfn(args:pointer) =
    linfo("noro thread created", args)
    #setupForeignThreadGc()
    return

linfo "corona initing ..."
noro_set_thread_createcb(noro_thread_createcbfn, nil)
noro_set_frame_funcs(cast[pointer](getFrame), cast[pointer](setFrame))
noroh = noro_init_and_wait_done()
linfo "corona inited done"

include "./gogoapi.nim"

# just like use spawn: gogo somefunc(a0, a1, a2)
# simple wrap gogo2 implemention macro
macro gogo*(stmt:typed) : untyped =
    result = quote do: gogo2(stmt)

proc corona_loop*() =
    while true:
        poll(5000)

include "tests/common.nim"
if isMainModule:
    testloop()

{.hint[XDeclaredButNotUsed]:off.}

# usage:
#    import corona/corona
#    gogo somefunc(1, 2, "abc")

