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

/* Civil date from days since 1970-01-01 (Howard Hinnant). */
static void log_civil_from_days(int days, int* y, int* m, int* d) {
    days += 719468;
    int era = (days >= 0 ? days : days - 146096) / 146097;
    unsigned doe = (unsigned)(days - era * 146097);
    unsigned yoe = (doe - doe/1460 + doe/36524 - doe/146096) / 365;
    int y0 = (int)yoe + era * 400;
    unsigned doy = doe - (365*yoe + yoe/4 - yoe/100);
    unsigned mp = (5*doy + 2) / 153;
    *d = (int)(doy - (153*mp + 2)/5 + 1);
    *m = (int)(mp + (mp < 10 ? 3 : -9));
    *y = y0 + (*m <= 2);
}

/* UTC+8 time parts without gmtime_r/localtime_r/strftime/tzset locks. */
static void log_time_parts(time_t t, int* y, int* mon, int* d, int* hh, int* mm, int* ss) {
    long long sec = (long long)t + 8 * 3600; /* UTC+8 */
    long long days = sec / 86400;
    int daysec = (int)(sec % 86400);
    if (daysec < 0) { daysec += 86400; days -= 1; }
    log_civil_from_days((int)days, y, mon, d);
    *hh = daysec / 3600;
    *mm = (daysec % 3600) / 60;
    *ss = daysec % 60;
}

/* UTC+8 HH:MM:SS without gmtime_r/localtime_r/strftime/tzset locks. */
static void log_time_hms(char* buf, size_t sz) {
    int y, mon, d, hh, mm, ss;
    log_time_parts(time(NULL), &y, &mon, &d, &hh, &mm, &ss);
    snprintf(buf, sz, "%02d:%02d:%02d", hh, mm, ss);
}

/* UTC+8 YYYY-MM-DD HH:MM:SS without gmtime_r/localtime_r/strftime/tzset locks. */
static void log_time_ymdhms(char* buf, size_t sz) {
    int y, mon, d, hh, mm, ss;
    log_time_parts(time(NULL), &y, &mon, &d, &hh, &mm, &ss);
    snprintf(buf, sz, "%04d-%02d-%02d %02d:%02d:%02d", y, mon, d, hh, mm, ss);
}

