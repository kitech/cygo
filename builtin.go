package main

import (
	ir "github.com/llir/llvm/ir"
	irconstant "github.com/llir/llvm/ir/constant"
	irenum "github.com/llir/llvm/ir/enum"
	irtypes "github.com/llir/llvm/ir/types"
	irvalue "github.com/llir/llvm/ir/value"
)

type builtins struct {
	append,
	exit,
	malloc,
	memcpy,
	printf,
	strncmp,
	write *ir.Func
}

func (b *builtins) Exit(t *translator) *ir.Func {
	if b.exit == nil {
		b.exit = t.m.NewFunc("_exit",
			irtypes.Void,
			ir.NewParam("status", irtypes.I32),
		)
	}
	return b.exit
}

func (b *builtins) Printf(t *translator) *ir.Func {
	if b.printf == nil {
		b.printf = t.m.NewFunc("dprintf",
			irtypes.Void,
			ir.NewParam("fd", irtypes.I32),
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

		b.malloc.ReturnAttrs = append(b.malloc.ReturnAttrs, irenum.ReturnAttrNoAlias)
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

func (b *builtins) Append(t *translator) *ir.Func {
	if b.append == nil {
		// TODO(pwaller): Different word than 'more'.
		// TODO(pwaller): More consistent naming convention (more first, or ptr/len first?)

		irPtr := ir.NewParam("ptr", irtypes.I8Ptr)
		irLen := ir.NewParam("len", irtypes.I64)
		irCap := ir.NewParam("cap", irtypes.I64)
		irMorePtr := ir.NewParam("moreptr", irtypes.I8Ptr)
		irMoreLen := ir.NewParam("morelen", irtypes.I64)
		irElemSize := ir.NewParam("elemsize", irtypes.I64)

		b.append = t.m.NewFunc(
			"append",
			irtypes.NewStruct(irtypes.I8Ptr, irtypes.I64, irtypes.I64),
			irPtr, irLen, irCap, irMorePtr, irMoreLen, irElemSize,
		)

		entry := b.append.NewBlock("entry")
		doInsert := b.append.NewBlock("doInsert")
		doResize := b.append.NewBlock("doResize")

		irNewLen := entry.NewAdd(irLen, irMoreLen)
		needResize := entry.NewICmp(irenum.IPredUGE, irNewLen, irCap)

		entry.NewCondBr(needResize, doResize, doInsert)

		// Insert path.
		{
			blk := doInsert

			// Copy appended elements.
			irPtrStart := blk.NewIntToPtr(
				// irPtr + offset
				blk.NewAdd(
					blk.NewPtrToInt(irPtr, irtypes.I64),
					blk.NewMul(irLen, irElemSize), // offset
				),
				irtypes.I8Ptr,
			)
			irExtraSize := blk.NewMul(irMoreLen, irElemSize)
			blk.NewCall(b.Memcpy(t), irPtrStart, irMorePtr, irExtraSize)

			// Return new slice value.
			irNewPtr := irPtr
			irNewCap := irCap
			irNewSlice := makeStruct(doInsert, irNewPtr, irNewLen, irNewCap)
			blk.NewRet(irNewSlice)
		}

		// Resize path.
		{
			blk := doResize

			irNewCap := irNewLen // TODO(pwaller): Better growth function.
			irNewCapSize := blk.NewMul(irNewCap, irElemSize)

			// Allocate new capacity.
			irNewPtr := blk.NewCall(b.Malloc(t), irNewCapSize)

			// Copy the original slice data into newly alloc'd memory.
			irOrigSize := blk.NewMul(irLen, irElemSize)
			blk.NewCall(b.Memcpy(t), irNewPtr, irPtr, irOrigSize)

			// Copy appended elements.
			irNewPtrMore := blk.NewIntToPtr(
				// irNewPtr + offset
				blk.NewAdd(
					blk.NewPtrToInt(irNewPtr, irtypes.I64),
					blk.NewMul(irLen, irElemSize), // offset
				),
				irtypes.I8Ptr,
			)
			irExtraSize := blk.NewMul(irMoreLen, irElemSize)
			blk.NewCall(b.Memcpy(t), irNewPtrMore, irMorePtr, irExtraSize)

			// Return new slice value.
			irNewSlice := makeStruct(blk, irNewPtr, irNewLen, irNewCap)
			blk.NewRet(irNewSlice)
		}
	}
	return b.append
}

func makeStruct(irBlock *ir.Block, values ...irvalue.Value) (ret irvalue.Value) {
	var types []irtypes.Type
	for _, v := range values {
		types = append(types, v.Type())
	}
	ret = irconstant.NewUndef(irtypes.NewStruct(types...))
	for i, v := range values {
		ret = irBlock.NewInsertValue(ret, v, uint64(i))
	}
	return ret
}
