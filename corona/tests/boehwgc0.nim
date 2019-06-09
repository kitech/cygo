
type needgcty0 = ref object
    f0: int
    f1: float
    f2: string

proc test_boehmgc_finalizer(x:needgcty0) =
    linfo("finalizer", cast[pointer](x))
    return

proc test_boehmgc0() : needgcty0 =
    var v : needgcty0
    v.new(test_boehmgc_finalizer)
    v.f0 = 123
    v.f1 = 456
    v.f2 = "hehehehe"
    sleep(5000)
    linfo("done")
    return v

#var onegcv : needgcty0
proc test_boehmgc1() =
    var gcveq = newseq[needgcty0]()
    for i in 0..3:
        var v = test_boehmgc0()
        # gcveq.add v
        linfo(i, cast[pointer](v))
        #if i == 0: onegcv = v
        GC_fullcollect()
        v = nil # not set nil, not gced the last one!!!
        sleep(3000)
    while gcveq.len > 0: gcveq.delete(0)
    GC_fullcollect()
    for i in 0..5:
        GC_fullcollect()
        sleep(3000)
    linfo("done")
    return

proc runtest_boehmgc0() =
    noro_post(test_boehmgc1, nil)
    return

