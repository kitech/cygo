package iopoller

/*

#include <stdio.h>
#include <errno.h>
#include <time.h>
#include <sys/time.h>
#include <pthread.h>

#include <unistd.h>
#include <fcntl.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
#include <sys/epoll.h>
#include <sys/timerfd.h>
#include <sys/eventfd.h>
#include <signal.h>

   extern int pipe2();

*/
import "C"

// import sync
// import time

// import vcp.rtcom
// import vcp
// import vcp.mlog
// import vcp.epoll
// import vcp.iohook
// import vcp.futex


import (
	"epoll"
	"rtcom"
	"futex"
	"iohook"
)

// const vnil = voidptr(0)

func Keepme() {}

///////////////////

// before vinit
func pre_gc_init() {
}
func pre_main_init(yr *rtcom.Resumer) {
    println("iopoll premain init")
    resumer = yr
}
func pre_main_deinit() {

}

func init() {
    npl = &NetPoller{}
    npl.initccc()
}

var npl *NetPoller
var resumer *rtcom.Resumer

////////////////
// just Fiber header fields
struct FiberMin {
    grid int // coroutine id
    mcid int // machine id
}

struct Timeval {
    tv_sec int64
    tv_usec int64
}

struct Evdata{
    evtype uint32
    // data &FiberMin = vnil // fiber*
	data *FiberMin
    grid int
    mcid int
    ytype int
    fd int64 // fd or ns or hostname
    out voidptr // void** out; //
    errcode int // int *errcode;
    seqno int64 // long seqno;
    tv *Timeval // struct timeval tv; // absolute time
    usec int64 // = tv.tv_sec*USEC + tv.tv_usec
    evt voidptr // struct event* evt;
}
struct Evdata2 {
    evtype uint32
    //dr &Evdata = vnil
    //dw &Evdata = vnil
    //dt &Evdata = vnil
	dr *Evdata
	dw *Evdata
	dt *Evdata
}

struct NetPoller {
    //mut:
    epfd int// = -1
    seqno int64// = 10000
    evfds []*Evdata2
	// tmupfd [2]int // TODO compiler
    tmupfd [2]int // int tmupfd[2]; // timer add refresh notify epoll_wait pipe
    tmupevfd int// = -1
    timers *PQueue// = &PQueue{} // []&Evdata2 // in order
    //tmerlk &futex.Mutex = futex.newMutex()
	tmerlk *futex.Mutex
    watchers map[string]*Evdata2 // ev_watcher*??? =>
    //mu &sync.RwMutex = sync.new_rwmutex()
	mu *futex.Mutex
}


struct PQueue {
    items []*Evdata
}
func newPQueue() *PQueue {
	q := &PQueue{}
	return q
}
//fn (this &PQueue) len() int { return this.items.len }

func start() {
    // go poller_main()
	var arg voidptr
	// var th C.pthread_t // TODO compiler
	// th := &C.pthread_t{} // TODO compiler
	var th uint64
	C.pthread_create(&th, nil, poller_main, arg)
}

// vlib/builtin/cfns.c.v
// fn C.signal() int

func mysighandler(sig int) int {
    println("signal $sig")
    return 0
}
func poller_main(arg voidptr) {
    // C.signal(C.SIGPWR, mysighandler)
    // C.signal(C.SIGXCPU, mysighandler)
    npl.loop()
}

// func (this *NetPoller) init() { // TODO compiler, init() name not work
func (this *NetPoller) initccc() {
	this.seqno = 10000
	this.tmerlk = futex.newMutex()
	this.mu = futex.newMutex()
	this.timers = newPQueue()

    // this.evfds = []*Evdata2{len:999}
	var zeroptr voidptr
	this.evfds.appendn(&zeroptr, 999)

    fd := C.epoll_create1(C.EPOLL_CLOEXEC)
    this.epfd = fd

    rv := 0
    rv = C.pipe2(this.tmupfd, C.O_NONBLOCK)
    assert (rv == 0)

	// TODO compiler crash
	//fd0 := this.tmupfd[0]
	//fd1 := this.tmupfd[1]
	//println(fd0, fd1)

    tmupevfd := C.eventfd(0, C.EFD_CLOEXEC|C.EFD_NONBLOCK)
    this.tmupevfd = tmupevfd
    //mlog.info(@FILE, @LINE, "epfd $fd $tmupevfd")
}

const (
    SEC = 1
    MSEC = 1000
    USEC = 1000000
    NSEC = 1000000000
)

//[typedef]
//struct C.sigset_t {}
//fn C.sigemptyset() int
//fn C.sigaddset() int

