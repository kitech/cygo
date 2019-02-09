package main

import (
	"fmt"
	gotypes "go/types"
	"log"
	"os"
	"sort"

	"github.com/llir/llvm/ir"
	irconstant "github.com/llir/llvm/ir/constant"
	irenum "github.com/llir/llvm/ir/enum"
	irtypes "github.com/llir/llvm/ir/types"
	irvalue "github.com/llir/llvm/ir/value"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type translator struct {
	prog *ssa.Program
	m    ir.Module

	goToIRValue map[ssa.Value]irvalue.Value

	builtinPrintln, builtinMalloc, builtinStrNCmp *ir.Func

	constantStrings map[string]irconstant.Constant
	goToIRTypeCache map[gotypes.Type]irtypes.Type
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"."}
	}
	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	initial, err := packages.Load(cfg, args...)
	if err != nil {
		log.Fatal("packages.Load: ", err)
	}
	if packages.PrintErrors(initial) > 0 {
		log.Fatalf("packages contain errors")
	}

	prog, _ := ssautil.AllPackages(initial, 0)
	prog.Build()

	t := translator{
		prog: prog,
		m: ir.Module{
			TargetTriple: "x86_64-pc-linux-gnu",
		},
		constantStrings: map[string]irconstant.Constant{},
		goToIRValue:     map[ssa.Value]irvalue.Value{},
		goToIRTypeCache: map[gotypes.Type]irtypes.Type{},
	}

	packages.Visit(initial, func(p *packages.Package) bool {
		return true
	}, func(p *packages.Package) {
		ssaPkg := prog.Package(p.Types)
		t.emitPackage(ssaPkg)

		if p.Name == "main" {
			ssaMain := ssaPkg.Func("main")
			ssaInit := ssaPkg.Func("init")
			irMain := t.goToIRValue[ssaMain].(*ir.Func)
			irInit := t.goToIRValue[ssaInit].(*ir.Func)

			var insts []ir.Instruction
			insts = append(insts, ir.NewCall(irInit))
			insts = append(insts, irMain.Blocks[0].Insts...)
			irMain.Blocks[0].Insts = insts
		}
	})

	os.Stdout.WriteString(t.m.String())
}

func sortedMembers(nameToMember map[string]ssa.Member) (members []ssa.Member) {
	for _, m := range nameToMember {
		members = append(members, m)
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].String() < members[j].String()
	})
	return members
}

func (t *translator) emitPackage(p *ssa.Package) {
	log.Println(p.Pkg.Path())

	var (
		funcs      []*ssa.Function
		globs      []*ssa.Global
		namedTypes []*ssa.Type
	)

	for _, m := range sortedMembers(p.Members) {
		// log.Println("Consider member", m.String())
		switch m := m.(type) {
		case *ssa.Function:
			funcs = append(funcs, m)

		case *ssa.Global:
			globs = append(globs, m)

		case *ssa.NamedConst:
			continue // No representation in llir, for now.

		case *ssa.Type:
			gotypesObj, ok := m.Object().Type().(*gotypes.Named)
			if !ok {
				log.Printf("skipping methods of type %q: %T", m.String(), m.Object().Type())
				continue // Nothing to represent for now.
			}
			for i, n := 0, gotypesObj.NumMethods(); i < n; i++ {
				funcs = append(funcs, p.Prog.FuncValue(gotypesObj.Method(i)))
			}

			namedTypes = append(namedTypes, m)

			continue // No representation in llir, for now.

		default:
			panic(fmt.Errorf("unhandled member: %T", m))
		}
	}

	// First resolve named types, since these can be recursive and other things
	// may depend on them.
	log.Println("TODO: need to dealwith recursive named types")

	for _, g := range globs {
		t.emitGlobal(g)
	}

	// Process functions in two passes; we need to be able to refer to other
	// functions while generating call instructions.
	var irFuncs []*ir.Func
	for _, f := range funcs {
		irFunc := t.emitFunctionDecl(f)
		irFuncs = append(irFuncs, irFunc)
	}

	for i, goFunc := range funcs {
		irFunc := irFuncs[i]
		t.emitFunctionBody(irFunc, goFunc)
	}
}

