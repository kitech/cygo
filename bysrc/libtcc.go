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
	"unsafe"
)

type Tcc struct {
	cobj *C.TCCState
}

func newTcc() *Tcc {
	tcc := &Tcc{}
	cobj := C.tcc_new()
	tcc.cobj = cobj

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
	defer C.free(unsafe.Pointer(cstr))
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
func (tcc *Tcc) AddFile(filename string) int {
	cfilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cfilename))
	rv := C.tcc_add_file(tcc.cobj, cfilename)
	return int(rv)
}
func (tcc *Tcc) CompileStr(buf string) int {
	cbuf := C.CString(buf)
	defer C.free(unsafe.Pointer(cbuf))
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
	defer C.free(unsafe.Pointer(cfilename))
	rv := C.tcc_output_file(tcc.cobj, cfilename)
	return int(rv)
}

func (tcc *Tcc) AddLibdir(dir string) int {
	cdir := C.CString(dir)
	defer C.free(unsafe.Pointer(cdir))
	rv := C.tcc_add_library_path(tcc.cobj, cdir)
	return int(rv)
}

func (tcc *Tcc) AddLib(name string) int {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
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
	defer C.free(unsafe.Pointer(cfilename))
	mod := "w+"
	cmod := C.CString(mod)

	rv := C.freopen(cfilename, cmod, C.stdout)
	log.Println(rv, rv != nil)
	gopp.Assert(rv != nil, "wtfff")
	return rv
}

func restorestdout(cfp *C.FILE) {
	cfilename := C.CString("/dev/tty")
	defer C.free(unsafe.Pointer(cfilename))
	mod := "w"
	cmod := C.CString(mod)
	defer C.free(unsafe.Pointer(cmod))

	rv := C.freopen(cfilename, cmod, C.stdout)
	log.Println(rv, rv != nil)
	gopp.Assert(rv != nil, "wtfff")
	C.fclose(cfp)
}

///
func tccpp(codebuf string, filename string, incdirs []string) error {
	srcfile := filename + ".nopp.c"
	err := ioutil.WriteFile(srcfile, []byte(codebuf), 0644)
	gopp.ErrPrint(err, filename)
	defer os.Remove(srcfile)

	// filename += ".fly.c"
	tcc := newTcc()
	rv := tcc.AddSysIncdir("/usr/include")
	// tcc.AddSysIncdir("/usr/lib/gcc/x86_64-pc-linux-gnu/9.2.0/include")
	tcc.AddSysIncdir("/usr/lib/tcc/include")
	tcc.AddIncdirs(incdirs...)
	tcc.AddLibdir("/usr/lib")
	// tcc.AddLibdir("/usr/lib/gcc/x86_64-pc-linux-gnu/9.2.0")
	tcc.AddLib("c")
	// rv := tcc.AddFile("/usr/lib/crtn.o")
	// log.Println(rv)
	// rv = tcc.SetOutputFile(filename)
	// log.Println(rv, filename)
	// tcc.SetOutputType(TCC_OUTPUT_PREPROCESS)
	// tcc.SetOutputType(TCC_OUTPUT_MEMORY)
	tcc.SetOptions("-o " + filename)
	tcc.SetOptions("-v -E")

	// rv = tcc.SetOutputFile(filename)
	// log.Println(rv, filename)

	cfp := redirstdout2file(filename)
	rv = tcc.CompileStr(codebuf)
	restorestdout(cfp)
	log.Println(rv, filename)

	// argv := []string{"-o", filename}
	// argc := len(argv)
	// argc = 0
	// log.Println(argv)
	// tcc.SetOptions(strings.Join(argv, " "))
	// rv = tcc.Run(argc, argv)
	// log.Println("saved", rv, filename, gopp.FileSize(filename))
	if rv < 0 {
		return fmt.Errorf("run error %d", rv)
	}
	log.Println("saved", filename, gopp.FileSize(filename))
	return nil
}

func tccpp2(codebuf string, filename string, incdirs []string) error {
	srcfile := filename + ".nopp.c"
	err := ioutil.WriteFile(srcfile, []byte(codebuf), 0644)
	gopp.ErrPrint(err, filename)
	defer os.Remove(srcfile)
	cmdo := exec.Command("tcc", "-E", "-o", filename, srcfile)
	err = cmdo.Run()
	gopp.ErrPrint(err, cmdo.Path, cmdo.Args)

	return err
}
