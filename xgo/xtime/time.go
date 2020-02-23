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

func Unix() int64 {
	uts := C.time(0)
	return int64(uts)
}

type Duration int64

type Time struct {
	unix int64 // usec, not the same with Go
	zone int
}

func Now() *Time {
	var tv = &C.struct_timeval{}
	rv := C.gettimeofday(tv, 0)
	t := &Time{}
	sec := tv.tv_sec
	usec := tv.tv_usec
	newts := sec*US + usec
	t.unix = newts
	t.unix = tv.tv_sec*US + tv.tv_usec
	if tmzone == -1 {
		tmzone = zoneno()
	}
	t.zone = tmzone
	return t
}

var tmzone int = -1

func zoneno() int {
	rv := C.timezone
	return rv
}

func (t *Time) Unix() int64 {
	return t.unix
}

func (t *Time) Iszero() bool {
	// sure := ifelse(t != nil && t.unix > 0, false, true) // TODO compiler
	if t != nil && t.unix > 0 {
		return false
	}
	return true
}

func (t *Time) Since(t2 *Time) Duration {
	duri := t.unix - t2.unix
	// duro := Duration(duri) // TODO compiler
	return duri
}

func Since(t2 *Time) Duration {
	t := Now()
	duri := t.unix - t2.unix
	// duro := Duration(duri) // TODO compiler
	return duri
}

func (dur Duration) String() string {
	var dur2 int64 = int64(dur)
	daydur := int64(3600 * 24 * US)
	hourdur := int64(3600 * US)
	mindur := int64(60 * US)
	secdur := int64(US)

	days := dur2 / daydur
	hours := (dur2 % daydur) / hourdur
	mins := (dur2 % hourdur) / mindur
	secs := (dur2 % mindur) / secdur
	msecs := (dur2 % secdur) / MS

	// TODO reduce allocation
	var str string
	seenpfx := false
	if days > 0 {
		str += days.repr() + "d"
		seenpfx = true
	}
	if hours > 0 || seenpfx {
		str += hours.repr() + "h"
		seenpfx = true
	}
	if mins > 0 || seenpfx {
		str += mins.repr() + "m"
		seenpfx = true
	}
	if secs > 0 || seenpfx {
		str += secs.repr() + "s"
		seenpfx = true
	}
	if msecs > 0 {
		str += msecs.repr() + "ms"
	}
	return str
}

func (t *Time) Format(format string) string {
	return ""
}

// yyyy-mm-dd hh:MM:ss.iii
func (t *Time) Format1(withms bool) string {
	return ""
}

func (t *Time) Tostr1() string {
	var ep usize
	ep = t.unix / US

	// var tmo *C.struct_tm // TODO compiler
	tmo := C.localtime(&ep)
	buf := malloc3(32)
	C.sprintf(buf, "%04d-%02d-%02d %02d:%02d:%02d".ptr,
		tmo.tm_year+1900, tmo.tm_mon, tmo.tm_mday, tmo.tm_hour, tmo.tm_min, tmo.tm_sec)
	return gostring(buf)
}

// with msec
func (t *Time) Tostr2() string {
	var ep int64
	ep = t.unix / US
	msec := (t.unix % US) / MS

	// var tmo *C.struct_tm // TODO compiler
	tmo := C.localtime(&ep)
	buf := malloc3(32)
	C.sprintf(buf, "%04d-%02d-%02d %02d:%02d:%02d.%03d".ptr,
		tmo.tm_year+1900, tmo.tm_mon, tmo.tm_mday,
		tmo.tm_hour, tmo.tm_min, tmo.tm_sec, msec)
	return gostring(buf)
}

func (t *Time) Toiso() string {
	var ep usize
	ep = t.unix / US

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
	// tmrman.tmradd = make(chan *timeritem, 128) // TODO compiler
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
	if readys.len > 0 {
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
