package main

import (
	"fmt"
	"go/token"
	"go/types"
	gotypes "go/types"
	"log"
	"math/big"
	"runtime"

	"golang.org/x/tools/go/ssa"

	ir "github.com/llir/llvm/ir"
	irconstant "github.com/llir/llvm/ir/constant"
	irenum "github.com/llir/llvm/ir/enum"
	irtypes "github.com/llir/llvm/ir/types"
	irvalue "github.com/llir/llvm/ir/value"
)

func (t *translator) emitInstr(irBlock *ir.Block, goInst ssa.Instruction) {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		p := t.prog.Fset.Position(goInst.Pos())

		log.Printf("panic while processing %s at %s", goInst, p)
		for i, op := range goInst.Operands(nil) {
			irOp := t.goToIRValue[*op]
			log.Printf("  operand %d: %T: %v: %v -- %T: %v", i, *op, (*op).Type(), *op, irOp, irOp)
		}

		panic(err)
	}()

	switch goInst := goInst.(type) {
	case *ssa.Alloc:
		t.emitAlloc(irBlock, goInst)
	case *ssa.BinOp:
		t.emitBinOp(irBlock, goInst)
	case *ssa.Call:
		t.emitCall(irBlock, goInst)
	case *ssa.ChangeInterface:
		t.emitChangeInterface(irBlock, goInst)
	case *ssa.ChangeType:
		t.emitChangeType(irBlock, goInst)
	case *ssa.Convert:
		t.emitConvert(irBlock, goInst)
	case *ssa.DebugRef:
		t.emitDebugRef(irBlock, goInst)
	case *ssa.Defer:
		t.emitDefer(irBlock, goInst)
	case *ssa.Extract:
		t.emitExtract(irBlock, goInst)
	case *ssa.Field:
		t.emitField(irBlock, goInst)
	case *ssa.FieldAddr:
		t.emitFieldAddr(irBlock, goInst)
	case *ssa.Go:
		t.emitGo(irBlock, goInst)
	case *ssa.If:
		t.emitIf(irBlock, goInst)
	case *ssa.Index:
		t.emitIndex(irBlock, goInst)
	case *ssa.IndexAddr:
		t.emitIndexAddr(irBlock, goInst)
	case *ssa.Jump:
		t.emitJump(irBlock, goInst)
	case *ssa.Lookup:
		t.emitLookup(irBlock, goInst)
	case *ssa.MakeChan:
		t.emitMakeChan(irBlock, goInst)
	case *ssa.MakeClosure:
		t.emitMakeClosure(irBlock, goInst)
	case *ssa.MakeInterface:
		t.emitMakeInterface(irBlock, goInst)
	case *ssa.MakeMap:
		t.emitMakeMap(irBlock, goInst)
	case *ssa.MakeSlice:
		t.emitMakeSlice(irBlock, goInst)
	case *ssa.MapUpdate:
		t.emitMapUpdate(irBlock, goInst)
	case *ssa.Next:
		t.emitNext(irBlock, goInst)
	case *ssa.Panic:
		t.emitPanic(irBlock, goInst)
	case *ssa.Phi:
		t.emitPhi(irBlock, goInst)
	case *ssa.Range:
		t.emitRange(irBlock, goInst)
	case *ssa.Return:
		t.emitReturn(irBlock, goInst)
	case *ssa.RunDefers:
		t.emitRunDefers(irBlock, goInst)
	case *ssa.Select:
		t.emitSelect(irBlock, goInst)
	case *ssa.Send:
		t.emitSend(irBlock, goInst)
	case *ssa.Slice:
		t.emitSlice(irBlock, goInst)
	case *ssa.Store:
		t.emitStore(irBlock, goInst)
	case *ssa.TypeAssert:
		t.emitTypeAssert(irBlock, goInst)
	case *ssa.UnOp:
		t.emitUnOp(irBlock, goInst)
	default:
		panic(fmt.Errorf("unimplemented: goInst: %T: %v", goInst, goInst))
	}
}

func (t *translator) emitAlloc(irBlock *ir.Block, a *ssa.Alloc) {
	if a.Heap {
		t.emitAllocHeap(irBlock, a)
		return
	}

	goAllocedType := a.Type().Underlying().(*types.Pointer).Elem()
	irAllocedType := t.goToIRType(goAllocedType)
	irAlloca := irBlock.NewAlloca(irAllocedType)
	irBlock.NewStore(irconstant.NewZeroInitializer(irAllocedType), irAlloca)
	t.goToIRValue[a] = irAlloca
}

func (t *translator) emitAllocHeap(irBlock *ir.Block, a *ssa.Alloc) {
	goElemType := a.Type().Underlying().(*gotypes.Pointer).Elem()
	sz := sizeof(goElemType)
	irVoidPtr := irBlock.NewCall(
		t.builtins.Malloc(t),
		irconstant.NewInt(irtypes.I64, sz),
	)
	irPtr := irBlock.NewBitCast(irVoidPtr, t.goToIRType(a.Type()))
	t.goToIRValue[a] = irPtr

	irElemZero := irconstant.NewZeroInitializer(t.goToIRType(goElemType))
	irBlock.NewStore(irElemZero, irPtr)
}

func (t *translator) emitBinOp(irBlock *ir.Block, b *ssa.BinOp) {
	if !gotypes.Identical(b.X.Type(), b.Y.Type()) && // Types must be identical
		!((b.Op == token.SHL || b.Op == token.SHR) && // Or it's a shift and the right operand is unsigned.
			!isSigned(b.Y.Type().Underlying())) {
		panic(fmt.Errorf("unmatched types in BinOp: %v != %v in %v",
			b.X.Type(), b.Y.Type(), b))
	}

	goParamType := b.X.Type()

	switch b.Op {
	case token.EQL, token.NEQ, token.GEQ, token.LEQ, token.GTR, token.LSS:
		t.emitBinOpCmp(irBlock, goParamType, b)

	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
		t.emitBinOpArith(irBlock, goParamType, b)

	case token.AND, token.AND_NOT, token.OR, token.XOR, token.SHL, token.SHR:
		t.emitBinOpLogical(irBlock, goParamType, b)

	default:
		panic(fmt.Errorf("unimplemented: emitBinOp: %v", b.Op))
	}
}

