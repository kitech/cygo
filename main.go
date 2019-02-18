package main

import (
	"fmt"
	gotypes "go/types"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

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

	builtins builtins

	constantStrings map[string]irconstant.Constant
	goToIRTypeCache map[gotypes.Type]irtypes.Type
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"."}
	}

	if len(args) >= 1 {
		switch args[0] {
		case "run":
			err := run(args[1:])
			if err != nil {
				log.Fatal(err)
			}
			return
		case "build":
			err := build("./a.out", args[1:])
			if err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	err := lower(os.Stdout, args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	fdAOut, err := ioutil.TempFile("", "a.*.out")
	if err != nil {
		return fmt.Errorf("ioutil.TempFile: %v", err)
	}
	exePath := fdAOut.Name()
	_ = fdAOut.Close()
	defer os.Remove(exePath)

	err = build(exePath, args)
	if err != nil {
		return err
	}

	exe := exec.Command(exePath)
	exe.Stdout = os.Stdout
	exe.Stderr = os.Stderr
	return exe.Run()
}

func build(exePath string, args []string) error {
	fd, err := ioutil.TempFile("", "x_*.ll")
	if err != nil {
		return fmt.Errorf("ioutil.TempFile: %v", err)
	}
	defer os.Remove(fd.Name())
	defer fd.Close()

	err = lower(fd, args)
	if err != nil {
		return fmt.Errorf("lower: %v", err)
	}

	clang := exec.Command(
		"clang",
		"-Wno-override-module",
		"-O3",
		"-o", exePath,
		fd.Name(),
	)
	clang.Stdout = os.Stdout
	clang.Stderr = os.Stderr
	err = clang.Run()
	if err != nil {
		return fmt.Errorf("clang: %v", err)
	}
	return nil
}

func lower(out io.Writer, args []string) error {
	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	initial, err := packages.Load(cfg, args...)
	if err != nil {
		return fmt.Errorf("packages.Load: %v", err)
	}
	if packages.PrintErrors(initial) > 0 {
		return fmt.Errorf("packages contain errors")
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
			t.emitEntryPoint(ssaPkg)
		}
	})

	_, err = io.WriteString(out, t.m.String())
	return err
}

// emitMainInitCall inserts a call to init() at the top of the main() func.
func (t *translator) emitEntryPoint(p *ssa.Package) {
	goMain := p.Func("main")
	if goMain == nil {
		panic("no main")
	}
	goInit := p.Func("init")
	if goInit == nil {
		panic("no init")
	}
	irMain := t.goToIRValue[goMain].(*ir.Func)
	irInit := t.goToIRValue[goInit].(*ir.Func)

	irEntryPoint := t.m.NewFunc("main", irtypes.I32)
	irBlock := irEntryPoint.NewBlock("entry")
	irBlock.NewCall(irInit)
	irBlock.NewCall(irMain)
	irBlock.NewRet(irconstant.NewInt(irtypes.I32, 0))
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
	// log.Println("emitPackage:", p.Pkg.Path(), p.Pkg.Name())

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
				if m.String() != "unsafe.Pointer" {
					log.Printf("skipping methods of type %q: %T",
						m.String(), m.Object().Type())
				}
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

	for _, goFunc := range funcs {
		t.emitFunctionBody(goFunc)
	}
}

func (t *translator) emitFunctionDecl(f *ssa.Function) *ir.Func {
	for _, anonF := range f.AnonFuncs {
		t.emitFunctionDecl(anonF)
	}

	var irParams []*ir.Param
	var irLoadFreeVars *ir.Block

	if len(f.FreeVars) != 0 {
		irLoadFreeVars = ir.NewBlock("loadFreeVars")

		irClosureEnvAsI8Ptr := ir.NewParam("$closureEnv", irtypes.I8Ptr)
		irParams = append(irParams, irClosureEnvAsI8Ptr)

		var irFreeVarTypes []irtypes.Type
		for _, goFreeVar := range f.FreeVars {
			irFreeVarTypes = append(irFreeVarTypes, t.goToIRType(goFreeVar.Type()))
		}
		irClosureEnvPtrType := irtypes.NewPointer(irtypes.NewStruct(irFreeVarTypes...))
		irClosureEnvPtr := irLoadFreeVars.NewBitCast(irClosureEnvAsI8Ptr, irClosureEnvPtrType)
		for i, goFreeVar := range f.FreeVars {
			irZero := irconstant.NewInt(irtypes.I32, 0)
			irIdx := irconstant.NewInt(irtypes.I32, int64(i))
			irFreeVarPtr := irLoadFreeVars.NewGetElementPtr(irClosureEnvPtr, irZero, irIdx)
			irFreeVar := irLoadFreeVars.NewLoad(irFreeVarPtr)
			t.goToIRValue[goFreeVar] = irFreeVar
		}
	}

	// Note: this includes the reciever on methods.
	for _, goParam := range f.Params {
		irPType := t.goToIRType(goParam.Type())
		irP := ir.NewParam(goParam.Name(), irPType)
		irParams = append(irParams, irP)
		t.goToIRValue[goParam] = irP
	}

	// TODO(pwaller): note that irSig is going to be incorrect in the presence of free vars...
	irSig := t.goToIRType(f.Signature).(*irtypes.StructType).
		Fields[0].(*irtypes.PointerType).
		ElemType.(*irtypes.FuncType)

	// irRetType := t.goToIRType()

	irFuncName := f.String()
	irFunc := t.m.NewFunc(irFuncName, irSig.RetType, irParams...)

	if irLoadFreeVars != nil {
		// Terminator is set later.
		irFunc.Blocks = append(irFunc.Blocks, irLoadFreeVars)
	}

	if irFuncName != "main" { // for dead code elimination, mark everything but main private.
		irFunc.Linkage = irenum.LinkagePrivate
	}

	t.goToIRValue[f] = irFunc
	return irFunc
}

