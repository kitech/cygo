package os

/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/syscall.h>
#include <sys/utsname.h>

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
	d := float64(1)
	ch1 := C.char(1)
	println(envp0)
	println(envp00)
	return arr
}

func Getenv(name string) string {
	var val string
	p := C.getenv(name.ptr)
	if p != nil {
		val = gostring(p)
	}
	return val
}
func Setenv(name string, value string, override bool) bool {
	rv := C.setenv(name.ptr, value.ptr, override)
	return rv == 0
}
func Unsetenv(name string) bool {
	rv := C.unsetenv(name.ptr)
	return rv == 0
}

func Paths() []string {
	var res []string
	p := C.getenv("PATH".ptr)
	if p == nil {
		return res
	}
	line := gostring(p)
	if line.index(":") >= 0 {
		res = line.split(":")
	} else {
		res = line.split(";")
	}
	return res
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
	//var diro voidptr // TODO compiler
	var diro *C.struct_dirent = nil //voidptr
	diro = C.opendir(dir.ptr)
	defer C.closedir(diro)
	for {
		item := C.readdir(diro)
		if item == nil {
			break
		}
		cdname := item.d_name
		dname := gostring(cdname)
		res.append(&dname)
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
	rv := C.remove(filename.ptr)
	if rv != COK {
		return newoserr1()
	}
	return nil
}

// default safe, write to temp and then rename
func WriteFile(filename string, data []byte) error {
	mode := "w+"
	var fp *C.FILE = nil
	fp = C.fopen(filename.ptr, mode.ptr)
	// fp := C.fopen(filename.ptr, mode.ptr) //TODO compiler
	if fp != nil {
		return newoserr1()
	}
	defer C.fclose(fp)
	len := data.len
	rv := C.fwrite(data.ptr, len, 1, fp)
	if rv != len {
		return newoserr1()
	}
	return nil
}

func ReadFile(filename string) ([]byte, error) {
	mode := "r"
	var fp *C.FILE = nil
	fp = C.fopen(filename.ptr, mode.ptr)
	if fp != nil {
		return nil, newoserr1()
	}
	defer C.fclose(fp)
	var res []byte
	buf := make([]byte, 8192)
	for {
		rv := C.fread(buf.ptr, buf.len, 1, fp)
		if rv > 0 {
			res = append(res, buf[:rv]...)
		}
		isferr := C.ferror(fp)
		if isferr == 1 {
			return nil, newoserr1()
		}
		iseof := C.feof(fp)
		if iseof == 1 {
			break
		}
	}
	return res, nil
}

func FileSize(filename string) int64 {
	var st = &C.struct_stat{}
	rv := C.stat(filename.ptr, st)
	return st.st_size
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

func Realpath(s string) string {
	respath := make([]byte, PATH_MAX)
	rv := C.realpath(s.ptr, respath.ptr)
	if rv == nil {
		return s
	}
	// return string(respath) // TODO compiler
	return gostring(rv)
}
func Readlink(s string) string {
	respath := make([]byte, PATH_MAX)
	rv := C.readlink(s.ptr, respath.ptr, PATH_MAX)
	if rv < 0 {
		err := newoserr1()
		println(err.Error())
		return s
	}
	return gostringn(respath.ptr, rv)
}
func Wkdir() string {
	var s string
	var buf voidptr
	for blen := 8; blen <= PATH_MAX; blen *= 2 {
		buf = realloc3(buf, blen)
		rv := C.getcwd(buf, blen)
		if rv != nil {
			s = gostring(buf)
			break
		}
	}
	return s
}
func Tmpdir() string {
	var s string
	s = Getenv("TMPDIR")
	s = ifelse(s.len == 0, "/tmp", s)
	return s
}

///// file stream
struct File {
	fp *C.FILE
	opened bool
	readonly bool
}

// readonly
func Open(filename string) *File {
	mode := "r"
	fo := OpenFile(filename, mode)
	return fo
}
// mode="rwx..."
func OpenFile(filename string, mode string) *File {
	//mode := "r"
	var fp *C.FILE = nil
	fp = C.fopen(filename.ptr, mode.ptr)
	if fp == nil {
		return nil
	}
	fo := &File{}
	fo.fp = fp
	fo.opened = true
	fo.readonly = mode == "r"
	return fo
}

func (fo *File) Write(data byteptr, len int) int64 {
	if fo.readonly {
		return -1
	}
	n := C.fwrite(data, len, 1, fo.fp)
	if n < 0 {
		return n
	}
	return len
}
func (fo *File) WriteString(data string) int64 {
	return fo.Write(data.ptr, data.len)
}

func (fo *File) Close() bool {
	if !fo.opened{
		return true
	}
	fo.opened = false
	C.fclose(fo.fp)
	return true
}

func (fo *File) Read(data byteptr, len int) int {
	n := C.fread(data, len, 1, fo.fp)
	if n < 0 {
		return n
	}
	return len
}

func (fo *File) Seek(pos int) bool {
	n := C.fseek(fo.fp, pos, C.SEEK_SET)
	return n==0
}
func (fo *File) Seekend() bool {
	n := C.fseek(fo.fp, 0, C.SEEK_END)
	return n==0
}


/////////////
type Utsname struct {
	sysname    string
	nodename   string
	release    string
	version    string
	machine    string
	domainname string
}

func (uto *Utsname) String() string {
	return uto.sysname + " " + uto.nodename + " " +
		uto.release + " " + uto.version + " " + uto.machine
}

func Uname() string {
	uts := &C.struct_utsname{}
	rv := C.uname(uts)
	if rv != 0 {
		println(Errmsg())
		return ""
	}
	uto := &Utsname{}
	uto.sysname = gostring(uts.sysname)
	uto.nodename = gostring(uts.nodename)
	uto.release = gostring(uts.release)
	uto.version = gostring(uts.version)
	uto.machine = gostring(uts.machine)
	// uto.domainname = gostring(uts.domainname)
	return uto.String()
}

func Umask(mask int) int {
	rv := C.umask(0)
	return rv
}

func Abort() { C.abort() }

func Keep() {}
