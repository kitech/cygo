package xlog

/*
#cgo LDFLAGS: -ldwarf

#include <execinfo.h>
#include <libdwarf/dwarf.h>
#include <libdwarf/libdwarf.h>
#include <libelf.h>

*/
import "C"
import (
	"xgo/xstrings"
)

type Frame struct {
	Btdepth int

	Funcname string
	Mglname  string
	Funcaddr voidptr // unsafe.Pointer
	Addrhex  string
	Offaddr  voidptr
	Offhex   string
	File     string
	Line     string
	Lineno   int

	Sframe string
}

func BacktraceLines() []string {
	// var buf = make([]byte, 100)
	buf1 := []byte{}
	buf := C.cxmalloc(200)
	nr := C.backtrace(buf, 200/8)
	// println("nr=", nr)
	symarr := C.backtrace_symbols(buf, nr)
	defer C.free(symarr)

	frames := []string{}
	for i := 0; i < nr; i++ {
		symit := (byteptr)(symarr[i])
		symstr := C.GoString(symit)
		frames = append(frames, symstr)
	}

	return frames
}
func line2frame(line string) *Frame {
	frm := &Frame{}
	frm.Sframe = line

	mglname := line.left("+")
	mglname = mglname.right("(")
	frm.Mglname = mglname
	frm.Funcname = mglname

	addrhex := line.right("[")
	addrhex = addrhex.left("]")
	frm.Addrhex = addrhex
	addrint := xstrings.ParseHex(addrhex)
	frm.Funcaddr = addrint

	offhex := line.right("+")
	offhex = offhex.left(")")
	frm.Offhex = offhex
	offint := xstrings.ParseHex(offhex)
	frm.Offaddr = offint

	return frm
}
func lines2frame2(lines []string) []*Frame {
	res := []*Frame{}
	for idx := 0; idx < lines.len(); idx++ {
		line := lines[idx]
		frm := line2frame(line)
		frm.Btdepth = idx
		res = append(res, frm)
	}

	for idx, line := range lines {
	}
	return res
}

// backtrace without file/line
func Backtrace() []*Frame {
	lines := BacktraceLines()
	frms := lines2frame2(lines)
	return frms
}

// backtrace with file/line
func Callers() []*Frame {
	frms := Backtrace()
	for idx := 0; idx < frms.len(); idx++ {
		frm := frms[idx]
		file, lineno := addr2line1(frm.Funcaddr)
		frm.File = file
		frm.Lineno = lineno
		frm.Line = lineno.repr()
	}
	return frms
}
