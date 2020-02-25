package xnet

type Listener interface {
	Accept() (int, error)
}

type Conn interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
}