func (this *NetPoller) loop() {
    rv := 0

    evt := &epoll.Event{}
    evt.events = epoll.IN
    evt.data.fd = this.tmupevfd

    rv = C.epoll_ctl(this.epfd, epoll.CTL_ADD, this.tmupevfd, evt)
	//println(this.epfd, this.tmupevfd, rv)
    assert(rv == 0)

    for {
        revt := &epoll.Event{}
        timeout := this.next_timeout()
        //timeout = if timeout < 0 { MSEC *10} else {timeout}
        //timeout = if timeout <= 31 { 31 } else { timeout }
		timeout = ifelse(timeout<0, MSEC*10, timeout)
		timeout = ifelse(timeout<=31, 31, timeout)

        sigset :=  &C.sigset_t{}
        C.sigaddset(sigset, C.SIGPWR)
        C.sigaddset(sigset, C.SIGXCPU)
        // mlog.info(@FILE, @LINE, "epoll_waiting ...", timeout)
		println("epoll_waiting ...", timeout)
        rv = epoll.wait(this.epfd, revt, 1, timeout)
        //rv = C.epoll_pwait(this.epfd, &revt, 1, timeout, &sigset)
        eno := C.errno
        //mlog.info(@FILE, @LINE, "waitret", rv, eno, this.timers.len())
        if rv < 0 {
            if eno == C.EINTR { continue }
            assert(1==2)
        }
        assert(rv >= 0)

        if rv == 0 {
            this.dispatch_timers()
            continue
        }

        evfd := revt.data.fd
        evts := revt.events
        //C.printf("gotevt %d %d %d\n", evfd, evts, this.tmupevfd)
        if evfd == this.tmupevfd {
            val := uint64(0)
            for {
                // TODO, maybe hooked, but not coroutine proc???
                rv = C.read(this.tmupevfd, &val, sizeof(uint64(0)))
                if rv <= 0 {break}
            }
            this.dispatch_timers()
            continue
        }
        rv = C.epoll_ctl(this.epfd, epoll.CTL_DEL, evfd, nil)

        // pthread_mutex_lock(&np->evmu);
        d2 := this.evfds[evfd]
        if d2 != nil {
            //this.evfds[evfd] = nil // clear
			var d2 *Evdata2 = nil // TODO compiler
			this.evfds[evfd] = d2
            // linfo("clear fd %d\n", evfd);
        }
        // pthread_mutex_unlock(&np->evmu);
        if d2 == nil {
            // linfo("wtf, fd not found %d\n", evfd);
        }else{
            // extern void netpoller_crnev_globcb(int epfd, int fd, int events, void* cbarg);
            // netpoller_crnev_globcb(np->epfd, evfd, evts, d2);
            // C.printf("evglobcb %d %d %d %p\n", this.epfd, evfd, evts, d2)
            this.evglobcb(this.epfd, evfd, evts, d2)
        }
        this.dispatch_timers()

        if false {
            // linfo("ohno, rv=%d\n", rv);
        }
    }
}

// 1/1000 秒, used in epoll_wait
func (thisp *NetPoller) next_timeout() int {
    timers := thisp.timers
    var d *Evdata
    tmlen := 0
    thisp.tmerlk.mlock()
    tmlen = timers.len()
    if timers.len() > 0 {
        d = timers.items[0]
    }
    thisp.tmerlk.munlock()
    if d != nil {
        tv := &Timeval{}
        C.gettimeofday(tv, nil)
        usec := tv.tv_sec * USEC + tv.tv_usec
        diffus := d.usec - usec
        // mlog.info(@FILE, @LINE, "epwait", d.data.grid, d.data.mcid, diffus/1000, tmlen, d.seqno)
        return int(diffus/1000)
    }
    return 500 // -1 // 500
}

// return dispatched count
func (thisp *NetPoller) dispatch_timers() int {
	this := thisp
    if thisp.timers.len() == 0 {
        return 0
    }

    tv := &Timeval{}
    C.gettimeofday(tv, nil)
    usec := tv.tv_sec * USEC + tv.tv_usec

    expires := []*Evdata{}
    thisp.tmerlk.mlock()
    for {
        tmcnt := thisp.timers.len()
        if tmcnt == 0 {
            break
        }
        d := thisp.timers.items[0]
        // mlog.info(@FILE, @LINE, d.usec, usec, d.usec <= usec, d.usec-usec)
        if d.usec <= usec {
            // thisp.timers.pop()
            this.timers.items.delete(0)
			expires.append(&d)
            // mlog.info(@FILE, @LINE, thisp.timers.len(), expires.len, d.data.mcid)
            //break
        }else {
            break
        }
    }
    thisp.tmerlk.munlock()

    cnt := expires.len
    if cnt == 0 { return cnt }
    len := thisp.timers.len()
	// println("dispatched timers cnt, left", cnt, len)
    for i := 0; i < cnt; i++{
        d := expires[i]
		// println("iopop", d.data.grid, d.data.mcid, len, d.seqno)
        thisp.resume(d)
    }
    for d in expires {} // logic bug?
    return cnt
}

