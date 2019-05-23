
proc usleep(usec:int) : int {.importc.}


proc test_usleep0() =
    var btime = times.now()
    linfo("before usleep", btime)
    #discard usleep(1000000)
    discard usleep(300000)
    linfo("after usleep", times.now()-btime)
    return

noro_post(test_usleep0, nil)


