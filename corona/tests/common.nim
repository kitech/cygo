import times

# get a non-block, yieldable sleep proc
proc usleepc(usec:int) : int {.importc:"usleep", discardable.}

proc nowt0() : DateTime = times.fromUnix(epochTime().int64).utc()
proc nowt1() : int64 = epochTime().int64


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

include "./tcpcon0.nim"
include "./usleep0.nim"
include "./manyroutines.nim"
include "./chan0.nim"
include "./chan1.nim"

# test Loop
proc testtick(cnter:int) =
    if cnter mod 12 == 1: linfo("tickout", cnter)
    if cnter mod 6 == 1:
        #runtest_tcpcon0()
        #runtest_usleep((cnter/6).int + 1)
        discard
    # if cnter > 2: break
    # runtest_manyroutines_tick(cnter)
    # if cnter == 0: runtest_chan1(cnter)
    if cnter == 1: runtest_tcpcon0()
    # test_chan0()
    return

proc testloop() =
    # umain()
    var cnter = 0
    while true:
        testtick(cnter)
        cnter += 1
        poll(500)
