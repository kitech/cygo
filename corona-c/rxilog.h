/**
 * Copyright (c) 2017 rxi
 *
 * This library is free software; you can redistribute it and/or modify it
 * under the terms of the MIT license. See `log.c` for details.
 */

#ifndef RXI_LOG_H
#define RXI_LOG_H

#include <stdio.h>
#include <stdarg.h>

#define LOG_VERSION "0.1.0"

typedef void (*log_LockFn)(void *udata, int lock);

enum { LOG_TRACE, LOG_DEBUG, LOG_INFO, LOG_WARN, LOG_ERROR, LOG_FATAL };


///// more powerful not need fmt string, no malloc
#include "va_args_iterators/pp_iter.h"

int log_log_nofmt(int level, const char* file, int line, int argc, int vallens[], int tyids[], ...);

#define cxlog_args_eachcb(arg,idx) \
    tyid = ctypeidof(arg); tyids[idx]=tyid; vallens[idx]=sizeof(arg); // \
    // printf("hehehhe %d, tyid=%d\n",idx,tyid);

// not support 0 args
// some times, args need ((usize)42)
#define log_log_pack(level, file, line, ...) ({ \
    int argc = PP_NARG(__VA_ARGS__); \
    int tyids[argc+1]; void* argvals[argc+1];\
    int vallens[argc+1]; int tyid = 0; \
    PP_EACH_IDX(cxlog_args_eachcb, __VA_ARGS__); \
    log_log_nofmt(level, file, line, argc, vallens, tyids, __VA_ARGS__); \
})
// printf("argc=%d, %d\n", argc, 0);

// test and log part
#define log_debugif(cond, ...) if((cond)) log_log_pack(LOG_DEBUG, __FILE__, __LINE__, __VA_ARGS__)
#define log_infoif(cond, ...) if((cond)) log_log_pack(LOG_INFO,  __FILE__, __LINE__, __VA_ARGS__)
#define log_warnif(cond, ...) if((cond)) log_log_pack(LOG_WARN,  __FILE__, __LINE__, __VA_ARGS__)
#define log_errorif(cond, ...) if((cond)) log_log_pack(LOG_ERROR, __FILE__, __LINE__, __VA_ARGS__)
#define log_fatalif(cond, ...) if((cond)) log_log_pack(LOG_FATAL, __FILE__, __LINE__, __VA_ARGS__)

///////// replace/switch underly impl
#define log_log_impl log_log_pack
// #define log_log_impl log_log

//////// public api part
#define log_trace(...) log_log_impl(LOG_TRACE, __FILE__, __LINE__, __VA_ARGS__)
#define log_debug(...) log_log_impl(LOG_DEBUG, __FILE__, __LINE__, __VA_ARGS__)
#define log_info(...)  log_log_impl(LOG_INFO,  __FILE__, __LINE__, __VA_ARGS__)
#define log_warn(...)  log_log_impl(LOG_WARN,  __FILE__, __LINE__, __VA_ARGS__)
#define log_error(...) log_log_impl(LOG_ERROR, __FILE__, __LINE__, __VA_ARGS__)
#define log_fatal(...) log_log_impl(LOG_FATAL, __FILE__, __LINE__, __VA_ARGS__)

void log_set_udata(void *udata);
void log_set_lock(log_LockFn fn);
void log_set_fp(FILE *fp);
void log_set_level(int level);
void log_set_quiet(int enable);

void log_log(int level, const char *file, int line, const char *fmt, ...);

#endif
