#ifndef MINLOG_H
#define MINLOG_H

void crn_simlog(int level, const char *filename, int line, const char* funcname, const char *fmt, ...);
// internal use
void crn_simlog2(int level, const char *filename, int line, const char* funcname, const char *fmt, ...);


#define LOGLVL_FATAL 0
#define LOGLVL_ERROR 1
#define LOGLVL_WARN 2
#define LOGLVL_INFO 3
#define LOGLVL_DEBUG 4
#define LOGLVL_VERBOSE 5
#define LOGLVL_TRACE 6

#ifdef NRDEBUG
#define SHOWLOG 1
#else
#define SHOWLOG 0
#endif


#define linfo(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_INFO, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define linfo2(fmt, ...)                                                \
    if (SHOWLOG) { crn_simlog2(LOGLVL_INFO, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define lfatal(fmt, ...)                                                \
    if (SHOWLOG) { crn_simlog(LOGLVL_FATAL, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define lerror(fmt, ...)                                                \
    if (SHOWLOG) { crn_simlog(LOGLVL_ERROR, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define lwarn(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_WARN, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define ldebug(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_DEBUG, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define lverb(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_VERBOSE, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }
#define ltrace(fmt, ...)                                                 \
    if (SHOWLOG) { crn_simlog(LOGLVL_TRACE, __FILE__, __LINE__, __func__, fmt, __VA_ARGS__); }


#endif /* MINLOG_H */

