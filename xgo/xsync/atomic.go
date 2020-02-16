package xsync

/*
#include <stdint.h>
#include <stdbool.h>
#include <stdatomic.h>

int mylang_sync_atomic_addint(int* v, int delta) {
    // extern int __atomic_fetch_add(int*, int, int);
    // return __atomic_fetch_add(v, delta, memory_order_seq_cst);
    return atomic_fetch_add(v, delta);
}
uint32_t mylang_sync_atomic_addu32(uint32_t* v, uint32_t delta) {
    // extern __atomic_fetch_add 表示符号不存在
    // extern uint32_t __atomic_fetch_add(uint32_t*, uint32_t, int);
    return atomic_fetch_add(v, delta);
}
int32_t mylang_sync_atomic_addi32(int32_t* v, int32_t delta) {
    return atomic_fetch_add(v, delta);
}
uint64_t mylang_sync_atomic_addu64(uint64_t* v, uint64_t delta) {
    return atomic_fetch_add(v, delta);
}
int64_t mylang_sync_atomic_addi64(int64_t* v, int64_t delta) {
    return atomic_fetch_add(v, delta);
}
usize mylang_sync_atomic_addusize(usize* v, usize delta) {
    return atomic_fetch_add(v, delta);
}

bool mylang_sync_atomic_notbool(bool* v) {
    return atomic_compare_exchange_strong(v, v, !*v);
}

bool mylang_sync_atomic_casbool(bool* v, bool oldval, bool newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool mylang_sync_atomic_casint(int* v, int oldval, int newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool mylang_sync_atomic_casu32(uint32_t* v, uint32_t oldval, uint32_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool mylang_sync_atomic_casu64(uint64_t* v, uint64_t oldval, uint64_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool mylang_sync_atomic_casusize(usize* v, usize oldval, usize newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool mylang_sync_atomic_casuptr(uintptr_t* v, uintptr_t oldval, uintptr_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool mylang_sync_atomic_casptr(void** v, void* oldval, void* newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool mylang_sync_atomic_casi32(int32_t* v, int32_t oldval, int32_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool mylang_sync_atomic_casi64(int64_t* v, int64_t oldval, int64_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}

int mylang_sync_atomic_swapint(int* v0, int newval) {
    return atomic_exchange(v0, newval);
}
uint32_t mylang_sync_atomic_swapu32(uint32_t* v0, uint32_t newval) {
    return atomic_exchange(v0, newval);
}
uint64_t mylang_sync_atomic_swapu64(uint64_t* v0, uint64_t newval) {
    return atomic_exchange(v0, newval);
}
usize mylang_sync_atomic_swapusize(usize* v0, usize newval) {
    return atomic_exchange(v0, newval);
}
uintptr_t mylang_sync_atomic_swapuptr(uintptr_t* v0, uintptr_t newval) {
    return atomic_exchange(v0, newval);
}
void* mylang_sync_atomic_swapptr(void** v0, void* newval) {
    return atomic_exchange(v0, newval);
}
int32_t mylang_sync_atomic_swapi32(int32_t* v0, int32_t newval) {
    return atomic_exchange(v0, newval);
}
int64_t mylang_sync_atomic_swapi64(int64_t* v0, int64_t newval) {
    return atomic_exchange(v0, newval);
}

void mylang_sync_atomic_setbool(bool* v, bool val) {
    atomic_store(v, val);
}
void mylang_sync_atomic_setint(int* v, int val) {
    atomic_store(v, val);
}
void mylang_sync_atomic_setu32(uint32_t* v, uint32_t val) {
    atomic_store(v, val);
}
void mylang_sync_atomic_setu64(uint64_t* v, uint64_t val) {
    atomic_store(v, val);
}
void mylang_sync_atomic_setusize(usize* v, usize val) {
    atomic_store(v, val);
}
void mylang_sync_atomic_setuptr(uintptr_t* v, uintptr_t val) {
    atomic_store(v, val);
}
void mylang_sync_atomic_setptr(void** v, void* val) {
    atomic_store(v, val);
}
void mylang_sync_atomic_seti32(int32_t* v, int32_t val) {
    atomic_store(v, val);
}
void mylang_sync_atomic_seti64(int64_t* v, int64_t val) {
    atomic_store(v, val);
}

bool mylang_sync_atomic_getbool(bool* v) {
    return atomic_load(v);
}
int mylang_sync_atomic_getint(int* v) {
    return atomic_load(v);
}
uint32_t mylang_sync_atomic_getu32(uint32_t* v) {
    return atomic_load(v);
}
uint64_t mylang_sync_atomic_getu64(uint64_t* v) {
    return atomic_load(v);
}
usize mylang_sync_atomic_getusize(usize* v) {
    return atomic_load(v);
}
uintptr_t mylang_sync_atomic_getuptr(uintptr_t* v) {
    return atomic_load(v);
}
void* mylang_sync_atomic_getptr(void** v) {
    return atomic_load(v);
}
int32_t mylang_sync_atomic_geti32(int32_t* v) {
    return atomic_load(v);
}
int64_t mylang_sync_atomic_geti64(int64_t* v) {
    return atomic_load(v);
}

*/
import "C"

