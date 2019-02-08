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

func (t *translator) goConstToIR(
	irBlock *ir.Block,
	goConst *ssa.Const,
) irvalue.Value {
	if goConst.Value == nil {
		// Constant zero.
		switch t := t.goToIRType(goConst.Type()).(type) {
		case *irtypes.StructType:
			return irconstant.NewZeroInitializer(t)
		case *irtypes.PointerType:
			return irconstant.NewNull(t)
		}
	}

	constAsStr := goConst.Value.ExactString()
	goConstType := goConst.Type()

	switch {
	case isBool(goConstType):
		irConst, _ := irconstant.NewIntFromString(irtypes.I1, constAsStr)
		return irConst

	case isString(goConstType):
		s := goconstant.StringVal(goConst.Value)
		irConstantS := t.constantString(irBlock, s)

		strT := irtypes.NewStruct(irtypes.I8Ptr, irtypes.I64)
		irLen := irconstant.NewInt(irtypes.I64, int64(len(s)))

		var v irvalue.Value = irconstant.NewZeroInitializer(strT)
		v = irBlock.NewInsertValue(v, irConstantS, 0)
		v = irBlock.NewInsertValue(v, irLen, 1)
		return v

	case isInteger(goConstType):
		irTyp := t.goToIRType(goConstType).(*irtypes.IntType)
		irConst, err := irconstant.NewIntFromString(irTyp, constAsStr)
		if err != nil {
			panic(err)
		}
		return irConst

	case isFloat(goConstType):
		irTyp := t.goToIRType(goConstType).(*irtypes.FloatType)
		return irconstant.NewFloat(irTyp, goConst.Float64())

	default:
		msg := "unimplemented constant: %v: %s"
		panic(fmt.Sprintf(msg, goConst.Value.Kind(), goConst))
	}
}
