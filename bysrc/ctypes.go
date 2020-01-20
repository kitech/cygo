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

// must *ast.Ident
func fakecfunc(funcnameidt ast.Expr, fcpkg *types.Package) *types.Func {
	idtname := fmt.Sprintf("%v", funcnameidt)
	ty1 := types.NewCtype(fmt.Sprintf("%v__ctype", idtname))
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

// must *ast.Ident
func fakecvar(varnameidt ast.Expr, fcpkg *types.Package) *types.Var {
	idtname := fmt.Sprintf("%v", varnameidt)
	ty1 := types.NewCtype(fmt.Sprintf("%v__ctype", idtname))
	var1 := types.NewVar(token.NoPos, fcpkg, idtname, ty1)
	return var1
}

func fakecconst(cstnameidt ast.Expr, fcpkg *types.Package) *types.Const {
	idtname := fmt.Sprintf("%v", cstnameidt)
	ty1 := types.NewCtype(fmt.Sprintf("%v__const__ctype", idtname))
	var val1 constant.Value
	cst1 := types.NewConst(token.NoPos, fcpkg, idtname, ty1, val1)
	return cst1
}

func fakecstruct(stnameidt ast.Expr, fcpkg *types.Package) *types.Named {
	idt := stnameidt.(*ast.Ident)
	// assert(ok)
	// idtname := fmt.Sprintf("%v", stnameidt)
	st1 := types.NewStruct(nil, nil)
	nty1 := types.NewTypeName(token.NoPos, fcpkg, idt.Name, nil)
	ty2 := types.NewNamed(nty1, st1, nil)
	return ty2
}