func (thisp *NetPoller) resume(d *Evdata) {
    assert (resumer.resume_one != nil)

    dd := d.data
    ytype := d.ytype
    grid := d.grid
    mcid := d.mcid
    // evdata_free(d);
    // evdata2_free(d2);

	resume_one := resumer.resume_one
    resume_one(dd, ytype, grid, mcid)
	// resumer.resume_one(dd, ytype, grid, mcid) // TODO compiler
	// cgen: rtcom__Resumer_resume_one(dd, ytype, grid, mcid)
}

func (thisp *NetPoller) evglobcb(epfd int, fd int, events uint32, cbarg *Evdata2) {
    np := thisp

    // linfo("fd=%d events=%d cbarg=%p %p\n", fd, events, cbarg, 0);
    d2 := cbarg
    assert (d2 != nil)
    dv := []*Evdata{}

	events64 := int64(events)
    newev := uint32(0)
    rv := 0
    if d2.evtype == CXEV_TIMER {
        // close(fd)
        //dv << d2.dt
        d2.dt = nil
    } else if d2.evtype == CXEV_IO {
        if (events64 & epoll.IN) != 0 || (events64 & epoll.HUP) != 0 || (events64 & epoll.ERR) != 0 {
            if d2.dr != nil {
                //dv << d2.dr
                d2.dr = nil
            }
        }
        if (events64 & epoll.OUT) != 0 || (events64 & epoll.HUP) != 0 || (events64 & epoll.ERR) != 0 {
            if d2.dw != nil {
                // dv << d2.dw
                d2.dw = nil
            }
        }
        if (events64 & epoll.IN) == 0 && (events64 & epoll.OUT) == 0  {
            C.printf("woo, close, error??? %d %d %d(%d) %d(%d)\n",
                     fd, events, events64&epoll.HUP, epoll.IN, events64&epoll.ERR, epoll.OUT)
        }
        if d2.dr != nil { newev |= uint32(epoll.IN) }
        if d2.dw != nil { newev |= uint32(epoll.OUT) }
        if newev != 0 {
            np.evfds[fd] = d2
        }
        if newev != 0 {
            evt := &epoll.Event{}
            evt.events = newev | uint32(epoll.ET)
            evt.data.fd = fd
            rv = C.epoll_ctl(np.epfd, epoll.CTL_ADD, fd, &evt)
        }
    }else{
        C.printf("wtf fd=%d %d %d %d\n", fd, CXEV_IO, CXEV_TIMER, CXEV_DNS_RESOLV)
        // linfo("wtf fd=%d r=%d w=%d\n", fd, events&EPOLLIN, events&EPOLLOUT);
        C.printf("wtf fd=%d %p %d %p %p %p\n", fd, d2, d2.evtype, d2.dr, d2.dw, d2.dt)
        //assert(1==2)
    }

    dvcnt := dv.len
    assert(dvcnt > 0)
    for idx, d in dv {
        //thisp.resume(d)
    }
}

////// user trigger
const (
    // CXEV_IO = uint32(0x1 << 5)
    // CXEV_TIMER = uint32(0x1 << 6)
    // CXEV_DNS_RESOLV = uint32(0x1 << 7)
	CXEV_IO = 0x1 << 5
	CXEV_TIMER = 0x1 << 6
	CXEV_DNS_RESOLV = 0x1 << 7
)

func newEvdata(evt uint32, gr *FiberMin) *Evdata{
    if gr == nil { abort() }
    this := &Evdata{}
    this.evtype = evt
    this.data = gr
    this.grid = gr.grid
    this.mcid = gr.mcid
	this.tv = &Timeval{}
    return this
}
func newEvdata2(evtype uint32) *Evdata2 {
    this := &Evdata2{}
    this.evtype = evtype
    return this
}

