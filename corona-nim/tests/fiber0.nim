
proc test_fiber0() =
    return

proc test_fiber1(s:string) =
    return

proc test_fiber2(s:cstring) =
    linfo(s)
    sleep(5000)
    return

proc test_fiber3(v: float) =
    linfo("whttt")
    for i in 0..5:
        linfo(v, i)
        sleep(1000)
    sleep(2000)
    return

proc runtest_fiber0(cnt:int) =
    linfo "hehehe"
    #gogo2 test_fiber()
    #gogo2 test_fiber1("abc")
    var cs : cstring = "abc"
    #gogo2 test_fiber2(cs)
    gogo2 test_fiber3(5.678)
