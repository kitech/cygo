import random

proc test_usleep0() =
    var btime = times.now()
    linfo("before usleep", btime)
    #discard usleep(1000000)
    discard usleepc(30000000)
    linfo("after usleep", times.now()-btime)
    return

proc test_usleep1(arg:pointer) =
    var btime = nowt0() # times.now()
    var tno = cast[int](arg)
    linfo("before usleep", btime, tno)
    for i in 0..50:
        discard usleepc(rand(320)*10000)
        linfo("inloop usleep", i, tno)
    linfo("after usleep", nowt0()-btime, tno, getFrame()==nil)
    return

#crn_post(test_usleep0, nil)
#crn_post(test_usleep1, cast[pointer](5))
proc runtest_usleep(cnt:int) =
    crn_post(test_usleep1, cast[pointer](cnt))

