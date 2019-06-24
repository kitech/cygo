
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

    crn_post(hlpunblock, cast[pointer](c1))
    var casi = goselect1(hcs)
    linfo(casi, c1.toelem(hcs[casi].hcelem))

    return

proc test_chan_select_nocase1() =
    var arr : array[0,scase]
    var rv = goselect1(arr)
    doAssert(1==2)
    return

# with macro goselect
proc test_chan_select_nocase2() =
    expandMacros: goselectv6: discard
    return

# only expand, need to take look the result
proc test_chan_goselect_macrov5() =
    var c0 = newchan(int, 1)
    var c1 = newchan(int32, 1)
    var c2 = newchan(float, 1)
    var c3 = newchan(string, 1)
    var c4 = newchan(pointer, 1)
    var sc0 = newscase(c0, caseRecv)
    var sc1 = newscase(c1, caseRecv)
    var sc2 = newscase(c2, caseRecv)
    var sc3 = newscase(c3, caseRecv)
    var sc4 = newscase(c4, caseRecv)

    var valis = newseq[int](3)
    var val0 : int
    expandMacros: goselectv5:
        scase c0 <- val0:
            echo "000"
            echo "aaa"
            discard
        scase val0 = <- c0:
            echo "111"
            echo "bbb"
            discard
        scase <- c0:
            echo "222"
            echo "ccc"
            discard
        scase valis[0] = <- c0:
            echo "333"
            echo "ddd"
            discard
        default:
            echo "444"
            echo "eee"
            discard
        scase c1 <- 32:
            echo "555"
            echo "fff"
            discard
        scase c3 <- "sss":
            echo "666"
            echo "ggg"
            discard
        scase c4 <- nil:
            echo "777"
            echo "hhh"
            discard
        scase c2 <- 5.678:
            echo "888"
            echo "jjj"
            discard
    return

proc test_chan_goselect_macro() =
    return

proc runtest_chan1(cnter:int) =
    test_chan10()
    # test_chan11()
    # test_chan12()
    # test_chan13()
    # test_chan14()
    # test_chan15()
    # test_chan16()
    # crn_post(test_chan_select_all_block_recv, nil)
    test_chan_goselect_macrov5()
    # crn_post test_chan_select_nocase1, nil
    # crn_post test_chan_select_nocase2, nil
    return

