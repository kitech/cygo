package sched

/*
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/mman.h>

__thread int co_sched_mcid = -1;
__thread int co_sched_grid = -1;
__thread void* co_sched_mcobj = 0;
__thread void* co_sched_grobj = 0;


void temp_print_sched(int which) {
  switch (which) {
  case 0:
    printf("sched pregc init ...\n");
    break;
  case 1:
    printf("sched pregc init done\n");
    break;
  default:
    printf("sched wtt %d\n", which);
    break;
  }
}
*/
import "C"

import (
	"iohook"
	"iopoller"
	"rtcom"
	"futex"
	"atomic"
	"coro"
	"vmm"
)

func Keepme() {
	if false {
		rtcom.Keepme()
		iohook.Keepme()
		iopoller.Keepme()
	}
}

// gc not inited
func pre_gc_init() {
	// cannot alloc here
	C.temp_print_sched(0)
	// C.printf("sched pregc init\n") // TODO
	//yielder := rtcom.Yielder{} // TODO
	// yielder := &rtcom.Yielder{}
	// yielder.incoro = incoro
	// yielder.getcoro = getcoro
	// yielder.yield = onyield
	// yielder.yield_multi = onyield_multi
	// resumer := &rtcom.Resumer{}
	// resumer.resume_one = onresume
	// rtcom.pre_gc_init(&yielder, &resumer, (voidptr)(0))
	// iohook.pre_main_init(rtcom.yielder(), (voidptr)(0))

	rtcom.pre_gc_init(incoro, getcoro, onyield, onyield_multi,
		onresume, nil, nil, nil, nil)
	C.temp_print_sched(1)
	iohook.pre_main_init(incoro, getcoro, onyield, onyield_multi,
		nil, nil, nil, nil)
	C.temp_print_sched(1)
}

// gc inited
func pre_main_init() {
	println("sched premain init")
	yielder := rtcom.yielder()
	resumer := rtcom.resumer()
	iopoller.pre_main_init(resumer)
	// chan1.pre_main_init(yielder, resumer, voidptr(0))
	println("y&r", yielder, resumer)
}
func post_main_deinit() {
	// TODO需要退出coroutine以及procer
}

func init() {
	tm := C.time(nil)
	C.srand(&tm)
	iopoller.start()

	// myself
	//schedobj = &Schedule{}
	schedobj = newSchedule()
	schedobj.init_machines()
}

//////////////////

// return 0 or >0
func incoro() int {
	mcid := getmcid()
	if mcid == -1 {
		return 0
	}
	return mcid
}
func getcoro() voidptr {
	return C.co_sched_grobj
}

func onyield(fdns int64, ytype int) int {
    //mlog.info(@FILE, @LINE, "fdns", fdns, "ytype", ytype)
    mcid := 0
    var mcobj *Machine = nil
    machine_get(&mcid, &mcobj)
    grid := 0
    var grobjx voidptr
    fiber_get(&grid, &grobjx)
    grobj := (*Fiber)(grobjx)
    isco := incoro()
    // mlog.info(@FILE, @LINE, fdns, ytype, grid, grobjx, mcid, isco)

	var grmin *iopoller.FiberMin = nil
	grmin = grobjx

    // grobj.savestack()
    if ytype == iohook.YIELD_TYPE_SLEEP {
        iopoller.yieldfd(fdns, ytype, grobjx)
    } else if ytype == iohook.YIELD_TYPE_USLEEP {
        iopoller.yieldfd(fdns, ytype, grobjx)
    } else if ytype == iohook.YIELD_TYPE_CHAN_RECV {
    } else if ytype == iohook.YIELD_TYPE_CHAN_SEND {
    } else if ytype == iohook.YIELD_TYPE_CONNECT ||
        ytype == iohook.YIELD_TYPE_RECV ||
        ytype == iohook.YIELD_TYPE_READ {
        iopoller.yieldfd(fdns, ytype, grobjx)
    } else {
        panic("unimpl $fdns $ytype")
    }
    /*
    match ytype {
        iohook.YIELD_TYPE_CHAN_RECV {}
        else{}
    }
    */
    grobj.swapback(ytype)
	return 0
}