func (this *PQueue) push(d *Evdata) {
    found := false
    for i, item in this.items {
        if d.usec < item.usec {
            // this.items.insert(i, d) // TODO
			this.items.insert(i, &d)
            found = true
            break
        }
    }
    // if !found { this.items << d }
	if !found { this.items.append(&d) }
}
func (this *PQueue) pop() *Evdata {
    if this.items.len == 0 {
        return nil
    }
    item := this.items[0]
    this.items.delete(0)
    return item
}
func (this *PQueue) first() *Evdata {
    if this.items.len == 0 {
        return nil
    }
    return this.items[0]
}
func (thisp *PQueue) len() int { return thisp.items.len }

func (this *NetPoller) add_timer(ns int64, ytype int, gr *FiberMin) {
    np := this

    d := newEvdata(CXEV_TIMER, gr)
    d.ytype = ytype
    d.fd = ns
    C.gettimeofday(d.tv, nil)
    usec := d.tv.tv_sec*USEC + d.tv.tv_usec + int64(ns/1000)
    d.usec = usec
    d.tv.tv_sec = usec / USEC
    d.tv.tv_usec = usec % USEC
    d.seqno = np.seqno
	np.seqno ++

    rv := 0
    rv0 := 0
    // crn_pre_gclock_proc(__func__);
    // pthread_mutex_lock(&np->evmu);
    // rv = pqueue_push(np->timers, d);
    np.tmerlk.mlock()
    rv0 = np.timers.len()
    np.timers.push(d)
    rv = np.timers.len()
    np.tmerlk.munlock()
    // mlog.info(@FILE, @LINE, "iopush", d.data.grid, d.data.mcid, rv0, rv, ns, d.seqno)
    assert (rv == rv0+1)
    // pthread_mutex_unlock(&np->evmu);
    // crn_post_gclock_proc(__func__);
    tmval := uint64(1)
    rv = C.write(np.tmupevfd, &tmval, sizeof(uint64(0)))
    ok := rv == sizeof(uint64(0))
    if !ok {
        C.printf("add error %d %ld %d\n", rv, ns, gr.grid)
        // evdata2_free(tmer);
        // evdata_free(d);
        // assert(rv == 0);
        return
    }

    // linfo("timer add %d d=%p %ld sec=%d nsec=%d\n", tmfd, d, ns, ts.tv_sec, ts.tv_nsec);
}


func (this *NetPoller) add_writefd(fd int64, ytype int, gr *FiberMin) {
    np := this
    d := newEvdata(CXEV_IO, gr)
    d.ytype = ytype
    d.fd = fd

    rv := 0
    // crn_pre_gclock_proc(__func__);
    // pthread_mutex_lock(&np->evmu);
    inuse := np.evfds[fd] != nil
    if inuse {
        d2 := np.evfds[fd]
        dwuse := d2.dw != nil
		samefib := dwuse && d2.dw.data == gr
        override := false
        if dwuse && samefib {
        }else{
            override = true
            d2.dw = d
            newev := epoll.ET
            if d2.dr != nil { newev |= epoll.IN }
            newev |= epoll.OUT
            rv = C.epoll_ctl(np.epfd, epoll.CTL_DEL, fd, 0)
            // assert(rv == 0);
            evt := &epoll.Event{}
            evt.events = newev
            evt.data.fd = int(fd)
            rv = C.epoll_ctl(np.epfd, epoll.CTL_ADD, fd, &evt)
        }
        if dwuse && samefib && !override {
            // ignored operation
        }else{
            C.printf("add w reset %d dwuse=%d samefib=%d override=%d\n",
                   fd, dwuse, samefib, override)
        }
    }else{
        d2 := newEvdata2(d.evtype)
        d2.dw = d
        evt := &epoll.Event{}
        evt.events = epoll.OUT | epoll.ET
        evt.data.fd = int(fd)
        np.evfds[fd] = d2
        rv = C.epoll_ctl(np.epfd, epoll.CTL_ADD, fd, &evt)
        // linfo("add w new %d\n", fd);
    }
    // pthread_mutex_unlock(&np->evmu);
    // crn_post_gclock_proc(__func__);
    if rv != 0 {
        C.printf("add error %d %d %d\n", rv, fd, gr.grid)
        // evdata2_free(evt);
        // evdata_free(d);
        // assert(rv == 0);
        return
    }

    // linfo("evwrite add d=%p %ld\n", d, fd);
}
func (this *NetPoller) add_readfd(fd int64, ytype int, gr *FiberMin) {
    np := this
    d := newEvdata(CXEV_IO, gr)
    d.ytype = ytype
    d.fd = fd

    // crn_pre_gclock_proc(__func__);
    rv := 0
    // pthread_mutex_lock(&np->evmu);
    inuse := np.evfds[fd] != nil
    // assert (inuse == 0);
    if inuse {
        d2 := np.evfds[fd]
        druse := d2.dr != nil
		samefib := druse && d2.dr.data == gr
        override := false
        if druse && samefib {
            // ignore ok?
        }else{
            override = true
            d2.dr = d
            newev := epoll.ET
            if d2.dw != nil { newev |= epoll.OUT }
            newev |= epoll.IN
            rv = C.epoll_ctl(np.epfd, epoll.CTL_DEL, fd, 0)
            assert(rv == 0)
            evt := &epoll.Event{}
            evt.events = newev
            evt.data.fd = int(fd)
            rv = C.epoll_ctl(np.epfd, epoll.CTL_ADD, fd, &evt)
        }
        if druse && samefib && !override {
            // ignored operation
        }else{
            C.printf("add r reset %d druse=%d samefib=%d override=%d\n",
                     fd, druse, samefib, override)
        }
    }else{
        d2 := newEvdata2(d.evtype)
        d2.dr = d
        evt := &epoll.Event{}
        evt.events = epoll.IN | epoll.ET
        evt.data.fd = int(fd)
        np.evfds[fd] = d2
        rv = C.epoll_ctl(np.epfd, epoll.CTL_ADD, fd, &evt)
        // linfo("add r new %d\n", fd);
    }
    // pthread_mutex_unlock(&np->evmu);
    // crn_post_gclock_proc(__func__);
    if rv != 0 {
        C.printf("add error %d %d %d\n", rv, fd, gr.grid)
        // evdata2_free(d2);
        // evdata_free(d);
        // assert(rv == 0);
        return
    }

    if d != nil {
        // linfo("event_add d=%p fd=%d ytype=%d rv=%d\n", d, fd, ytype, rv);
    }
}

