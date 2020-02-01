package main

/*
#cgo LDFLAGS: -ltcc -ldl
#include <libtcc.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

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
	cfilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cfilename))
	rv := C.tcc_output_file(tcc.cobj, cfilename)
	return int(rv)
}

func (tcc *Tcc) Run(argc int, argv []string) int {
	rv := C.tcc_run(tcc.cobj, 0, nil)
	return int(rv)
}

///
func tccpp(codebuf string, filename string, incdirs []string) int {
	tcc := newTcc()
	tcc.AddSysIncdir("/usr/include")
	tcc.AddIncdirs(incdirs...)
	tcc.SetOutputType(TCC_OUTPUT_PREPROCESS)
	tcc.SetOutputFile(filename)
	tcc.CompileStr(codebuf)
	rv := tcc.Run(0, nil)
	return int(rv)
}
