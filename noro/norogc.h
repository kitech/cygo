#ifndef _NORO_GC_H_
#define _NOGO_GC_H_

#define USE_BDWGC

#ifdef USE_BDWGC

#include <gc/gc.h>
const char* noro_gc_event_name(GC_EventType evty);

void* noro_gc_malloc(size_t size);
void* noro_gc_realloc(void* ptr, size_t size);
void noro_gc_free(void* ptr);

#endif

void* noro_raw_malloc(size_t size);
void* noro_raw_realloc(void* ptr, size_t size);
void noro_raw_free(void* ptr);

#define noro_malloc_st(/*typedesc*/st) (st*)noro_raw_malloc(sizeof(st))
#define convto(/*typedesc*/ st, /*var*/ var) (st*)(var)

#endif

