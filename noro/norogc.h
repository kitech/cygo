#ifndef _NORO_GC_H_
#define _NOGO_GC_H_

#define USE_BDWGC

#ifdef USE_BDWGC

#include <gc/gc.h>
#define NORO_MALLOC(size) GC_MALLOC(size)
#define NORO_FREE(ptr) GC_FREE(ptr)
#define NORO_REALLOC(obj, new_size) GC_REALLOC(obj, new_size)

#else

#include <stdlib.h>
#define NORO_MALLOC(size) calloc(1, size)
#define NORO_FREE(ptr) free(ptr)
#define NORO_REALLOC(obj, new_size) realloc(obj, new_size)

#endif

#endif
