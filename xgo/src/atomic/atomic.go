package atomic

/*
#cgo LDFLAGS: -latomic

#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdatomic.h>

extern bool __atomic_compare_exchange_4(void* x, void* expected, int y, bool weak, int mo, int mo2);
extern bool __atomic_compare_exchange_8(void* x, void* expected, int64_t y, bool weak, int mo, int mo2);

extern unsigned long long __atomic_load_8(void* x, int mo);
extern void __atomic_store_8(void* x, unsigned long long y, int mo);
extern unsigned long long __atomic_fetch_add_8(void* x, unsigned long long y, int mo);
extern unsigned long long __atomic_fetch_sub_8(void* x, unsigned long long y, int mo);

extern unsigned int __atomic_load_4(void* x, int mo);
extern void __atomic_store_4(void* x, unsigned int y, int mo);
extern unsigned int __atomic_fetch_add_4(void* x, unsigned int y, int mo);
extern unsigned int __atomic_fetch_sub_4(void* x, unsigned int y, int mo);

*/
import "C"

func Keepme() {}

func hhhe() {
	//C.atomic_compare_exchange_strong()
	//C.atomic_fetch_add()
}

// CmpXchg32 cas of int32
func CmpXchg32(v *int, oldval int, newval int) bool {
	rv := C.__atomic_compare_exchange_4(v, &oldval, newval, 0, C.__ATOMIC_SEQ_CST, C.__ATOMIC_SEQ_CST)
	return rv
}

// CmpXchg64 cas of int64
func CmpXchg64(v *int64, oldval int64, newval int64) bool {
	rv := C.__atomic_compare_exchange_8(v, &oldval, newval, 0, C.__ATOMIC_SEQ_CST, C.__ATOMIC_SEQ_CST)
	return rv
}

func Load64(v *int64) int64 {
	return C.__atomic_load_8(v, C.__ATOMIC_SEQ_CST)
}
func Store64(v *int64, newval int64) {
	C.__atomic_store_8(v, newval, C.__ATOMIC_SEQ_CST)
}
func FetchAdd64(v *int64, delta int64) int64 {
	rv := C.__atomic_fetch_add_8(v, delta, C.__ATOMIC_SEQ_CST)
	return rv
}
func FetchSub64(v *int64, delta int64) int64 {
	rv := C.__atomic_fetch_sub_8(v, delta, C.__ATOMIC_SEQ_CST)
	return rv
}

func Load32(v *int) int {
	return C.__atomic_load_4(v, C.__ATOMIC_SEQ_CST)
}
func Store32(v *int, newval int) {
	C.__atomic_store_4(v, newval, C.__ATOMIC_SEQ_CST)
}
func FetchAdd32(v *int, delta int) int {
	rv := C.__atomic_fetch_add_4(v, delta, C.__ATOMIC_SEQ_CST)
	return rv
}
func FetchSub32(v *int, delta int) int {
	rv := C.__atomic_fetch_sub_4(v, delta, C.__ATOMIC_SEQ_CST)
	return rv
}
