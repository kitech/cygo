package xasm

const (
	MOVL = iota + 1
	ADDL
	SUBL

	IMULL
	XADDL

	MOV
	SUB
	ADD

	LOCK

	EAX
	EBX
	ECX
	EDX
)

func keep() {}