void log_log(int level, const char *file, int line, const char *fmt, ...) {
  if (level < L.level) {
    return;
  }

  const char* filemid = log_log_file_trim(file, 1);
  int fmtlen = strlen(fmt);
  int fmt_has_newline = (fmtlen > 0) && (fmt[fmtlen - 1] == '\n');

  /* Acquire lock */
  // lock();   /* never hold the log mutex across write()/stdio: GC STW may suspend this thread */

  /* Log to stderr: fixed 2048-byte stack buffer, truncate on overflow,
   * single write(), and no log mutex held across the write. */
  if (!L.quiet) {
    char tbuf[2048];
    char buf[16];
    log_time_hms(buf, sizeof(buf));

    int bpos;
#ifdef LOG_USE_COLOR
    bpos = snprintf(tbuf, sizeof(tbuf),
      "%s %s%-5s\x1b[0m \x1b[90m%s:%d:\x1b[0m ",
      buf, level_colors[level], level_names[level], filemid, line);
#else
    bpos = snprintf(tbuf, sizeof(tbuf), "%s %-5s %s:%d: ",
      buf, level_names[level], filemid, line);
#endif
    if (bpos < 0) bpos = 0;
    if (bpos > (int)sizeof(tbuf) - 1) bpos = (int)sizeof(tbuf) - 1;

    va_list args;
    va_start(args, fmt);
    int n = vsnprintf(tbuf + bpos, sizeof(tbuf) - bpos, fmt, args);
    va_end(args);
    if (n < 0) n = 0;
    if (n >= (int)sizeof(tbuf) - bpos) {
      bpos = (int)sizeof(tbuf) - 1;   /* truncated */
    } else {
      bpos += n;
    }
    if (!fmt_has_newline && bpos < (int)sizeof(tbuf) - 1) {
      tbuf[bpos++] = '\n';
    }
    tbuf[bpos] = '\0';
    if (bpos > 0) write(STDERR_FILENO, tbuf, (size_t)bpos);
  }

  /* Log to file: same stack-buffer policy; stdio outside the log mutex */
  if (L.fp) {
    char fbuf[2048];
    char buf[32];
    log_time_ymdhms(buf, sizeof(buf));

    int fpos = snprintf(fbuf, sizeof(fbuf), "%s %-5s %s:%d: ",
      buf, level_names[level], filemid, line);
    if (fpos < 0) fpos = 0;
    if (fpos > (int)sizeof(fbuf) - 1) fpos = (int)sizeof(fbuf) - 1;

    va_list args;
    va_start(args, fmt);
    int n = vsnprintf(fbuf + fpos, sizeof(fbuf) - fpos, fmt, args);
    va_end(args);
    if (n < 0) n = 0;
    if (n >= (int)sizeof(fbuf) - fpos) {
      fpos = (int)sizeof(fbuf) - 1;   /* truncated */
    } else {
      fpos += n;
    }
    if (!fmt_has_newline && fpos < (int)sizeof(fbuf) - 1) {
      fbuf[fpos++] = '\n';
    }
    fbuf[fpos] = '\0';
    if (fpos > 0) {
      fwrite(fbuf, 1, (size_t)fpos, L.fp);
      fflush(L.fp);
    }
  }

  /* Release lock */
  // unlock();
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
// bounded append helper: snprintf returns the WOULD-BE length, so raw
// accumulation overflows buf once it fills up; clamp the return value
static int logno_append(char* buf, int bufsz, int pos, const char* fmt, ...) {
    if (pos < 0) { pos = 0; }
    if (pos > bufsz - 1) { return pos; }
    va_list args;
    va_start(args, fmt);
    int n2 = vsnprintf(buf+pos, bufsz-pos-1, fmt, args);
    va_end(args);
    if (n2 < 0) { n2 = 0; }
    if (n2 >= bufsz - pos) { n2 = bufsz - pos - 1; }
    return pos + n2;
}
int log_log_nofmt(int level, const char *file, int line, int vallens[], int tyids[], int argcx, ...) {
    // printf("got in normal func, arc=%d\n", argc);
    char buf[4096] = {0};
    int pos = 0;

    // time level file line
    const char* filemid = log_log_file_trim(file, 1);
    // pos += logno_append(buf, sizeof(buf), pos, "%s:%d ", filemid, line);

    // args
    va_list args;
    va_start(args, argcx);
    for (int idx=0; idx<argcx; idx++) {
        int tyid = tyids[idx];
        const char* tystr = ctypeid_tostr(tyid);
        // printf("arg%d tyid=%d\n", idx, tyid);

        // pos += logno_append(buf, sizeof(buf), pos, "%d: ", idx);
        switch (tyid) {
        case ctypeid_int: {
            int val = va_arg(args, int);
            pos += logno_append(buf, sizeof(buf), pos, "%d", val);
            // if (val==4) cxpanic(0, "www");
            }
            break;
        case ctypeid_uchar: {
            uchar val = va_arg(args, unsigned char);
            pos += logno_append(buf, sizeof(buf), pos, "%u", val);
            }
            break;
        case ctypeid_char: {
            char val = va_arg(args, char);
            pos += logno_append(buf, sizeof(buf), pos, "%c", val);
            }
            break;
        case ctypeid_double: {
            double val = va_arg(args, double);
            pos += logno_append(buf, sizeof(buf), pos, "%g", val);
            }
            break;
        case ctypeid_float: {
            double val = va_arg(args, double);
            pos += logno_append(buf, sizeof(buf), pos, "%g", val);
            }
            break;
        case ctypeid_ulong: {
            ulong val = va_arg(args, ulong);
            pos += logno_append(buf, sizeof(buf), pos, "%lu", val);
            }
            break;
        case ctypeid_long: {
            long val = va_arg(args, long);
            pos += logno_append(buf, sizeof(buf), pos, "%ld", val);
            }
            break;
        case ctypeid_charptr: {
            char* val = va_arg(args, char*);
            // todo if first arg is fmt string forword to old log_log, or just omit it
            if (idx==0 && strchr(val, '%') && !strstr(val, "%%")) {
                // log_log(LOG_WARN, filemid, line, "maybe old usage %s", val);
            }
            pos += logno_append(buf, sizeof(buf), pos, "%s", val);
            }
            break;
        case ctypeid_charptrptr: {
            char** val = va_arg(args, char**);
            pos += logno_append(buf, sizeof(buf), pos, "%p", val);
            }
            break;
        case ctypeid_bool: {
            _Bool val = va_arg(args, _Bool);
            pos += logno_append(buf, sizeof(buf), pos, "%s", val?"true":"false");
            }
            break;
        case ctypeid_voidptr: {
            void* val = va_arg(args, void*);
            pos += logno_append(buf, sizeof(buf), pos, "%p", val);
            }
            break;
        default: // must all used for va_arg adder inner offset
            { int val = va_arg(args, int); }
            pos += logno_append(buf, sizeof(buf), pos, "???(%d,%s)", tyid, tystr);
            break;
        }
        // pos += logno_append(buf, sizeof(buf), pos, "(%s=%d)", ctypeid_tostr(tyid), tyid);
        pos += logno_append(buf, sizeof(buf), pos, " ");
    }
    va_end(args);
    // printf("logline lenth: %d\n", pos);
    pos += logno_append(buf, sizeof(buf), pos, " \x1b[0m \x1b[90mlen: %d, argc: %d\x1b[0m", pos, argcx);
    // printf("wttt %d, %d %s\n", pos, strlen(buf), buf);
    // puts(buf); fflush(stderr);
    log_log(level, filemid, line, "%s", buf);

    return pos;
}
