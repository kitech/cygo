package coro

// modernc.org/cc not support
// #include "/home/me/oss/src/cxrt/xgo/src/coro/coro.h"

/*
#cgo CFLAGS: -DCORO_ASM

extern void coro_create();
// extern void coro_transfer(void*, void*); // signature not match error
extern void coro_transfer();
extern void libcoro_destroy(void*);
*/
import "C"

func Keepme() {}

struct Context {
	// void **sp; /* must be at offset 0 */
	sp voidptr
}

func New(f voidptr, arg voidptr, sptr voidptr, ssze usize) *Context {
	ctx := &Context{}
	// C.coro_create(voidptr(ctxp), f, arg, sptr, ssze) // TODO
	C.coro_create((voidptr)(ctx), f, arg, sptr, ssze)
	return ctx
}

func NewMain() *Context {
	ctx := &Context{}
	// C.coro_create(voidptr(ctxp), f, arg, sptr, ssze) // TODO
	C.coro_create((voidptr)(ctx), nil, nil, nil, 0)
	return ctx
}

func (ctx *Context) Swapto(ctx2 *Context) {
	C.coro_transfer((voidptr)(ctx), (voidptr)(ctx2))
}

func (ctx *Context) Destroy() {
	C.libcoro_destroy(ctx)
}

/////////////////
func newctx() voidptr {
    cot := mallocgc(64)
    return cot
}

func create(cot voidptr, fp voidptr, arg voidptr, sptr voidptr, ssze int) {
    C.coro_create(cot, fp, arg, sptr, usize(ssze))
}
func create2(fp voidptr, arg voidptr, sptr voidptr, ssze int) voidptr {
    cot := mallocgc(64)
    C.coro_create(cot, fp, arg, sptr, usize(ssze))
    return cot
}

// fn C.coro_transfer() int
// [inline]

func transfer(prev voidptr, next voidptr) {
    // C.libcoro_transfer(prev, next)
    C.coro_transfer(prev, next)
}

func destroy(cot voidptr) {
    C.libcoro_destroy(cot)
}

