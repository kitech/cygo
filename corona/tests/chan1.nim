
# test public chan api
import strutils
import tables

# test tostr
proc test_chan10() =
    var c = makechan(int, 5)
    assert(($c).startsWith("chan[int; 5]@0x"))
    c = nil
    #c.send(8)
    #var v = c.recv()
    #c <- 9
    #var v2 = <- c
    #<- c

# test int
proc test_chan11() =
    var c = makechan(int, 5)
    var br = c.send(8)
    assert(br == true)
    var rv = c.recv()
    assert(rv == 8, rv.repr)
    return

# test string
proc test_chan12() =
    var c = makechan(string, 5)
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
    var c = makechan(tstobj, 5)
    var o = tstobj(a:567, b : "abc", c: 999.9)
    var br = c.send(o)
    assert(br == true)
    var rv = c.recv()
    assert(rv == o, rv.repr)
    return

# test seq/table
proc test_chan14() =
    var c = makechan(seq[int], 5)
    var o = newseq[int](3)
    var br = c.send(o)
    assert(br == true)
    var rv = c.recv()
    assert(rv == o, rv.repr)
    return

# test seq/TableRef
proc test_chan15() =
    var c = makechan(TableRef[int, int], 5)
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
    var c = makechan(int, 5)
    c <- 9
    var rv = c.recv()
    assert(rv == 9, rv.repr)

    c.send(8)
    rv = <- c
    assert(rv == 8, rv.repr)
    return

proc runtest_chan1(cnter:int) =
    test_chan10()
    # test_chan11()
    # test_chan12()
    # test_chan13()
    # test_chan14()
    # test_chan15()
    test_chan16()
    return