func onyield_multi(ytype int, cnt int, fds *int64, ytypes *int) int {
    //mlog.info(@FILE, @LINE, "ytype", ytype, "cnt", cnt)
    mcid := 0
    var mcobj *Machine
    machine_get(&mcid, &mcobj)
    grid := 0
    var grobjx voidptr
    fiber_get(&grid, &grobjx)
    grobj := (*Fiber)(grobjx)
    isco := incoro()
    // mlog.info(@FILE, @LINE, fdns, ytype, grid, grobjx, mcid, isco)

	var grmin *iopoller.FiberMin
	grmin = grobjx

    // curl
    if ytype == iohook.YIELD_TYPE_UUPOLL {
        for i := 0; i < cnt; i++ {
            // mlog.info(@FILE, @LINE, cnt, fds[i], ytypes[i])
            iopoller.yieldfd(fds[i], ytypes[i], grmin)
        }
        // x11
    } else if ytype == iohook.YIELD_TYPE_RECVMSG_TIMEOUT {
        for i := 0; i < cnt; i++ {
            // mlog.info(@FILE, @LINE, cnt, fds[i], ytypes[i])
            iopoller.yieldfd(fds[i], ytypes[i], grmin)
        }
    } else {
        panic("unimpl ytype $ytype cnt $cnt")
    }

    grobj.swapback(ytype)
	return 0
}

func onresume(grx voidptr, ytype int, grid int, mcid int) {
    //mlog.info(@FILE, @LINE, "resume $ytype", grx, grid, mcid)
    grobj := (*Fiber)(grx)
    assert (grobj.grid == grid)
    mcobj := grobj.mcobj
    if mcobj.mcid != mcid {
        // mlog.info(@FILE, @LINE)
    }
    assert (mcobj.mcid == mcid)

    grobj.set_state(coresumed)
    // grobj.state = .coresumed
    grobj.wakecnt ++
    mcobj.wake(ytype)
}

/////////////////////
// __thread local
func machine_set(mcid int, mcobj voidptr) int {
	C.co_sched_mcid = mcid
	C.co_sched_mcobj = mcobj
	return 0
}
func fiber_set(mcid int, mcobj voidptr) int {
	C.co_sched_grid = mcid
	C.co_sched_grobj = mcobj
	return 0
}
func getmcid() int { return C.co_sched_mcid }
func machine_get(mcid *int, mcobj *voidptr) int {
	*mcid = C.co_sched_mcid
	*mcobj = C.co_sched_mcobj
	return 0
}
func fiber_get(mcid *int, mcobj *voidptr) int {
	*mcid = C.co_sched_grid
	*mcobj = C.co_sched_grobj
	return 0
}

/////////////////////////
struct Schedule {
    mainmc *Machine// = nil
    osths map[int]*Machine
    mcidno int
    gridno int// = 100
    // amu &futex.Mutex = futex.newMutex()
	amu *futex.Mutex
}

func newSchedule() *Schedule {
	this := &Schedule{}
	this.amu = futex.newMutex()
	return this
}

var (
    schedobj *Schedule// = &Schedule(0)
    ylder2 *rtcom.Yielder// = &rtcom.Yielder(0)
    // 这两个似乎都不好与GC配合
    useshrstk = false // has some problem, disable now
    usemmapstk = false
)

struct Machine {
    mcid int
	//futo &futex.Futex = futex.newFutex()
	futo *futex.Futex
    wakety int // WakeType
    parkty int // WakeType
    mainco voidptr
    wakecnt int // 通知的时候，可能接不到

    taskq []*Fiber // not complete initialed
    // amu &futex.Mutex = futex.newMutex()
	amu *futex.Mutex
    workq []*Fiber
    rungr *Fiber// = nil
    shrstk voidptr // one shrstk per osthread

	// systhread
	systh uint64 // pthread_t
}
func newMachine() *Machine {
	this := &Machine{}
	this.futo = futex.newFutex()
	this.amu = futex.newMutex()
	return this
}

/*
enum Costate {
    coready
    corunning
    coyielded
    coresumed
    codone
}
*/
const (
    coready = 0
    corunning = 1
    coyielded = 2
    coresumed = 3
    codone = 4
)

struct Fiber {
    grid int
    mcid int

    // channel, sudog struct???
    elem voidptr // channel recv/send var addr
    fromgr *Fiber// = 0
    channel voidptr // sending channel
    releasetime int64
    param voidptr
    isselect bool

