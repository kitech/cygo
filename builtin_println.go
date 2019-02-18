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

	goArgs := c.Call.Args
	for i, goArg := range goArgs {
		if i != 0 {
			irBlock.NewCall(t.builtins.Printf(t), space)
		}

		if isString(goArg.Type()) {
			// Strings use write() so that nul bytes can be written.
			t.emitWriteString(irBlock, goArg)
			continue
		}

		fmt, val := t.makePrintArg(irBlock, goArg)
		irBlock.NewCall(t.builtins.Printf(t), fmt, val)
	}
	irBlock.NewCall(t.builtins.Printf(t), newLine)
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
