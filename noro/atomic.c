#include <stdint.h>
#include <stdbool.h>
#include <stdatomic.h>

#include "atomic.h"

int atomic_addint(int* v, int delta) {
    return atomic_fetch_add(v, delta);
}
uint32_t atomic_addu32(uint32_t* v, uint32_t delta) {
    return atomic_fetch_add(v, delta);
}
int32_t atomic_addi32(int32_t* v, int32_t delta) {
    return atomic_fetch_add(v, delta);
}
uint64_t atomic_addu64(uint64_t* v, uint64_t delta) {
    return atomic_fetch_add(v, delta);
}
int64_t atomic_addi64(int64_t* v, int64_t delta) {
    return atomic_fetch_add(v, delta);
}

bool atomic_notbool(bool* v) {
    return atomic_compare_exchange_strong(v, v, !*v);
}

bool atomic_casint(int* v, int oldval, int newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool atomic_casu32(uint32_t* v, uint32_t oldval, uint32_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool atomic_casu64(uint64_t* v, uint64_t oldval, uint64_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool atomic_casuptr(uintptr_t* v, uintptr_t oldval, uintptr_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool atomic_casptr(void** v, void* oldval, void* newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool atomic_casi32(int32_t* v, int32_t oldval, int32_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}
bool atomic_casi64(int64_t* v, int64_t oldval, int64_t newval) {
    return atomic_compare_exchange_strong(v, &oldval, newval);
}

int atomic_swapint(int* v0, int newval) {
    return atomic_exchange(v0, newval);
}
uint32_t atomic_swapu32(uint32_t* v0, uint32_t newval) {
    return atomic_exchange(v0, newval);
}
uint64_t atomic_swapu64(uint64_t* v0, uint64_t newval) {
    return atomic_exchange(v0, newval);
}
uintptr_t atomic_swapuptr(uintptr_t* v0, uintptr_t newval) {
    return atomic_exchange(v0, newval);
}
void* atomic_swapptr(void** v0, void* newval) {
    return atomic_exchange(v0, newval);
}
int32_t atomic_swapi32(int32_t* v0, int32_t newval) {
    return atomic_exchange(v0, newval);
}
int64_t atomic_swapi64(int64_t* v0, int64_t newval) {
    return atomic_exchange(v0, newval);
}

void atomic_setbool(bool* v, bool val) {
    atomic_store(v, val);
}
void atomic_setint(int* v, int val) {
    atomic_store(v, val);
}
void atomic_setu32(uint32_t* v, uint32_t val) {
    atomic_store(v, val);
}
void atomic_setu64(uint64_t* v, uint64_t val) {
    atomic_store(v, val);
}
void atomic_setuptr(uintptr_t* v, uintptr_t val) {
    atomic_store(v, val);
}
void atomic_setptr(void** v, void* val) {
    atomic_store(v, val);
}
void atomic_seti32(int32_t* v, int32_t val) {
    atomic_store(v, val);
}
void atomic_seti64(int64_t* v, int64_t val) {
    atomic_store(v, val);
}

bool atomic_getbool(bool* v) {
    return atomic_load(v);
}
int atomic_getint(int* v) {
    return atomic_load(v);
}
uint32_t atomic_getu32(uint32_t* v) {
    return atomic_load(v);
}
uint64_t atomic_getu64(uint64_t* v) {
    return atomic_load(v);
}
uintptr_t atomic_getuptr(uintptr_t* v) {
    return atomic_load(v);
}
void atomic_getptr(void** v) {
    atomic_load(v);
}
int32_t atomic_geti32(int32_t* v) {
    return atomic_load(v);
}
int64_t atomic_geti64(int64_t* v) {
    return atomic_load(v);
}