    // state Costate
    state uint32
    ytype int
    cofn *CoFunc
    stk voidptr// = 0 // bottom
    stksz int
    coctx voidptr
    coctx0 voidptr
    //ctime time.Time
    //yieldtm time.Time
    mcobj *Machine// = nil
    stki *vmm.StackInfo
    //stksav StackSave
	stksav *StackSave
    wakecnt int // 从执行到结束
	mcsi *vmm.StackInfo
}

struct StackSave {
    stkmem voidptr// = nil
    memsz int// = 0
}

struct CoFunc {
    this voidptr
    fnptr2 func(this voidptr, arg voidptr) // TODO compiler
    fnptr func(arg voidptr)
    fnarg voidptr
}
func (this *CoFunc) call() {
    if this.this != nil {
        // this.fnptr2(this.this, this.fnarg)// TODO compiler
		fnptr := this.fnptr2
		fnptr(this.this, this.fnarg)
    }else{
        //this.fnptr(this.fnarg) // TODO compiler
		fnptr := this.fnptr
		fnptr(this.fnarg)
    }
}

/////////////////////////////////
func newFiber(ff *CoFunc) *Fiber {
    id := schedobj.nextgrid()
	this := &Fiber{}
	this.grid = id
	this.cofn = ff
	this.stki = &vmm.StackInfo{}
    //nowt := time.now()
	return this
}
func (thisp *Fiber) swapback(ytype int) {
    this := thisp
    if this.set_state_ifeq(coyielded, corunning) {
        this.ytype = ytype
    }else{
        state := this.get_state()
        //mlog.info(@FILE, @LINE, "not running? some resume me?", state)
        //panic("ooooo")
    }

    if useshrstk { this.savestack() }
    //vmm.set_stackbottom2(this.mcsi)
    coro.transfer(this.coctx, this.coctx0)
}
func (thisp *Fiber) set_state_ifeq(new_state uint32, expect_state uint32) bool {
    //rv := C.atomic_compare_exchange_strong_u32(&thisp.state, &expect_state, new_state)
	rv := atomic.CmpXchg32(&thisp.state, expect_state, new_state)
    return rv
}
func (thisp *Fiber) set_state(state uint32) {
    // C.atomic_store_u32(&thisp.state, state)
	atomic.Store32(&thisp.state, state)
}
func (thisp *Fiber) get_state() uint32 {
    // state := C.atomic_load_u32(&thisp.state)
	state := atomic.Load32(&thisp.state)
    return state
}

func (thisp *Fiber) setstki(stksz int) {
    this := thisp
    this.stksz = stksz
    // not need care handle, gc as null handle in current thread
    this.stki.membase = this.stk // just gc wanted bottom
    this.stki.stksz = this.stksz
    this.stki.stktop = voidptr(usize(this.stk) + usize(this.stksz))
}
func (thisp *Fiber) destroy() {
    this := thisp
    if !useshrstk {
        this.stackguard(false)
        //vmm.freegc(this.stk)
        //vmm.freemp(this.stk, this.stksz)
    }else{
        this.stk = nil
    }
    //nowt := time.now()
    //mlog.info(@FILE, @LINE, "coexit", this.grid, this.mcid, (nowt-this.ctime).str())
	println("coexit", this.grid, this.mcid)
}

/*
corutine stack layout:
top -------------
    |
    |  using
    |
dummy -----------
    |  empty
    |
guard -----------
    |  guard
bottom ----------
*/

