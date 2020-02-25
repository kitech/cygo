package xos

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

func Lookup(p string) (string, error) {
	return "", nil
}
