package main

/*
#cgo LDFLAGS: -ltcc -ldl
#include <libtcc.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdio.h>

int tcc_run2(TCCState* state, int argc, uintptr_t argv) {

char** argv2 = (char**)argv;
for (int i = 0; i < argc; i++) {
printf("%d %s\n", i, argv2[i]);
}
char* argv3[] = {
"-E", "-o", "/tmp/heheh.c", NULL,
};
// return tcc_run(state, argc, (char**)argv);
return tcc_run(state, 3, (char**)argv3);
}
*/
import "C"
import (
	"fmt"
	"gopp"
	"gopp/cgopp"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unsafe"

	"github.com/thoas/go-funk"
)

type Tcc struct {
	cobj *C.TCCState
}

func newTcc() *Tcc {
	tcc := &Tcc{}
	cobj := C.tcc_new()
	tcc.cobj = cobj

	runtime.SetFinalizer(tcc, tcc_finalizer)
	return tcc
}

func tcc_finalizer(objx interface{}) {

}

func (tcc *Tcc) delete() {
	C.tcc_delete(tcc.cobj)
}

///
func (tcc *Tcc) SetOptions(str string) {
	cstr := C.CString(str)
	defer cgopp.Cfree3(cstr)
	C.tcc_set_options(tcc.cobj, cstr)
}

///
func (tcc *Tcc) AddIncdir(dir string) int {
	rv := C.tcc_add_include_path(tcc.cobj, C.CString(dir))
	return int(rv)
}
func (tcc *Tcc) AddIncdirs(dirs ...string) {
	for _, dir := range dirs {
		tcc.AddIncdir(dir)
	}
}
func (tcc *Tcc) AddSysIncdir(dir string) int {
	rv := C.tcc_add_sysinclude_path(tcc.cobj, C.CString(dir))
	return int(rv)
}
func (tcc *Tcc) AddSysIncdirs(dirs ...string) {
	for _, dir := range dirs {
		tcc.AddSysIncdir(dir)
	}
}

///
// temporary disable this for signature not match in some env
func (tcc *Tcc) AddFile(filename string) int {
	cfilename := C.CString(filename)
	defer cgopp.Cfree3(cfilename)
	log.Panicln("todo")
	rv := 0
	// rv := C.tcc_add_file(tcc.cobj, cfilename)
	return int(rv)
}
func (tcc *Tcc) CompileStr(buf string) int {
	cbuf := C.CString(buf)
	defer cgopp.Cfree3(cbuf)
	rv := C.tcc_compile_string(tcc.cobj, cbuf)
	return int(rv)
}

const TCC_OUTPUT_MEMORY = 1     /* output will be run in memory (default) */
const TCC_OUTPUT_EXE = 2        /* executable file */
const TCC_OUTPUT_DLL = 3        /* dynamic library */
const TCC_OUTPUT_OBJ = 4        /* object file */
const TCC_OUTPUT_PREPROCESS = 5 /* only preprocess (used internally) */

func (tcc *Tcc) SetOutputType(typ int) {
	C.tcc_set_output_type(tcc.cobj, C.int(typ))
}

func (tcc *Tcc) SetOutputFile(filename string) int {
	log.Println(filename)
	cfilename := C.CString(filename)
	defer cgopp.Cfree3(cfilename)
	rv := C.tcc_output_file(tcc.cobj, cfilename)
	return int(rv)
}

func (tcc *Tcc) AddLibdir(dir string) int {
	cdir := C.CString(dir)
	defer cgopp.Cfree3(cdir)
	rv := C.tcc_add_library_path(tcc.cobj, cdir)
	return int(rv)
}

func (tcc *Tcc) AddLib(name string) int {
	cname := C.CString(name)
	defer cgopp.Cfree3(cname)
	rv := C.tcc_add_library(tcc.cobj, cname)
	return int(rv)
}

func (tcc *Tcc) Run(argc int, argv []string) int {
	cargv := cgopp.CStrArrFromStrs(argv)
	p2 := uintptr(unsafe.Pointer(uintptr(cargv.ToC())))

	rv := C.tcc_run2(tcc.cobj, C.int(argc), (C.uintptr_t)(p2))
	return int(rv)
}

