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

#include <sys/types.h>
#include <dirent.h>

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

// import "errors"

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

func Mkdir(dir string) error {
	rv := C.mkdir(dir.ptr, 0755)
	if rv != 0 {
		println("TODO error", Errmsg())
		return nil
	}
	return nil
}
func MkdirAll(dir string) error {
	rv := C.mkdir(dir.ptr, 0755)
	if rv != 0 {
		println("TODO error", Errmsg())
		return nil
	}
	return nil
}

func Rmdir(dir string) error {
	rv := C.rmdir(dir.ptr)
	if rv != 0 {
		println("TODO error", Errmsg())
		return nil
	}
	return nil
}

func RmdirAll(dir string) error {
	rv := C.rmdir(dir.ptr)
	if rv != 0 {
		println("TODO error", Errmsg())
		return nil
	}
	return nil
}

func Listdir(dir string) []string {
	var res []string
	var diro voidptr
	diro = C.opendir(dir.ptr)
	defer C.closedir(diro)
	for {
		item := C.readdir(diro)
		if item == nil {
			break
		}
		cdname := item.d_name
		dname := gostring(cdname)
		res.append(dname)
	}

	return res
}

// func Copy(from, to string) error { // TODO compiler
func Copy(from string, to string) error { // TODO compiler
	return nil
}

func Move(from string, to string) error {
	return nil
}

func Remove(filename string) error {
	return nil
}

func WriteFile(filename string, data []byte) error {
	return nil
}

func ReadFile(filename string) ([]byte, error) {
	return nil, nil
}

func FileExist(filename string) bool {
	rv := C.access(filename.ptr, C.F_OK)
	return rv == 0
}
func IsReadable(filename string) bool {
	rv := C.access(filename.ptr, C.R_OK)
	return rv == 0
}
func IsWritable(filename string) bool {
	rv := C.access(filename.ptr, C.W_OK)
	return rv == 0
}
func IsExcutable(filename string) bool {
	rv := C.access(filename.ptr, C.X_OK)
	return rv == 0
}

func Umask(mask int) int {
	rv := C.umask(0)
	return rv
}

func Keep() {}