func (t *translator) emitFunctionBody(f *ssa.Function) {
	for _, anonF := range f.AnonFuncs {
		t.emitFunctionBody(anonF)
	}

	irFunc := t.goToIRValue[f].(*ir.Func)

	if len(f.Blocks) == 0 {
		// This function has no blocks; it is likely implemented in assembly.
		// Look for an equivalent function with lower-case name, that's where
		// the pure go implementation usually resides.
		goGenericImpl := f.Package().Func(strings.ToLower(f.Name()))
		if goGenericImpl != nil {
			irGenericImpl := t.goToIRValue[goGenericImpl]
			var irArgs []irvalue.Value
			for _, irParam := range irFunc.Params {
				irArgs = append(irArgs, irParam)
			}
			irBlock := irFunc.NewBlock("")
			irRetValue := irBlock.NewCall(irGenericImpl, irArgs...)
			irBlock.Term = irBlock.NewRet(irRetValue)
		} else {
			log.Println("emitting empty function body...", f.String())
			irBlock := irFunc.NewBlock("")
			irRetValue := irconstant.NewZeroInitializer(irFunc.Sig.RetType)
			irBlock.Term = irBlock.NewRet(irRetValue)
		}
	}

	// Bulk of translation happens here, except for terminators and phis which
	// can't be hooked up until their targets are constructed. So that happens
	// below.
	blockMap := map[*ssa.BasicBlock]*ir.Block{}
	for _, goBB := range f.Blocks {
		blockMap[goBB] = t.emitBlock(irFunc, goBB)
	}

	// Fixup Phi incoming edges.
	for _, goBB := range f.Blocks {
		irBlock := blockMap[goBB]

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

	// if len(f.Blocks) == 0 {
	// 	// No blocks to convert, but they may have been synthesized.
	// 	return
	// }

	// Fixup branching terminators.
	// for bbIdx, irBB := range irFunc.Blocks {
	// goBB := f.Blocks[bbIdx]
	for _, goBB := range f.Blocks {
		irBB := blockMap[goBB]

		switch irTerm := irBB.Term.(type) {
		case *ir.TermBr:
			irTerm.Target = irFunc.Blocks[goBB.Succs[0].Index]

		case *ir.TermCondBr:
			irTerm.TargetTrue = irFunc.Blocks[goBB.Succs[0].Index]
			irTerm.TargetFalse = irFunc.Blocks[goBB.Succs[1].Index]
		}
	}

	if irFunc.Blocks[0].Term == nil {
		irFunc.Blocks[0].NewBr(blockMap[f.Blocks[0]])
	}
}

func (t *translator) emitBlock(irFunc *ir.Func, goBB *ssa.BasicBlock) *ir.Block {
	irBlock := irFunc.NewBlock(fmt.Sprintf("bb_%03d", goBB.Index))
	for _, goInst := range goBB.Instrs {
		t.emitInstr(irBlock, goInst)
	}

	if irBlock.Term == nil {
		lastI := goBB.Instrs[len(goBB.Instrs)-1]
		panic(fmt.Sprintf("terminator not set, should be %T", lastI))
	}
	return irBlock
}

func (t *translator) emitGlobal(g *ssa.Global) {
	name := g.String()
	goElemType := g.Type().Underlying().(*gotypes.Pointer).Elem()
	irElemType := t.goToIRType(goElemType)
	irG := t.m.NewGlobalDef(name, irconstant.NewZeroInitializer(irElemType))
	// TODO(pwaller): Different types of linkage?
	irG.Linkage = irenum.LinkagePrivate
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