func (thisp *Fiber) savestack() {
    this := thisp
    stktop := 0
    stksz := 0
    usesz := 0

    dummy := byte(0) ///// stack pos
    //stktop := voidptr(size_t(this.stk) + size_t(dftstksz)) // this.stki.stktop
    stktop = this.stki.stktop
    stksz = this.stki.stksz
    usesz = int(usize(voidptr(stktop)) - usize(voidptr(&dummy)))
    if false {
		//mlog.info(@FILE, @LINE, this.grid, "top", stktop, "dummy", voidptr(&dummy),
		//    "usesz", usesz, "totsz", stksz)
		println(this.grid, "top", stktop, "dummy", &dummy,
			"usesz", usesz, "totsz", stksz)
    }
    if usesz < 0 {
        C.printf("usesz<0 %d\n", usesz)
        C.abort()
    }
    if usesz > stksz {
        C.printf("usesz>stksz %d>%d\n", usesz,stksz)
        C.abort()
    }
    if this.stksav.memsz < usesz {
        //this.stksav.stkmem = malloc(usesz+1)
		this.stksav.stkmem = mallocuc(usesz+1)
        this.stksav.memsz = usesz
    }
    C.memcpy(this.stksav.stkmem, voidptr(&dummy), usesz)
}
func (thisp *Fiber) stackoverflow_check() {
    this := thisp
    stktop := this.stki.stktop
    stksz := this.stki.stksz
    usesz := 0
    usept := 0
    dummy := byte(0) // stack pos
    usesz = int(usize(voidptr(stktop)) - usize(voidptr(&dummy)))
    usept = usesz * 100 / stksz
    if usept > 75 {
        //mlog.info(@FILE, @LINE, "need more stack", thisp.grid, usept)
    }
}
func overflow_check() {
    grid := 0
    var grobjx *Fiber = nil
    fiber_get(&grid, &grobjx)
    grobj := (*Fiber)(grobjx)
    grobj.stackoverflow_check()
}

// fn C.mprotect() int
// report when overflow, Cannot access memory at address
func (thisp *Fiber) stackguard(on bool) {
    if true {
        return
    }
    addr := thisp.stk
    guardsz := C.sysconf(C._SC_PAGESIZE) // must round to multiple of pagesize
    if on {
        C.memset(addr, 0, guardsz)
        rv := C.mprotect(addr, guardsz, C.PROT_READ)
        assert (rv == 0)
        // test 1
        okaddr := voidptr(usize(addr) + usize(guardsz))
        C.memcpy(okaddr, &rv, sizeof(rv))
        // below should segfault
        // guardaddr := voidptr(size_t(addr) + size_t(1024*4-1))
        // C.memcpy(guardaddr, &rv, sizeof(rv))
        // C.memcpy(addr, &rv, sizeof(rv))
    }else{
        // rv := C.mprotect(addr, guardsz, C.PROT_READ | C.PROT_WRITE|C.PROT_EXEC)
        rv := C.mprotect(addr, guardsz, C.PROT_READ | C.PROT_WRITE)
        assert (rv == 0)
    }
}

/////////////////////////////
func (thisp *Schedule) init_machines() {
    this := thisp
    for i := 1; i <= 3 ; i++ {
		m := newMachine()
        m.mcid = i
        if useshrstk {
            // m.shrstk = vmm.mallocuc(dftstksz)
            //m.shrstk = vmm.mallocmp(dftstksz)
			m.shrstk = mallocuc(dftstksz)
        }
        this.osths[i] = m
        // go coruner_proc(m)
		C.pthread_create(&m.systh, nil, coruner_proc, m)
    }

	m := newMachine()
    m.mcid = -2
    this.mainmc = m
    //go coctrl_proc(m)
	C.pthread_create(&m.systh, nil, coctrl_proc, m)
}

func comainfp(argx voidptr) {
    co := (*Fiber)(argx)
    co.cofn.call()
    // co.state = .codone
    co.set_state(codone)
	//mlog.info(@FILE, @LINE, "cofn done", co.grid, co.mcid)
	println("cofn done", co.grid, co.mcid)
    coro.transfer(co.coctx, co.coctx0)
}

// not complete initialed fibers
// taskq manager
func (thisp *Machine) addnew(gr *Fiber) {
    this := thisp
    gr2 := gr
    gr2.mcid = this.mcid
    this.amu.mlock()
    //this.taskq << gr
    //assert(1==2)
    this.taskq.append(&gr)
    this.amu.munlock()
}
func (thisp *Machine) getnew() *Fiber {
    this := thisp
    var gr *Fiber = nil
    this.amu.mlock()
    if this.taskq.len > 0 {
        gr = this.taskq[0]
        this.taskq.delete(0)
    }
    this.amu.munlock()
    return gr
}
func (thisp *Machine) cntnew() int {
    this := thisp
    len := 0
    this.amu.mlock()
    len = this.taskq.len
    this.amu.munlock()
    return len
}

