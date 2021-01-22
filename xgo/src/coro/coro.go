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

