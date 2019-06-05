
# test public chan api
import strutils
import tables

# test tostr
proc test_chan10() =
    var c = newchan(int, 5)
    assert(($c).startsWith("chan[int; 5]@0x"))
    c = nil
    #c.send(8)
    #var v = c.recv()
    #c <- 9
    #var v2 = <- c
    #<- c

# test int
proc test_chan11() =
    var c = newchan(int, 5)
    var br = c.send(8)
    assert(br == true)
    var rv = c.recv()
    assert(rv == 8, rv.repr)
    return

# test string
proc test_chan12() =
    var c = newchan(string, 5)
    var br = c.send("abc")
    assert(br == true)
    var rv = c.recv()
    assert(rv == "abc", rv.repr)
    return

type tstobj = ref object
    a: int
    b: string
    c: float
    d: pointer

# test struct
proc test_chan13() =
    var c = newchan(tstobj, 5)
    var o = tstobj(a:567, b : "abc", c: 999.9)
    var br = c.send(o)
    assert(br == true)
    var rv = c.recv()
    assert(rv == o, rv.repr)
    return

# test seq/table
proc test_chan14() =
    var c = newchan(seq[int], 5)
    var o = newseq[int](3)
    var br = c.send(o)
    assert(br == true)
    var rv = c.recv()
    assert(rv == o, rv.repr)
    return

# test seq/TableRef
proc test_chan15() =
    var c = newchan(TableRef[int, int], 5)
    var o : TableRef[int,int]
    o = newTable[int,int]()
    o.add(1, 2)
    o.add(3, 4)
    o.add(5, 6)
    var br = c.send(o)
    assert(br == true)
    var rv = c.recv()
    assert(rv == o, rv.repr)
    return

# test operator `send` and `recv`
proc test_chan16() =
    var c = newchan(int, 5)
    c <- 9
    var rv = c.recv()
    assert(rv == 9, rv.repr)

    c.send(8)
    rv = <- c
    assert(rv == 8, rv.repr)
    return

# test select operator
# all block recv
proc test_chan_select_all_block_recv() =
    var hlpunblock = proc(c1p: pointer) =
        var c1 = cast[chan[pointer]](c1p)
        usleepc(2000000)
        linfo("send one to unblock select")
        c1.send(cast[pointer](3))
        return

    var c0 = newchan(int, 0)
    var c1 = newchan(pointer, 0)
    var c2 = newchan(float, 0)

    var hcs = newseq[scase](0)
    hcs.add(scase(hc: c0.hc, kind: caseRecv))
    hcs.add(scase(hc: c1.hc, kind: caseRecv))
    hcs.add(scase(hc: c2.hc, kind: caseRecv))

    noro_post(hlpunblock, cast[pointer](c1))
    var casi = select1(hcs)
    linfo(casi, c1.toelem(hcs[casi].hcelem))

    return

proc test_chan_select_nocase() =
    return

proc runtest_chan1(cnter:int) =
    test_chan10()
    # test_chan11()
    # test_chan12()
    # test_chan13()
    # test_chan14()
    # test_chan15()
    # test_chan16()
    noro_post(test_chan_select_all_block_recv, nil)
    return