// workq manager
func (thisp *Machine) append(gr *Fiber) {
    this := thisp
    this.amu.mlock()
    //this.workq << gr
    //assert(1==2)
    this.workq.append(&gr)
    this.amu.munlock()
}
func (thisp *Machine) popnext() *Fiber {
    this := thisp
    var gr *Fiber = nil // = sched__Fiber_new_zero
    idx := -1
    haswakecnt := 0
    this.amu.mlock()
    for i := this.workq.len-1; i >= 0; i-- {
        tr := this.workq[i]
        if tr == nil { C.abort() }
        state := tr.get_state()
        if state == coready || state == coresumed {
            if gr == nil {
                gr = tr
                idx = i
                //break
            }
        }else if tr.wakecnt > 0 {
            haswakecnt ++
        }
    }
    if gr != nil { this.workq.delete(idx) }
    this.amu.munlock()
    if haswakecnt>0 { // so many this case???
        // mlog.info(@FILE, @LINE, "wakecnt>0 but state notok", haswakecnt)
		println("wakecnt>0 but state notok", haswakecnt)
    }
    return gr
}


func (thisp  *Machine) corofy(grp *Fiber) voidptr {
    co := grp
    co.mcobj = thisp
    co.coctx0 = thisp.mainco
    coctx := coro.newctx()
    co.coctx = coctx
    coro.create(coctx, comainfp, co, co.stk, co.stksz)
    return coctx
}
func coruner_proc(arg *Machine) {
    mysi := vmm.get_my_stackbottom()
    machine_set(arg.mcid, arg)
    mymcid := 0
    var mymcobjx *Machine = nil
    machine_get(&mymcid, &mymcobjx)
    //mlog.info(@FILE, @LINE, mymcid, mymcobjx, voidptr(&mymcid))
    mymcobj := (*Machine)(mymcobjx)
    myfuto := mymcobj.futo
    mainco := coro.newctx()
    mymcobj.mainco = mainco
    //mymcobj.shrstk = vmm.mallocuc(dftstksz)
    coro.create(mainco, nil, nil, nil, 0)

    for {
        needpark := true
        var grobj *Fiber = nil
        grobj = mymcobj.popnext()
        if grobj != nil {
            needpark = false
        }else{
            if mymcobj.cntnew() > 0 {
                needpark = false
            }
        }

        if needpark {
            mymcobj.wakecnt = 0
            myfuto.park()
            // mlog.info(@FILE, @LINE, mymcid, "waked", mymcobj.wakecnt, mymcobj.taskq.len)
			println(mymcid, "waked", mymcobj.wakecnt, mymcobj.taskq.len)
        }

        for {
            gr := mymcobj.getnew()
            if gr == nil { break }
            // mlog.info(@FILE, @LINE, "left", mymcobj.taskq.len)
            if useshrstk {
                gr.stk = mymcobj.shrstk
                gr.setstki(dftstksz)
            }
            coctx := mymcobj.corofy(gr)
            gr.mcsi = &mysi
            mymcobj.append(gr)
        }
        if grobj == nil {
            grobj = mymcobj.popnext()
        }
        if grobj != nil {
            mymcobj.rungr = grobj
            grobj.set_state(corunning)
            grobj.wakecnt = 0
            //C.printf("hhhhh %d\n", 540)
            if useshrstk {
                C.memset(mymcobj.shrstk, 0, dftstksz-5000)
                //grobj.stk = mymcobj.shrstk
                //C.printf("copystkback (%p, %p, %d)\n",
                  //       grobj.stk, grobj.stksav.stkmem, grobj.stksav.memsz)
                if grobj.stksav.stkmem != nil {
                C.memcpy(mymcobj.shrstk, grobj.stksav.stkmem, grobj.stksav.memsz)
                }
            }
            fiber_set(grobj.grid, grobj)
            //vmm.alloc_lock()
            vmm.set_stackbottom(grobj.stki) // 没有这个设置根本就不回收 coroutine运行时分配的内存
            coro.transfer(mainco, grobj.coctx)
            vmm.set_stackbottom2(mysi) //mysi // 没有这个设置根本就不回收 coroutine运行时分配的内存
            fiber_set(-1, nil)
            if useshrstk {
                //grobj.stk = nil
            }
            mymcobj.rungr = nil
            // mlog.info(@FILE, @LINE, "swap back", grobj.grid,
               //       grobj.state.str(), grobj.ytype, grobj.wakecnt)
            if grobj.get_state() == codone {
                // destroy
                grobj.destroy()
            }else{
                mymcobj.append(grobj)
            }
        }
    }

    // mlog.info(@FILE, @LINE, mymcid, "exit")
	println(mymcid, "exit")
}