func (t *translator) emitBinOpLogical(
	irBlock *ir.Block,
	goParamType gotypes.Type,
	b *ssa.BinOp,
) {
	irX := t.translateValue(irBlock, b.X)
	irY := t.translateValue(irBlock, b.Y)

	switch b.Op {
	case token.AND:
		t.goToIRValue[b] = irBlock.NewAnd(irX, irY)
	case token.AND_NOT:
		// TODO(pwaller): Correctness test
		irOnes := irConstantOnes(int(sizeof(goParamType) * 8))
		irNotY := irBlock.NewXor(irY, irOnes)
		t.goToIRValue[b] = irBlock.NewAnd(irX, irNotY)
	case token.OR:
		t.goToIRValue[b] = irBlock.NewOr(irX, irY)
	case token.XOR:
		t.goToIRValue[b] = irBlock.NewXor(irX, irY)
	case token.SHL:
		if sizeof(b.X.Type()) > sizeof(b.Y.Type()) {
			irY = irBlock.NewZExt(irY, irX.Type())
		} else if sizeof(b.X.Type()) < sizeof(b.Y.Type()) {
			irY = irBlock.NewTrunc(irY, irX.Type())
		}
		t.goToIRValue[b] = irBlock.NewShl(irX, irY)
	case token.SHR:
		if sizeof(b.X.Type()) > sizeof(b.Y.Type()) {
			irY = irBlock.NewZExt(irY, irX.Type())
		} else if sizeof(b.X.Type()) < sizeof(b.Y.Type()) {
			irY = irBlock.NewTrunc(irY, irX.Type())
		}
		if isSigned(goParamType) {
			t.goToIRValue[b] = irBlock.NewAShr(irX, irY)
			return
		}
		t.goToIRValue[b] = irBlock.NewLShr(irX, irY)
	default:
		panic(fmt.Errorf("emitBinOpArith: unknown op %q", b.Op))
	}
}

func (t *translator) emitBinOpArith(
	irBlock *ir.Block,
	goParamType gotypes.Type,
	b *ssa.BinOp,
) {
	if isInteger(goParamType) {
		t.emitBinOpArithInt(irBlock, goParamType, b)
		return
	}

	if isFloat(goParamType) {
		t.emitBinOpArithFloat(irBlock, goParamType, b)
		return
	}

	if isString(goParamType) {
		if b.Op != token.ADD {
			panic(fmt.Errorf("emitBinOpArith: unknown op %q on strings", b.Op))
		}

		irX := t.translateValue(irBlock, b.X)
		irY := t.translateValue(irBlock, b.Y)

		irXPtr := irBlock.NewExtractValue(irX, 0)
		irXLen := irBlock.NewExtractValue(irX, 1)
		irYPtr := irBlock.NewExtractValue(irY, 0)
		irYLen := irBlock.NewExtractValue(irY, 1)

		irNewLen := irBlock.NewAdd(irXLen, irYLen)
		irNewPtr := irBlock.NewCall(t.builtins.Malloc(t), irNewLen)
		irNewPtrAsInt := irBlock.NewPtrToInt(irNewPtr, irtypes.I64)

		irNewPtrYOff := irBlock.NewIntToPtr(
			irBlock.NewAdd(irNewPtrAsInt, irXLen), irtypes.I8Ptr,
		)

		irBlock.NewCall(t.builtins.Memcpy(t), irNewPtr, irXPtr, irXLen, irconstant.False)
		irBlock.NewCall(t.builtins.Memcpy(t), irNewPtrYOff, irYPtr, irYLen, irconstant.False)

		var irSum irvalue.Value = irconstant.NewUndef(t.goToIRType(b.Type()))
		irSum = irBlock.NewInsertValue(irSum, irNewPtr, 0)
		irSum = irBlock.NewInsertValue(irSum, irNewLen, 1)

		t.goToIRValue[b] = irSum
		return
	}

	panic(fmt.Errorf("emitBinOpArith: unknown op %q", b.Op))
}

func (t *translator) emitBinOpArithFloat(
	irBlock *ir.Block,
	goParamType gotypes.Type,
	b *ssa.BinOp,
) {
	irX := t.translateValue(irBlock, b.X)
	irY := t.translateValue(irBlock, b.Y)

	switch b.Op {
	case token.ADD:
		t.goToIRValue[b] = irBlock.NewFAdd(irX, irY)
	case token.SUB:
		t.goToIRValue[b] = irBlock.NewFSub(irX, irY)
	case token.MUL:
		t.goToIRValue[b] = irBlock.NewFMul(irX, irY)
	case token.QUO:
		t.goToIRValue[b] = irBlock.NewFDiv(irX, irY)
	case token.REM:
		t.goToIRValue[b] = irBlock.NewFRem(irX, irY)

	default:
		panic(fmt.Errorf("emitBinOpArithFloat: unknown op %q", b.Op))
	}
}

func (t *translator) emitBinOpArithInt(
	irBlock *ir.Block,
	goParamType gotypes.Type,
	b *ssa.BinOp,
) {
	irX := t.translateValue(irBlock, b.X)
	irY := t.translateValue(irBlock, b.Y)

	switch b.Op {
	case token.ADD:
		t.goToIRValue[b] = irBlock.NewAdd(irX, irY)
	case token.SUB:
		t.goToIRValue[b] = irBlock.NewSub(irX, irY)
	case token.MUL:
		t.goToIRValue[b] = irBlock.NewMul(irX, irY)
	case token.QUO:
		if isSigned(goParamType) {
			t.goToIRValue[b] = irBlock.NewSDiv(irX, irY)
			return
		}
		t.goToIRValue[b] = irBlock.NewUDiv(irX, irY)
	case token.REM:
		if isSigned(goParamType) {
			t.goToIRValue[b] = irBlock.NewSRem(irX, irY)
			return
		}
		t.goToIRValue[b] = irBlock.NewURem(irX, irY)

	default:
		panic(fmt.Errorf("emitBinOpArithInt: unknown op %q", b.Op))
	}
}

type signedness int

func getSigned(goType gotypes.Type) signedness {
	if isSigned(goType) {
		return signed
	}
	return unsigned
}

const (
	signed signedness = iota
	unsigned
)

var (
	goSignedToOpToIPred = map[signedness]map[token.Token]irenum.IPred{
		signed: {
			token.EQL: irenum.IPredEQ,
			token.NEQ: irenum.IPredNE,
			token.GTR: irenum.IPredSGT,
			token.LSS: irenum.IPredSLT,
			token.GEQ: irenum.IPredSGE,
			token.LEQ: irenum.IPredSLE,
		},
		unsigned: {
			token.EQL: irenum.IPredEQ,
			token.NEQ: irenum.IPredNE,
			token.GTR: irenum.IPredUGT,
			token.LSS: irenum.IPredULT,
			token.GEQ: irenum.IPredUGE,
			token.LEQ: irenum.IPredULE,
		},
	}
	goOpToFPred = map[token.Token]irenum.FPred{
		token.EQL: irenum.FPredOEQ,
		token.NEQ: irenum.FPredONE,
		token.GTR: irenum.FPredOGT,
		token.LSS: irenum.FPredOLT,
		token.GEQ: irenum.FPredOGE,
		token.LEQ: irenum.FPredOLE,
	}
)

