/*
 * Copyright (c) 2017 rxi
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to
 * deal in the Software without restriction, including without limitation the
 * rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 * sell copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */

#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>
#include <time.h>

#include "rxilog.h"

static struct {
  void *udata;
  log_LockFn lock;
  FILE *fp;
  int level;
  int quiet;
} L;


static const char *level_names[] = {
  "TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"
};

#ifdef LOG_USE_COLOR
static const char *level_colors[] = {
  "\x1b[94m", "\x1b[36m", "\x1b[32m", "\x1b[33m", "\x1b[31m", "\x1b[35m"
};
#endif


static void lock(void)   {
  if (L.lock) {
    L.lock(L.udata, 1);
  }
}


static void unlock(void) {
  if (L.lock) {
    L.lock(L.udata, 0);
  }
}


void log_set_udata(void *udata) {
  L.udata = udata;
}


void log_set_lock(log_LockFn fn) {
  L.lock = fn;
}


void log_set_fp(FILE *fp) {
  L.fp = fp;
}


void log_set_level(int level) {
  L.level = level;
}


void log_set_quiet(int enable) {
  L.quiet = enable ? 1 : 0;
}


void log_log(int level, const char *file, int line, const char *fmt, ...) {
  if (level < L.level) {
    return;
  }

  /* Acquire lock */
  lock();

  /* Get current time */
  time_t t = time(NULL);
  struct tm *lt = localtime(&t);

  /* Log to stderr */
  if (!L.quiet) {
    va_list args;
    char buf[16];
    buf[strftime(buf, sizeof(buf), "%H:%M:%S", lt)] = '\0';
#ifdef LOG_USE_COLOR
    fprintf(
      stderr, "%s %s%-5s\x1b[0m \x1b[90m%s:%d:\x1b[0m ",
      buf, level_colors[level], level_names[level], file, line);
#else
    fprintf(stderr, "%s %-5s %s:%d: ", buf, level_names[level], file, line);
#endif
    va_start(args, fmt);
    vfprintf(stderr, fmt, args);
    va_end(args);
    fprintf(stderr, "\n");
    fflush(stderr);
  }

  /* Log to file */
  if (L.fp) {
    va_list args;
    char buf[32];
    buf[strftime(buf, sizeof(buf), "%Y-%m-%d %H:%M:%S", lt)] = '\0';
    fprintf(L.fp, "%s %-5s %s:%d: ", buf, level_names[level], file, line);
    va_start(args, fmt);
    vfprintf(L.fp, fmt, args);
    va_end(args);
    fprintf(L.fp, "\n");
    fflush(L.fp);
  }

  /* Release lock */
  unlock();
}

// more powerful, no fmt string needed
#include "cxtypedefs.h"
#include <stdarg.h>
#include <stdio.h>

// sepcnt, 0, 1 works fine
static const char* log_log_file_trim(const char* file, int sepcnt) {
    char sep = '/';
    char* ptr = file+strlen(file);
    int cnt = (sepcnt<0 || sepcnt > 99) ? 1 : sepcnt;
    for (; ptr != file; ptr--) {
        if (*ptr == sep) {
            if (cnt == 0) {
                ptr++; // after current sep
                break;
            }else{
                cnt--;
            }
        }
    }

    return ptr;
}

static int log_log_snprintf_arg(char* buf, int len, int idx, int tyid, void* argptr) {

    return 0;
}

// todo more case tyid
// tyids if ctypeid enum values
// vallens used for cannot correct recognize literal char 'x'
int log_log_nofmt(int level, const char *file, int line, int argc, int vallens[], int tyids[], ...) {
    // printf("got in normal func, arc=%d\n", argc);
    char buf[4096] = {0};
    int pos = 0;

    // time level file line
    char* filemid = log_log_file_trim(file, 1);
    // pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%s:%d ", filemid, line);

    // args
    va_list args;
    va_start(args, tyids);
    for (int idx=0; idx<argc; idx++) {
        int tyid = tyids[idx];
        // printf("arg%d tyid=%d\n", idx, tyid);

        // pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%d: ", idx);
        switch (tyid) {
        case ctypeid_int: {
            int val = va_arg(args, int);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%d", val);
            }
            break;
        case ctypeid_char: {
            char val = va_arg(args, char);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%c", val);
            }
            break;
        case ctypeid_double: {
            double val = va_arg(args, double);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%g", val);
            }
            break;
        case ctypeid_ulong: {
            ulong val = va_arg(args, ulong);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%lu", val);
            }
            break;
        case ctypeid_long: {
            long val = va_arg(args, long);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%ld", val);
            }
            break;
        case ctypeid_charptr: {
            char* val = va_arg(args, char*);
            // todo if first arg is fmt string forword to old log_log, or just omit it
            if (idx==0 && strchr(val, '%') && !strstr(val, "%%")) {
                // log_log(LOG_WARN, filemid, line, "maybe old usage %s", val);
            }
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%s", val);
            }
            break;
        case ctypeid_bool: {
            _Bool val = va_arg(args, _Bool);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%s", val?"true":"false");
            }
            break;
        case ctypeid_voidptr: {
            void* val = va_arg(args, void*);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%p", val);
            }
            break;
        default: // must all used for va_arg adder inner offset
            { int val = va_arg(args, int); }
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "???");
            break;
        }
        // pos += snprintf(buf+pos, sizeof(buf)-pos-1, "(%s=%d)", ctypeid_tostr(tyid), tyid);
        pos += snprintf(buf+pos, sizeof(buf)-pos-1, " ");
    }
    va_end(args);
    // printf("logline lenth: %d\n", pos);
    pos += snprintf(buf+pos, sizeof(buf)-pos-1, " len: %d, argc: %d", pos, argc);
    // puts(buf); fflush(stderr);
    log_log(level, filemid, line, "%s", buf);
    return pos;
}
