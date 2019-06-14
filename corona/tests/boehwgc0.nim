
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
# 与预期相符。虽然最后一个变量在返回之后才回收的
proc test_boehmgc1() =
    var gcveq = newseq[needgcty0]()
    for i in 0..3:
        var v = test_boehmgc0()
        # gcveq.add v
        linfo(i, cast[pointer](v))
        #if i == 0: onegcv = v
        sleep(3000)
    while gcveq.len > 0: gcveq.delete(0)
    linfo("done")
    return

# 在goroutine中申请内存，然后yield 10秒，看其是否会被回收
# 在 yield时间范围内并不会回收，在函数结束以后会回收。和预期行为一致。
proc test_boehmgc2() =
    var v : needgcty0
    v.new(test_boehmgc_finalizer)

    sleep(10000)
    linfo("done")
    return

# 在 yield 之前被回收，和预期行为一致。
proc test_boehmgc3() =
    var v : needgcty0
    v.new(test_boehmgc_finalizer)
    v = nil

    sleep(10000)
    linfo("done")
    return

# 永远不会被回收，和预期行为一致。
var gcv4 : pointer
proc test_boehmgc4() =
    var v : needgcty0
    v.new(test_boehmgc_finalizer)
    gcv4 = cast[pointer](v)

    sleep(10000)
    linfo("done")
    return

# test5/6, 预期5秒后会被回收
# TODO 和预期不符合，还是在10秒后回收的???
proc test_boehmgc5(vp:pointer) =
    sleep(5000)
    return
proc test_boehmgc6() =
    var v : needgcty0
    v.new(test_boehmgc_finalizer)
    var vp = cast[pointer](v)
    linfo("newed", vp)
    noro_post(test_boehmgc5, vp)
    v = nil
    vp = nil

    sleep(10000)
    linfo("done")
    return

# test7/8, 预期15秒后会被回收
# 和预期行为一致
proc test_boehmgc7(vp:pointer) =
    sleep(15000)
    return
proc test_boehmgc8() =
    var v : needgcty0
    v.new(test_boehmgc_finalizer)
    var vp = cast[pointer](v)
    linfo("newed", vp)
    noro_post(test_boehmgc7, vp)
    v = nil
    vp = nil

    sleep(10000)
    linfo("done")
    return

proc runtest_boehmgc0() =
    noro_post(test_boehmgc1, nil)
    #noro_post(test_boehmgc6, nil)
    #noro_post(test_boehmgc8, nil)
    return

