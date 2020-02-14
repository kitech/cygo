package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"unsafe"
)

// 与 types.Basic结构相同，强制转换为types.Basic使用
type C2BasicType struct {
	kind int // BasicKind
	info int // BasicInfo
	name string
}

var ctypeno int = 200
var ctypetys = map[string]types.Type{}

func NewCtype(tyname string) types.Type {
	if ty, ok := ctypetys[tyname]; ok {
		return ty
	}
	tyno := ctypeno
	ctypeno++

	ty := &C2BasicType{}
	ty.name = tyname
	ty.kind = tyno
	ty.info = int(types.IsOrdered | types.IsNumeric | types.IsConstType)

	var tyx = (*types.Basic)(unsafe.Pointer(ty))
	ctypetys[tyname] = tyx
	return tyx
}

type canytype struct {
	name    string
	underty types.Type
}

func newcantype(name string) *canytype {
	ty := &canytype{}
	ty.name = name
	return ty
}
func newcanytype2(idt ast.Expr) *canytype {
	return newcantype(fmt.Sprintf("%v", idt))
}
func (caty *canytype) Underlying() types.Type { return caty.underty }
func (caty *canytype) String() string         { return caty.name }

func newtyandval(typ types.Type) types.TypeAndValue {
	tyandval := types.TypeAndValue{}
	tyandval.Type = typ
	return tyandval
}

var fcpkg = types.NewPackage("C", "C")

// must *ast.Ident
func fakecfunc2(nameidt ast.Expr, fcpkg *types.Package, rety types.Type) *types.Func {
	idtname := fmt.Sprintf("%v", nameidt)
	ty1 := rety
	var1 := types.NewVar(token.NoPos, fcpkg, "", ty1)
	f1rets := types.NewTuple(var1)
	iftype := types.NewInterfaceType(nil, nil)
	iflst := types.NewSlice(iftype)
	var2 := types.NewVar(token.NoPos, fcpkg, "args", iflst)
	prms := types.NewTuple(var2)
	prms = nil // fix some array like gened out, don't use variadic signature
	f1sig := types.NewSignature(nil, prms, f1rets, false)
	f1 := types.NewFunc(token.NoPos, fcpkg, idtname, f1sig)
	return f1
}

func fakecconst2(nameidt ast.Expr, fcpkg *types.Package, typ types.Type) *types.Const {
	idtname := fmt.Sprintf("%v", nameidt)
	ty1 := typ
	var val1 constant.Value
	val1 = constant.MakeInt64(0)
	cst1 := types.NewConst(token.NoPos, fcpkg, idtname, ty1, val1)
	return cst1
}

func fakecvar2(tnameidt ast.Expr, fcpkg *types.Package, typ types.Type) *types.Var {
	idtname := fmt.Sprintf("%v", tnameidt)
	ty1 := typ
	cst1 := types.NewVar(token.NoPos, fcpkg, idtname, ty1)
	return cst1
}

func fakecstruct(nameidt ast.Expr, fcpkg *types.Package) *types.Named {
	idt := nameidt.(*ast.Ident)
	// assert(ok)
	// idtname := fmt.Sprintf("%v", stnameidt)
	st1 := types.NewStruct(nil, nil)
	nty1 := types.NewTypeName(token.NoPos, fcpkg, idt.Name, nil)
	ty2 := types.NewNamed(nty1, st1, nil)
	return ty2
}

func fakectype(nameidt ast.Expr, fcpkg *types.Package, typ types.Type) *types.Named {
	idt := nameidt.(*ast.Ident)
	st1 := typ
	nty1 := types.NewTypeName(token.NoPos, fcpkg, idt.Name, nil)
	ty2 := types.NewNamed(nty1, st1, nil)
	return ty2
}