func (t *translator) emitBinOpCmp(
	irBlock *ir.Block,
	goParamType gotypes.Type,
	b *ssa.BinOp,
) {
	irX := t.translateValue(irBlock, b.X)
	irY := t.translateValue(irBlock, b.Y)

	switch {
	case isInteger(goParamType) || isBool(goParamType):
		irPred, ok := goSignedToOpToIPred[getSigned(goParamType)][b.Op]
		if !ok {
			panic(fmt.Errorf("emitBinOpCmp/isInteger: no such op: %q", b.Op))
		}
		t.goToIRValue[b] = irBlock.NewICmp(irPred, irX, irY)

	case isString(goParamType):
		t.emitBinOpCmpStr(irBlock, b)
		return

	case isFloat(goParamType):
		irPred, ok := goOpToFPred[b.Op]
		if !ok {
			panic(fmt.Errorf("emitBinOpCmp/isInteger: no such op: %q", b.Op))
		}
		t.goToIRValue[b] = irBlock.NewFCmp(irPred, irX, irY)

	// TODO(pwaller): Test with "nil != interface"...
	case isInterface(goParamType):
		// Can only compare interface with equality or non-equality.
		if b.Op != token.EQL && b.Op != token.NEQ {
			panic(fmt.Errorf("unimplemented: emitBinOpCmp: %v", goParamType))
		}

		iPred := irenum.IPredEQ
		if b.Op == token.NEQ {
			iPred = irenum.IPredNE
		}

		irXIfaceData := irBlock.NewExtractValue(irX, 1)
		irYIfaceData := irBlock.NewExtractValue(irY, 1)
		irXInt := irBlock.NewPtrToInt(irXIfaceData, irtypes.I64)
		irYInt := irBlock.NewPtrToInt(irYIfaceData, irtypes.I64)

		t.goToIRValue[b] = irBlock.NewICmp(iPred, irXInt, irYInt)

	case isPointer(goParamType) || isChan(goParamType):
		iPred := irenum.IPredEQ
		if b.Op == token.NEQ {
			iPred = irenum.IPredNE
		}

		// TODO(pwaller): Pointer comparisons are actually allowed, no need to convert to int.
		irXInt := irBlock.NewPtrToInt(irX, irtypes.I64)
		irYInt := irBlock.NewPtrToInt(irY, irtypes.I64)

		t.goToIRValue[b] = irBlock.NewICmp(iPred, irXInt, irYInt)

	case isStruct(goParamType):
		log.Printf("unimplemented: emitBinOpCmp: struct: %T %v", goParamType, goParamType)
		t.doTrap(irBlock)
		t.goToIRValue[b] = irconstant.NewUndef(irtypes.I1)

	case isComplex(goParamType):
		log.Printf("unimplemented: complex comparison")
		t.doTrap(irBlock)
		t.goToIRValue[b] = irconstant.NewUndef(irtypes.I1)

	case isSignature(goParamType):
		log.Printf("unimplemented: signature comparison")
		t.doTrap(irBlock)
		t.goToIRValue[b] = irconstant.NewUndef(irtypes.I1)

	case isSlice(goParamType):
		log.Printf("unimplemented: slice comparison")
		t.doTrap(irBlock)
		t.goToIRValue[b] = irconstant.NewUndef(irtypes.I1)

	case isArray(goParamType):
		log.Printf("unimplemented: array comparison")
		t.doTrap(irBlock)
		t.goToIRValue[b] = irconstant.NewUndef(irtypes.I1)

	case isMap(goParamType):
		log.Printf("unimplemented: map comparison")
		t.doTrap(irBlock)
		t.goToIRValue[b] = irconstant.NewUndef(irtypes.I1)

	default:
		msg := "unimplemented: emitBinOpCmp: %T: %v"
		panic(fmt.Errorf(msg, goParamType, goParamType))
	}
}

func (t *translator) emitBinOpCmpStr(
	irBlock *ir.Block,
	b *ssa.BinOp,
) {
	irX := t.translateValue(irBlock, b.X)
	irY := t.translateValue(irBlock, b.Y)

	irXPtr := irBlock.NewExtractValue(irX, 0)
	irYPtr := irBlock.NewExtractValue(irY, 0)
	irXLen := irBlock.NewExtractValue(irX, 1)
	irYLen := irBlock.NewExtractValue(irY, 1)

	// Get the minimum length of the two strings
	irCond := irBlock.NewICmp(irenum.IPredULE, irXLen, irYLen)
	irMinLen := irBlock.NewSelect(irCond, irXLen, irYLen)

	irStrncmp := t.builtins.Strncmp(t)
	irStrNCmpRet := irBlock.NewCall(irStrncmp, irXPtr, irYPtr, irMinLen)

	irConstZero := irconstant.NewInt(irStrncmp.Sig.RetType.(*irtypes.IntType), 0)

	irPred, ok := goSignedToOpToIPred[signed][b.Op]
	if !ok {
		panic(fmt.Errorf("unimplemented: str comparison with %q", b.Op))
	}

	t.goToIRValue[b] = irBlock.NewICmp(irPred, irStrNCmpRet, irConstZero)
}

func (t *translator) emitCall(irBlock *ir.Block, c *ssa.Call) {
	if c.Call.IsInvoke() {
		t.doTrap(irBlock)
		t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
		log.Printf("unimplemented: emitCall & c.Call.IsInvoke(): %v", c)
		return
	}

	goBuiltin, ok := c.Call.Value.(*ssa.Builtin)
	if ok {
		t.emitCallBuiltin(irBlock, goBuiltin, c)
		return
	}

	var irArgs []irvalue.Value
	for _, goArg := range c.Call.Args {
		irArg := t.translateValue(irBlock, goArg)
		irArgs = append(irArgs, irArg)
	}

	switch goCallee := c.Call.Value.(type) {
	case *ssa.Function:
		// Not using translateValue machinary here.
		irCallee := t.goToIRValue[goCallee]
		// irCallee := t.translateValue(irBlock, goCallee)
		t.goToIRValue[c] = irBlock.NewCall(
			irCallee,
			irArgs...,
		)

	case *ssa.MakeClosure:
		t.doTrap(irBlock)
		t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
		log.Println("unimplemented: MakeClosure call")
		// panic(fmt.Errorf("unimplemented: MakeClosure call"))

	default:
		log.Println("unimplemented: non-static call")

		log.Println("unimplemented: non-static call")
		t.doTrap(irBlock)
		t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
		// panic(fmt.Errorf("unimplemented: non-static call"))
	}
}

func (t *translator) emitCallStatic(irBlock *ir.Block, c *ssa.Call, irArgs []irvalue.Value) {

}

func (t *translator) emitCallClosure(irBlock *ir.Block, c *ssa.Call, irArgs []irvalue.Value) {
	irCallee := t.translateValue(irBlock, c.Call.Value)

	irClosure := irBlock.NewExtractValue(irCallee, 1)
	irCallee = irBlock.NewExtractValue(irCallee, 0)
	irArgs = append([]irvalue.Value{irClosure}, irArgs...)
	irCall := irBlock.NewCall(
		irCallee,
		irArgs...,
	)
	t.goToIRValue[c] = irCall
}