// when ytype is SLEEP/USLEEP/NANOSLEEP, fd is the nanoseconds
func yieldfd(fd int64, ytype int, gr *FiberMin) {
    assert(ytype > iohook.YIELD_TYPE_NONE)
    assert(ytype < iohook.YIELD_TYPE_MAX)

    // struct timeval tv = {0, 123};
    /* match ytype { */
    /*     iohook.YIELD_TYPE_SLEEP {} */
    /*     iohook.YIELD_TYPE_MSLEEP {} */
    /*     iohook.YIELD_TYPE_USLEEP {} */
    /*     iohook.YIELD_TYPE_NANOSLEEP {} */
    /*     // event_base_loopbreak(gnpl__->loop); */
    /*     // event_base_loopexit(gnpl__->loop, &tv); */
    /* } */
	// println("fd", fd, "ytype", ytype, "gr", gr)

    ns := int64(0)
    if  ytype == iohook.YIELD_TYPE_SLEEP {
        ns = fd*1000000000
        npl.add_timer(ns, ytype, gr)
    } else if ytype == iohook.YIELD_TYPE_MSLEEP {
        ns = fd*1000000
        npl.add_timer(ns, ytype, gr)
    } else if ytype == iohook.YIELD_TYPE_USLEEP {
        ns = fd*1000
        npl.add_timer(ns, ytype, gr)
    } else if ytype == iohook.YIELD_TYPE_NANOSLEEP{
        ns = fd
        npl.add_timer(ns, ytype, gr)
    } else if ytype == iohook.YIELD_TYPE_CHAN_SEND{
        assert(1==2)// cannot process this type
        npl.add_timer(1000, ytype, gr)
    } else if ytype == iohook.YIELD_TYPE_CHAN_RECV {
        assert(1==2)// cannot process this type
        npl.add_timer(1000, ytype, gr)
    } else if ytype == iohook.YIELD_TYPE_CONNECT ||
        ytype == iohook.YIELD_TYPE_WRITE ||
        ytype == iohook.YIELD_TYPE_WRITEV ||
        ytype == iohook.YIELD_TYPE_SEND ||
        ytype == iohook.YIELD_TYPE_SENDTO ||
        ytype == iohook.YIELD_TYPE_SENDMSG {
            npl.add_writefd(fd, ytype, gr)
            // case YIELD_TYPE_READ: case YIELD_TYPE_READV:
            // case YIELD_TYPE_RECV: case YIELD_TYPE_RECVFROM: case YIELD_TYPE_RECVMSG:
    } else if ytype == iohook.YIELD_TYPE_GETADDRINFO {
        //    netpoller_dnsresolv((char*)fd, ytype, gr);
    } else {
        // linfo("add reader fd=%d ytype=%d=%s\n", fd, ytype, yield_type_name(ytype));
        assert(fd >= 0)
        npl.add_readfd(fd, ytype, gr)
    }
}

