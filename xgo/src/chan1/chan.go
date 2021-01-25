package chan1

/*
   #include <stdio.h>
   #include <stdlib.h>
   #include <string.h>

   void temp_print_chan1(int no) {
      switch (no) {
        case 0:
          printf("chan1 premain init\n");
          break;
        default:
          printf("chan1 print what %d\n", no);
          break;
      }
   }

   void chan1_avoid_cxcallable_resume(void* fnptr, void* gr, int ytype, int grid, int mcid) {
       void(*fnobj)() = fnptr;
       fnobj(gr, ytype, grid, mcid);
   }

   void chan1_avoid_cxcallable_yield(void* fnptr, int64_t fdns, int ytype) {
       void(*fnobj)() = fnptr;
       fnobj(fdns, ytype);
   }
   void* chan1_avoid_cxcallable_getcoro(void* fnptr) {
       void* (*fnobj)() = fnptr;
       return fnobj();
   }
*/
import "C"
import "rtcom"
import "futex"
import "iohook"
import "atomic"

// import vcp.mlog
// import vcp.rtcom
// import vcp.futex
// import vcp.iohook

// const vnil = voidptr(0)

func Keepme() {}

func pre_main_init(yielder *rtcom.Yielder, resumer *rtcom.Resumer, allocer voidptr) {
    //C.printf("chan1 premain init\n")
	C.temp_print_chan1(0)
    ylder = yielder
    rsmer = resumer
    alcer = allocer
}

func post_main_deinit() {
}

var (
    ylder *rtcom.Yielder
    alcer voidptr
    rsmer *rtcom.Resumer
)

func init() {

}

//////////////////
// unsafe channel, v not support generic enough
struct Chan1Impl {
    len int // alias qcount
    cap int // alias datasize
    elemsize int
    //elemtype string
	elemtype byteptr
    // elems []voidptr
    // elems array
    elems voidptr// = 0

    //
    closed uint32 // atomic
    sendx int
    recvx int
    recvq []*Fiber // fiber link list
    sendq []*Fiber // fiber link list

    // mu &futex.Mutex = futex.newMutex()
	mu *futex.Mutex
}

struct Chan1 {
	//    mut:
    ci *Chan1Impl// = 0
}

// just Fiber header fields
struct Fiber {
    grid int // coroutine id
	//mut:
    mcid int // machine id

    elem voidptr // channel recv/send var addr
    fromgr *Fiber// = 0
    channel *Chan1Impl// = 0 // sending channel
    releasetime int64
    param voidptr
    isselect bool
}

struct Waitq {
    //mut:
    objs []voidptr
}

func new(cap int, elemsz int, tyname byteptr) *Chan1 {
	ci := &Chan1Impl{}
	ci.cap = ifelse(cap < 0, 0, cap)
	ci.elemsize = elemsz
	ci.elemtype = tyname
	ci.mu = futex.newMutex()

	if ci.cap > 0 {
		ci.elems = mallocgc(ci.cap*ci.elemsize)
	}

	ch := &Chan1{}
	ch.ci = ci
	return ch
}

// not encourage usage
// fn C.__new_array() array
// fn C.array_set() int
// fn C.array_get() voidptr

// pub fn new<T>(cap int) Chan1 {
//     tv := T{}
//     mut ci := &Chan1Impl{}
//     ci.cap = if cap < 0 { 0 } else {cap}
//     ci.elemsize = int(sizeof(tv))
//     ci.elemtype = T.name
//     if ci.cap > 0 {
//         // ci.elems = []voidptr{len: ci.cap}
//         // ci.elems = C.__new_array(ci.cap, ci.cap, ci.elemsize)
//         ci.elems = malloc(ci.cap*ci.elemsize)
//     }
//     return Chan1{ci}
// }
// pub fn (ch Chan1) len() int { return ch.ci.len }
// pub fn (ch Chan1) cap() int { return ch.ci.cap }
// pub fn (ch Chan1) elemsize() int { return ch.ci.elemsize }
func (ch*Chan1) len() int { return ch.ci.len }
func (ch*Chan1) cap() int { return ch.ci.cap }
func (ch*Chan1) elemsize() int { return ch.ci.elemsize }