func (t *translator) emitCallBuiltin(
	irBlock *ir.Block,
	goBuiltin *ssa.Builtin,
	c *ssa.Call,
) {
	goArgs := c.Call.Args
	switch goBuiltin.Name() {
	case "len":
		if len(goArgs) != 1 {
			panic(fmt.Errorf("len() only accepts one argument, got %d", len(goArgs)))
		}

		goArg := goArgs[0]

		if isMap(goArg.Type()) {
			log.Printf("unimplemented: len(map)")
			t.doTrap(irBlock)
			t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
			return
		}

		irArg := t.translateValue(irBlock, goArg)

		const lenFieldIdx = 1
		irLen := irBlock.NewExtractValue(irArg, lenFieldIdx)
		t.goToIRValue[c] = irLen

	case "cap":
		if len(goArgs) != 1 {
			panic(fmt.Errorf("cap() only accepts one argument, got %d", len(goArgs)))
		}

		goArg := goArgs[0]
		irArg := t.translateValue(irBlock, goArg)

		const capFieldIdx = 2
		irCap := irBlock.NewExtractValue(irArg, capFieldIdx)
		t.goToIRValue[c] = irCap

	case "append":
		t.emitCallBuiltinAppend(irBlock, c)

	case "println", "print":
		t.emitCallBuiltinPrintln(irBlock, c)

	case "copy":
		t.emitCallBuiltinCopy(irBlock, c)

	default:
		// TODO(pwaller): A number of missing builtins.
		log.Printf("unimplemented: emitCallBuiltin: %v", goBuiltin.Name())
		t.doTrap(irBlock)
		t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
		// panic(fmt.Errorf("unimplemented: emitCallBuiltin: %v", goBuiltin.Name()))
	}
}

func (t *translator) emitCallBuiltinCopy(
	irBlock *ir.Block,
	c *ssa.Call,
) {
	// TODO(pwaller): improve element size computation.
	goElemSize := sizeof(c.Call.Args[0].Type().Underlying().(*gotypes.Slice).Elem())
	irElemSize := irconstant.NewInt(irtypes.I64, goElemSize)

	irDst := t.translateValue(irBlock, c.Call.Args[0])
	irSrc := t.translateValue(irBlock, c.Call.Args[1])

	irDstPtr := irBlock.NewExtractValue(irDst, 0)
	irSrcPtr := irBlock.NewExtractValue(irSrc, 0)
	irDstLen := irBlock.NewExtractValue(irDst, 1)
	irSrcLen := irBlock.NewExtractValue(irSrc, 1)

	irLenDstLTSrc := irBlock.NewICmp(irenum.IPredULE, irDstLen, irSrcLen)
	irCopyLen := irBlock.NewSelect(irLenDstLTSrc, irDstLen, irSrcLen)

	irCopySize := irBlock.NewMul(irCopyLen, irElemSize)
	irBlock.NewCall(t.builtins.Memcpy(t),
		irBlock.NewBitCast(irDstPtr, irtypes.I8Ptr),
		irBlock.NewBitCast(irSrcPtr, irtypes.I8Ptr),
		irCopySize,
		irconstant.False,
	)

	t.goToIRValue[c] = irCopyLen
}

func (t *translator) emitCallBuiltinAppend(
	irBlock *ir.Block,
	c *ssa.Call,
) {
	irAppendee := t.translateValue(irBlock, c.Call.Args[0])
	irAppendeePtr := irBlock.NewExtractValue(irAppendee, 0)
	irAppendeeLen := irBlock.NewExtractValue(irAppendee, 1)
	irAppendeeCap := irBlock.NewExtractValue(irAppendee, 2)

	irAppended := t.translateValue(irBlock, c.Call.Args[1])
	irAppendedPtr := irBlock.NewExtractValue(irAppended, 0)
	irAppendedLen := irBlock.NewExtractValue(irAppended, 1)

	// TODO(pwaller): It's probably wrong to use the Go elem size here.
	// But we don't currently have an easy way to compute the llir one.
	// See https://github.com/llir/llvm/issues/66
	goElemSize := sizeof(c.Call.Args[0].Type().Underlying().(*gotypes.Slice).Elem())
	irElemSize := irconstant.NewInt(irtypes.I64, goElemSize)

	irNewGenericSlice := irBlock.NewCall(
		t.builtins.Append(t),
		irBlock.NewBitCast(irAppendeePtr, irtypes.I8Ptr),
		irAppendeeLen,
		irAppendeeCap,
		irBlock.NewBitCast(irAppendedPtr, irtypes.I8Ptr),
		irAppendedLen,
		irElemSize,
	)

	irPtrType := irAppendeePtr.Type()
	t.goToIRValue[c] = makeStruct(
		irBlock,
		irBlock.NewBitCast(irBlock.NewExtractValue(irNewGenericSlice, 0), irPtrType),
		irBlock.NewExtractValue(irNewGenericSlice, 1),
		irBlock.NewExtractValue(irNewGenericSlice, 2),
	)
}

func (t *translator) emitChangeInterface(irBlock *ir.Block, c *ssa.ChangeInterface) {
	log.Printf("unimplemented: emitChangeInterface")
	t.doTrap(irBlock)
	t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
}

func (t *translator) emitChangeType(irBlock *ir.Block, c *ssa.ChangeType) {
	t.goToIRValue[c] = t.translateValue(irBlock, c.X)
}

func (t *translator) emitConvertInt(irBlock *ir.Block, c *ssa.Convert) {
	from, to := c.X.Type(), c.Type()
	sizeFrom, sizeTo := sizeof(from), sizeof(to)
	fromV, toT := t.translateValue(irBlock, c.X), t.goToIRType(to)

	if isFloat(to) {
		if isSigned(from) {
			t.goToIRValue[c] = irBlock.NewSIToFP(fromV, toT)
		} else {
			t.goToIRValue[c] = irBlock.NewUIToFP(fromV, toT)
		}
		return
	}
	if isString(to) {
		t.doTrap(irBlock)
		t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
		log.Printf("unimplemented: emitConvertInt to string")
		return
	}

	if isUnsafePointer(to) {
		t.goToIRValue[c] = irBlock.NewIntToPtr(fromV, toT)
		return
	}

	if !isInteger(to) {
		panic(fmt.Errorf("unimplemented: emitConvertInt: %v <- %v", to, from))
	}

	switch {
	case sizeTo == sizeFrom:
		// passthrough, e.g. int(int64(x)) where sizeof(int)==sizeof(int64).
		t.goToIRValue[c] = t.translateValue(irBlock, c.X)

	case sizeTo < sizeFrom:
		t.goToIRValue[c] = irBlock.NewTrunc(fromV, toT)
	case sizeTo > sizeFrom:
		// TODO(pwaller): What happens in case of differing signedness?
		if isSigned(from) {
			t.goToIRValue[c] = irBlock.NewSExt(fromV, toT)
		} else {
			t.goToIRValue[c] = irBlock.NewZExt(fromV, toT)
		}

	default:
		panic("unreachable")
	}
}

func (t *translator) emitConvertFloat(irBlock *ir.Block, c *ssa.Convert) {
	from, to := c.X.Type(), c.Type()
	sizeFrom, sizeTo := sizeof(from), sizeof(to)
	fromV, toT := t.translateValue(irBlock, c.X), t.goToIRType(to)

	if isInteger(to) {
		if isSigned(to) {
			t.goToIRValue[c] = irBlock.NewFPToSI(fromV, toT)
		} else {
			t.goToIRValue[c] = irBlock.NewFPToUI(fromV, toT)
		}
		return
	}

	if isFloat(to) {
		switch {
		case sizeTo == sizeFrom:
			msg := "emitConvert float with equal sizes %d: %v <- %v"
			panic(fmt.Errorf(msg, sizeFrom, to, from))

		case sizeTo < sizeFrom:
			t.goToIRValue[c] = irBlock.NewFPTrunc(fromV, toT)
			return

		case sizeTo > sizeFrom:
			t.goToIRValue[c] = irBlock.NewFPExt(fromV, toT)
			return

		default:
			panic("unreachable")
		}

	}

	panic(fmt.Errorf("unimplemented: emitConvertFloat: %v <- %v", to, from))
}