// freopen("/dev/tty", "w", stdout); /*for gcc, ubuntu*/
// freopen("CON", "w", stdout); /*Mingw C++; Windows*/

func redirstdout2file(filename string) *C.FILE {
	cfilename := C.CString(filename)
	defer cgopp.Cfree3(cfilename)
	mod := "w+"
	cmod := C.CString(mod)

	rv := C.freopen(cfilename, cmod, C.stdout)
	log.Println(rv, rv != nil)
	gopp.Assert(rv != nil, "wtfff")
	return rv
}

func restorestdout(cfp *C.FILE) {
	cfilename := C.CString("/dev/tty")
	defer cgopp.Cfree3(cfilename)
	mod := "w"
	cmod := C.CString(mod)
	defer cgopp.Cfree3(cmod)

	// TODO not work on github ubuntu runner
	rv := C.freopen(cfilename, cmod, C.stdout)
	if rv == nil {
		C.fclose(cfp)
	} else {
		log.Println(rv, rv != nil)
		// gopp.Assert(rv != nil, "wtfff")
	}
}

///
// TODO stdio.h:27: error: include file 'bits/libc-header-start.h' not found
func tccpp(codebuf string, filename string, incdirs []string) error {
	const (
		ppk_tccfly = 1
		ppk_tcccmd = 2
		ppk_gcccmd = 3
	)
	var ppkind = ppk_tcccmd
	switch ppkind {
	case ppk_tccfly:
		return tccppfly(codebuf, filename, incdirs)
	case ppk_tcccmd:
		return tccppcmd(codebuf, filename, incdirs)
	case ppk_gcccmd: // TODO still not work!!!
		return gccppcmd(codebuf, filename, incdirs)
	default:
		panic("not support")
	}
}

func tccppfly(codebuf string, filename string, incdirs []string) error {
	tcc := newTcc()
	rv := tcc.AddSysIncdir("/usr/include")
	tcc.AddSysIncdir("/usr/lib/tcc/include")
	tcc.AddIncdirs(incdirs...)
	tcc.AddLibdir("/usr/lib")
	tcc.AddLib("c")
	// rv := tcc.AddFile("/usr/lib/crtn.o")
	// log.Println(rv)
	// rv = tcc.SetOutputFile(filename) // crash use with -E
	// log.Println(rv, filename)
	// tcc.SetOutputType(TCC_OUTPUT_PREPROCESS)
	// tcc.SetOutputType(TCC_OUTPUT_MEMORY)
	tcc.SetOptions("-o " + filename)
	tcc.SetOptions("-v -E")
	tcc.SetOptions("-DGC_THREADS")

	cfp := redirstdout2file(filename)
	rv = tcc.CompileStr(codebuf)
	restorestdout(cfp)
	log.Println(rv, filename)
	tcc.delete()

	if rv < 0 {
		return fmt.Errorf("run error %d", rv)
	}
	if gopp.FileSize(filename) == 0 {
		return fmt.Errorf("empty cppout file", filename)
	}
	return nil
}

func xccppsave(codebuf string, filename string) (string, error) {
	srcfile := filename + ".nopp.c"
	err := ioutil.WriteFile(srcfile, []byte(codebuf), 0644)
	gopp.ErrPrint(err, filename)

	return srcfile, err
}

// contains -Dfoo=1 -I bar
func xccpp_cmd_args(isgcc bool, srcfile, filename string) []string {
	var args []string
	args = append(args, gopp.IfElseStr(isgcc, "gcc", "tcc"))
	for _, incdir := range get_compile_incdirs(false) {
		args = append(args, "-I", incdir)
	}
	for _, defitem := range get_compile_predefs() {
		args = append(args, defitem)
	}
	args = append(args, "-E", "-o", filename, srcfile)
	return args
}

