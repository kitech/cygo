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

#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>
#include <time.h>
#include <assert.h>
#include <unistd.h>

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


int log_set_level(int level) {
    int oldval = L.level;
    L.level = level;
    return oldval;
}


void log_set_quiet(int enable) {
  L.quiet = enable ? 1 : 0;
}

// sepcnt, 0, 1 works fine
static const char* log_log_file_trim(const char* file, int sepcnt) {
    char sep = '/';
    char* ptr = (char*)(file+strlen(file));
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

void log_log(int level, const char *file, int line, const char *fmt, ...) {
  if (level < L.level) {
    return;
  }

  const char* filemid = log_log_file_trim(file, 1);
  int fmtlen = strlen(fmt);
  int fmt_has_newline = fmt[fmtlen-1] == '\n';
  char tbuf[4096] = {0};
  int bpos = 0;

  // todo __lll_lock_wait_private deadlock
  /* Get current time */
  int zone = 8;
  time_t t = time(NULL);
  struct tm lt0 = {0};
  // struct tm *lt = localtime_r(&t, &lt0);
  struct tm *lt = gmtime_r(&t, &lt0);
  lt->tm_hour = (lt->tm_hour+zone)%24;

  /* Acquire lock */
  lock();

  /* Log to stderr */
  if (!L.quiet) {
    va_list args;
    char buf[16];
    buf[strftime(buf, sizeof(buf), "%H:%M:%S", lt)] = '\0';
#ifdef LOG_USE_COLOR
    // fprintf(
    //   stderr, "%s %s%-5s\x1b[0m \x1b[90m%s:%d:\x1b[0m ",
    //   buf, level_colors[level], level_names[level], filemid, line);
    bpos += snprintf(tbuf+bpos, sizeof(tbuf)-bpos,
      "%s %s%-5s\x1b[0m \x1b[90m%s:%d:\x1b[0m ",
      buf, level_colors[level], level_names[level], filemid, line);
#else
    // fprintf(stderr, "%s %-5s %s:%d: ", buf, level_names[level], filemid, line);
    bpos += snprintf(tbuf+bpos, sizeof(tbuf)-bpos, "%s %-5s %s:%d: ", buf, level_names[level], filemid, line);
#endif
    va_start(args, fmt);
    // vfprintf(stderr, fmt, args);
    bpos += vsnprintf(tbuf+bpos, sizeof(tbuf)-bpos, fmt, args);
    va_end(args);
    if (!fmt_has_newline) bpos+= snprintf(tbuf+bpos, sizeof(tbuf)-bpos, "\n");
    assert (bpos<sizeof(tbuf));
    // extern int (*write_f)();
    // write_f(STDERR_FILENO, tbuf, bpos);
    write(STDERR_FILENO, tbuf, bpos);
    // fprintf(stderr, "%s", tbuf);
    // fflush(stderr);
  }

  /* Log to file */
  if (L.fp) {
    va_list args;
    char buf[32];
    buf[strftime(buf, sizeof(buf), "%Y-%m-%d %H:%M:%S", lt)] = '\0';
    fprintf(L.fp, "%s %-5s %s:%d: ", buf, level_names[level], filemid, line);
    va_start(args, fmt);
    vfprintf(L.fp, fmt, args);
    va_end(args);
    if (!fmt_has_newline) fprintf(L.fp, "\n");
    fflush(L.fp);
  }

  /* Release lock */
  unlock();
}

// more powerful, no fmt string needed
#include "cxtypedefs.h"
#include <stdarg.h>
#include <stdio.h>


static int log_log_snprintf_arg(char* buf, int len, int idx, int tyid, void* argptr) {

    return 0;
}

// todo more case tyid
// tyids if ctypeid enum values
// vallens used for cannot correct recognize literal char 'x'
int log_log_nofmt(int level, const char *file, int line, int vallens[], int tyids[], int argcx, ...) {
    // printf("got in normal func, arc=%d\n", argc);
    char buf[4096] = {0};
    int pos = 0;

    // time level file line
    const char* filemid = log_log_file_trim(file, 1);
    // pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%s:%d ", filemid, line);

    // args
    va_list args;
    va_start(args, argcx);
    for (int idx=0; idx<argcx; idx++) {
        int tyid = tyids[idx];
        const char* tystr = ctypeid_tostr(tyid);
        // printf("arg%d tyid=%d\n", idx, tyid);

        // pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%d: ", idx);
        switch (tyid) {
        case ctypeid_int: {
            int val = va_arg(args, int);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%d", val);
            // if (val==4) cxpanic(0, "www");
            }
            break;
        case ctypeid_uchar: {
            uchar val = va_arg(args, unsigned char);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%u", val);
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
        case ctypeid_float: {
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
        case ctypeid_charptrptr: {
            char** val = va_arg(args, char**);
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "%p", val);
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
            pos += snprintf(buf+pos, sizeof(buf)-pos-1, "???(%d,%s)", tyid, tystr);
            break;
        }
        // pos += snprintf(buf+pos, sizeof(buf)-pos-1, "(%s=%d)", ctypeid_tostr(tyid), tyid);
        pos += snprintf(buf+pos, sizeof(buf)-pos-1, " ");
    }
    va_end(args);
    // printf("logline lenth: %d\n", pos);
    pos += snprintf(buf+pos, sizeof(buf)-pos-1, " \x1b[0m \x1b[90mlen: %d, argc: %d\x1b[0m", pos, argcx);
    // printf("wttt %d, %d %s\n", pos, strlen(buf), buf);
    // puts(buf); fflush(stderr);
    log_log(level, filemid, line, "%s", buf);

    return pos;
}
