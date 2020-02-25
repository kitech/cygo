package xos

/*
#include <errno.h>
#include <string.h>
*/
import "C"

const (
	COK = 0
)

func Errno() int {
	// because cgo cannot refer to errno directly, so use another way
	return *C.__errno_location()
}

func Errmsg() string {
	eno := Errno()
	emsg := C.strerror(eno)
	var emsg2 charptr = emsg
	return string(emsg2)
}

func Errmsgof(eno int) string {
	emsg := C.strerror(eno)
	var emsg2 charptr = emsg
	return string(emsg2)
}

type oserror struct {
	eno  int
	emsg string
}

func newoserr1() *oserror {
	eno := Errno()
	err := &oserror{}
	err.eno = eno
	return err
}

func newoserr(eno int) *oserror {
	err := &oserror{}
	err.eno = eno
	return err
}

func (err *oserror) Error() string {
	var emsg string
	if err == nil || err.eno == COK {
		return emsg
	}
	if err.emsg.len == 0 {
		emsg = "OSErr " + err.eno.repr() + ": " + Errmsgof(err.eno)
		err.emsg = emsg
	} else {
		emsg = err.emsg
	}
	return emsg
}