func tccppcmd(codebuf string, filename string, incdirs []string) error {
	srcfile, err := xccppsave(codebuf, filename)
	defer os.Remove(srcfile)
	args := xccpp_cmd_args(false, srcfile, filename)
	cmdo := exec.Command(args[0], args[1:]...)
	errout, err := cmdo.CombinedOutput()
	gopp.ErrPrint(err, cmdo.Path, cmdo.Args, string(errout))

	return err
}

func gccppcmd(codebuf string, filename string, incdirs []string) error {
	srcfile, err := xccppsave(codebuf, filename)
	defer os.Remove(srcfile)
	args := xccpp_cmd_args(true, srcfile, filename)
	cmdo := exec.Command(args[0], args[1:]...)
	errout, err := cmdo.CombinedOutput()
	gopp.ErrPrint(err, cmdo.Path, cmdo.Args, string(errout))

	return err
}

var preincdirs = []string{
	// cxrtroot + "/src",
	cxrtroot + "/3rdparty/cltc/src",
	cxrtroot + "/3rdparty/cltc/src/include",
	// cxrtroot+"/3rdparty/tcc",
	"/usr/include/gc",
	"/usr/include/curl",
}
var presysincs = []string{"/usr/include", "/usr/local/include",
	"/usr/include/x86_64-linux-gnu/", // ubuntu
}

const codepfx = "#include <stdio.h>\n" +
	"#include <stdlib.h>\n" +
	"#include <string.h>\n" +
	"#include <errno.h>\n" +
	"#include <pthread.h>\n" +
	"#include <time.h>\n" +
	// "#include <cxrtbase.h>\n" +
	"\n"

var cxrtroot = "/home/none/oss/cxrt" // not exist one, force dynamic populate

var cxrtincs = []string{
	// "src",
	"3rdparty/cltc/src", "3rdparty/cltc/include"}

// 使用单独init函数名
func init() { init_cxrtroot() }
func init_cxrtroot() {
	if !gopp.FileExist(cxrtroot) {
		gopaths := gopp.Gopaths()
		for _, gopath := range gopaths {
			d := gopath + "/src/cxrt" // github actions runner
			if gopp.FileExist(d) {
				cxrtroot = d
				break
			}
		}
	}
}

// default tcc
func get_compile_incdirs(isgcc bool) []string {
	var incdirs []string
	wkdir, err := os.Getwd()
	gopp.ErrPrint(err)
	incdirs = append(incdirs, wkdir)

	for _, item := range cxrtincs {
		d := cxrtroot + "/" + item
		if funk.Contains(incdirs, d) {
			continue
		}
		incdirs = append(incdirs, d)
	}
	if isgcc {
		incdirs = append(incdirs, "/usr/lib/gcc/x86_64-pc-linux-gnu/9.2.1/include")
	} else {
		incdirs = append(incdirs, cxrtroot+"/3rdparty/tcc")
		incdirs = append(incdirs, "/usr/lib/tcc/include/")
	}
	return incdirs
}

// -Dfoo=1 -Dbar=2
func get_compile_predefs() []string {
	predefs := "-DGC_THREADS" // TODO auto from #cgo pragma
	return strings.Fields(predefs)
}

// "-DFOO=1 -DBAR -DBAZ=fff"
func cp2_split_predefs(predefs string) map[string]interface{} {
	items := strings.Split(predefs, " ")
	res := map[string]interface{}{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		gopp.Assert(strings.HasPrefix(item, "-D"), "wtfff", item)
		item = item[2:]
		kv := strings.Split(item, "=")
		if len(kv) == 1 {
			res[item] = 1
		} else {
			if gopp.IsInteger(kv[1]) {
				res[kv[0]] = gopp.MustInt(kv[1])
			} else {
				res[kv[0]] = kv[1]
			}
		}
	}
	for k, v := range res {
		log.Println("predefsm", k, v, reftyof(v))
	}

	return res
}

// C preprocessor
type Cpreprocessor interface {
	String(codebuf string, filename string, incdirs ...string) error
	File(codefile string, filename string, incdirs ...string) error
}

// C compile env populator
type CcompileEnv struct {
	incdirs []string
	sysincs []string
	predefs string
}
