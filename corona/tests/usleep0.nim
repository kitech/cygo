import random

proc usleep(usec:int) : int {.importc.}


proc test_usleep0() =
    var btime = times.now()
    linfo("before usleep", btime)
    #discard usleep(1000000)
    discard usleep(30000000)
    linfo("after usleep", times.now()-btime)
    return

proc nowt0() : DateTime = times.fromUnix(epochTime().int64).utc()
proc nowt1() : int64 = epochTime().int64
proc test_usleep1(arg:pointer) =
    var btime = nowt0() # times.now()
    var tno = cast[int](arg)
    linfo("before usleep", btime, tno)
    for i in 0..50:
        discard usleep(rand(320)*10000)
        linfo("inloop usleep", i, tno)
    linfo("after usleep", nowt0()-btime, tno, getFrame()==nil)
    return

#noro_post(test_usleep0, nil)
#noro_post(test_usleep1, cast[pointer](5))
proc runtest_usleep(cnt:int) =
    noro_post(test_usleep1, cast[pointer](cnt))

