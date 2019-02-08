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
		fmtStr = t.constantString(irBlock, "%d")

	case isString(goType):
		fmtStr = t.constantString(irBlock, "%s")

	case isBool(goType):
		fmtStr = t.constantString(irBlock, "%s")
		strTrue := t.constantString(irBlock, "true")
		strFalse := t.constantString(irBlock, "false")
		val = irBlock.NewSelect(t.translateValue(irBlock, goArg), strTrue, strFalse)
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

	// msg := "makePrintArg: unimplemented goArg type: %T: %v"
	// panic(fmt.Errorf(msg, goArg, goArg.Type()))
	// fmtStr := t.constantString("")
	// switch goType := goType.(type)
}

// constantString makes a global array representing the string s.
func (t *translator) constantString(irBlock *ir.Block, s string) irvalue.Value {
	if c, ok := t.constantStrings[s]; ok {
		return c
	}

	irConstantS := irconstant.NewCharArrayFromString(s + "\x00")
	irGlobal := t.m.NewGlobalDef("$const_str_"+s, irConstantS)
	irC := irBlock.NewBitCast(irGlobal, irtypes.I8Ptr)
	t.constantStrings[s] = irC
	return irC
}