func (t *translator) emitConvertSlice(irBlock *ir.Block, c *ssa.Convert) {
	from, to := c.X.Type(), c.Type()
	fromV, toT := t.translateValue(irBlock, c.X), t.goToIRType(to)

	switch {
	// TODO(pwaller): []rune to string
	case isString(to) && from.Underlying().(*gotypes.Slice).Elem().String() == "byte": // s string := b []byte
		// Grab pointer and len.
		irSlicePtr := irBlock.NewExtractValue(fromV, 0)
		irLen := irBlock.NewExtractValue(fromV, 1)

		irNewStringPtr := t.copyBytes(irBlock, irSlicePtr, irLen)

		// Construct string type.
		var irStr irvalue.Value = irconstant.NewUndef(toT)
		irStr = irBlock.NewInsertValue(irStr, irNewStringPtr, 0)
		irStr = irBlock.NewInsertValue(irStr, irLen, 1)
		t.goToIRValue[c] = irStr

	default:
		t.doTrap(irBlock)
		t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
		log.Printf("unimplemented: emitConvertSlice: %v <- %v", to, from)
		// panic(fmt.Errorf("unimplemented: emitConvertSlice: %v <- %v", to, from))
	}
}

// copyBytes copies irLen bytes from irPtr into newly allocated memory.
func (t *translator) copyBytes(
	irBlock *ir.Block, irPtr, irLen irvalue.Value,
) irvalue.Value {
	irNewPtr := irBlock.NewCall(t.builtins.Malloc(t), irLen)
	irBlock.NewCall(t.builtins.Memcpy(t), irNewPtr, irPtr, irLen, irconstant.False)
	return irNewPtr
}

func (t *translator) emitConvertComplex(irBlock *ir.Block, c *ssa.Convert) {
	t.doTrap(irBlock)
	t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
	log.Printf("unimplemented: emitConvertComplex: %v", c)
}

func (t *translator) emitConvertString(irBlock *ir.Block, c *ssa.Convert) {
	from, to := c.X.Type(), c.Type()
	fromV, toT := t.translateValue(irBlock, c.X), t.goToIRType(to)

	switch {
	case isSlice(to) && to.Underlying().(*gotypes.Slice).Elem().String() == "byte": // b []byte := s string
		// Grab pointer and len.
		irSlicePtr := irBlock.NewExtractValue(fromV, 0)
		irLen := irBlock.NewExtractValue(fromV, 1)

		irNewStringPtr := t.copyBytes(irBlock, irSlicePtr, irLen)

		// Construct slice type.
		var irSlice irvalue.Value = irconstant.NewUndef(toT)
		irSlice = irBlock.NewInsertValue(irSlice, irNewStringPtr, 0)
		irSlice = irBlock.NewInsertValue(irSlice, irLen, 1)
		irSlice = irBlock.NewInsertValue(irSlice, irLen, 2)
		t.goToIRValue[c] = irSlice

	default:
		t.doTrap(irBlock)
		t.goToIRValue[c] = irconstant.NewUndef(t.goToIRType(c.Type()))
		log.Printf("unimplemented: emitConvertSlice: %v <- %v", to, from)
		// panic(fmt.Errorf("unimplemented: emitConvertSlice: %v <- %v", to, from))
	}
}

func (t *translator) emitConvert(irBlock *ir.Block, c *ssa.Convert) {
	from, to := c.X.Type(), c.Type()
	if gotypes.Identical(from.Underlying(), to.Underlying()) {
		panic("this branch firing...")
	}
	fromV, toT := t.translateValue(irBlock, c.X), t.goToIRType(to)

	switch {
	case isComplex(from):
		t.emitConvertComplex(irBlock, c)

	case isInteger(from):
		t.emitConvertInt(irBlock, c)

	case isFloat(from):
		t.emitConvertFloat(irBlock, c)

	case isString(from):
		t.emitConvertString(irBlock, c)

	case isPointer(from):
		if isInteger(to) {
			t.goToIRValue[c] = irBlock.NewPtrToInt(fromV, toT)
			return
		}

		t.goToIRValue[c] = irBlock.NewBitCast(fromV, toT)

	case isSlice(from):
		t.emitConvertSlice(irBlock, c)

	default:
		panic(fmt.Errorf("unimplemented: emitConvert: %v <- %v", to, from))
	}
}

func (t *translator) emitDebugRef(irBlock *ir.Block, d *ssa.DebugRef) {
	log.Printf("unimplemented: emitDebugRef")
}

func (t *translator) emitDefer(irBlock *ir.Block, d *ssa.Defer) {
	log.Printf("unimplemented: emitDefer")
}

func (t *translator) emitExtract(irBlock *ir.Block, e *ssa.Extract) {
	irElem := irBlock.NewExtractValue(t.translateValue(irBlock, e.Tuple), uint64(e.Index))
	t.goToIRValue[e] = irElem
}

func (t *translator) emitField(irBlock *ir.Block, f *ssa.Field) {
	irX := t.translateValue(irBlock, f.X)
	t.goToIRValue[f] = irBlock.NewExtractValue(irX, uint64(f.Field))
}

func (t *translator) emitFieldAddr(irBlock *ir.Block, f *ssa.FieldAddr) {
	irX := t.translateValue(irBlock, f.X)
	irZero := irconstant.NewInt(irtypes.I32, 0)
	irIndex := irconstant.NewInt(irtypes.I32, int64(f.Field))
	t.goToIRValue[f] = irBlock.NewGetElementPtr(irX, irZero, irIndex)
}

func (t *translator) emitGo(irBlock *ir.Block, g *ssa.Go) {
	log.Printf("unimplemented: emitGo")
}

func (t *translator) emitIf(irBlock *ir.Block, i *ssa.If) {
	irBlock.Term = &ir.TermCondBr{
		Cond: t.translateValue(irBlock, i.Cond),
		// These are set once blocks are known.
		// TargetTrue:
		// TargetFalse:
	}
}

func (t *translator) emitIndex(irBlock *ir.Block, i *ssa.Index) {
	irX := t.translateValue(irBlock, i.X)
	irIndex := t.translateValue(irBlock, i.Index)
	// TODO(pwaller): Woah yuck, have to do an alloca? This could use a lot of
	// stack :(. Not sure what else we can do here. LLVM seemingly has no notion
	// of dynamic index into an array value. Perhaps the best we can do is move
	// the alloca up to the point of definition and do some sort of escape
	// analysis. Grim. For now, do the horrible thing.
	irXPtr := irBlock.NewAlloca(irX.Type())
	irBlock.NewStore(irX, irXPtr)
	irPtr := irBlock.NewGetElementPtr(irXPtr, irconstant.NewInt(irtypes.I32, 0), irIndex)
	t.goToIRValue[i] = irBlock.NewLoad(irPtr)
}

