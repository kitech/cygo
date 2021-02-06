package builtin

/*
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdarg.h>
#include <unistd.h>
#include <time.h>
#include <stdbool.h>
#include <stdint.h>

   extern void crn_init_and_wait_done();
   // 如果在其他地方import了 SCHED包，则自动开启
#ifdef CYGO_ENABLE_SCHED
   extern void sched__pre_gc_init(); // 前置声明，builtin需要反向依赖 sched包了
   extern void sched__pre_main_init(); // 前置声明，builtin需要反向依赖 sched包了
#else
   // dummy one
   static void sched__pre_gc_init(){}
   static void sched__pre_main_init(){}
#endif

void println2(const char* filename, int lineno, const char* funcname, const char* fmt, ...) {
    static __thread char obuf[712] = {0};
    const char* fbname = strrchr(filename, '/');
    if (fbname != nilptr) { fbname = fbname + 1; }
    else { fbname = filename; }

    int len = snprintf(obuf, sizeof(obuf)-1, "%s:%d:%s ", fbname, lineno, funcname);

    va_list arg;
    va_start (arg, fmt);
    len += vsnprintf(obuf+len,sizeof(obuf)-len-1,fmt,arg);
    va_end (arg);
    obuf[len++] = '\n';

    write(STDERR_FILENO, obuf, len);
}

*/
import "C"

///////////////

//export crn_init_and_wait_done
func crn_init_and_wait_done() {

}

///////////////

//export cxrt_get_argv
func get_argv() *byteptr { return cxargv }

//export cxrt_get_argc
func get_argc() int { return cxargc }

func get_envp() *byteptr {
	envpp := C.__environ
	return envpp
}

var cxrt_inited int
var cxargc int
var cxargv *byteptr

//export cxrt_init_env
func init_env(argc int, argv *byteptr) {
	if cxrt_inited > 0 {
		C.printf("%s:%d %s already inited %d\n",
			C.__FILE__, C.__LINE__, C.__FUNCTION__, cxrt_inited)
		return
	}
	cxrt_inited = C.time(nil)
	cxargc = argc
	cxargv = argv

	C.sched__pre_gc_init()
	memory_init()
	C.sched__pre_main_init()

	// cxrt_init_gc_env();
	C.crn_init_and_wait_done()
}

//export println2_wip
func println2(filename byteptr, lineno int, funcname byteptr, fmt byteptr) {

}

// void println2(const char* filename, int lineno, const char* funcname, const char* fmt, ...);