// random one
func (thisp *Schedule) pickmc() *Machine {
    //mcidx := rand.intn(thisp.osths.len)
	// thisp.osths.len
	cnt := 3
	mcidx := C.rand() % cnt
    assert (mcidx >= 0)
    // mlog.info(@FILE, @LINE, mcidx)
    cnter := -1
    var mcobj *Machine = nil
    mcid := 0
    for key, obj in thisp.osths {
        cnter++
        if cnter == mcidx {
            mcobj = obj
            break
        }
    }
    return mcobj
}

func coctrl_proc(arg *Machine) {
    mymcid := arg.mcid
    mymcobjx := voidptr(arg)
    mymcobj := (*Machine)(arg)
    myfuto := mymcobj.futo
    scheder := schedobj

    //mlog.info(@FILE, @LINE, mymcid, mymcobjx)
	println(mymcid, mymcobjx)
    for {
        myfuto.park()
        //mlog.info(@FILE, @LINE, mymcid, "waked", mymcobj.wakety.str())
		println(mymcid, "waked", mymcobj.wakety)
        if mymcobj.cntnew() == 0 {
            continue
        }
        for {
            curgr := mymcobj.getnew()
            if curgr == nil { break }
			println(curgr)
			println(curgr.mcid)

            mcobj := scheder.pickmc()
            mcid := mcobj.mcid
            //mlog.info(@FILE, @LINE, "gr#${curgr.grid} move to mc#$mcid")
			println("gr", curgr.grid, "move to mc#", mcid)
            mcobj.addnew(curgr)
            mcobj.wake(newtask)
        }
    }

    //mlog.info(@FILE, @LINE, mymcid, "exit")
	println(mymcid, "exit")
}

func (thisp *Schedule) nextgrid() int {
    this := thisp
    no := 0
    this.amu.mlock()
    no = this.gridno
    this.gridno = this.gridno + 1
    this.amu.munlock()
    // mlog.info(@FILE, @LINE, "nextid", no)
    return no
}

const dftstksz = 128*1024

func post(f voidptr, arg voidptr) {
    post2(nil, f, arg)
}
func post2(this voidptr, f voidptr, arg voidptr) {
    post3(this, f, arg, dftstksz)
}

func post3(this voidptr, f voidptr, arg voidptr, stksz int) {
    sch := schedobj
    //stksz2 := if stksz <= 0 { dftstksz } else { stksz }
	stksz2 := ifelse(stksz <= 0, dftstksz, stksz)
    //ff := &CoFunc{this, f, f, arg} // TODO gxcallable!!!
    ff := &CoFunc{}
    clos := gxcallable_new(f, this)
    ff.this = this
    ff.fnptr2 = clos
    ff.fnptr = clos
    ff.fnarg = arg

    gr := newFiber(ff)
    if !useshrstk {
        //gr.stk = vmm.mallocuc(stksz2)
        //gr.stk = vmm.mallocmp(stksz2)
		gr.stk = mallocuc(stksz2)
        gr.setstki(stksz2)
        gr.stackguard(true)
    }
    sch.mainmc.addnew(gr)
    sch.mainmc.wake(newtask)
}

//enum WakeType {
const (
    unknown = 0
    newtask
    iocanread
    iocanwrite
    ioconnected
    ioclosed
    timerto
)

func (thisp *Machine) wake(ty int) {
    this := thisp
    this.wakety = ty
    this.wakecnt ++
    thisp.futo.wake()
}

// Features
// [x] stack guard
// [ ] shared stack, less memory usage
// [ ] dynamic increase stack size, sigaltstack
// [ ] mmap

/// 共享栈协程保存与恢复原理，https://blog.csdn.net/liushengxi_root/article/details/85114692
/// 共享栈的坑，栈变量地址重复问题，https://masutangu.com/2018/12/10/libco-share-stack/#%E5%85%B1%E4%BA%AB%E6%A0%88%E6%A8%A1%E5%BC%8F%E9%9A%90%E8%97%8F%E7%9A%84%E5%9D%91

// 类似项目
// https://github.com/idealvin/co
// https://github.com/owt5008137/libcopp
// https://github.com/SasLuca/libco/tree/master/source