func (t *translator) emitIndexAddr(irBlock *ir.Block, i *ssa.IndexAddr) {
	goXType := i.X.Type().Underlying()
	switch goXType := goXType.(type) {
	case *gotypes.Slice:
		irSlice := t.translateValue(irBlock, i.X)
		irPtr := irBlock.NewExtractValue(irSlice, 0)
		irIndex := t.translateValue(irBlock, i.Index)
		t.goToIRValue[i] = irBlock.NewGetElementPtr(irPtr, irIndex)

	case *gotypes.Pointer:
		_, ok := goXType.Elem().Underlying().(*gotypes.Array)
		if !ok {
			panic(fmt.Errorf("unhandled emitIndexArray: %T: %v", goXType, goXType))
		}

		irArrayPtr := t.translateValue(irBlock, i.X)
		irIndex := t.translateValue(irBlock, i.Index)
		irZero := irconstant.NewInt(irtypes.I64, 0)
		t.goToIRValue[i] = irBlock.NewGetElementPtr(irArrayPtr, irZero, irIndex)

	default:
		panic(fmt.Errorf("unhandled emitIndexArray: %T: %v", goXType, goXType))
	}
}

func (t *translator) emitJump(irBlock *ir.Block, j *ssa.Jump) {
	irBlock.Term = &ir.TermBr{
		// These are set once blocks are known.
		// Target:
	}
}

func (t *translator) emitLookup(irBlock *ir.Block, l *ssa.Lookup) {
	if isString(l.X.Type()) {
		irStr := t.translateValue(irBlock, l.X)
		irIdx := t.translateValue(irBlock, l.Index)
		irStrBytes := irBlock.NewExtractValue(irStr, 0)
		// TODO(pwaller): Emit bounds checks?

		irGEP := irBlock.NewGetElementPtr(irStrBytes, irIdx)
		irLookup := irBlock.NewLoad(irGEP)
		t.goToIRValue[l] = irLookup

		return
	} // else, it's a map.

	// switch l.Type.
	// switch l.X.Type().(type) {
	// case *gotypes.String:
	// 	case *
	// }

	t.doTrap(irBlock)
	t.goToIRValue[l] = irconstant.NewUndef(t.goToIRType(l.Type()))
	log.Printf("unimplemented: emitLookupMap")
}

func (t *translator) emitMakeChan(irBlock *ir.Block, m *ssa.MakeChan) {
	t.doTrap(irBlock)
	t.goToIRValue[m] = irconstant.NewUndef(t.goToIRType(m.Type()))
	log.Printf("unimplemented: emitMakeChan")
}

func (t *translator) emitMakeClosure(irBlock *ir.Block, m *ssa.MakeClosure) {
	log.Printf("unimplemented: emitMakeClosure")
	// log.Printf("makeClosure: %T: %v", m.Type(), m.Type())

	// irBindingTypes = append(irBindingTypes, irtypes.NewPointer(t.goToIRType(m.Type())))
	var irBindingTypes []irtypes.Type
	for _, goBinding := range m.Bindings {
		irBindingTypes = append(irBindingTypes, t.goToIRType(goBinding.Type()))
	}

	irClosureEnvType := irtypes.NewStruct(irBindingTypes...)

	var irClosureEnv irvalue.Value = irconstant.NewUndef(irClosureEnvType)
	for i, goBinding := range m.Bindings {
		irBinding := t.translateValue(irBlock, goBinding)
		irClosureEnv = irBlock.NewInsertValue(irClosureEnv, irBinding, uint64(i))
	}

	irClosureEnvI8Ptr := irBlock.NewCall(
		t.builtins.Malloc(t),
		// TODO(pwaller): Compute actual size. Think we might want to form the
		// go struct and use gosizes, perhaps... See:
		// https://github.com/llir/llvm/issues/66 - and the idea to use
		// llvm.objectsize.
		irconstant.NewInt(irtypes.I64, 1024),
	)

	irBlock.NewStore(irClosureEnv, irBlock.NewBitCast(irClosureEnvI8Ptr, irtypes.NewPointer(irClosureEnvType)))

	// { %funcType FuncPtr, i8* ClosureEnv }
	var irClosure irvalue.Value = irconstant.NewUndef(t.goToIRType(m.Type()))
	irFn := t.goToIRValue[m.Fn]
	// log.Printf("irFn = %T %v -- %T - %v", irFn, irFn, m.Fn, m.Fn)
	// t.translateValue(irBlock, m.Fn)
	if irFn != nil {
		irClosure = irBlock.NewInsertValue(irClosure, irFn, 0)
	} else {
		log.Printf("unimplemented closure type: (e.g. Type{}.Foo)")
	}

	irClosure = irBlock.NewInsertValue(irClosure, irClosureEnvI8Ptr, 1)

	t.goToIRValue[m] = irClosure
}

func (t *translator) emitMakeInterface(irBlock *ir.Block, m *ssa.MakeInterface) {
	log.Printf("unimplemented: emitMakeInterface")
	t.goToIRValue[m] = irconstant.NewUndef(t.goToIRType(m.Type()))
	t.doTrap(irBlock)
	return

	// ms := t.prog.MethodSets.MethodSet(m.X.Type())
	// log.Printf("%T: %v", ms, ms)
	// t.goToIRValue[m] = irconstant.NewUndef(t.goToIRType(m.Type()))
	var irValue irvalue.Value = irconstant.NewUndef(t.goToIRType(m.Type()))
	irX := t.translateValue(irBlock, m.X)
	if _, ok := m.X.(*ssa.Function); ok {
		irX = t.goToIRValue[m.X]
		if irX == nil {
			irX = irconstant.NewUndef(irtypes.I8Ptr)
			// panic("blarg")
		}
	} else if !isPointer(m.X.Type()) {
		// t.builtins.Malloc(t)
		irX = irconstant.NewUndef(irtypes.I8Ptr)
	}
	irXI8Ptr := irBlock.NewBitCast(irX, irtypes.I8Ptr)
	irValue = irBlock.NewInsertValue(irValue, irXI8Ptr, 1)
	t.goToIRValue[m] = irValue

}

func (t *translator) emitMakeMap(irBlock *ir.Block, m *ssa.MakeMap) {
	log.Printf("unimplemented: emitMakeMap")
	t.doTrap(irBlock)
	t.goToIRValue[m] = irconstant.NewUndef(t.goToIRType(m.Type()))
}

func (t *translator) emitMakeSlice(irBlock *ir.Block, m *ssa.MakeSlice) {
	irLen := t.translateValue(irBlock, m.Len)
	if !irLen.Type().Equal(irtypes.I64) {
		irLen = irBlock.NewZExt(irLen, irtypes.I64)
	}
	irCap := t.translateValue(irBlock, m.Cap)
	if !irCap.Type().Equal(irtypes.I64) {
		irCap = irBlock.NewZExt(irCap, irtypes.I64)
	}

	// TODO(pwaller): improve element size computation.
	goElemType := m.Type().Underlying().(*gotypes.Slice).Elem()
	irElemType := t.goToIRType(goElemType)
	goElemSize := sizeof(goElemType)
	irElemSize := irconstant.NewInt(irtypes.I64, goElemSize)

	irAllocSize := irBlock.NewMul(irCap, irElemSize)
	irPtrI8 := irBlock.NewCall(t.builtins.Malloc(t), irAllocSize)
	irPtr := irBlock.NewBitCast(irPtrI8, irtypes.NewPointer(irElemType))

	t.goToIRValue[m] = makeStruct(irBlock, irPtr, irLen, irCap)
}

