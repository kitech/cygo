
proc test_chan0impl() =
    var hc = hchan_new(0)
    assert(hc != nil, "cannot be nil")
    var closed = hchan_is_closed(hc)
    assert(closed == false)
    closed = hchan_close(hc)
    assert(closed == true, $closed)
    closed = hchan_is_closed(hc) # because already zeroed
    assert(closed == false, $closed)
    return

proc test_chan1impl(hc:pointer) =
    assert(hc != nil)
    for i in 0..5:
        var dat0 = cast[pointer](5+i)
        linfo("hc sending", dat0)
        assert(hchan_len(hc) == 0)
        var rv = hchan_send(hc, dat0)
        linfo("send rv", rv)
        sleep(1000)
    linfo("send done")
    discard hchan_close(hc)
    return

proc test_chan2impl() =
    var hc = hchan_new(0)
    noro_post(test_chan1impl, hc)
    var dat0 : pointer
    for i in 0..5:
        sleep(500)
        assert(hchan_len(hc) == 0)
        var rv = hchan_recv(hc, dat0.addr)
        linfo("recv rv", rv, cast[int](dat0))
        sleep(500)
    linfo("recv done")
    return

proc test_chan0() =
    #noro_post(test_chan0impl, nil)
    noro_post(test_chan2impl, nil)
    return

