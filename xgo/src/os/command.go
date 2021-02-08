package os

/*
#include <errno.h>
*/
import "C"

type Cmd struct {
	cmd  string
	args []string
}

func Command(cmd string, args ...string) *Cmd {
	cmdo := &Cmd{}
	cmdo.cmd = cmd
	cmdo.args = args
	return cmdo
}

func (cmd *Cmd) Run() error {
	return nil
}

func (cmd *Cmd) Output() error {
	return nil
}

// err and out
func (cmd *Cmd) Errout() error {
	return nil
}

func Lookup(exename string) (string, error) {
	paths := Paths()
	for _, dir := range paths {
		exepath := dir + PathSep + exename
		if FileExist(exepath) {
			return exepath, nil
		}
	}

	return "", newoserr(C.ENOENT)
}
