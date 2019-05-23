
proc usleep(usec:int) : int {.importc.}


proc test_usleep0() =
    linfo("before usleep", times.now())
    discard usleep(1000000)
    linfo("after usleep", times.now())
    return

noro_post(test_usleep0, nil)


