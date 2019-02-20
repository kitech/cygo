package main

import (
	"fmt"
	goconstant "go/constant"

	"golang.org/x/tools/go/ssa"

	ir "github.com/llir/llvm/ir"
	irconstant "github.com/llir/llvm/ir/constant"
	irtypes "github.com/llir/llvm/ir/types"
	irvalue "github.com/llir/llvm/ir/value"
)

func (t *translator) emitCallBuiltinPrintln(
	irBlock *ir.Block,
	c *ssa.Call,
) {
	newLine := t.constantString(irBlock, "\n")
	space := t.constantString(irBlock, " ")
	irStderr := irconstant.NewInt(irtypes.I32, 2)

	goArgs := c.Call.Args
	for i, goArg := range goArgs {
		if i != 0 {
			irBlock.NewCall(t.builtins.Printf(t), irStderr, space)
		}

		if isString(goArg.Type()) {
			// Strings use write() so that nul bytes can be written.
			t.emitWriteString(irBlock, goArg)
			continue
		}
		if isSlice(goArg.Type()) {
			irSlice := t.translateValue(irBlock, goArg)
			irPtr := irBlock.NewExtractValue(irSlice, 0)
			irLen := irBlock.NewExtractValue(irSlice, 1)
			irCap := irBlock.NewExtractValue(irSlice, 2)

			irBlock.NewCall(
				t.builtins.Printf(t),
				irStderr,
				t.constantString(irBlock, "[%d/%d]%p"),
				irLen, irCap, irPtr,
			)
			continue
		}
		if isComplex(goArg.Type()) {
			irComplex := t.translateValue(irBlock, goArg)
			irReal := irBlock.NewExtractValue(irComplex, 0)
			irImag := irBlock.NewExtractValue(irComplex, 1)

			irBlock.NewCall(
				t.builtins.Printf(t),
				irStderr,
				t.constantString(irBlock, "(%f+%fi)"),
				irReal, irImag,
			)
			continue
		}

		fmt, val := t.makePrintArg(irBlock, goArg)
		irBlock.NewCall(t.builtins.Printf(t), irStderr, fmt, val)
	}
	irBlock.NewCall(t.builtins.Printf(t), irStderr, newLine)
}

func (t *translator) emitWriteString(
	irBlock *ir.Block,
	goArg ssa.Value,
) {
	irStr := t.translateValue(irBlock, goArg)
	irStrPtr := irBlock.NewExtractValue(irStr, 0)
	irStrLen := irBlock.NewExtractValue(irStr, 1)

	irStderr := irconstant.NewInt(irtypes.I32, 2)

	irBlock.NewCall(t.builtins.Write(t), irStderr, irStrPtr, irStrLen)
}

func (t *translator) makePrintArg(
	irBlock *ir.Block,
	goArg ssa.Value,
) (
	fmtStr, val irvalue.Value,
) {
	goType := goArg.Type()
	switch {
	case isInteger(goType):
		fmt := "d"
		if !isSigned(goType) {
			fmt = "u"
		}
		if sizeof(goType) == 8 {
			fmt = "ll" + fmt
		}

		fmtStr = t.constantString(irBlock, "%"+fmt)

	case isFloat(goType):
		fmtStr = t.constantString(irBlock, "%+e")

	case isString(goType):
		fmtStr = t.constantString(irBlock, "%s")

	case isBool(goType):
		fmtStr = t.constantString(irBlock, "%s")

		strTrue := t.constantString(irBlock, "true")
		strFalse := t.constantString(irBlock, "false")

		val = irBlock.NewSelect(
			t.translateValue(irBlock, goArg),
			strTrue,
			strFalse,
		)

		return fmtStr, val

	case isPointer(goType):
		fmtStr = t.constantString(irBlock, "%p")

	case isInterface(goType):
		fmtStr = t.constantString(irBlock, "%p")

		val = irBlock.NewExtractValue(
			t.translateValue(irBlock, goArg),
			0,
		)

	default:
		panic(fmt.Errorf("makePrintArg: unknown type: %T: %v: %v", goType, goType, goArg))
	}

	switch goArg := goArg.(type) {
	case *ssa.Const:
		switch {
		case isString(goArg.Type()):
			val = t.constantString(irBlock, goconstant.StringVal(goArg.Value))
			return fmtStr, val
		}
	}

	val = t.translateValue(irBlock, goArg)
	return fmtStr, val
}