func (t *translator) emitMapUpdate(irBlock *ir.Block, m *ssa.MapUpdate) {
	t.doTrap(irBlock)
	// t.goToIRValue[m] = irconstant.NewUndef(t.goToIRType(m.Type()))
	log.Printf("unimplemented: emitMapUpdate")
}

func (t *translator) emitNext(irBlock *ir.Block, n *ssa.Next) {
	log.Printf("Next: %v %v", n, n.Type())
	t.doTrap(irBlock)
	t.goToIRValue[n] = irconstant.NewUndef(t.goToIRType(n.Type()))
	log.Printf("unimplemented: emitNext")
}

func (t *translator) emitPanic(irBlock *ir.Block, p *ssa.Panic) {
	irBlock.NewCall(
		t.builtins.Printf(t),
		irconstant.NewInt(irtypes.I32, 2),
		t.constantString(irBlock, "panic(TODO(pwaller): better message)\n"),
	)
	irBlock.NewCall(
		t.builtins.Exit(t),
		irconstant.NewInt(irtypes.I32, 42),
	)
	irBlock.NewUnreachable()
}

func (t *translator) emitPhi(irBlock *ir.Block, p *ssa.Phi) {
	irPhi, ok := t.goToIRValue[p]
	if ok {
		if irPhi == nil {
			panic("nil phi?")
		}
		// TODO(pwaller): HACK - revisit.
		irBlock.Insts = append(irBlock.Insts, irPhi.(ir.Instruction))
		return
	}

	// log.Printf("unimplemented: emitPhi")
	irPhi1 := &ir.InstPhi{} // populated when function is complete.
	irPhi1.Typ = t.goToIRType(p.Type())
	irBlock.Insts = append(irBlock.Insts, irPhi1)
	t.goToIRValue[p] = irPhi1
}

func (t *translator) emitRange(irBlock *ir.Block, r *ssa.Range) {
	log.Printf("unimplemented: emitRange")
}

func (t *translator) emitReturn(irBlock *ir.Block, r *ssa.Return) {
	var retVal irvalue.Value
	switch {
	case len(r.Results) == 0:

	case len(r.Results) == 1:
		retVal = t.translateValue(irBlock, r.Results[0])

	default:
		retVal = irconstant.NewZeroInitializer(irBlock.Parent.Sig.RetType)
		for i, goResult := range r.Results {
			irResult := t.translateValue(irBlock, goResult)
			retVal = irBlock.NewInsertValue(retVal, irResult, uint64(i))
		}
	}
	irBlock.Term = ir.NewRet(retVal)
}

func (t *translator) emitRunDefers(irBlock *ir.Block, r *ssa.RunDefers) {
	// TODO(pwaller): A no-op for now.
	log.Printf("unimplemented: emitRunDefers")
}

func (t *translator) emitSelect(irBlock *ir.Block, s *ssa.Select) {
	t.doTrap(irBlock)
	t.goToIRValue[s] = irconstant.NewUndef(t.goToIRType(s.Type()))
	log.Printf("unimplemented: emitSelect")
}

func (t *translator) emitSend(irBlock *ir.Block, s *ssa.Send) {
	log.Printf("unimplemented: emitSend")
	t.doTrap(irBlock)
}

func (t *translator) emitSliceOfString(irBlock *ir.Block, s *ssa.Slice) {
	// TODO(pwaller): Deduplication of hi/lo logic with emitSliceOfSlice?
	// TODO(pwaller): Bounds checks.
	irX := t.translateValue(irBlock, s.X)
	irSlicePtr := irBlock.NewExtractValue(irX, 0)
	irOrigLen := irBlock.NewExtractValue(irX, 1)

	var irLo irvalue.Value = irconstant.NewInt(irtypes.I64, 0)
	var irHi irvalue.Value = irOrigLen
	var irNewSlicePtr irvalue.Value = irSlicePtr

	if s.Low != nil {
		irLo = t.translateValue(irBlock, s.Low)
		if !irLo.Type().Equal(irtypes.I64) {
			irLo = irBlock.NewZExt(irLo, irtypes.I64)
		}
		irSlicePtrInt := irBlock.NewPtrToInt(irSlicePtr, irtypes.I64)
		irNewSlicePtrInt := irBlock.NewAdd(irSlicePtrInt, irLo)
		irNewSlicePtr = irBlock.NewIntToPtr(irNewSlicePtrInt, irtypes.I8Ptr)
	}
	if s.High != nil {
		irHi = t.translateValue(irBlock, s.High)
		if !irHi.Type().Equal(irtypes.I64) {
			irHi = irBlock.NewZExt(irHi, irtypes.I64)
		}
	}
	irNewLen := irBlock.NewSub(irHi, irLo)

	var irS irvalue.Value = irconstant.NewUndef(t.goToIRType(s.Type()))
	irS = irBlock.NewInsertValue(irS, irNewSlicePtr, 0)
	irS = irBlock.NewInsertValue(irS, irNewLen, 1)

	t.goToIRValue[s] = irS
}

func (t *translator) emitSliceOfSlice(irBlock *ir.Block, s *ssa.Slice) {
	// TODO(pwaller): Bounds checks.
	// TODO(pwaller): Copying of underlying data.
	irX := t.translateValue(irBlock, s.X)
	irSlicePtr := irBlock.NewExtractValue(irX, 0)
	irOrigLen := irBlock.NewExtractValue(irX, 1)

	var irLo irvalue.Value = irconstant.NewInt(irtypes.I64, 0)
	var irHi irvalue.Value = irOrigLen
	var irNewSlicePtr irvalue.Value = irSlicePtr

	if s.Low != nil {
		irLo = t.translateValue(irBlock, s.Low)
		if !irLo.Type().Equal(irtypes.I64) {
			irLo = irBlock.NewZExt(irLo, irtypes.I64)
		}
		irSlicePtrInt := irBlock.NewPtrToInt(irSlicePtr, irtypes.I64)

		// TODO(pwaller): improve element size computation.
		goElemSize := sizeof(s.Type().Underlying().(*gotypes.Slice).Elem())
		irElemSize := irconstant.NewInt(irtypes.I64, goElemSize)
		irLowOffset := irBlock.NewMul(irLo, irElemSize)

		irNewSlicePtrInt := irBlock.NewAdd(irSlicePtrInt, irLowOffset)
		irNewSlicePtr = irBlock.NewIntToPtr(irNewSlicePtrInt, irSlicePtr.Type())
	}
	if s.High != nil {
		irHi = t.translateValue(irBlock, s.High)
		if !irHi.Type().Equal(irtypes.I64) {
			irHi = irBlock.NewZExt(irHi, irtypes.I64)
		}
	}
	irNewLen := irBlock.NewSub(irHi, irLo)

	var irS irvalue.Value = irconstant.NewUndef(t.goToIRType(s.Type()))
	irS = irBlock.NewInsertValue(irS, irNewSlicePtr, 0)
	irS = irBlock.NewInsertValue(irS, irNewLen, 1)
	irS = irBlock.NewInsertValue(irS, irNewLen, 2)

	t.goToIRValue[s] = irS
}

