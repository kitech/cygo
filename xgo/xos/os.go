package xos

/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>

#include <sys/syscall.h>

int xos_gettid3() {
#ifdef SYS_gettid
    // extern int syscall(int); // fix hidden definition
    int tid = syscall(SYS_gettid);
    return tid;
#else
#error "SYS_gettid unavailable on this system"
    return 0;
#endif
}

   extern int cxrt_get_argc();
   extern char** cxrt_get_argv();
*/
import "C"

// import "unsafe"

const (
	DftMode  = 0644
	DftMask  = 0022
	PATH_MAX = 256 // C.PATH_MAX
)

const (
	PathSep = "/"
)

func Gettid() int {
	return C.xos_gettid3()
}

func Touch(path string) bool {
	fp := C.open(path.ptr, C.O_RDWR|C.O_CREAT, 0644)
	if fp < 0 {
		return false
	}
	C.close(fp)
	return true
}

func Environ() []string {
	arr := []string{}
	envp := C.__environ
	envp0 := envp[0]
	envp00 := envp[0][0]
	b := C.O_RDWR
	c := C.int(1)
	d := C.double(1)
	ch1 := C.char(1)
	println(envp0)
	println(envp00)
	return arr
}

func Exit(code int) {
	C.exit(code)
}

func Args() []string {
	var args []string
	argc := C.cxrt_get_argc()
	argvpp := C.cxrt_get_argv()
	println(argc)
	for i := 0; i < argc; i++ {
		argp := argvpp[i]
		arg := gostring(argp)
		args = append(args, arg)
	}
	return args
}

func Keep() {}
