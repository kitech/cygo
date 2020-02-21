package builtin

/*
// #include <gc.h>

extern void* crn_gc_malloc(size_t);
extern void* crn_gc_realloc(void*, size_t);
extern void crn_gc_free(void*);
*/
import "C"

//export cxmalloc
func bdwgc_malloc(n usize) voidptr {
	ptr := C.crn_gc_malloc(n)
	return ptr
}

//export cxrealloc
func bdwgc_realloc(ptr voidptr, size usize) voidptr {
	ptr2 := C.crn_gc_realloc(ptr, size)
	return ptr2
}

//export cxfree
func bdwgc_free(ptr voidptr) {
	C.crn_gc_free(ptr)
}

//export cxcalloc
func bdwgc_calloc(blocks usize, size usize) voidptr {
	ptr := C.crn_gc_malloc(blocks * size)
	return ptr
}

//export cxstrdup
func bdwgc_strdup(str byteptr) byteptr {
	ds := bdwgc_malloc(C.strlen(str) + 1)
	C.strcpy(ds, str)
	return ds
}

//export cxstrndup
func bdwgc_strndup(str byteptr, n int) byteptr {
	ds := bdwgc_malloc(n + 1)
	C.strncpy(ds, str, n)
	return ds
}

func bdwgc_memdup(ptr voidptr, sz int) voidptr {
	dp := bdwgc_malloc(sz)
	memcpy3(dp, ptr, sz)
	return dp
}