// ep = elem pointer
func (ch *Chan1) send0(ep voidptr, block bool) bool {
    ci := ch.ci
    var mysg *Fiber = nil
    // mysg = ylder.getcoro() // TODO compiler
	mysg = C.chan1_avoid_cxcallable_getcoro(ylder.getcoro)

    ci.mu.mlock()
    if ci.closed != 0 {
        ci.mu.munlock()
        panic("send on closed channel")
    }

    if ci.recvq.len > 0 {
        sg := ci.recvq[0]
        ci.recvq.delete(0)
        ch.send_direct0(ci.elemsize, sg, ci.mu, ep)
        return true
    }

    if ci.len < ci.cap {
        // 只是主线程sleep中断太多，过早退出进程，然而在退出时gcdeinit了
        // if C.GC_is_init_called() == 0 {
        //     C.printf("some err %d %d\n", mysg.grid, mysg.mcid)
        //     C.abort()
        // }
        // println("sndbuf $ci.len, ci.cap")
        // 导致 GC_init()多次调用，然后崩溃
        // mlog.info(@FILE, @LINE, "sndbuf", ci.len, ci.cap) ///
        // C.printf("sndbuf %d %d\n", ci.len, ci.cap) // OK
        // ci.elems.set(ci.sendx, ep) // private
        // C.array_set(&ci.elems, ci.sendx, ep)
        offptr := voidptr(usize(ci.elems)+usize(ci.sendx*ci.elemsize))
        C.memcpy(offptr, ep, ci.elemsize)
        ci.sendx++
        if ci.sendx == ci.cap {
            ci.sendx = 0
        }
        ci.len ++
        ci.mu.munlock()
        return true
    }

    if !block{
        ci.mu.munlock()
        return false
    }

    // blocking part
    mysg.elem = ep
    mysg.channel = ci
    // ci.sendq << mysg
	ci.sendq.append(&mysg)

    ci.mu.munlock()
    //mlog.info(@FILE, @LINE, "sndblk", ci.len, ci.cap)
    //ylder.yield(0, iohook.YIELD_TYPE_CHAN_SEND) // TODO compiler
	C.chan1_avoid_cxcallable_yield(ylder.yield, 0, iohook.YIELD_TYPE_CHAN_SEND)

    mysg.param = nil
    mysg.channel = nil
    return true
}

// just like golang send
func (ch *Chan1) send_direct0(elemsize int, sgx *Fiber, mu *futex.Mutex, ep voidptr) {
    sg := sgx
    if sg.elem != nil {
        ch.send_direct1(elemsize, sgx, ep)
        sg.elem = nil
        sg.fromgr = nil
    }
    mu.munlock()
    sg.param = sg
	C.chan1_avoid_cxcallable_resume(rsmer.resume_one, sg, iohook.YIELD_TYPE_CHAN_RECV, sg.grid, sg.mcid)
    //rsmer.resume_one(sg, iohook.YIELD_TYPE_CHAN_RECV, sg.grid, sg.mcid) // TODO compiler
}
func (ch *Chan1) send_direct1(elemsize int, sgx *Fiber, src voidptr) {
    C.memcpy(sgx.elem, src, elemsize)
}

// [inline]
// pub fn (ch Chan1) send<T>(v T) {
//     assert int(sizeof(v)) == ch.ci.elemsize
//     ch.send0(voidptr(&v), true)
// }
// pub fn (ch Chan1) sendnb<T>(v T) bool {
//     assert int(sizeof(v)) == ch.ci.elemsize
//     ch.send0(voidptr(&v), false)
//     return false
// }

// usage send(&var)
func (ch *Chan1) send(v voidptr) bool {
	ch.send0(v, true)
	return true
}
func (ch *Chan1) sendnb(v voidptr) bool {
	ch.send0(v, false)
	return true
}

// // 没有编译器帮助，这个接收语法太难用了

