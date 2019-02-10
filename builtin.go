package main

import (
	ir "github.com/llir/llvm/ir"
	irtypes "github.com/llir/llvm/ir/types"
)

type builtins struct {
	printf, malloc, memcpy, strncmp, write *ir.Func
}

func (b *builtins) Printf(t *translator) *ir.Func {
	if b.printf == nil {
		b.printf = t.m.NewFunc("printf",
			irtypes.Void,
			ir.NewParam("fmt", irtypes.I8Ptr),
		)
		b.printf.Sig.Variadic = true
	}
	return b.printf
}
func (b *builtins) Malloc(t *translator) *ir.Func {
	if b.malloc == nil {
		b.malloc = t.m.NewFunc(
			"malloc",
			irtypes.I8Ptr,
			ir.NewParam("size", irtypes.I64),
		)
	}
	return b.malloc
}
func (b *builtins) Memcpy(t *translator) *ir.Func {
	if b.memcpy == nil {
		b.memcpy = t.m.NewFunc(
			"memcpy",
			irtypes.Void,
			ir.NewParam("dst", irtypes.I8Ptr),
			ir.NewParam("src", irtypes.I8Ptr),
			ir.NewParam("n", irtypes.I64),
		)
	}
	return b.memcpy
}
func (b *builtins) Strncmp(t *translator) *ir.Func {
	if b.strncmp == nil {
		b.strncmp = t.m.NewFunc(
			"strncmp",
			irtypes.I32,
			ir.NewParam("x", irtypes.I8Ptr),
			ir.NewParam("y", irtypes.I8Ptr),
			ir.NewParam("n", irtypes.I64),
		)
	}
	return b.strncmp
}
func (b *builtins) Write(t *translator) *ir.Func {
	if b.write == nil {
		b.write = t.m.NewFunc("write",
			irtypes.Void,
			ir.NewParam("fd", irtypes.I32),
			ir.NewParam("buf", irtypes.I8Ptr),
			ir.NewParam("count", irtypes.I64),
		)
	}
	return b.write
}
