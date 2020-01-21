package time

/*
#include <time.h>
#include <sys/time.h>
*/
import "C"
import "unsafe"

func dummying() {
	var p unsafe.Pointer
}

type Duration int64

type Location struct {
}

type Time struct {
	wall uint64
	ext  int64
	loc  *Location
}

const (
	hasMonotonic = 1 << 63
	// maxWall      = wallToInternal + (1<<33 - 1) // year 2157
	// minWall      = wallToInternal               // year 1885
	nsecMask  = 1<<30 - 1
	nsecShift = 30
)

func Now() *Time {
	return unix()
}

func unix() *Time {
	tp := &C.struct_timeval{}
	C.gettimeofday(tp, nil)
	println(tp.tv_sec, tp.tv_usec)
	return Unix(tp.tv_sec, tp.tv_usec)
}

func Unix(sec int64, nsec int64) *Time {
	t := &Time{}
	t.wall = sec
	t.wall = t.wall * 1000000000
	var us uint64 = nsec * 1000
	t.wall = t.wall | us
	return t
}

func (t *Time) Unix() int64 { return t.wall / 1000000000 }

func (t *Time) Sub(t2 *Time) int64 {
	return t.wall - t2.wall
}

func (t *Time) String() string {

}

func Sleep(d Duration) {

}

func after_fiber(c chan *Time) { c <- nil }
func After(d Duration) chan *Time {
	c := make(chan *Time)
	go after_fiber(c)
	return c
}
