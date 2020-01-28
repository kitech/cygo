package xtime

/*
#include <time.h>
#include <sys/time.h>
#include <unistd.h>
*/
import "C"

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
	// t.unix = tv.tv_sec*US + tv.tv_usec // TODO
	t.unix = uts * US
	t.zone = zone()
	return t
}

func zone() int {
	rv := C.timezone
	return rv
}

func (t *Time) Format() {

}

func ParseIso(s string) *Time {
	return nil
}

func Keep() {}
