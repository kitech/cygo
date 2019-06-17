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

import nimlog
import nimplus

include "coro.nim"
include "hook.nim"
include "ascproj.nim"


var coronah : pointer
proc crn_set_thread_createcb(fnptr:pointer, args:pointer) {.importc.}
proc crn_set_frame_funcs(getter, setter : pointer) {.importc.}
proc crn_init_and_wait_done():pointer {.importc.}
proc crn_get_goid():cint {.importc.}
proc crn_post(fnptr:pointer, args:pointer) {.importc.}
proc crn_malloc(size:csize) : pointer {.importc.}
proc hchan_new(cap:int) : pointer {.importc.}
proc hchan_close(hc:pointer) : bool {.importc.}
proc hchan_is_closed(hc:pointer) : bool {.importc.}
proc hchan_send(hc:pointer, data:pointer) : bool {.importc.}
proc hchan_recv(hc:pointer, data:ptr pointer) : bool {.importc.}
proc hchan_len(hc:pointer) : int {.importc.}
proc hchan_cap(hc:pointer) : int {.importc.}
proc goselect(rcasi: ptr cint, cas0: pointer, ncases:cint) : bool {.importc.}

proc crn_thread_createcbfn(args:pointer) =
    #linfo("corona thread created", args)
    #setupForeignThreadGc()
    return

linfo "corona initing ..."
crn_set_thread_createcb(crn_thread_createcbfn, nil)
crn_set_frame_funcs(cast[pointer](getFrame), cast[pointer](setFrame))
coronah = crn_init_and_wait_done()
linfo "corona inited done"

proc goid*():int =
    ## get current fiber id
    crn_get_goid()

include "./gogoapi.nim"
include "./gochanapi.nim"

# simple wrap gogo2 implemention macro
macro gogo*(funccallexpr: untyped) : untyped =
    ## Just like a spawn: `gogo somefunc(a0, a1, a2)`
    result = quote do: gogo2 `funccallexpr`

# keep keywords
macro go*(funccallexpr: untyped) : untyped =
    ## Just like a spawn: `gogo somefunc(a0, a1, a2)`
    result = quote do: gogo2 `funccallexpr`

# public channel apis. see gochanapi.nim
# proc makechan*(T: typedesc, cap:int) : chan[T]
# proc send*[T](c: chan[T], v : T) : bool {.discardable.}
# proc recv*[T](c: chan[T]) : T {.discardable.}
# proc cap*[T](c: chan[T]) : int
# proc len*[T](c: chan[T]) : int
# proc closed*[T](c: chan[T]) : bool
# proc `$`*[T](c : chan[T]) : string
# # alias of send: c <- v
# proc `<-`*[T](c: chan[T], v: T)
# # alias of recv: var v = <- c
# proc `<-`*[T](c : chan[T]) : T {.discardable.}

macro goselect*(select_case_expr: untyped) : untyped =
    result = quote do: goselectv6 `select_case_expr`

# keep keywords
macro select*(select_case_expr: untyped) : untyped =
    ## Just like go select:
    ##
    ## **Examples:**
    ##
    ## .. code-block::
    ##   goselect:
    ##     scase <- ch0: discard          # recv but nosave
    ##     scase v0 = <- ch0: discard
    ##     scase vec[0] = <- ch0: discard
    ##     scase ch1 <- 42: discard
    ##     scase ch2 <- "foo": discard
    ##     scase ch3 <- cast[pointer](42): discard
    ##     default: discard
    ##
    ## .. code-block::
    ##   goselect: discard                # block current fiber forever
    ##
    result = quote do: goselectv6 `select_case_expr`

# usage: corona.loop
proc loop*() =
    ## If you haven't other loop, use this
    while true:
        if false: poll(5000)
        else: sleep(5000)

include "tests/common.nim"
if isMainModule:
    testloop()

{.push hint[XDeclaredButNotUsed]:off.}

# usage:
#    import corona/corona
#    gogo somefunc(1, 2, "abc")

