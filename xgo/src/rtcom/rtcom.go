package rtcom

/*
#cgo LDFLAGS: -lpthread

#include <stdio.h>

extern int rtcom_pre_gc_init();
extern int rtcom_pre_gc_init2();
extern void* rtcom_yielder_get();
extern void* rtcom_resumer_get();
extern void* rtcom_allocator_get();

void temp_print_rtcom(int which) {
switch (which) {
case 0:
printf("rtcom pregc init ...\n");
break;
case 1:
printf("rtcom pregc init done\n");
break;
default:
printf("rtcom wtt %d\n", which);
break;
}

}

*/
import "C"

func Keepme() {}

struct Yielder {
	incoro func() int
	getcoro func() voidptr
	yield func(int64, int) int
	yield_multi func(int, int, *int64, *int) int
}
struct Resumer {
    resume_one func(grobj voidptr, ytype int, grid int, mcid int)
}
struct Allocator {
    mallocfn func(usize) voidptr
    callocfn func(usize, usize) voidptr
    reallocfn func(voidptr, usize) voidptr
    freefn func(voidptr)
}

// fn C.rtcom_pre_gc_init() int

// 被sched调用初始化
func Pre_gc_init000(yielderx *Yielder, resumerx *Resumer, allocerx *Allocator) {
    C.printf("rtcom pregc init\n")
    C.rtcom_pre_gc_init(yielderx, 8*4, // sizeof(Yielder),
		resumerx, 8*1, // sizeof(Resumer),
		allocerx, 8*4) //sizeof(Allocator))
}
func pre_gc_init(incoro voidptr, getcoro voidptr, onyield voidptr, onyield_multi voidptr,
	resumeone voidptr,
	mallocfn voidptr, callocfn voidptr, reallocfn voidptr, freefn voidptr) {
    // C.printf("rtcom pregc init\n")
	C.temp_print_rtcom(0)
	C.rtcom_pre_gc_init2(incoro, getcoro, onyield, onyield_multi,
		resumeone, mallocfn, callocfn , reallocfn, freefn)
	C.temp_print_rtcom(1)
}

func pre_main_init() {

}

// return persistent pointer, can save, not need copy
// fn C.rtcom_yielder_get() voidptr
func yielder() *Yielder {
    rv := C.rtcom_yielder_get()
    return (*Resumer)(rv)
}
// fn C.rtcom_resumer_get() voidptr
func resumer() *Resumer {
    rv := C.rtcom_resumer_get()
    return (*Resumer)(rv)
}
// fn C.rtcom_allocator_get() voidptr
func allocer() *Allocator {
    rv := C.rtcom_allocator_get()
    return (*Allocator)(rv)
}

