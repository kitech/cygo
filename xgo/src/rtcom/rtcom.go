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
    // incoro fn() int
    // getcoro fn() voidptr
    // yield fn(i64, int) int
    // yield_multi fn(int, int, &i64, &int) int
	incoro voidptr
	getcoro voidptr
	yield voidptr
	yield_multi voidptr
}
struct Resumer {
    // resume_one fn(grobj voidptr, ytype int, grid int, mcid int)
	resume_one voidptr
}
struct Allocator {
    // mallocfn fn(size_t) voidptr
    // callocfn fn(size_t, size_t) voidptr
    // reallocfn fn(voidptr, size_t) voidptr
    // freefn fn(voidptr)

	mallocfn voidptr
	callocfn voidptr
	reallocfn voidptr
	freefn voidptr
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
    return rv
}
// fn C.rtcom_resumer_get() voidptr
func resumer() *Resumer {
    rv := C.rtcom_resumer_get()
    return rv
}
// fn C.rtcom_allocator_get() voidptr
func allocer() *Allocator {
    rv := C.rtcom_allocator_get()
    return rv
}

