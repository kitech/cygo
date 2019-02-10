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
	if t.builtinPrintln == nil {
		pl := t.m.NewFunc("printf", irtypes.Void, ir.NewParam("fmt", irtypes.I8Ptr))
		pl.Sig.Variadic = true
		t.builtinPrintln = pl
	}

	newLine := t.constantString(irBlock, "\n")
	space := t.constantString(irBlock, " ")

	goArgs := c.Call.Args
	for i, goArg := range goArgs {
		if i != 0 {
			irBlock.NewCall(t.builtinPrintln, space)
		}
		fmt, val := t.makePrintArg(irBlock, goArg)
		irBlock.NewCall(t.builtinPrintln, fmt, val)
	}
	irBlock.NewCall(t.builtinPrintln, newLine)
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
