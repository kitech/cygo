package xgc

/*
#cgo LDFLAGS: -lgc
#cgo CFLAGS: -DGC_MALLOC

#include <gc/gc.h>
*/
import "C"

var debug = false
var debug_replace = true

const (
	_GC_EXTRAS = "unsupported" // C.GC_EXTRAS
	__FILE__   = C.__FILE__
	__LINE__   = C.__LINE__
	// __FUNC__   = C.__FUNC__ // not work
	// __func__ = C.__func__ // not work
)

func Init() {
	C.GC_init()
}
func Inited() bool {
	rv := C.GC_is_init_called()
	return rv == 1
}
func Deinit() {
	C.GC_deinit()
}

func Malloc(n int) voidptr {
	if debug {
		// return C.GC_debug_malloc(n, C.GC_EXTRAS)
		// if GC_ADD_CALLER
		// return C.GC_debug_malloc(n, C.GC_RETURN_ADDR, C.__FILE__, C.__LINE__)
		return C.GC_debug_malloc(n, C.__FILE__, C.__LINE__)
	} else {
		return C.GC_malloc(n)
	}
}
func Realloc(ptr voidptr, n int) voidptr {
	if debug {
		return C.GC_debug_realloc(ptr, n, C.__FILE__, C.__LINE__)
	} else {
		return C.GC_realloc(ptr, n)
	}
}
func Free(ptr voidptr) {
	if debug {
		C.GC_debug_free(ptr)
	} else {
		C.GC_free(ptr)
	}
}
func MallocUncollectable(n int) voidptr { return C.GC_malloc_uncollectable(n) }
func Collect()                          { C.GC_gcollect() }
func Disabled() bool                    { return 1 == C.GC_is_disabled() }
func Enable()                           { C.GC_enable() }
func Disable()                          { C.GC_disable() }
func EnableIncremental()                { C.GC_enable_incremental() }
func IsIncremental() bool               { return 1 == C.GC_is_incremental_mode() }
func SetFinalizer(ptr voidptr, fnptr voidptr, ud voidptr) {
	var oldfn voidptr
	var oldud voidptr
	if debug {
		C.GC_debug_register_finalizer(ptr, fnptr, ud, &oldfn, &oldud)
	} else {
		C.GC_register_finalizer(ptr, fnptr, ud, &oldfn, &oldud)
	}
}

type StackBase struct {
	Ptr voidptr
	pad voidptr
}

func GetMyStackbottom() *StackBase {
	csb := &C.struct_GC_stack_base{} // TODO compiler
	// var csb = &C.struct_GC_stack_base{}
	handle := C.GC_get_my_stackbottom(csb)
	sb := &StackBase{}
	sb.Ptr = csb.mem_base
	// return sb
	return nil
}
func SetMyBottom(gchandle voidptr, sb *StackBase) {
	csb := &C.struct_GC_stack_base{}
	csb.mem_base = sb.Ptr
	C.GC_set_stackbottom(gchandle, csb)
}

func AllowRegisterThreads() {
	C.GC_allow_register_threads()
}
func ThreadIsRegistered() bool {
	return 1 == C.GC_thread_is_registered()
}
func UnregisterMyThread() {
	C.GC_unregister_my_thread()
}

func RegisterMyThread(sb *StackBase) {
	C.GC_register_my_thread(sb)
}

func CallWithAllocLock(fnptr voidptr, cbval voidptr) {
	C.GC_call_with_alloc_lock(fnptr, cbval)
}

func SetOnCollectionEvent(fnptr voidptr) {
	C.GC_set_on_collection_event(fnptr)
}
func SetOnThreadEvent(fnptr voidptr) {
	C.GC_set_on_thread_event(fnptr)
}
func SetFreeSpaceDivisor(val int) {
	C.GC_set_free_space_divisor(val)
}
func GetFreeSpaceDivisor() int {
	return C.GC_get_free_space_divisor()
}

// tools
func IsHeapPtr(ptr voidptr) bool {
	return 1 == C.GC_is_heap_ptr(ptr)
}

// stats
func Version() uint        { return C.GC_get_version() }
func VersionStr() string   { return "" }
func GetGCNo() int         { return C.GC_get_gc_no() }
func GetFreeBytes() int    { return C.GC_get_free_bytes() }
func GetHeapSize() int     { return C.GC_get_heap_size() }
func GetBytesSinceGC() int { return C.GC_get_bytes_since_gc() }
func GetMemoryUse() int    { return C.GC_get_memory_use() }
func GetNonGCBytes() int   { return C.GC_get_non_gc_bytes() }
func GetTotalBytes() int   { return C.GC_get_total_bytes() }

func Keep() {}
