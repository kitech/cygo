import random

proc test_manyroutines1(arg:pointer) =
    var btime = nowt0() # times.now()
    var tno = cast[int](arg)
    linfo("before usleep", btime, tno)
    for i in 0..50:
        discard usleepc(rand(320)*10000)
        linfo("inloop usleep", i, tno)
    linfo("after usleep", nowt0()-btime, tno)
    return


# 每个tick是500ms
# 每tick启动1个，大概应该能够保持120个
proc runtest_manyroutines_tick(cnt:int) =
    if cnt < 120:
        crn_post(test_manyroutines1, cast[pointer](cnt))
    else:
        if cnt mod 2 == 1:
            crn_post(test_manyroutines1, cast[pointer](cnt))

