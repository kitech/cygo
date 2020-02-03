package xtime

/*
#include <time.h>
#include <sys/time.h>
#include <unistd.h>
*/
import "C"

// import "xgo/xtime"

const (
	SEC = 1
	MS  = 1000
	US  = 1000 * 1000
	NS  = 1000 * 1000 * 1000
)

func Sleep(sec int) int {
	rv := C.sleep(sec)
	return rv
}

func Sleepms(msec int) int {
	var ts0 = &C.struct_timespec{}
	var ts1 = &C.struct_timespec{}

	ts0.tv_sec = msec / MS
	ts0.tv_nsec = (msec % MS) * US
	rv := C.nanosleep(ts0, ts1)
	return rv
}

type Duration int64

type Time struct {
	unix int64 // usec
	zone int
}

func Now() *Time {
	var tv = &C.struct_timeval{}
	rv := C.gettimeofday(tv, 0)
	uts := C.time(0)
	t := &Time{}
	sec := tv.tv_sec
	usec := tv.tv_usec
	newts := sec*US + usec
	t.unix = newts
	t.unix = tv.tv_sec*US + tv.tv_usec
	t.unix = uts * US
	t.zone = zone()
	return t
}

func zone() int {
	rv := C.timezone
	return rv
}

func (t *Time) Unix() int64 {
	return t.unix
}

func (t *Time) Iszero() bool {
	if t != nil && t.unix > 0 {
		return false
	}
	return true
}

func (t *Time) Format(format string) string {
	return ""
}

// yyyy-mm-dd hh:MM:ss.iii
func (t *Time) Format1(withms bool) string {
	return ""
}

func ParseIso(s string) *Time {
	return nil
}

//
type timeritem struct {
	// btime   *xtime.Time
	btime   int64
	timeout Duration
	f       voidptr
	// f2      func() // TODO compiler
}
type timerman struct {
	// itemmu *xsync.Mutex
	itemmu C.pthread_mutex_t
	items  []*timeritem
	tmradd chan *timeritem
}

func (trm *timerman) lock() {
	C.pthread_mutex_lock(&trm.itemmu)
}
func (trm *timerman) unlock() {
	C.pthread_mutex_unlock(&trm.itemmu)
}

func newtimeritem(timeout Duration, f voidptr) *timeritem {
	item := &timeritem{}
	item.timeout = timeout
	item.f = f
	item.btime = C.time(0)
	return item
}

var tmrman *timerman

func init() {
	// init_timerman_proc()
}
func init_timerman_proc() {
	tmrman = &timerman{}
	// tmrman.itemmu = &xsync.Mutex{}
	tmrman.tmradd = make(chan *timeritem, 128)
	go timerman_dotask_proc()
	go timerman_recv_proc()
}

func timerman_dotask_proc() {
	for i := 0; ; i++ {
		Sleep(1)
		timerman_dotask_proc1()
	}
}
func timerman_dotask_proc1() {
	var readys []*timeritem
	var lefts []*timeritem
	nowts := C.time(0)
	tmrman.lock()
	for _, item := range tmrman.items {
		if (item.btime + int64(item.timeout)) >= nowts {
			readys = append(readys, item)
		} else {
			lefts = append(lefts, item)
		}
	}
	if readys.len() > 0 {
		tmrman.items = lefts
	}
	tmrman.unlock()
	for _, item := range readys {
		var cbfn func()
		cbfn = item.f
		// cbfn()
		cbfn()
	}
}
func timerman_recv_proc() {
	select {
	case item := <-tmrman.tmradd:
		tmrman.lock()
		tmrman.items = append(tmrman.items, item)
		tmrman.unlock()
	}
}

func AfterFunc(timeout Duration, f voidptr) {

}

func After(timeout Duration) <-chan int {
	return nil
}

func Keep() {}
