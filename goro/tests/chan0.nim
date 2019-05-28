
# test new and close
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

# test non buffer send/recv
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

# test buffered send without recv
proc test_chan3impl() =
    var hc = hchan_new(5)
    assert(hchan_cap(hc) == 5)
    var dat0 : pointer
    for i in 0..4:
        dat0 = cast[pointer](i)
        discard hchan_send(hc, dat0)
        var clen = hchan_len(hc)
        assert(clen == i+1)
    discard hchan_close(hc)
    return

# test buffered send to full and then continues recv
proc test_chan4impl(hc:pointer) =
    assert(hc != nil)
    for i in 0..4:
        var dat0 = cast[pointer](5+i)
        var rv = hchan_recv(hc, dat0.addr)
        assert(dat0 == cast[pointer](5+i))
    assert(hchan_len(hc) == 0)
    return

proc test_chan5impl() =
    var hc = hchan_new(5)
    var dat0 : pointer
    for i in 0..4:
        var dat0 = cast[pointer](5+i)
        var rv = hchan_send(hc, dat0)
    noro_post(test_chan4impl, hc)
    sleep(2000)
    discard hchan_close(hc)
    return

# test multiple sender, one recver
proc test_chan6impl(hc:pointer) =
    assert(hc != nil)
    for i in 0..4:
        var dat0 = cast[pointer](5+i)
        var rv = hchan_send(hc, dat0)
    linfo("sender done", hc)
    return

proc test_chan7impl() =
    var hc = hchan_new(0)
    noro_post(test_chan6impl, hc)
    noro_post(test_chan6impl, hc)
    noro_post(test_chan6impl, hc)
    var dat0 : pointer
    var cnter = 0
    var rcval = newseq[int](0)
    for i in 0..14:
        var rv = hchan_recv(hc, dat0.addr)
        rcval.add(cast[int](dat0))
        cnter += 1
    linfo("recv done", cnter, rcval)
    discard hchan_close(hc)
    return


proc test_chan0() =
    #noro_post(test_chan0impl, nil)
    #noro_post(test_chan2impl, nil)
    #noro_post(test_chan3impl, nil)
    # noro_post(test_chan5impl, nil)
    noro_post(test_chan7impl, nil)
    return

