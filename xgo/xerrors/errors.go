package xerrors

type ezerror struct {
	s      string
	stacks []string
}

func (err *ezerror) Error() string {
	return err.s
}

func (err *ezerror) Stacks() []string {
	return err.stacks
}

func New(s string) error {
	var err error
	err = &ezerror{s}
	return err
}

func Wrap(err error, s string) error {
	var olderrpp **ezerror = &err
	var olderrp = *olderrpp

	olds := err.Error()
	var nerr error
	err1 := &ezerror{}
	err1.s = s
	err1.stacks = olderrp.stacks
	err1.stacks = append(err1.stacks, olds)
	nerr = err1
	return nerr
}

func Errorf(format string, args ...interface{}) error {
	return nil
}

func Keep() {

}