func (t *translator) emitSliceOfArray(irBlock *ir.Block, s *ssa.Slice) {
	// TODO(pwaller): Bounds checks.
	// TODO(pwaller): Copying of underlying data.
	irX := t.translateValue(irBlock, s.X)
	irZero := irconstant.NewInt(irtypes.I32, 0)
	irSlicePtr := irBlock.NewGetElementPtr(irX, irZero, irZero)

	arrayLen := s.X.Type().(*gotypes.Pointer).Elem().Underlying().(*gotypes.Array).Len()
	irOrigLen := irconstant.NewInt(irtypes.I64, arrayLen)

	var irLo irvalue.Value = irconstant.NewInt(irtypes.I64, 0)
	var irHi irvalue.Value = irOrigLen
	var irNewSlicePtr irvalue.Value = irSlicePtr

	if s.Low != nil {
		irLo = t.translateValue(irBlock, s.Low)
		if !irLo.Type().Equal(irtypes.I64) {
			irLo = irBlock.NewZExt(irLo, irtypes.I64)
		}
		irSlicePtrInt := irBlock.NewPtrToInt(irSlicePtr, irtypes.I64)

		// TODO(pwaller): improve element size computation.
		goElemSize := sizeof(s.Type().Underlying().(*gotypes.Slice).Elem())
		irElemSize := irconstant.NewInt(irtypes.I64, goElemSize)
		irLowOffset := irBlock.NewMul(irLo, irElemSize)

		irNewSlicePtrInt := irBlock.NewAdd(irSlicePtrInt, irLowOffset)
		irNewSlicePtr = irBlock.NewIntToPtr(irNewSlicePtrInt, irSlicePtr.Type())
	}
	if s.High != nil {
		irHi = t.translateValue(irBlock, s.High)
		if !irHi.Type().Equal(irtypes.I64) {
			irHi = irBlock.NewZExt(irHi, irtypes.I64)
		}
	}
	irNewLen := irBlock.NewSub(irHi, irLo)

	var irS irvalue.Value = irconstant.NewUndef(t.goToIRType(s.Type()))
	irS = irBlock.NewInsertValue(irS, irNewSlicePtr, 0)
	irS = irBlock.NewInsertValue(irS, irNewLen, 1)
	irS = irBlock.NewInsertValue(irS, irNewLen, 2)

	t.goToIRValue[s] = irS
}

func (t *translator) emitSlice(irBlock *ir.Block, s *ssa.Slice) {
	switch {
	case isSlice(s.X.Type()):
		t.emitSliceOfSlice(irBlock, s)

	case isPtrToArray(s.X.Type()):
		t.emitSliceOfArray(irBlock, s)

	case isString(s.X.Type()):
		t.emitSliceOfString(irBlock, s)

	default:
		// TODO(pwaller): Hack: not yet implemented.
		t.doTrap(irBlock)
		t.goToIRValue[s] = irconstant.NewUndef(t.goToIRType(s.Type()))

		log.Printf("unimplemented: emitSlice: %v %T: %v", s, s.X.Type(), s.X.Type())
	}
}

func (t *translator) emitStore(irBlock *ir.Block, s *ssa.Store) {
	src := t.translateValue(irBlock, s.Val)
	dst := t.translateValue(irBlock, s.Addr)
	irBlock.NewStore(src, dst)
}

func (t *translator) emitTypeAssert(irBlock *ir.Block, ta *ssa.TypeAssert) {
	if ta.CommaOk {
		commaOKType := irtypes.NewStruct(t.goToIRType(ta.AssertedType), irtypes.I1)
		// TODO(pwaller): Undef until we have an implementation.
		t.doTrap(irBlock)
		t.goToIRValue[ta] = irconstant.NewUndef(commaOKType)
		log.Printf("unimplemented: emitTypeAssert")
		return
	}

	// TODO(pwaller): Undef until we have an implementation.
	t.doTrap(irBlock)
	t.goToIRValue[ta] = irconstant.NewUndef(t.goToIRType(ta.AssertedType))
	log.Printf("unimplemented: emitTypeAssert")
}

func (t *translator) emitUnOp(irBlock *ir.Block, u *ssa.UnOp) {
	irX := t.translateValue(irBlock, u.X)
	goXType := u.X.Type()
	irXType := t.goToIRType(goXType)

	switch u.Op {
	case token.MUL: // '*'
		t.goToIRValue[u] = irBlock.NewLoad(irX)

	case token.SUB:
		switch {
		case isInteger(goXType):
			irZero := irconstant.NewInt(irXType.(*irtypes.IntType), 0)
			t.goToIRValue[u] = irBlock.NewSub(irZero, irX)

		case isFloat(goXType):
			irZero := irconstant.NewFloat(irXType.(*irtypes.FloatType), 0)
			t.goToIRValue[u] = irBlock.NewFSub(irZero, irX)

		default:
			panic(fmt.Errorf("unimplemented: UnOp: %q: %s", u.Op, u))
		}

	case token.NOT, token.XOR:
		if isInteger(goXType) || isBool(goXType) {
			irOnes := irConstantOnes(int(sizeof(goXType) * 8))
			t.goToIRValue[u] = irBlock.NewXor(irX, irOnes)
			return
		}

		panic(fmt.Errorf("unimplemented: UnOp: %q: %s; t = %v", u.Op, u, goXType))

	case token.ARROW:
		log.Printf("unimplemented: channel recv")
		t.doTrap(irBlock)
		t.goToIRValue[u] = irconstant.NewUndef(t.goToIRType(u.Type()))

	default:
		panic(fmt.Errorf("unimplemented: UnOp: %q: %s", u.Op, u))
	}
}

func irConstantOnes(nBits int) irvalue.Value {
	// Compute (2^bits)-1
	bigOnes := (&big.Int{})
	bigOnes.SetBit(bigOnes, nBits+1, 1)
	bigOnes = bigOnes.Sub(bigOnes, big.NewInt(1))

	return &irconstant.Int{
		Typ: irtypes.NewInt(uint64(nBits)),
		X:   bigOnes,
	}
}

func (t *translator) doTrap(irBlock *ir.Block, msg ...string) {
	pc, _, _, ok := runtime.Caller(1)
	callerName, callerFile, callerLine := "", "", 0
	if ok {
		frames := runtime.CallersFrames([]uintptr{pc})
		frame, _ := frames.Next()
		callerName = frame.Func.Name()
		callerFile, callerLine = frame.Func.FileLine(pc)
	}

	extra := ""
	if len(msg) != 0 {
		extra = msg[0]
	}

	irBlock.NewCall(
		t.builtins.Printf(t),
		irconstant.NewInt(irtypes.I32, 2),
		t.constantString(irBlock, "doTrap from %s %s:%d -- %s\n"),
		t.constantString(irBlock, callerName),
		t.constantString(irBlock, callerFile),
		irconstant.NewInt(irtypes.I32, int64(callerLine)),
		t.constantString(irBlock, extra),
	)

	irBlock.NewCall(t.builtins.Trap(t))
}