func (t *translator) emitFunctionDecl(f *ssa.Function) *ir.Func {
	var irParams []*ir.Param

	// Note: this includes the reciever on methods.
	for _, goParam := range f.Params {
		irPType := t.goToIRType(goParam.Type())
		irP := ir.NewParam(goParam.Name(), irPType)
		irParams = append(irParams, irP)
		t.goToIRValue[goParam] = irP
	}

	irSig := t.goToIRType(f.Signature).(*irtypes.FuncType)

	irFuncName := funcName(f)
	irFunc := t.m.NewFunc(irFuncName, irSig.RetType, irParams...)

	if len(f.Blocks) == 0 {
		// For functions with no body, just return zero.
		log.Println("emitting empty function body...", f.String())
		irBlock := irFunc.NewBlock("")
		irRetValue := irconstant.NewZeroInitializer(irSig.RetType)
		irBlock.Term = irBlock.NewRet(irRetValue)
	}

	if irFuncName != "main" { // for dead code elimination, mark everything but main private.
		// irFunc.Linkage = irenum.LinkagePrivate
	}

	t.goToIRValue[f] = irFunc
	return irFunc
}

func funcName(f *ssa.Function) string {
	if f.Name() == "main" {
		// There can be one main...
		return f.Name()
	}
	return f.String()
}

func (t *translator) emitFunctionBody(irFunc *ir.Func, f *ssa.Function) {
	// Bulk of translation happens here, except for terminators and phis which
	// can't be hooked up until their targets are constructed. So that happens
	// below.
	for _, goBB := range f.Blocks {
		t.emitBlock(irFunc, goBB)
	}

	// Fixup Phi incoming edges.
	for i, goBB := range f.Blocks {
		irBlock := irFunc.Blocks[i]

		for _, goInstr := range goBB.Instrs {
			goPhi, ok := goInstr.(*ssa.Phi)
			if !ok {
				continue
			}
			irPhi := t.goToIRValue[goPhi].(*ir.InstPhi)

			for j, goEdgeValue := range goPhi.Edges {
				irEdgeValue := t.translateValue(irBlock, goEdgeValue)
				irPhi.Incs = append(irPhi.Incs, &ir.Incoming{
					X:    irEdgeValue,
					Pred: irFunc.Blocks[goBB.Preds[j].Index],
				})
			}

			_ = irPhi.Type() // Compute phi type.
		}
	}

	if len(f.Blocks) == 0 {
		// No blocks to convert, but they may have been synthesized.
		return
	}

	// Fixup branching terminators.
	for bbIdx, irBB := range irFunc.Blocks {
		goBB := f.Blocks[bbIdx]

		switch irTerm := irBB.Term.(type) {
		case *ir.TermBr:
			irTerm.Target = irFunc.Blocks[goBB.Succs[0].Index]

		case *ir.TermCondBr:
			irTerm.TargetTrue = irFunc.Blocks[goBB.Succs[0].Index]
			irTerm.TargetFalse = irFunc.Blocks[goBB.Succs[1].Index]
		}
	}
}

func (t *translator) emitBlock(irFunc *ir.Func, goBB *ssa.BasicBlock) {
	irBlock := irFunc.NewBlock(fmt.Sprintf("bb_%03d", goBB.Index))
	for _, goInst := range goBB.Instrs {
		t.emitInstr(irBlock, goInst)
	}

	if irBlock.Term == nil {
		lastI := goBB.Instrs[len(goBB.Instrs)-1]
		panic(fmt.Sprintf("terminator not set, should be %T", lastI))
	}
}

func (t *translator) emitGlobal(g *ssa.Global) {
	name := g.Name()
	goElemType := g.Type().Underlying().(*gotypes.Pointer).Elem()
	irElemType := t.goToIRType(goElemType)
	irG := t.m.NewGlobal(name, irElemType)
	// TODO(pwaller): Different types of linkage?
	irG.Linkage = irenum.LinkageExternal
	t.goToIRValue[g] = irG
}

// translateValue takes a go ssa.Value and gives an irvalue.Value.
func (t *translator) translateValue(
	irBlock *ir.Block,
	goValue ssa.Value,
) irvalue.Value {
	irValue, ok := t.goToIRValue[goValue]
	if ok {
		return irValue
	}

	switch goValue := goValue.(type) {
	case *ssa.Const:
		irConst := t.goConstToIR(irBlock, goValue)
		t.goToIRValue[goValue] = irConst
		return irConst

	case *ssa.Builtin:
		panic(fmt.Sprintf("use of builtin %v not in call", goValue.Name()))

	case *ssa.Phi:
		// It's a forward reference.
		irPhi := &ir.InstPhi{} // populated after all instructions emitted.
		irPhi.Typ = t.goToIRType(goValue.Type())
		t.goToIRValue[goValue] = irPhi
		return irPhi

	default:
		panic(fmt.Sprintf("unknown goValue: %T: %v", goValue, goValue))
	}
}
