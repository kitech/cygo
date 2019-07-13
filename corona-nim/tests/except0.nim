import os

proc test_exc0() =
    linfo "hehehehhe"
    raise newException(OSError, "test exc msg")

proc test_exc1() =
    try:
        test_exc0()
    except:
        linfo getCurrentExceptionMsg()
    for i in 0..5: sleep(1000)
    return

proc runtest_exc0(cnter: int) =
    crn_post(test_exc1, nil)
    return
