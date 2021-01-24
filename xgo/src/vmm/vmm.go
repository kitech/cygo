package vmm

/*
#include <gc/gc.h>
*/
import "C"

struct StackInfo {
	membase voidptr
    regbase voidptr

    // extra
    handle voidptr
    stksz int
    stktop voidptr // membase + stksz
}

func get_my_stackbottom() *StackInfo {
    this := &StackInfo {}
    // alloc_lock() // hang!!!
    this.handle = C.GC_get_my_stackbottom(this)
    // alloc_unlock()
    return this
}
func set_stackbottom(si *StackInfo) {
    alloc_lock()
    C.GC_set_stackbottom(si.handle, &si)
    alloc_unlock()
}
func set_stackbottom2(si *StackInfo) {
    //alloc_lock()
    C.GC_set_stackbottom(si.handle, si)
    //alloc_unlock()
}

func alloc_lock() { C.GC_alloc_lock() }
func alloc_unlock() { C.GC_alloc_unlock() }