func (ch *Chan1) recv0(ep voidptr, block bool)  {
    ci := ch.ci
    // two return value
    selected := false
    received := false

    if ci == nil {
        if !block {
            return
        }
        //mlog.info(@FILE, @LINE, "park forever")
		println("park forever")
        // park
        // ylder.yield(0, iohook.YIELD_TYPE_CHAN_RECV_CLOSED)
		C.chan1_avoid_cxcallable_yield(ylder.yield, 0, iohook.YIELD_TYPE_CHAN_RECV_CLOSED)
        panic("unreachable")
    }

    // mlog.info(@FILE, @LINE, "recv lock")
    ci.mu.mlock()

    if ci.closed != 0 && ci.len == 0 {
        //mlog.info(@FILE, @LINE, "recv unlock, closed")
		println("recv unlock, closed")
        ci.mu.munlock()
        if ep != nil {
            // typedmemclr()
        }
        selected = true
        return
    }

    if ci.sendq.len > 0 {
        // mlog.info(@FILE, @LINE, "recv sendq>0, direct")
        sg := ci.sendq[0]
        ci.sendq.delete(0)
        ch.recv_direct0(sg, ep, ci.mu)
        selected = true
        received = true
        return
    }

    if ci.len > 0 {
        // mlog.info(@FILE, @LINE, "rcvbuf len>0, read", ci.len)
        offptr := voidptr(usize(ci.elems)+usize(ci.recvx*ci.elemsize))
        qp := offptr
        // qp := C.array_get(ci.elems, ci.recvx)
        if ep != nil {
            C.memcpy(ep, qp, ci.elemsize)
        }
        C.memset(qp, 0, ci.elemsize)
        ci.recvx++
        if ci.recvx == ci.cap {
            ci.recvx = 0
        }
        ci.len--
        ci.mu.munlock()
        selected = true
        received = true
        return
    }

    if !block {
        //mlog.info(@FILE, @LINE, "need block, return")
		println("need block, return")
        ci.mu.munlock()
        return
    }

    var mysg *Fiber = nil
    // mysg = ylder.getcoro()// TODO compiler
	mysg = C.chan1_avoid_cxcallable_getcoro(ylder.getcoro)
    mysg.elem = ep
    mysg.channel = ci
    //ci.recvq << mysg
	ci.recvq.append(&mysg)

    // mlog.info(@FILE, @LINE, "need park", mysg.grid, mysg.mcid)
    ci.mu.munlock()
    //ylder.yield(0, iohook.YIELD_TYPE_CHAN_RECV) // TODO compiler
	C.chan1_avoid_cxcallable_yield(ylder.yield, 0, iohook.YIELD_TYPE_CHAN_RECV)
    // mlog.info(@FILE, @LINE, "recv park waked", mysg.grid, mysg.mcid)

    mysg.elem = nil
    mysg.channel = nil

    closed := false // gp.param == vnil
    closed = mysg.param == nil
    selected = true
    received = !closed
    return
}
func (ch *Chan1) recv_direct0(sgx *Fiber, ep voidptr, mu *futex.Mutex, ) {
    ci := ch.ci
    sg := sgx

    if ci.cap == 0 {
        if ep != nil {
            // recv_direct1
            C.memcpy(ep, sg.elem, ci.elemsize)
        }
    }else{
        offptr := voidptr(usize(ci.elems)+usize(ci.recvx*ci.elemsize))
        qp := offptr
        // qp := C.array_get(ci.elems, ci.recvx)
        if ep != nil {
            C.memcpy(ep, qp, ci.elemsize)
        }
        C.memcpy(qp, sg.elem, ci.elemsize) // ????
        ci.recvx ++
        if ci.recvx == ci.cap {
            ci.recvx = 0
        }
        ci.sendx =  ci.recvx
    }

    sg.elem = nil
    mu.munlock()
    sg.param = sg
	C.chan1_avoid_cxcallable_resume(rsmer.resume_one, sg, iohook.YIELD_TYPE_CHAN_RECV, sg.grid, sg.mcid)
    // rsmer.resume_one(sg, iohook.YIELD_TYPE_CHAN_RECV, sg.grid, sg.mcid)
}


// // not safe
// // usage: ret := int(0); ch.recv(&ret)
// [inline]
// pub fn (ch Chan1) recv1(ret voidptr) {
//     ch.recv0(ret, true)
// }
// pub fn (ch Chan1) recvnb1(ret voidptr) {
// }
// usage recv1(&var)
func (ch *Chan1) recv1(ret voidptr){
	ch.recv0(ret, true)
}
func (ch *Chan1) recvnb1(ret voidptr){
	ch.recv0(ret, false)
}

// // safe
// // usage: ret = ch.recv2(ret)
// pub fn (ch Chan1) recv2<T>(v T) T {
//     return v
// }
// pub fn (ch Chan1) recvnb2<T>(v T) T {
// }
// usage varptr = recv1()
func (ch *Chan1) recv2() voidptr{
	ret := mallocgc(ch.ci.elemsize)
	ch.recv0(ret, true)
	return ret
}
func (ch *Chan1) recvnb2() voidptr{
	ret := mallocgc(ch.ci.elemsize)
	ch.recv0(ret, false)
	return ret
}


// // safe
// // usage: ret = ch.recv3<int>()
// pub fn (ch Chan1) recv3<T>() T {
//     v := T{}
//     return v
// }
// pub fn (ch Chan1) recvnb3<T>() T {
//     v := T{}
//     return v
// }

func (ch *Chan1) close() {
	ok := atomic.CmpXchg32(&ch.ci.closed, 0, 1)
	if ok {
		// clean
	}
}
func (ch *Chan1) closed() bool {
	return atomic.Load32(&ch.ci.closed) == 1
}

// // this works
// fn (ch Chan1) sendval<T>(v T) {
//     println(v)
// }
// // this works
// fn (ch Chan1) recvval<T>(v T) T {
//     v2 := T{}
//     println(v)
//     return v2
// }
// // this works, ret := ch.recvval2<int>()
// fn (ch Chan1) recvval2<T>() T {
//     v2 := T{}
//     println(v2)
//     return v2
// }

////////
/*
struct Chan2<T> {
    pub mut:
    x []T
}
fn new2<T>() Chan2<T> {
    mut ch := Chan2<T>{}
    return ch
}
*/
