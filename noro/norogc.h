#ifndef _NORO_GC_H_
#define _NOGO_GC_H_

#define USE_BDWGC

#ifdef USE_BDWGC

#include <gc/gc.h>
#define NORO_MALLOC(size) GC_MALLOC(size)
#define NORO_FREE(ptr) GC_FREE(ptr)

#else

#include <stdlib.h>
#define NORO_MALLOC(size) calloc(1, size)
#define NORO_FREE(ptr) free(ptr)

#endif

#endif
