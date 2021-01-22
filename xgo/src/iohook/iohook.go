package iohook

/*
#cgo CFLAGS: -D_GNU_SOURCE

extern void iohook_initHook();
extern void iohook_initHook2();
*/
import "C"

func Keepme() {}

struct Yielder {
    // int (*incoro)();
    // void* (*getcoro)();
    // int (*yield)(long, int);
    // int (*yield_multi)(int, int, long*, int*);

	incoro voidptr
	getcoro voidptr
	yield voidptr
	yield_multi voidptr
}

func pre_main_init000(y voidptr, allocer voidptr) {
    // C.iohook_initHook(y, sizeof(rtcom.Yielder), allocer)
	C.iohook_initHook(y, 8*4, allocer)
}

func pre_main_init(incoro voidptr, getcoro voidptr, onyield voidptr, onyield_multi voidptr,
	mallocfn voidptr, callocfn voidptr, reallocfn voidptr, freefn voidptr) {
	C.iohook_initHook2(incoro , getcoro , onyield , onyield_multi ,
		mallocfn , callocfn , reallocfn , freefn )
}

func post_main_deinit() {

}
