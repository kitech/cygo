package main

import (
	"fmt"
	goconstant "go/constant"
	"strings"

	"golang.org/x/tools/go/ssa"

	ir "github.com/llir/llvm/ir"
	irconstant "github.com/llir/llvm/ir/constant"
	irenum "github.com/llir/llvm/ir/enum"
	irtypes "github.com/llir/llvm/ir/types"
	irvalue "github.com/llir/llvm/ir/value"
)

// TODO(pwaller): Should not require irBlock parameter. I had it here because I didn't know how to construct constant strings, but now I've figured that out.

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
		irLen := irconstant.NewInt(irtypes.I64, int64(len(s)))
		stty := irtypes.NewStruct(irtypes.I8Ptr, irtypes.I64)
		return irconstant.NewStruct(stty, irConstantS, irLen)

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

	case isComplex(goConstType):
		irFloatTyp := t.goToIRType(goConstType).(*irtypes.StructType).Fields[0].(*irtypes.FloatType)

		c := goConst.Complex128()

		stty := irtypes.NewStruct(irtypes.Float, irtypes.Float)
		return irconstant.NewStruct(stty,
			irconstant.NewFloat(irFloatTyp, real(c)),
			irconstant.NewFloat(irFloatTyp, imag(c)),
		)

	case isUnsafePointer(goConstType):
		irInt := irconstant.NewInt(irtypes.I64, goConst.Int64())
		return irconstant.NewIntToPtr(irInt, t.goToIRType(goConstType))

	default:
		msg := "unimplemented constant: %v: %s"
		panic(fmt.Sprintf(msg, goConst.Value.Kind(), goConst))
	}
}

// constantString makes a global array representing the string s.
func (t *translator) constantString(irBlock *ir.Block, s string) irconstant.Constant {
	irC, ok := t.constantStrings[s]
	if ok {
		return irC
	}

	nul := string([]byte{0}) // Constant string "\x00" is a problem

	irConstantS := irconstant.NewCharArrayFromString(s + nul)

	// TODO(pwaller): Ensure non-colliding names (foo_ and foo\x00).
	// This makes having the constant string "\x00" in the program a problem.
	constName := strings.ReplaceAll(s, nul, "<nul>")

	irGlobal := t.m.NewGlobalDef("$const_str_"+constName, irConstantS)
	irGlobal.Immutable = true
	irGlobal.Linkage = irenum.LinkagePrivate

	irZero := irconstant.NewInt(irtypes.I32, 0)
	irC = irconstant.NewGetElementPtr(irGlobal, irZero, irZero)
	t.constantStrings[s] = irC
	return irC
}