func Addint(v *int, delta int) int {
	// return C.atomic_addint(v, delta) // not working
	// return C.atomic_fetch_add(v, delta)
	rv := C.mylang_sync_atomic_addint(v, delta)
	return rv
}
func Addu32(v *u32, delta u32) u32 {
	rv := C.mylang_sync_atomic_addu32(v, delta)
	return rv
}
func Addu64(v *u64, delta u64) u64 {
	rv := C.mylang_sync_atomic_addu64(v, delta)
	return rv
}
func Addusize(v *usize, delta usize) usize {
	rv := C.mylang_sync_atomic_addusize(v, delta)
	return rv
}

func Casint(v *int, old int, new int) bool {
	rv := C.mylang_sync_atomic_casint(v, old, new)
	return rv
}
func Casu32(v *u32, old u32, new u32) bool {
	rv := C.mylang_sync_atomic_casu32(v, old, new)
	return rv
}
func Casu64(v *u64, old u64, new u64) bool {
	rv := C.mylang_sync_atomic_casu64(v, old, new)
	return rv
}
func Casusize(v *usize, old usize, new usize) bool {
	rv := C.mylang_sync_atomic_casusize(v, old, new)
	return rv
}
func Casvptr(v *voidptr, old voidptr, new voidptr) bool {
	rv := C.mylang_sync_atomic_casuptr(v, old, new)
	return rv
}

func Swapint(v *int, new int) int {
	rv := C.mylang_sync_atomic_swapint(v, new)
	return rv
}
func Swapu32(v *u32, new u32) u32 {
	rv := C.mylang_sync_atomic_swapu32(v, new)
	return rv
}
func Swapu64(v *u64, new u64) u64 {
	rv := C.mylang_sync_atomic_swapu64(v, new)
	return rv
}
func Swapusize(v *usize, new usize) usize {
	rv := C.mylang_sync_atomic_swapusize(v, new)
	return rv
}
func Swapvptr(v *voidptr, new voidptr) voidptr {
	rv := C.mylang_sync_atomic_swapuptr(v, new)
	return rv
}

func Setint(v *int, new int) {
	C.mylang_sync_atomic_setint(v, new)
}
func Setu32(v *u32, new u32) {
	C.mylang_sync_atomic_setu32(v, new)
}
func Setu64(v *u64, new u64) {
	C.mylang_sync_atomic_setu64(v, new)
}
func Setusize(v *usize, new usize) {
	C.mylang_sync_atomic_setusize(v, new)
}
func Setvptr(v *voidptr, new voidptr) {
	C.mylang_sync_atomic_setuptr(v, new)
}

func Getint(v *int) int {
	rv := C.mylang_sync_atomic_getint(v)
	return rv
}
func Getu32(v *u32) u32 {
	rv := C.mylang_sync_atomic_getu32(v)
	return rv
}
func Getu64(v *u64) u64 {
	rv := C.mylang_sync_atomic_getu64(v)
	return rv
}
func Getusize(v *usize) usize {
	rv := C.mylang_sync_atomic_getusize(v)
	return rv
}
func Getvptr(v *voidptr) voidptr {
	rv := C.mylang_sync_atomic_getuptr(v)
	return rv
}
