#ifndef _ATOMIC_H_
#define _ATOMIC_H_

// need -std=c11

#include <stdint.h>

int atomic_addint(int* v, int delta);
uint32_t atomic_addu32(uint32_t* v, uint32_t delta);
int32_t atomic_addi32(int32_t* v, int32_t delta) ;
uint64_t atomic_addu64(uint64_t* v, uint64_t delta);
int64_t atomic_addi64(int64_t* v, int64_t delta) ;

bool atomic_notbool(bool* v) ;
bool atomic_casbool(bool* v, bool oldval, bool newval);
bool atomic_casint(int* v, int oldval, int newval) ;
bool atomic_casu32(uint32_t* v, uint32_t oldval, uint32_t newval) ;
bool atomic_casu64(uint64_t* v, uint64_t oldval, uint64_t newval) ;
bool atomic_casuptr(uintptr_t* v, uintptr_t oldval, uintptr_t newval) ;
bool atomic_casptr(void** v, void* oldval, void* newval) ;
bool atomic_casi32(int32_t* v, int32_t oldval, int32_t newval);
bool atomic_casi64(int64_t* v, int64_t oldval, int64_t newval) ;

int atomic_swapint(int* v0, int newval) ;
uint32_t atomic_swapu32(uint32_t* v0, uint32_t newval) ;
uint64_t atomic_swapu64(uint64_t* v0, uint64_t newval) ;
uintptr_t atomic_swapuptr(uintptr_t* v0, uintptr_t newval) ;
void* atomic_swapptr(void** v0, void* newval) ;
int32_t atomic_swapi32(int32_t* v0, int32_t newval) ;
int64_t atomic_swapi64(int64_t* v0, int64_t newval) ;

void atomic_setbool(bool* v, bool val) ;
void atomic_setint(int* v, int val) ;
void atomic_setu32(uint32_t* v, uint32_t val) ;
void atomic_setu64(uint64_t* v, uint64_t val) ;
void atomic_setuptr(uintptr_t* v, uintptr_t val) ;
void atomic_setptr(void** v, void* val) ;
void atomic_seti32(int32_t* v, int32_t val);
void atomic_seti64(int64_t* v, int64_t val) ;

bool atomic_getbool(bool* v) ;
int atomic_getint(int* v) ;
uint32_t atomic_getu32(uint32_t* v);
uint64_t atomic_getu64(uint64_t* v) ;
uintptr_t atomic_getuptr(uintptr_t* v);
void* atomic_getptr(void** v) ;
int32_t atomic_geti32(int32_t* v) ;
int64_t atomic_geti64(int64_t* v) ;

#endif

