package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"gopp"
	"log"
	"strings"
)

// imppath not include /xxx/src part
func find_builtin_path(builtin_imppath string) string {
	gopaths := gopp.Gopaths()
	builtin_pkgpath := ""
	for _, gopath := range gopaths {
		pkgpath := gopath + "/src/" + builtin_imppath
		if gopp.FileExist(pkgpath) {
			builtin_pkgpath = pkgpath
			break
		}
	}
	return builtin_pkgpath
}

func cltbuiltin_methods(builtin_pkgpath string) map[string][]*ast.FuncDecl {
	mths := map[string][]*ast.FuncDecl{}
	bipc := doparse(builtin_pkgpath, "")
	for name, pkgo := range bipc.pkgs {
		if false {
			log.Println(name, pkgo.Name)
		}
		for filename, fio := range pkgo.Files {
			if false {
				log.Println(pkgo.Name, filename, fio != nil)
			}
			for idx, declx := range fio.Decls {
				if false {
					log.Println(idx, reftyof(declx))
				}
				fndecl, ok := declx.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if fndecl.Recv == nil {
					continue
				}
				recvty_expr := fndecl.Recv.List[0].Type
				recvty_str := typexpr2tyname(recvty_expr)
				if false {
					log.Println(recvty_expr, reftyof(recvty_expr), recvty_str, fndecl.Name)
				}
				switch recvty_str {
				case "string":
					mths[recvty_str] = append(mths[recvty_str], fndecl)
				case "mirmap":
					mths["map"] = append(mths["map"], fndecl)
				case "mirarray":
					mths["array"] = append(mths["array"], fndecl)
				default:
					mths[recvty_str] = append(mths[recvty_str], fndecl)
				}
			}
		}
	}
	return mths
}

func fill_builtin_methods(bimths map[string][]*ast.FuncDecl) {
	log.Println(types.DumpBuiltinMethods())
	defer log.Println(types.DumpBuiltinMethods())

	for tyname, fndecls := range bimths {
		var typobj = tyname2typobj(tyname, nil)
		if typobj == nil {
			log.Println("unkty", tyname)
			continue
		}

		for _, fndecl := range fndecls {
			retys := []types.Type{}
			retynames := []string{}
			if fndecl.Type.Results == nil {
				retys = append(retys, nil)
				retynames = append(retynames, "void")
			} else {
				for _, fldo := range fndecl.Type.Results.List {
					tyname := typexpr2tyname(fldo.Type)
					typobj := tyname2typobj(tyname, fldo.Type)
					if typobj == nil {
						log.Println(tyname, reftyof(fldo.Type))
					}
					retys = append(retys, typobj)
					retynames = append(retynames, tyname)
				}
			}
			retargs := []*types.Var{}
			for idx, rety := range retys {
				tyname := retynames[idx]
				retarg := types.NewVar(token.NoPos, nil, "", rety)
				retargs = append(retargs, retarg)
				if false {
					log.Println(tyname, rety)
					log.Println(tyname, retarg)
				}
			}
			retuple := types.NewTuple(retargs...)

			// gen *types.Func
			recvvar := types.NewVar(token.NoPos, nil, "this", typobj)
			var prmtuple *types.Tuple
			sig := types.NewSignature(recvvar, prmtuple, retuple, false)
			fnty := types.NewFunc(token.NoPos, nil, fndecl.Name.Name, sig)
			if false {
				log.Println(recvvar)
				log.Println(retuple)
			}
			if strings.Contains(fmt.Sprintf("%v", sig), "panic") {
				log.Println("addmth", typobj, fndecl.Name, "//", fnty, "//", fnty.Name())
			}
			types.AddBuiltinMethod(typobj, fnty)
		}

	}
}

func typexpr2tyname(tyexpr ast.Expr) string {
	tystr := ""
	switch recvty := tyexpr.(type) {
	case *ast.Ident:
		tystr = recvty.Name
	case *ast.StarExpr:
		tystr = recvty.X.(*ast.Ident).Name
	case *ast.ArrayType:
		tystr = "array"
	default:
		log.Panicln("wtfff", tyexpr, reftyof(tyexpr))
	}
	return tystr
}

func tyname2typobj(tyname string, tyexpr ast.Expr) types.Type {
	var typobj types.Type

	for _, tyo := range types.Typ {
		if tyo.Name() == tyname {
			typobj = tyo
			break
		}
	}
	if typobj != nil {
		return typobj
	}

	voidptrty := types.Typ[types.Voidptr]
	switch tyname {
	case "map":
		typobj = types.NewMap(voidptrty, voidptrty)
		// typobj = types.NewPointer(typobj)
	case "array":
		var elty types.Type
		if tyexpr != nil {
			ty := tyexpr.(*ast.ArrayType)
			name := typexpr2tyname(ty.Elt)
			elty = tyname2typobj(name, ty.Elt)
		}
		if elty != nil {
			typobj = types.NewSlice(elty)
		} else {
			typobj = types.NewSlice(voidptrty)
		}
	case "mirarray":
		typobj = types.NewSlice(voidptrty)
	case "f64":
		typobj = types.Typ[types.Float64]
	case "f32":
		typobj = types.Typ[types.Float32]
	case "usize":
		typobj = types.Typ[types.Usize]
	case "rune":
		typobj = types.Typ[types.Rune]
	case "byte":
		typobj = types.Typ[types.Byte]
	default:
		log.Println("unkty", tyname)
	}
	return typobj
}
