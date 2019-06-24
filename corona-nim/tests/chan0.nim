
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
    crn_post(test_chan1impl, hc)
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
    crn_post(test_chan4impl, hc)
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
    crn_post(test_chan6impl, hc)
    crn_post(test_chan6impl, hc)
    crn_post(test_chan6impl, hc)
    var dat0 : pointer
    var cnter = 0
    var rcval = newseq[int](0)
    for i in 0..14:
        var rv = hchan_recv(hc, dat0.addr)
        rcval.add(cast[int](dat0))
        cnter += 1
    linfo("recv done", cnter, rcval)
    discard hchan_close(hc)
    var valsum = 0
    for v in rcval: valsum += v
    assert(valsum == 3*(5+6+7+8+9) )
    return


# test one sender, multiple recver
var mulrcvcnt = 0
var rcval = newseq[int](0)
proc test_chan8impl(hc:pointer) =
    assert(hc != nil)
    var dat0 : pointer
    var valcnt = 0
    for i in 0..4:
        var rv = hchan_recv(hc, dat0.addr)
        atomicInc(mulrcvcnt)
        valcnt += 1
        # linfo("recved", cast[int](dat0), mulrcvcnt)
    linfo("recv done", hc, valcnt)
    assert(valcnt == 5)
    return

proc test_chan9impl() =
    var hc = hchan_new(0)
    crn_post(test_chan8impl, hc)
    crn_post(test_chan8impl, hc)
    crn_post(test_chan8impl, hc)
    var dat0 : pointer
    for i in 0..14:
        dat0 = cast[pointer](i+1)
        var rv = hchan_send(hc, dat0)
    linfo("send done")
    sleep(500)
    discard hchan_close(hc)
    assert(atomicInc(mulrcvcnt, 0) == 15)
    return


proc test_chan0() =
    #crn_post(test_chan0impl, nil)
    #crn_post(test_chan2impl, nil)
    #crn_post(test_chan3impl, nil)
    # crn_post(test_chan5impl, nil)
    # crn_post(test_chan7impl, nil)
    crn_post(test_chan9impl, nil)
    return

