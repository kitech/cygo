package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"gopp"
	"log"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/thoas/go-funk"
)

func init() {
	if false {
		debug.PrintStack()
	}
}

type g2nc struct {
	*basecomp

	sb     strings.Builder
	curpkg string
	pkgo   *ast.Package

	info *types.Info

	fnexcepts map[*ast.FuncDecl]*FuncExceptions
}

func (this *g2nc) initfields() {
	this.info = &this.psctx.info
	this.fnexcepts = this.psctx.fnexcepts

}
func (this *g2nc) genpkgs() {
	this.initfields()

	// pkgs order?
	for pname, pkg := range this.psctx.pkgs {
		this.geninclude_cfiles(pkg)

		pkg.Scope = ast.NewScope(nil)
		this.curpkg = pkg.Name
		this.pkgo = pkg

		this.genpkg(pname, pkg)
		this.calcClosureInfo(pkg.Scope, pkg)
		this.calcDeferInfo(pkg.Scope, pkg)
		this.genGostmtTypes(pkg.Scope, pkg)
		this.genChanTypes(pkg.Scope, pkg)
		this.genMultiretTypes(pkg.Scope, pkg)
		this.genFuncs(pkg)
	}

}

const pkgsep = "__"  // between pkg and type/function
const mthsep = "_"   // between type and method
const cuzero = "{0}" // pointer, array, struct?
const stzero = "{}"
const cmtpfx = "//"

func (c *g2nc) pkgpfx() string {
	pfx := ""
	if c.curpkg == "main" {
		pfx = c.curpkg + pkgsep
	} else {
		if c.psctx.pkgrename != "" {
			pfx = c.psctx.pkgrename
		} else {
			pfx = c.curpkg
		}
		pfx += pkgsep
	}
	return pfx
}

func (c *g2nc) geninclude_cfiles(pkg *ast.Package) {
	c.outf("// cfiles %d in %v", len(c.psctx.bdpkgs.CFiles), pkg.Name).outnl()
	for _, cfile := range c.psctx.bdpkgs.CFiles {
		filename := c.psctx.path + "/" + cfile
		filename2, err := filepath.Abs(filename)
		gopp.ErrPrint(err, filename)
		log.Println(filename2)
		c.outf("#include \"%s\"", filename2).outnl()
	}
}

func (c *g2nc) genpkg(name string, pkg *ast.Package) {
	log.Println("processing package", name)
	for name, f := range pkg.Files {
		c.genPredefsFile(pkg.Scope, name, f)
	}
	for stname, _ := range c.psctx.gomangles {
		if strings.HasPrefix(stname, "struct_{") {
			c.out("/* wtfff").outnl()
			c.outf("typedef struct %s %s", stname[7:], stname).outfh().outnl()
			c.out("*/").outnl()
		} else {
			stname = strings.ReplaceAll(stname, "_", "_")
			c.outf("typedef struct %s %s /*ooo*/", stname[7:], stname).outfh().outnl()
		}
	}
	c.genFunctypesDecl(pkg.Scope)
	for name, f := range pkg.Files {
		c.genfile(pkg.Scope, name, f)
	}
}
func (c *g2nc) genPredefsFile(scope *ast.Scope, name string, f *ast.File) {
	log.Println("processing", name)
	c.outf("// predefs %v", 111).outnl()
	for _, d := range f.Decls {
		switch r := d.(type) {
		case *ast.GenDecl:
			c.genPredefTypeDecl(scope, r)
			if r == nil {
			}
		}
	}
	c.outnl()
}

func (c *g2nc) genfile(scope *ast.Scope, name string, f *ast.File) {
	log.Println("processing", name)
	/*
		for idx, cmto := range f.Comments {
			log.Println(idx, len(f.Comments), cmto.Text())
		}
	*/

	// non-func decls
	for _, d := range f.Decls {
		switch r := d.(type) {
		case *ast.FuncDecl:
		default:
			c.genDecl(scope, d)
			if r == nil {
			}
		}
	}

	// decls order?
	// for _, d := range f.Decls {
	// 	c.genDecl(scope, d)
	// }
}
func (c *g2nc) calcClosureInfo(scope *ast.Scope, pkg *ast.Package) {
	fds := map[*ast.FuncDecl]int{}
	for _, fnlit := range c.psctx.closures {
		fd := upfindFuncDeclAst(c.psctx, fnlit, 0)
		if fd == nil {
			// maybe global
		}
		if _, ok := fds[fd]; !ok {
			fds[fd] = 1
		} else {
			fds[fd] += 1
		}
		cnter := fds[fd]
		closi := c.newclosinfo(fd, fnlit, cnter)
		c.closidx[fnlit] = closi
	}

}
func (c *g2nc) calcDeferInfo(scope *ast.Scope, pkg *ast.Package) {
	defers := map[*ast.FuncDecl][]*ast.DeferStmt{}
	for _, defero := range c.psctx.defers {
		tmpfd := upfindFuncDeclNode(c.psctx, defero, 0)
		idx := len(defers[tmpfd])
		defers[tmpfd] = append(defers[tmpfd], defero)
		deferi := newdeferinfo(defero, idx)
		deferi.fd = tmpfd
		c.deferidx[defero] = deferi
	}
}
func (c *g2nc) genGostmtTypes(scope *ast.Scope, pkg *ast.Package) {
	c.out("// gostmt types ", fmt.Sprintf("%d", len(c.psctx.gostmts))).outnl()
	defer c.outnl()
	for idx, gostmt := range c.psctx.gostmts {
		c.outf("// %d %v %v", idx, gostmt.Call.Fun, gostmt.Call.Args).outnl()
		c.genFiberStargs(scope, gostmt.Call)
		c.outnl()
	}
}
func (c *g2nc) genChanTypes(scope *ast.Scope, pkg *ast.Package) {
	c.out("// chan types ", fmt.Sprintf("%d", len(c.psctx.chanops))).outnl()
	defer c.outnl()

	gottys := map[string]bool{}
	// te: ast.SendStmt.Chan/ast.UnaryExpr.X
	for _, te := range c.psctx.chanops {
		goty := c.info.TypeOf(te)
		gopp.Assert(ischanty2(goty), "must chan")
		if _, ok := gottys[goty.String()]; ok {
			continue
		}
		c.genChanStargs(scope, te) // chan structure args
		gottys[goty.String()] = true
		c.outnl()
	}
}
func (c *g2nc) genMultiretTypes(scope *ast.Scope, pkg *ast.Package) {
	c.out("// multirets types ", fmt.Sprintf("%d", len(c.psctx.gostmts))).outnl()
	defer c.outnl()
	for idx, fd := range c.psctx.multirets {
		c.outf("// %d %v %v", idx, fd.Name, fd.Type.Results.NumFields()).outnl()
		c.outf("typedef struct %s%s_multiret_arg %s%s_multiret_arg",
			c.pkgpfx(), fd.Name.Name, c.pkgpfx(), fd.Name.Name).outfh().outnl()
		c.outf("struct %s%s_multiret_arg {", c.pkgpfx(), fd.Name.Name)

		cnter := 0
		for _, fld := range fd.Type.Results.List {
			for _, _ = range fld.Names {
				c.out(c.exprTypeName(scope, fld.Type)).outsp()
				c.out(tmpvarname2(cnter)).outfh().outnl()
				cnter++
			}
		}
		// c.genFieldList(scope, fd.Type.Results, false, true, ";", false)
		c.out("}").outfh().outnl()
		c.outnl()
	}
	for idx, fd := range c.psctx.multirets {
		c.outf("// %d %v %v", idx, fd.Name, fd.Type.Results.NumFields()).outnl()
		c.outf("%s%s_multiret_arg* %s%s_multiret_arg_new_zero() {",
			c.pkgpfx(), fd.Name.Name, c.pkgpfx(), fd.Name.Name).outnl()
		c.outf("%s%s_multiret_arg* obj = (%s%s_multiret_arg*)cxmalloc(sizeof(%s%s_multiret_arg))",
			c.pkgpfx(), fd.Name.Name, c.pkgpfx(), fd.Name.Name, c.pkgpfx(), fd.Name.Name).outfh().outnl()
		cnter := 0
		for _, fld := range fd.Type.Results.List {
			fldtyx := c.info.TypeOf(fld.Type)
			for _, _ = range fld.Names {
				recurzero := false
				switch fldty := fldtyx.(type) {
				case *types.Basic:
					if fldty.Kind() == types.String {
						recurzero = true
						c.outf("obj->%s=cxstring_new()", tmpvarname2(cnter)).outfh().outnl()
					}
				case *types.Slice:
					recurzero = true
					c.outf("obj->%s=cxarray3_new(0,", tmpvarname2(cnter))
					etystr := c.exprTypeNameImpl2(scope, fldty.Elem(), nil)
					c.outf("sizeof(%s))", etystr).outfh().outnl()
				}
				if !recurzero {
					c.out("//")
					c.out(c.exprTypeName(scope, fld.Type)).outsp()
					c.out(tmpvarname2(cnter)).outfh().outnl()
				}
				cnter++
			}
		}
		c.out("return obj").outfh().outnl()
		c.out("}").outnl()
	}
}

func (this *g2nc) genFuncs(pkg *ast.Package) {
	scope := pkg.Scope
	// ordered funcDeclsv
	for _, fd := range this.psctx.funcDeclsv {
		if fd == nil {
			log.Println("wtf", fd)
			continue
		}
		if fd.Name.Name == "init" {
			continue
		}
		this.genDecl(scope, fd)
	}

	this.genInitGlobvars(pkg.Scope, pkg)

	this.genInitFuncs(scope, pkg)
	if pkg.Name == "main" {
		this.genMainFunc(scope)
	}
}

func (this *g2nc) genDecl(scope *ast.Scope, d ast.Decl) {
	switch td := d.(type) {
	case *ast.FuncDecl:
		this.genPreFuncDecl(scope, td)
		this.genFuncDecl(scope, td)
		this.genPostFuncDecl(scope, td)
	case *ast.GenDecl:
		this.genGenDecl(scope, td)
	default:
		log.Println("unimplemented", reflect.TypeOf(d))
	}
}
func (c *g2nc) genPreFuncDecl(scope *ast.Scope, d *ast.FuncDecl) {
	for _, fnlit := range c.psctx.closures {
		fd2 := upfindFuncDeclAst(c.psctx, fnlit, 0)
		if fd2 != d {
			continue
		}

		closi := c.getclosinfo(fnlit)
		c.closidx[fnlit] = closi
		cnter := closi.idx

		pkgpfx := c.pkgpfx()
		c.outf("// %v", fnlit).outnl()
		c.out("typedef").outsp()
		c.out("struct").outsp()
		c.outf("%s%s_closure_arg_%d", pkgpfx, d.Name.Name, cnter).outsp()
		c.outf("%s%s_closure_arg_%d", pkgpfx, d.Name.Name, cnter).outfh().outnl()
		c.out("struct").outsp()
		c.outf("%s%s_closure_arg_%d", pkgpfx, d.Name.Name, cnter).outsp()
		c.out("{").outnl()
		for _, ido := range closi.idents {
			c.out(c.exprTypeName(scope, ido)).outsp()
			c.out(ido.Name).outfh().outnl()
		}
		c.out("}").outfh().outnl()

		c.out("typedef").outsp()
		c.genFieldList(scope, fnlit.Type.Results, true, false, "", true)
		c.outf("(*%s%s_closure_type_%d)(", pkgpfx, d.Name.Name, cnter)
		c.genFieldList(scope, fnlit.Type.Params, false, false, ",", false)
		c.outf("%s%s_closure_arg_%d*", pkgpfx, d.Name.Name, cnter).outsp()
		c.out(")")
		c.outfh().outnl()

		c.outf("%s%s_closure_arg_%d* %s%s_closure_arg_%d_new_zero() {",
			pkgpfx, d.Name.Name, cnter, pkgpfx, d.Name.Name, cnter)
		c.outf("  return (%s%s_closure_arg_%d*)cxmalloc(sizeof(%s%s_closure_arg_%d))",
			pkgpfx, d.Name.Name, cnter, pkgpfx, d.Name.Name, cnter).outfh().outnl()
		c.out("}").outnl()

		c.out("static").outsp()
		c.genFieldList(scope, fnlit.Type.Results, true, false, "", true)
		c.outsp()
		c.outf("%s%s_closure_%d(", pkgpfx, d.Name.Name, cnter)
		c.outf("%s%s_closure_arg_%d*", pkgpfx, d.Name.Name, cnter).outsp()
		c.out("clos", gopp.IfElseStr(fnlit.Type.Params.NumFields() > 0, ",", ""))
		c.genFieldList(scope, fnlit.Type.Params, false, true, ",", true)
		// c.genFieldList(scope *ast.Scope, flds *ast.FieldList, keepvoid bool, withname bool, linebrk string, skiplast bool)
		c.out(")")
		c.out("{").outnl()
		for _, ido := range closi.idents {
			c.out(c.exprTypeName(scope, ido)).outsp()
			c.out(ido.Name).outeq()
			c.out("clos", "->", ido.Name)
			c.outfh().outnl()
		}
		c.genBlockStmt(scope, fnlit.Body)
		// assign back in case modified
		for _, ido := range closi.idents {
			c.out("clos", "->", ido.Name).outeq()
			c.out(ido.Name)
			c.outfh().outnl()
		}
		c.out("}").outnl()
		c.outnl()
	}
}
func (c *g2nc) genPostFuncDecl(scope *ast.Scope, fd *ast.FuncDecl) {
	// gen fiber wrapper funcs
	for _, gostmt := range c.psctx.gostmts {
		// how compare called func is current func
		fe := gostmt.Call.Fun
		mat := false
		switch te := fe.(type) {
		case *ast.Ident:
			mat = te.Name == fd.Name.Name
		default:
			log.Println("todo", fe, reflect.TypeOf(fe))
		}
		if mat {
			c.genFiberStwrap(scope, gostmt.Call)
		}
	}
	ant := newAnnotation(fd.Doc)
	if ant.exported {
		c.genFuncDeclExported(scope, fd, ant)
	}
	if fd.Recv == nil && fd.Body != nil {
		c.genFuncDeclCallable(scope, fd, ant)
	}
}
func (c *g2nc) genFuncDeclExported(scope *ast.Scope, fd *ast.FuncDecl, ant *Annotation) {
	c.genFieldList(scope, fd.Type.Results, true, false, "", false)
	c.outsp().out(ant.exportname).out("(")
	if fd.Recv != nil {
		c.genExpr(scope, fd.Recv.List[0].Type)
		c.outsp().out("this")
		if len(fd.Type.Params.List) > 0 {
			c.out(",")
		}
	}
	c.genFieldList(scope, fd.Type.Params, false, true, ",", true)
	c.out(") {").outnl()
	if fd.Type.Results != nil {
		c.out("return").outsp()
	}

	if fd.Recv != nil {
		tystr := c.exprTypeName(scope, fd.Recv.List[0].Type)
		tystr = strings.Trim(tystr, "*")
		c.out(tystr)
		c.out(mthsep)
	} else {
		c.out(c.pkgpfx())
	}
	c.out(fd.Name.Name).out("(")
	if fd.Recv != nil {
		c.out("this")
		if len(fd.Type.Params.List) > 0 {
			c.out(",")
		}
	}
	for idx1, prm := range fd.Type.Params.List {
		for idx2, name := range prm.Names {
			c.out(name.Name)
			if idx2 == len(prm.Names)-1 && idx1 == len(fd.Type.Params.List)-1 {
			} else {
				c.out(",")
			}
		}
	}
	c.out(")").outfh().outnl()
	c.out("}").outnl().outnl()

}
func (c *g2nc) genFuncDeclCallable(scope *ast.Scope, fd *ast.FuncDecl, ant *Annotation) {
	// todo multirets
	ismret := fd.Type.Results.NumFields() >= 2
	if ismret {
		c.outf("%s%s_multiret_arg*", c.pkgpfx(), fd.Name.Name)
	} else {
		c.genFieldList(scope, fd.Type.Results, true, false, "", false)
	}
	c.outsp()
	c.outf("%s%s_gxcallable", c.pkgpfx(), fd.Name).out("(")
	gopp.Assert(fd.Recv == nil, "wtfff", fd.Recv)
	c.out("voidptr nilthis", gopp.IfElseStr(len(fd.Type.Params.List) > 0, ",", ""))
	c.genFieldList(scope, fd.Type.Params, false, true, ",", true)
	c.out(") {").outnl()
	if fd.Type.Results != nil {
		c.out("return").outsp()
	}
	if fd.Recv != nil {
	} else {
		c.out(c.pkgpfx())
	}
	c.out(fd.Name.Name).out("(")
	for idx1, prm := range fd.Type.Params.List {
		for idx2, name := range prm.Names {
			c.out(name.Name)
			if idx2 == len(prm.Names)-1 && idx1 == len(fd.Type.Params.List)-1 {
			} else {
				c.out(",")
			}
		}
	}
	c.out(")").outfh().outnl()
	c.out("}").outnl().outnl()
}
func (this *g2nc) genFuncDecl(scope *ast.Scope, fd *ast.FuncDecl) {
	ant := newAnnotation(fd.Doc)
	this.outf("// %v", ant.original).outnl()
	this.outf("// %v", this.exprpos(fd)).outnl()
	this.clinema(fd)
	if fd.Body == nil {
		log.Println("decl only func", fd.Name)
		this.out(gopp.IfElseStr(this.curpkg == "unsafe", "//", ""))
		this.out("extern").outsp()
		// return
	}
	// _Cfunc_xxx
	iswcfn := iswrapcfunc(this.exprstr(fd.Name))
	ismret := fd.Type.Results.NumFields() >= 2
	if iswcfn {
		this.outf("// %v %s", fd, exprpos(this.psctx, fd)).outnl()
	}

	fdname := fd.Name.Name
	pkgpfx := this.pkgpfx()
	if ismret {
		this.outf("%s%s_multiret_arg*", this.pkgpfx(), fd.Name.Name)
	} else {
		this.genFieldList(scope, fd.Type.Results, true, false, "", false)
	}
	this.outsp()
	if fd.Recv != nil {
		recvtystr := this.exprTypeName(scope, fd.Recv.List[0].Type)
		recvtystr = strings.TrimRight(recvtystr, "*")
		this.out(recvtystr + mthsep + fd.Name.String())
	} else {
		this.out(pkgpfx + fdname)
	}
	this.out("(")
	if fd.Recv != nil {
		this.genFieldList(scope, fd.Recv, false, true, ",", true)
		if len(fd.Recv.List[0].Names) == 0 {
			// empty arg name
			this.out(tmpvarname())
		}
		if fd.Type.Params != nil && fd.Type.Params.NumFields() > 0 {
			this.out(",")
		}
	}

	this.genFieldList(scope, fd.Type.Params, false, true, ",", true)
	this.out(")").outnl()
	if iswcfn {
		this.out("{").outnl()
		if fd.Type.Results.NumFields() > 0 {
			this.out("return").outsp()
		}
		this.outf("%s(", fd.Name.Name[7:])
		for idx1, arge := range fd.Type.Params.List {
			_, isptrty := arge.Type.(*ast.StarExpr)
			for idx2, name := range arge.Names {
				this.out(gopp.IfElseStr(isptrty, "(voidptr)", ""))
				this.outf("%s", name.Name)
				if idx1 == fd.Type.Params.NumFields()-1 && idx2 == len(arge.Names)-1 {
				} else {
					this.out(",")
				}
			}
		}
		this.out(")").outfh().outnl()
		this.out("}").outnl()
	} else if fd.Body != nil {
		gendeferprep := func() {
			this.out("// int array").outnl()
			this.out(gopp.IfElseStr(ant.nodefer, "//", ""))
			this.outf("builtin__cxarray3* deferarr=cxarray3_new(0, sizeof(int))").outfh().outnl()
		}
		gennamedrets := func() {
			this.out("//named returns").outnl()
			if fd.Type.Results == nil {
				return
			}

			for _, fld := range fd.Type.Results.List {
				fldtyx := this.info.TypeOf(fld.Type)
				for _, name := range fld.Names {
					this.out(this.exprTypeName(scope, fld.Type)).outsp()
					this.out(name.Name).outeq().out(cuzero).outfh().outnl()

					switch fldty := fldtyx.(type) {
					case *types.Basic:
						if fldty.Kind() == types.String {
							this.outf("%v=cxstring_new()", name).outfh().outnl()
						}
					case *types.Slice:
						this.outf("%v=cxarray3_new(0,", name)
						etystr := this.exprTypeNameImpl2(scope, fldty.Elem(), nil)
						this.outf("sizeof(%s))", etystr).outfh().outnl()
					}
				}
			}
		}
		gentoperr := func() {
			this.out("// uniform error like exception").outnl()
		}

		scope = ast.NewScope(scope)
		scope.Insert(ast.NewObj(ast.Fun, fd.Name.Name))
		if ismret {
			tvname := tmpvarname()
			tvidt := newIdent(tvname)
			this.multirets[fd] = tvidt
			this.out("{").outnl()
			gennamedrets()
			gentoperr()

			this.outf("%s%s_multiret_arg*", this.pkgpfx(), fd.Name.Name).outsp().out(tvname)
			this.outeq().outsp()
			this.outf("%s%s_multiret_arg_new_zero()", this.pkgpfx(), fd.Name.Name).outfh().outnl()
			cnter := 0
			for _, fld := range fd.Type.Results.List {
				for _, name := range fld.Names {
					this.out(name.Name).outeq()
					this.outf("%s->%s", tvname, tmpvarname2(cnter)).outfh().outnl()
					cnter++
				}
			}
			gendeferprep()
			this.genBlockStmt(scope, fd.Body)
			this.out("labmret:").outnl()
			this.out("return").outsp().out(tvname).outfh().outnl()
			this.out("}").outnl()
		} else {
			this.out("{").outnl()
			gendeferprep()
			gennamedrets()
			gentoperr()
			this.genBlockStmt(scope, fd.Body)
			this.out("}").outnl()
		}
	} else {
		this.outfh()
	}
	this.outnl()
}
func (c *g2nc) genMainFunc(scope *ast.Scope) {
	c.out("int main(int argc, char**argv) {").outnl()
	c.out("cxrt_init_env(argc, argv)").outfh().outnl()
	c.out("// TODO arguments populate").outnl()
	c.out("// globvars populate").outnl()
	c.out("extern void cxall_globvars_init()").outfh().outnl()
	c.out("cxall_globvars_init()").outfh().outnl()
	c.outf("%sglobvars_init()", c.pkgpfx()).outfh().outnl()
	c.out("extern void cxall_pkginit()").outfh().outnl()
	c.out("cxall_pkginit()").outfh().outnl()
	c.out("// all func init()").outnl()
	c.outf("%spkginit()", c.pkgpfx()).outfh().outnl()
	c.outf("main%smain()", pkgsep).outfh().outnl()
	c.out("return 0").outfh().outnl()
	c.out("}").outnl()
}

// per package
func (c *g2nc) genInitFuncs(scope *ast.Scope, pkg *ast.Package) {
	for idx, fd := range c.psctx.initFuncs {
		c.outf("// %s", c.exprpos(fd).String()).outnl()
		c.out("static").outsp()
		c.outf("void %spkginit_%d()", c.pkgpfx(), idx)
		c.genBlockStmt(scope, fd.Body)
	}
	c.outf("void %spkginit(){", c.pkgpfx()).outnl()
	for idx, _ := range c.psctx.initFuncs {
		c.outf("%spkginit_%d()", c.pkgpfx(), idx).outfh().outnl()
	}
	c.out("}").outnl().outnl()
}

// all packages
func (c *g2nc) genCallPkgGlobvarsInits(pkgs []string) {
	gopp.Assert(len(pkgs) > 0, "wtfff", len(pkgs)) // builtin
	c.out("void cxall_globvars_init() {").outnl()
	// last := pkgs[len(pkgs)-1] // builtin
	// pkgs = append([]string{last}, pkgs[:len(pkgs)-1]...)
	for _, pkg := range pkgs {
		c.outf("extern void %s%sglobvars_init()", pkg, pkgsep).outfh().outnl()
		c.outf("  %s%sglobvars_init()", pkg, pkgsep).outfh().outnl()
	}
	c.out("}").outnl()
}
func (c *g2nc) genCallPkgInits(pkgs []string) {
	c.out("void cxall_pkginit() {").outnl()
	// last := pkgs[len(pkgs)-1] // builtin
	// pkgs = append([]string{last}, pkgs[:len(pkgs)-1]...)
	for _, pkg := range pkgs {
		c.outf("extern void %s%spkginit()", pkg, pkgsep).outfh().outnl()
		c.outf("  %s%spkginit()", pkg, pkgsep).outfh().outnl()
	}
	c.out("}").outnl()
}

func (this *g2nc) genBlockStmt(scope *ast.Scope, stmt *ast.BlockStmt) {
	this.out("{").outnl()
	scope = ast.NewScope(scope)

	tailreturn := false
	var tailstmt ast.Stmt
	for idx, s := range stmt.List {
		this.genStmt(scope, s, idx)
		if idx == len(stmt.List)-1 {
			_, tailreturn = s.(*ast.ReturnStmt)
			if !tailreturn {
				tailstmt = s
			}
		}
	}

	pcn := this.psctx.cursors[stmt].Parent()
	_, isfuncblk := pcn.(*ast.FuncDecl)
	if isfuncblk {
		if !tailreturn {
			this.genDeferStmt(scope, tailstmt)
		}
		this.genExceptionStmt(scope, stmt)
	}
	this.out("}").outnl()
}

// clause index?
func (this *g2nc) genStmt(scope *ast.Scope, stmt ast.Stmt, idx int) {
	// log.Println(stmt, reflect.TypeOf(stmt))
	if stmt != nil {
		posinfo := this.exprpos(stmt).String()
		fields := strings.Split(posinfo, ":")
		if len(fields) > 1 {
			// this.outf("#line %s \"%s\"", fields[1], fields[0]).outnl()
			this.out("// ", posinfo).outnl()
		} else {
			this.out("// ", posinfo).outnl()
		}
		stmtstr := this.prtnode(stmt)
		if !strings.ContainsAny(strings.TrimSpace(stmtstr), "\n") {
			this.outf("// %s", stmtstr).outnl()
		}
		this.genStmtTmps(scope, stmt)
	}
	defer this.outnl()

	addfh := true
	switch t := stmt.(type) {
	case *ast.AssignStmt:
		this.genAssignStmt(scope, t)
		if len(t.Rhs) == 1 {
			if ce, ok := t.Rhs[0].(*ast.CallExpr); ok {
				lastvar := t.Lhs[len(t.Lhs)-1]
				log.Println(lastvar, reftyof(lastvar), len(t.Lhs))
				var ns = putscope(scope, ast.Var, "varname", lastvar)
				this.genCallExprExceptionJump(ns, ce)
			}
		}
	case *ast.ExprStmt:
		this.genExpr(scope, t.X)
		if ce, ok := t.X.(*ast.CallExpr); ok {
			this.genCallExprExceptionJump(scope, ce)
		}
	case *ast.GoStmt:
		this.genGoStmt(scope, t)
	case *ast.ForStmt:
		this.genForStmt(scope, t)
	case *ast.RangeStmt:
		this.genRangeStmt(scope, t)
	case *ast.IncDecStmt:
		this.genIncDecStmt(scope, t)
	case *ast.BranchStmt:
		this.genBranchStmt(scope, t)
	case *ast.DeclStmt:
		this.genDeclStmt(scope, t)
	case *ast.IfStmt:
		this.genIfStmt(scope, t)
	case *ast.BlockStmt:
		this.genBlockStmt(scope, t)
	case *ast.SwitchStmt:
		this.genSwitchStmt(scope, t)
	case *ast.CatchStmt:
		// this.genCatchStmt(scope, t)
	case *ast.CaseClause:
		// addfh = false
		this.genCaseClause(scope, t, idx)
	case *ast.SendStmt:
		this.genSendStmt(scope, t)
	case *ast.ReturnStmt:
		this.genReturnStmt(scope, t)
	case *ast.DeferStmt:
		this.genDeferStmtSet(scope, t)
	default:
		if stmt == nil { // empty block {}
		} else {
			log.Println("unknown", reflect.TypeOf(stmt), t)
		}
	}
	if addfh {
		this.outfh().outnl()
	}
}
func (c *g2nc) genStmtTmps(scope *ast.Scope, stmt ast.Stmt) {
	if nodes, ok := c.psctx.tmpvars[stmt]; ok {
		c.outf("// temporary vars %v", len(nodes)).outnl()
		for _, n := range nodes {
			switch en := n.(type) {
			case ast.Stmt:
				c.genStmt(scope, en, 0)
			case *ast.ValueSpec:
				c.genValueSpec(scope, en, 0)
			}
		}
	}
}

// TODO too large, how split
func (c *g2nc) genAssignStmt(scope *ast.Scope, s *ast.AssignStmt) {
	// log.Println(s.Tok.String(), s.Tok.Precedence(), s.Tok.IsOperator(), s.Tok.IsLiteral(), s.Lhs)
	for i := 0; i < len(s.Rhs); i++ {
		c.valnames[s.Rhs[i]] = s.Lhs[i]
		switch te := s.Lhs[i].(type) {
		case *ast.Ident:
			obj := ast.NewObj(ast.Var, te.Name)
			obj.Data = s.Rhs[i]
			scope.Insert(obj)
		}
		var ischrv = false
		var chexpr ast.Expr
		switch e := s.Rhs[i].(type) {
		case *ast.UnaryExpr:
			if e.Op.String() == "<-" {
				ischrv = true
				chexpr = e.X
			}
		default:
			// log.Println("unknown", reflect.TypeOf(e))
		}
		idxase, isidxas := s.Lhs[i].(*ast.IndexExpr) // index assign

		mytyx := c.info.TypeOf(s.Lhs[i])
		retyx := c.info.TypeOf(s.Rhs[i])
		if ischrv {
			if s.Tok == token.DEFINE {
				c.out(c.chanElemTypeName(chexpr, false)).outsp()
				c.genExpr(scope, s.Lhs[i])
				// c.outfh().outnl()
			}

			var ns = putscope(scope, ast.Var, "varname", s.Lhs[i])
			c.genExpr(ns, s.Rhs[i])
		} else if isidxas {
			if s.Tok == token.DEFINE {
				c.out(c.exprTypeName(scope, s.Rhs[i])).outsp()
			}
			var ns = putscope(scope, ast.Var, "varval", s.Rhs[i])
			c.genExpr(ns, s.Lhs[i])

			// TODO rewrite ast of map/arr elem assign to call
			idxerty := c.info.TypeOf(idxase.X)
			_, ismap := idxerty.(*types.Map)
			_, isarray := idxerty.(*types.Slice)
			if !ismap && !isarray {
				c.outeq()
				c.genExpr(ns, s.Rhs[i])
			}
		} else if istuple2(retyx) {
			tvname := tmpvarname()
			var ns = putscope(scope, ast.Var, "varname", newIdent(tvname))
			c.out(c.exprTypeName(scope, s.Rhs[i]))
			c.out(tvname).outeq()
			c.genExpr(ns, s.Rhs[i])
			c.outfh().outnl()
			for idx, te := range s.Lhs {
				if s.Tok == token.DEFINE {
					c.out(c.exprTypeName(scope, te)).outsp()
				}
				switch xe := te.(type) {
				case *ast.Ident:
					c.out(xe.Name).outeq()
				case *ast.SelectorExpr:
					c.genExpr(scope, xe)
				default:
					log.Panicln(te, reftyof(te), exprstr(te))
				}
				c.out(tvname).out("->").out(tmpvarname2(idx)).outfh().outnl()
			}
			c.outf("cxfree(%s)", tvname).outfh().outnl()
		} else if iserrorty2(retyx) {
			// log.Panicln("TODO waitdep, after full iface assign support")
			c.out("error*").outsp()
			c.genExpr(scope, s.Lhs[i])
			c.outeq()
			lety := c.info.TypeOf(s.Lhs[i])
			gopp.Assert(lety != nil, "wtfff", retyx, s.Rhs[i], s.Lhs[i])
			if retyx == mytyx {
				c.genExpr(scope, s.Rhs[i])
			} else {
				c.out("error_new_zero()").outfh().outnl()
				c.genExpr(scope, s.Lhs[i])
				c.outf("->thisptr").outeq()
				c.genExpr(scope, s.Rhs[i])
				c.outfh().outnl()
				c.genExpr(scope, s.Lhs[i])
				c.outf("->Error").outeq()
				c.outf("%s_Error", strings.Trim(c.exprTypeName(scope, s.Rhs[i]), "*"))
			}
		} else if isiface2(mytyx) {
			if retyx == mytyx {
				c.genExpr(scope, s.Rhs[i])
			} else {
				// iface assign
				c.genExpr(scope, s.Lhs[i])
				c.outeq()
				tystr := c.exprTypeName(scope, s.Lhs[i])
				tystr = strings.Trim(tystr, "*")
				c.outf("%s_new_zero()", tystr).outfh().outnl()
				c.genExpr(scope, s.Lhs[i])
				c.out("->thisptr /*ifaceas*/").outeq()
				c.genExpr(scope, s.Rhs[i])
				if isiface2(retyx) {
					c.out("->thisptr")
				}
				c.outfh().outnl()
				unty := mytyx.Underlying().(*types.Interface)
				for j := 0; j < unty.NumMethods(); j++ {
					mtho := unty.Method(j)
					c.genExpr(scope, s.Lhs[i])
					c.outf("->%v /*ifaceas*/ = ", mtho.Name())
					if isiface2(retyx) {
						c.genExpr(scope, s.Rhs[i])
						c.out("->thisptr")
					} else {
						retystr := c.exprTypeName(scope, s.Rhs[i])
						retystr = strings.TrimRight(retystr, "*")
						c.outf("%s_%s", retystr, mtho.Name())
					}
					c.outfh().outnl()
				}
			}
		} else if rety, ok := retyx.(*types.Signature); ok {
			log.Println(s.Lhs[i], retyx, reftyof(retyx), rety, reftyof(s.Rhs[i]))
			switch aty := s.Rhs[i].(type) {
			case *ast.FuncLit:
				closi := c.getclosinfo(aty)
				tmpvname := tmpvarname()
				c.outf("%s%s* %s", c.pkgpfx(), closi.argtyname, tmpvname).outeq()
				c.outf("%s%s_new_zero()", c.pkgpfx(), closi.argtyname).outfh().outnl()
				for _, idt := range closi.idents {
					c.outf("%s->%s=%s", tmpvname, idt.Name, idt.Name).outfh().outnl()
				}
				if s.Tok == token.DEFINE {
					c.out("gxcallable*").outsp()
				}
				c.genExpr(scope, s.Lhs[i])
				c.out("").outeq()
				c.outf("gxcallable_new((voidptr)&%s%s, %s)", c.pkgpfx(), closi.fnname, tmpvname)
			case *ast.SelectorExpr: // in case method
				ismth := true // how check
				selxtyx := c.info.TypeOf(aty.X)
				selxty2 := selxtyx.(*types.Pointer).Elem().(*types.Named).Underlying()
				selxty3 := selxty2.(*types.Struct)
				for j := 0; j < selxty3.NumFields(); j++ {
					fvx := selxty3.Field(j)
					// log.Println(j, fvx.Name(), fvx.Type())
					if fvx.Name() == aty.Sel.Name {
						ismth = false
						break
					}
				}
				// log.Println(selxtyx, reftyof(selxtyx), reftyof(selxty2))
				if !ismth { // should be field, so just direct assign
					if s.Tok == token.DEFINE {
						c.out("__typeof__(")
						c.genExpr(scope, aty)
						c.out(")").outsp()
					}
					c.genExpr(scope, s.Lhs[i])
					c.outeq()
					c.genExpr(scope, s.Rhs[i])
				} else {
					if s.Tok == token.DEFINE {
						c.out("gxcallable*").outsp()
					}
					c.genExpr(scope, s.Lhs[i])
					c.out("").outeq()
					tystr := c.exprTypeName(scope, aty.X)
					tystr = strings.TrimRight(tystr, "*")
					c.outf("gxcallable_new((voidptr)&%s%s%s,", tystr, mthsep, aty.Sel.Name)
					c.genExpr(scope, aty.X)
					c.out(")")
				}
			default:
				if idt, ok := s.Rhs[i].(*ast.Ident); ok && idt.Obj.Kind == ast.Fun {
					if s.Tok == token.DEFINE {
						c.out("gxcallable*").outsp()
					}
					c.genExpr(scope, s.Lhs[i])
					c.out("").outeq()
					c.outf("gxcallable_new((voidptr)&%s%s_gxcallable, nilptr)", c.pkgpfx(), idt.Name)
				} else {
					if s.Tok == token.DEFINE {
						c.out("__typeof__(")
						c.genExpr(scope, aty)
						c.out(")").outsp()
					}
					c.genExpr(scope, s.Lhs[i])
					c.outeq()
					c.genExpr(scope, s.Rhs[i])
				}
			}
		} else {
			if s.Tok == token.DEFINE {
				log.Println(s.Rhs[i], rety, s.Lhs)
				c.out(c.exprTypeName(scope, s.Rhs[i])).outsp()
			}
			c.genExpr(scope, s.Lhs[i])

			goty := c.info.TypeOf(s.Rhs[i])
			var ns = putscope(scope, ast.Var, "varname", s.Lhs[i])
			if s.Tok == token.DEFINE {
				c.outeq()
			} else if s.Tok == token.AND_NOT_ASSIGN {
				c.out(s.Tok.String()) // todo
			} else {
				if isstrty2(goty) && s.Tok == token.ADD_ASSIGN {
					c.outeq()
				} else {
					c.out(s.Tok.String())
				}
			}
			if isstrty2(goty) && s.Tok == token.ADD_ASSIGN {
				c.out("cxstring_add(")
				c.genExpr(scope, s.Lhs[i])
				c.out(",")
			}
			c.genExpr(ns, s.Rhs[i])
			if isstrty2(goty) && s.Tok == token.ADD_ASSIGN {
				c.out(")")
			}

			// c.outfh().outnl()
		}

		if i < len(s.Lhs)-1 {
			c.outfh().outnl()
		}
	}

}

func (this *g2nc) genGoStmt(scope *ast.Scope, stmt *ast.GoStmt) {
	// calleename := stmt.Call.Fun.(*ast.Ident).Name
	// this.genCallExpr(scope, stmt.Call)
	// define function in function in c?
	// this.genFiberStargs(scope, stmt.Call)
	// this.genFiberStwrap(scope, stmt.Call)
	this.genFiberWcall(scope, stmt.Call)
}
func (c *g2nc) genFiberStargs(scope *ast.Scope, e *ast.CallExpr) {
	var funame string
	switch te := e.Fun.(type) {
	case *ast.Ident:
		funame = e.Fun.(*ast.Ident).Name
		if _, ok := c.psctx.grstargs[funame]; ok {
			return
		}
	case *ast.FuncLit:
		closi := c.getclosinfo(te)
		funame = closi.fnname
	default:
		log.Println("todo", e, reflect.TypeOf(e.Fun))
	}

	c.out("typedef struct {")
	for idx, ae := range e.Args {
		fldname := fmt.Sprintf("a%d", idx)
		fldtype := c.exprTypeName(scope, ae)
		// log.Println(funame, fldtype, fldname, reflect.TypeOf(ae))
		c.out(fldtype).outsp().out(fldname).outfh().outnl()
	}
	c.out("}", funame+"_fiber_args").outfh().outnl()
}
func (c *g2nc) genFiberStwrap(scope *ast.Scope, e *ast.CallExpr) {
	funame := e.Fun.(*ast.Ident).Name
	if _, ok := c.psctx.grstargs[funame]; ok {
		return
	}
	c.psctx.grstargs[funame] = true

	fnobj := c.info.ObjectOf(e.Fun.(*ast.Ident))
	pkgo := fnobj.Pkg()

	stname := funame + "_fiber_args"
	c.out("static").outsp()
	c.out("void").outsp()
	c.out(gopp.IfElseStr(pkgo == nil, "", pkgo.Name()+pkgsep))
	c.out(funame+"_fiber", "(voidptr vpargs)").outnl()
	c.out("{").outnl()
	c.out(stname, "*args = (", stname, "*)vpargs").outfh().outnl()
	c.out(gopp.IfElseStr(pkgo == nil, "", pkgo.Name()+pkgsep))
	c.out(funame, "(")
	for idx, _ := range e.Args {
		fldname := fmt.Sprintf("args->a%d", idx)
		c.out(fldname)
		c.out(gopp.IfElseStr(idx == len(e.Args)-1, "", ","))
	}
	c.out(")").outfh().outnl()
	c.out("}").outnl().outnl()
}
func (c *g2nc) genFiberWcall(scope *ast.Scope, e *ast.CallExpr) {
	funame := e.Fun.(*ast.Ident).Name
	wfname := funame + "_fiber"
	stname := funame + "_fiber_args"

	fnobj := c.info.ObjectOf(e.Fun.(*ast.Ident))
	pkgo := fnobj.Pkg()

	c.out("// gogorun", funame).outnl()
	c.out("{")
	c.outf("%s* args = (%s*)cxmalloc(sizeof(%s))", stname, stname, stname).outfh().outnl()
	for idx, arg := range e.Args {
		c.outf("args->a%d", idx).outeq()
		c.genExpr(scope, arg)
		c.outfh().outnl()
	}
	pkgpfx := gopp.IfElseStr(pkgo == nil, "", pkgo.Name()+pkgsep)
	c.outf("cxrt_fiber_post(%s%s, args)", pkgpfx, wfname).outfh().outnl()
	c.out("}").outnl()
}

func (c *g2nc) genForStmt(scope *ast.Scope, s *ast.ForStmt) {
	isefor := s.Init == nil && s.Cond == nil && s.Post == nil // for {}
	tmpv := tmpvarname()
	c.outf("int %s = 0", tmpv).outfh().outnl()
	c.out("for (")
	c.genStmt(scope, s.Init, 0)
	// c.out(";") // TODO ast.AssignStmt has put ;
	if isefor {
		// c.out(";")
	}
	// c.genExpr(scope, s.Cond)
	c.out(";")
	c.out(")")

	c.out("{")
	c.outf("if (%s>0) {", tmpv).outnl()
	c.genStmt(scope, s.Post, 2)
	c.outf("} else { %s = 1; }", tmpv)
	c.outf("if (")
	if s.Cond == nil {
		c.out("1")
	} else {
		c.genExpr(scope, s.Cond)
	}
	c.outf(") {\n /* goon */\n } else {break;}").outnl()

	c.genBlockStmt(scope, s.Body)
	// c.genStmt(scope, s.Post, 2) // Post move to real post, to resolve ';' problem
	c.out("// TODO gc safepoint code").outnl()
	c.out("}")
}
func (c *g2nc) genRangeStmt(scope *ast.Scope, s *ast.RangeStmt) {
	varty := c.info.TypeOf(s.X)
	// log.Println(varty, reflect.TypeOf(varty))
	switch be := varty.(type) {
	case *types.Map:
		idxidstr := fmt.Sprintf("%v", s.Key)
		idxidstr = gopp.IfElseStr(s.Index == nil, tmpvarname(), idxidstr)
		idxidstr = gopp.IfElseStr(idxidstr == "_", tmpvarname(), idxidstr)

		keytystr := c.exprTypeName(scope, s.Key)
		valtystr := c.exprTypeName(scope, s.Value)

		c.out("{").outnl()
		c.outf("  int %s = -1", idxidstr).outfh().outnl()
		c.out("  HashTableIter htiter").outfh().outnl()
		c.out("  hashtable_iter_init(&htiter, ")
		c.genExpr(scope, s.X)
		c.out(")").outfh().outnl()
		c.out("  TableEntry *entry").outfh().outnl()
		c.out("  while (hashtable_iter_next(&htiter, &entry) != CC_ITER_END) {").outnl()
		c.outf("  %s++", idxidstr).outfh().outnl()
		keyvname := fmt.Sprintf("%v", s.Key)
		keyvname = gopp.IfElseStr(s.Key == nil, tmpvarname(), keyvname)
		keyvname = gopp.IfElseStr(keyvname == "_", tmpvarname(), keyvname)
		c.outf("    %s %v = entry->key", keytystr, keyvname).outfh().outnl()
		valvname := fmt.Sprintf("%v", s.Value)
		valvname = gopp.IfElseStr(s.Value == nil, tmpvarname(), valvname)
		valvname = gopp.IfElseStr(valvname == "_", tmpvarname(), valvname)
		c.outf("    %s %v = entry->value", valtystr, valvname).outfh().outnl()
		c.genBlockStmt(scope, s.Body)
		c.out("  }").outnl()
		c.out("// TODO gc safepoint code").outnl()
		c.out("}").outnl()
	case *types.Slice:
		if s.Key != nil && s.Value == nil {
			// fix form like: for x in arr
			s.Value = s.Key
			s.Key = nil
		}
		keyidstr := fmt.Sprintf("%v", s.Key)
		keyidstr = gopp.IfElseStr(s.Key == nil, tmpvarname(), keyidstr)
		keyidstr = gopp.IfElseStr(keyidstr == "_", tmpvarname(), keyidstr)

		c.out("{").outnl()
		tmparrsz := tmpvarname()
		c.outf("int %v = cxarray3_size(", tmparrsz)
		c.genExpr(scope, s.X)
		c.out(")").outfh().outnl()
		c.outf("  for (int %s = 0; %s < %v; %s++) {",
			keyidstr, keyidstr, tmparrsz, keyidstr).outnl()
		if s.Value != nil {
			valtystr := c.exprTypeName(scope, s.Value)
			c.outf("     %s %v = %v", valtystr, s.Value, cuzero).outfh().outnl()
			var tmpvar = tmpvarname()
			c.outf("    voidptr %s = %v", tmpvar, cuzero).outfh().outnl()
			c.outf("    %v = *(%v*)cxarray3_get_at(", tmpvar, valtystr)
			c.genExpr(scope, s.X)
			c.outf(", %s)", keyidstr).outfh().outnl()
			c.outf("%v = %v", s.Value, tmpvar).outfh().outnl()
		}
		c.genBlockStmt(scope, s.Body)
		c.out("  }").outnl()
		c.out("// TODO gc safepoint code").outnl()
		c.out("}").outnl()
		if be == nil {
		}
	// TODO Array/String
	default:
		if isstrty2(varty) {
			keyidstr := fmt.Sprintf("%v", s.Key)
			keyidstr = gopp.IfElseStr(keyidstr == "_", "idx", keyidstr)
			valtystr := c.exprTypeName(scope, s.Value)

			c.out("{").outnl()
			c.outf("  for (int %s = 0; %s < (%v)->len; %s++) {",
				keyidstr, keyidstr, s.X, keyidstr).outnl()
			c.outf("     %s %v = %v", valtystr, s.Value, cuzero).outfh().outnl()
			c.outf("    %v = (%v->ptr)[%s]", s.Value, s.X, keyidstr).outfh().outnl()
			c.genBlockStmt(scope, s.Body)
			c.out("  }").outnl()
			c.out("// TODO gc safepoint code").outnl()
			c.out("}").outnl()
		} else {
			log.Println("todo", s.Key, s.Value, s.X, varty)
		}
	}
}
func (c *g2nc) genIncDecStmt(scope *ast.Scope, s *ast.IncDecStmt) {
	c.genExpr(scope, s.X)
	if s.Tok.IsOperator() {
		c.out(s.Tok.String())
	}
}
func (c *g2nc) genBranchStmt(scope *ast.Scope, s *ast.BranchStmt) {
	if s.Tok == token.FALLTHROUGH {
		c.out("gxtvnextcase = 1; break")
	} else {
		c.out(s.Tok.String())
		if s.Label != nil {
			c.out(s.Label.Name)
		}
	}
}
func (c *g2nc) genDeclStmt(scope *ast.Scope, s *ast.DeclStmt) {
	c.genDecl(scope, s.Decl)
}

func (c *g2nc) genIfStmt(scope *ast.Scope, s *ast.IfStmt) {
	if s.Init != nil {
		c.genStmt(scope, s.Init, 0)
	}
	c.out("if (")
	c.genExpr(scope, s.Cond)
	c.out(")")
	c.genBlockStmt(scope, s.Body)
	if s.Else != nil {
		c.out("else").outsp()
		c.genStmt(scope, s.Else, 0)
	}
}
func (c *g2nc) genSwitchStmt(scope *ast.Scope, s *ast.SwitchStmt) {
	tagty := c.info.TypeOf(s.Tag)
	if tagty == nil {
		log.Println(tagty, c.exprpos(s))
	} else {
		log.Println(tagty, reflect.TypeOf(tagty), reflect.TypeOf(tagty.Underlying()))
	}
	// resolve real type
	switch tty := tagty.(type) {
	case *types.Named: // like type foo int
		tagty = tty.Underlying()
	}
	switch tty := tagty.(type) {
	case *types.Basic:
		if tty.Kind() == types.String {
			c.genSwitchStmtStr(scope, s)
		} else if tty.Info()&types.IsOrdered > 0 {
			// c.genSwitchStmtNum(scope, s)
			c.genSwitchStmtAsIf(scope, s)
		} else {
			log.Println("unknown", tagty, reflect.TypeOf(tagty))
		}
	default:
		if tagty == nil {
			c.genSwitchStmtIf(scope, s)
		} else {
			log.Println("unknown", tagty, reflect.TypeOf(tagty))
		}
	}

}

func (c *g2nc) genSwitchStmtNum(scope *ast.Scope, s *ast.SwitchStmt) {
	c.out("switch (")
	c.genExpr(scope, s.Tag)
	c.out(")")
	c.genBlockStmt(scope, s.Body)
}
func (c *g2nc) genSwitchStmtStr(scope *ast.Scope, s *ast.SwitchStmt) {
	log.Println(s.Tag)
	c.out("{ // switch str").outnl()
	if s.Init != nil {
		c.genStmt(scope, s.Init, 0)
	}
	lst := s.Body.List
	tmplabs := []string{}
	for range lst {
		tmplabs = append(tmplabs, tmpvarname())
	}
	for idx, stmtx := range lst {
		stmt := stmtx.(*ast.CaseClause)
		log.Println(stmt, reftyof(stmt))
		c.outf(gopp.IfElseStr(idx > 0, "else", "")).outsp()
		c.outf("// %v", exprpos(c.psctx, stmt)).outnl()
		c.outf("if (")
		for idx2, exprx := range stmt.List {
			c.out("cxstring_eq(")
			c.genExpr(scope, s.Tag)
			c.out(",")
			c.genExpr(scope, exprx)
			c.out(")")
			c.out(gopp.IfElseStr(idx2 < len(stmt.List)-1, "||", ""))
		}
		if len(stmt.List) == 0 { //default
			c.out("1")
		}
		c.outf(") {").outnl()
		c.outf("%s:", tmplabs[idx]).outfh().outnl()
		c.outf("int gxtvnextcase = 0").outfh().outnl()
		c.outf("do {").outnl()
		for idx2, s2 := range stmt.Body {
			c.genStmt(scope, s2, idx2)
		}
		c.outf("} while(0)").outfh().outnl()
		c.outf("if (gxtvnextcase==1) {").outnl()
		if idx >= len(lst)-1 {
			c.outf("// goto %s+1", tmplabs[idx]).outnl()
		} else {
			c.outf("goto %s", tmplabs[idx+1]).outfh().outnl()
		}
		c.outf("}").outnl()
		c.outf("}").outnl()
	}
	c.out("}").outnl()
}
func (c *g2nc) genCaseClause(scope *ast.Scope, s *ast.CaseClause, idx int) {
	log.Println(s.List, s.Body)
	if len(s.List) == 0 {
		// default
		c.out("default:").outnl()
		for idx, s_ := range s.Body {
			c.genStmt(scope, s_, idx)
		}
	} else {
		switch s.List[0].(type) {
		case *ast.BinaryExpr:
			c.genCaseClauseIf(scope, s, idx)
			return
		}

		// TODO precheck if have fallthrough
		for idx, ce := range s.List {
			c.out("case").outsp()
			c.genExpr(scope, ce)
			c.out(":").outnl()
			for idx2, be := range s.Body {
				c.genStmt(scope, be, idx2)
			}
			c.out("break").outfh().outnl()
			gopp.G_USED(idx)
		}
	}
}
func (c *g2nc) genSwitchStmtIf(scope *ast.Scope, s *ast.SwitchStmt) {
	log.Println(s.Tag, s.Body == nil)
	c.outf("// %v", reflect.TypeOf(s)).outnl()
	c.genBlockStmt(scope, s.Body)
}
func (c *g2nc) genCaseClauseIf(scope *ast.Scope, s *ast.CaseClause, idx int) {
	log.Println(s.List, s.Body)
	for _, expr := range s.List {
		log.Println(expr, reflect.TypeOf(expr))
	}
	if len(s.List) == 0 {
		// default
		c.out("default:").outnl()
		for idx, s_ := range s.Body {
			c.genStmt(scope, s_, idx)
		}
	} else {
		// TODO precheck if have fallthrough
		c.out("//", gopp.IfElseStr(idx > 0, "else", "")).outnl()
		c.out("if (")
		for idx, ce := range s.List {
			c.out("(")
			c.genExpr(scope, ce)
			c.out(")")
			if idx < len(s.List)-1 {
				c.out("||").outnl()
			}
			gopp.G_USED(idx)
		}
		c.out(") {").outnl()
		for idx, be := range s.Body {
			c.genStmt(scope, be, idx)
		}
		c.out("}").outnl()
	}
}

// TODO c switch too weak, use c if stmt
func (c *g2nc) genSwitchStmtAsIf(scope *ast.Scope, s *ast.SwitchStmt) {
	c.out("{ // switch asif").outnl()
	if s.Init != nil {
		c.genStmt(scope, s.Init, 0)
	}
	lst := s.Body.List
	tmplabs := []string{}
	for range lst {
		tmplabs = append(tmplabs, tmpvarname())
	}
	for idx, stmtx := range lst {
		stmt := stmtx.(*ast.CaseClause)
		log.Println(stmt, reftyof(stmt))
		c.outf(gopp.IfElseStr(idx > 0, "else", "")).outsp()
		c.outf("// %v", exprpos(c.psctx, stmt)).outnl()
		c.outf("if (")
		for idx2, exprx := range stmt.List {
			c.genExpr(scope, s.Tag)
			c.out(token.EQL.String())
			c.genExpr(scope, exprx)
			c.out(gopp.IfElseStr(idx2 < len(stmt.List)-1, "||", ""))
		}
		if len(stmt.List) == 0 { //default
			c.out("1")
		}
		c.outf(") {").outnl()
		c.outf("%s:", tmplabs[idx]).outfh().outnl()
		c.outf("int gxtvnextcase = 0").outfh().outnl()
		c.outf("do {").outnl()
		for idx2, s2 := range stmt.Body {
			c.genStmt(scope, s2, idx2)
		}
		c.outf("} while(0)").outfh().outnl()
		c.outf("if (gxtvnextcase==1) {").outnl()
		if idx >= len(lst)-1 {
			c.outf("// goto %s+1", tmplabs[idx]).outnl()
		} else {
			c.outf("goto %s", tmplabs[idx+1]).outfh().outnl()
		}
		c.outf("}").outnl()
		c.outf("}").outnl()
	}
	c.out("}").outnl()
}

// TODO c switch too weak, use c if stmt
func (c *g2nc) genCatchStmtAsIf(scope *ast.Scope, s *ast.CatchStmt) {
	c.out("{ // catch asif").outnl()
	if s.Init != nil {
		c.genStmt(scope, s.Init, 0)
	}
	lst := s.Body.List
	tmplabs := []string{}
	for range lst {
		tmplabs = append(tmplabs, tmpvarname())
	}
	for idx, stmtx := range lst {
		stmt := stmtx.(*ast.CaseClause)
		log.Println(stmt, reftyof(stmt))
		c.outf(gopp.IfElseStr(idx > 0, "else", "")).outsp()
		c.outf("// %v", exprpos(c.psctx, stmt)).outnl()
		c.outf("if (")
		for idx2, exprx := range stmt.List {
			c.genExpr(scope, s.Tag)
			c.out(token.EQL.String())
			c.genExpr(scope, exprx)
			c.out(gopp.IfElseStr(idx2 < len(stmt.List)-1, "||", ""))
		}
		if len(stmt.List) == 0 { //default
			c.out("1")
		}
		c.outf(") {").outnl()
		c.outf("%s:", tmplabs[idx]).outfh().outnl()
		c.outf("int gxtvnextcase = 0").outfh().outnl()
		c.outf("do {").outnl()
		for idx2, s2 := range stmt.Body {
			c.genStmt(scope, s2, idx2)
		}
		c.outf("} while(0)").outfh().outnl()
		c.outf("if (gxtvnextcase==1) {").outnl()
		if idx >= len(lst)-1 {
			c.outf("// goto %s+1", tmplabs[idx]).outnl()
		} else {
			c.outf("goto %s", tmplabs[idx+1]).outfh().outnl()
		}
		c.outf("}").outnl()
		c.outf("}").outnl()
	}
	c.out("}").outnl()
}

func (c *g2nc) genCatchStmt(scope *ast.Scope, s *ast.CatchStmt) {
	tagty := c.info.TypeOf(s.Tag)
	if tagty == nil {
		log.Println(tagty, c.exprpos(s))
	} else {
		log.Println(tagty, reflect.TypeOf(tagty), reflect.TypeOf(tagty.Underlying()))
	}
	// resolve real type
	c.genCatchStmtAsIf(scope, s)
	/*
		switch tty := tagty.(type) {
		case *types.Named: // like type foo int
			tagty = tty.Underlying()
		}
		switch tty := tagty.(type) {
		case *types.Basic:
			if tty.Kind() == types.String {
				c.genSwitchStmtStr(scope, s)
			} else if tty.Info()&types.IsOrdered > 0 {
				// c.genSwitchStmtNum(scope, s)
				c.genSwitchStmtAsIf(scope, s)
			} else {
				log.Println("unknown", tagty, reflect.TypeOf(tagty))
			}
		default:
			if tagty == nil {
				c.genSwitchStmtIf(scope, s)
			} else {
				log.Println("unknown", tagty, reflect.TypeOf(tagty))
			}
		}
	*/
}

func (c *g2nc) genCallExpr(scope *ast.Scope, te *ast.CallExpr) {
	// log.Println(te, te.Fun, reflect.TypeOf(te.Fun))
	scope = putscope(scope, ast.Fun, "infncall", te.Fun)
	fca := c.getCallExprAttr(scope, te)
	switch be := te.Fun.(type) {
	case *ast.Ident:
		funame := be.Name
		if funame == "make" {
			c.genCallExprMake(scope, te)
		} else if funame == "len" {
			c.genCallExprLen(scope, te)
		} else if funame == "cap" {
			c.genCallExprLen(scope, te)
			// panic("not supported " + funame)
		} else if funame == "append" {
			c.genCallExprAppend(scope, te)
		} else if funame == "delete" {
			c.genCallExprDelete(scope, te)
		} else if funame == "println" {
			c.genCallExprPrintln(scope, te)
		} else if c.funcistype(be) {
			c.genTypeCtor(scope, te)
		} else {
			var upfindsym func(s *ast.Scope, id *ast.Ident, lvl int) interface{}
			upfindsym = func(s *ast.Scope, id *ast.Ident, lvl int) interface{} {
				if s == nil {
					return nil
				}
				obj := s.Lookup(id.Name)
				if obj != nil {
					switch id2 := obj.Data.(type) {
					case *ast.Ident:
						return upfindsym(s, id2, 0)
					}
					return obj.Data
				}
				return upfindsym(s.Outer, id, lvl+1)
			}
			isclos := false
			_, isfnvar := te.Fun.(*ast.Ident)
			var fnlit *ast.FuncLit
			gotyx := c.info.TypeOf(te.Fun)
			log.Println(gotyx, reftyof(gotyx), te.Fun, reftyof(te.Fun))
			switch gotyx.(type) {
			case *types.Signature:
				symve := upfindsym(scope, be, 0)
				isclos = symve != nil && !isfnvar
				if symve != nil {
					if _, ok := symve.(*ast.FuncLit); ok {
						isclos = true
					}
				}
				if isclos {
					fnlit = symve.(*ast.FuncLit)
				}
			}
			if isclos {
				c.genCallExprClosure(scope, te, fnlit)
			} else if fca.isvardic {
				c.genCallExprVaridic(scope, te)
			} else {
				c.genCallExprNorm(scope, te)
			}
		}
	case *ast.SelectorExpr:
		if fca.isbuiltin &&
			funk.Contains([]string{"sizeof", "alignof", "offsetof", "assert"},
				fca.selfn.Sel.Name) {
			selname := fca.selfn.Sel.Name
			selname = gopp.IfElseStr(selname == "alignof", "_Alignof", selname)
			c.outf("%s(", selname)
			for idx, _ := range te.Args {
				c.genExpr(scope, te.Args[idx])
				c.out(gopp.IfElseStr(idx == 0 && len(te.Args) > 1, ",", ""))
			}
			c.out(")")
			break
		}
		if c.funcistype(te.Fun) {
			c.genTypeCtor(scope, te)
		} else if fca.isvardic {
			c.genCallExprVaridic(scope, te)
		} else {
			c.genCallExprNorm(scope, te)
		}
	case *ast.ArrayType:
		c.genTypeCtor(scope, te)
	case *ast.ParenExpr:
		c.out("(")
		log.Println(be.X, reflect.TypeOf(be.X))
		c.genExpr(scope, be.X)
		c.out(")")
		c.out("(")
		for idx, arge := range te.Args {
			c.genExpr(scope, arge)
			if idx < len(te.Args)-1 {
				c.out(",")
			}
		}
		c.out(")")
	case *ast.FuncLit:
		c.genCallExprClosure(scope, te, be)
	default:
		log.Println("todo", be, reflect.TypeOf(be))
	}
}
func (c *g2nc) genCallExprMake(scope *ast.Scope, te *ast.CallExpr) {
	log.Println("CallExpr", te.Fun)
	itep := te.Args[0]
	var lenep ast.Expr
	if len(te.Args) > 1 {
		lenep = te.Args[1]
	}

	log.Println(reflect.TypeOf(itep))
	switch ity := itep.(type) {
	case *ast.ChanType:
		log.Println("elemty", reflect.TypeOf(ity.Value), c.info.TypeOf(ity.Value))
		elemtyx := c.info.TypeOf(ity.Value)
		log.Println(elemtyx, reflect.TypeOf(elemtyx))
		switch elemty := elemtyx.(type) {
		case *types.Basic:
			switch elemty.Kind() {
			case types.Int:
				log.Println("it's chan, and elem int", lenep)
			default:
				log.Println("unknown", elemtyx, elemty)
			}
		default:
			log.Println("unknown", elemtyx, elemty)
		}
		c.out("cxrt_chan_new(")
		if lenep == nil {
			c.out("0")
		} else {
			c.genExpr(scope, lenep)
		}
		c.out(")")
	case *ast.ArrayType:
		gopp.Assert(len(te.Args) > 0, "wtfff", len(te.Args))
		elemtya := te.Args[0].(*ast.ArrayType).Elt
		// log.Println(te.Args[0], reftyof(te.Args[0]), elemtya, reftyof(elemtya))
		elemtyt := c.info.TypeOf(elemtya)
		c.outf("cxarray3_new(")
		if len(te.Args) > 1 {
			c.genExpr(scope, te.Args[1])
		} else {
			c.out("0")
		}
		etystr := c.exprTypeNameImpl2(scope, elemtyt, elemtya)
		c.outf(", sizeof(%v))", etystr)
	default:
		log.Println("unknown", itep, ity, lenep)
	}
}
func (c *g2nc) genCallExprLen(scope *ast.Scope, te *ast.CallExpr) {
	arg0 := te.Args[0]
	argty := c.info.TypeOf(arg0)
	if ismapty(argty.String()) {
		switch be := arg0.(type) {
		case *ast.Ident:
			c.outf("hashtable_size(%s)", be.Name)
		case *ast.SelectorExpr:
			c.out("hashtable_size(")
			c.genExpr(scope, be.X)
			c.out("->")
			c.genExpr(scope, be.Sel)
			c.out(")")
		default:
			log.Println("todo", reflect.TypeOf(arg0))
		}
	} else if isstrty(argty.String()) {
		c.out("cxstring_len(")
		c.genExpr(scope, arg0)
		c.out(")")
	} else if isslicety(argty.String()) || isarrayty(argty.String()) {
		funame := te.Fun.(*ast.Ident).Name
		if funame == "len" {
			c.outf("cxarray3_size(")
			c.genExpr(scope, arg0)
			c.out(")")
		} else if funame == "cap" {
			c.out("cxarray3_capacity(")
			c.genExpr(scope, arg0)
			c.out(")")
		} else {
			panic(funame)
		}
	} else {
		log.Println("todo", te.Args, argty)
	}
}
func (c *g2nc) genCallExprAppend(scope *ast.Scope, te *ast.CallExpr) {
	arg0 := te.Args[0]
	argty := c.info.TypeOf(arg0)
	if ismapty(argty.String()) {
		panic(argty.String())
		switch be := arg0.(type) {
		case *ast.Ident:
			c.outf("hashtable_size(%s)", be.Name)
		case *ast.SelectorExpr:
			c.out("hashtable_size(")
			c.genExpr(scope, be.X)
			c.out("->")
			c.genExpr(scope, be.Sel)
			c.out(")")
		default:
			log.Println("todo", reflect.TypeOf(arg0))
		}
	} else if isstrty(argty.String()) {
		panic(argty.String())
		c.out("cxstring_len(")
		c.genExpr(scope, arg0)
		c.out(")")
	} else if isslicety(argty.String()) || isarrayty(argty.String()) {
		// funame := te.Fun.(*ast.Ident).Name
		for idx := 1; idx < len(te.Args); idx++ {
			ae := te.Args[idx]
			if idx > 1 {
				c.genExpr(scope, arg0)
				c.outeq()
			}
			c.outf("cxarray3_append(")
			c.genExpr(scope, arg0)
			c.out(", (voidptr)&")
			c.genExpr(scope, ae)
			c.out(")").outfh().outnl()
		}

	} else {
		log.Println("todo", te.Args, argty)
	}
}
func (c *g2nc) genCallExprDelete(scope *ast.Scope, te *ast.CallExpr) {
	arg0 := te.Args[0]
	arg1 := te.Args[1]
	argty := c.info.TypeOf(arg0)
	if ismapty(argty.String()) {
		keystr := ""
		switch te := arg1.(type) {
		case *ast.BasicLit:
			switch te.Kind {
			case token.STRING:
				keystr = fmt.Sprintf("cxhashtable_hash_str(%s)", te.Value)
			default:
				log.Println("todo", te.Kind)
			}
		case *ast.Ident:
			keystr = c.exprstr(arg1)
		default:
			log.Println("todo", reflect.TypeOf(arg1), arg1, c.exprstr(arg1), c.exprpos(arg0))
		}
		c.outf("hashtable_remove(")
		c.genExpr(scope, arg0)
		c.outf(", (voidptr)(uintptr)%s, 0)", keystr).outfh().outnl()
	} else {
		log.Println("todo", te.Args, argty)
	}
}
func (c *g2nc) genCallExprPrintln(scope *ast.Scope, te *ast.CallExpr) {
	tmpnames := make([]string, len(te.Args))
	for idx, e1 := range te.Args {
		tety := c.info.TypeOf(e1)
		if isstrty2(tety) {
			switch tety.(type) {
			case *types.Basic:
				tname := tmpvarname()
				c.outf("cxstring* %s = ", tname)
				c.genExpr(scope, e1)
				c.outfh().outnl()
				tmpnames[idx] = tname
			}
		}
	}
	// c.genExpr(scope, te.Fun)
	c.out("println2")
	c.out("(__FILE__, __LINE__, __func__")
	c.out(gopp.IfElseStr(len(te.Args) > 0, ",", "")).outnl()
	if len(te.Args) > 0 {
		var tyfmts []string
		for _, e1 := range te.Args {
			tyfmt := c.exprTypeFmt(scope, e1)
			tyfmts = append(tyfmts, "%"+tyfmt)
		}
		c.out(fmt.Sprintf(`"%s"`, strings.Join(tyfmts, " ")))
		c.out(", ")
	}
	for idx, e1 := range te.Args {
		tety := c.info.TypeOf(e1)
		if isstrty2(tety) {
			c.outf("(%s)->len,", tmpnames[idx])
			c.outf("(%s)->ptr", tmpnames[idx])
		} else if iseface2(tety) {
			c.genExpr(scope, e1)
			c.out(".data")
		} else {
			c.genExpr(scope, e1)
		}
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")")

	// check if real need, ;\n
	cs := c.psctx.cursors[te]
	if cs.Name() != "Args" {
		// c.outfh().outnl()
	}
}
func (c *g2nc) getCallExprAttr(scope *ast.Scope, te *ast.CallExpr) *FuncCallAttr {
	fca := &FuncCallAttr{}
	fca.selfn, fca.isselfn = te.Fun.(*ast.SelectorExpr)
	if fca.isselfn {
		// var selidt *ast.Ident
		// selidt, isidt = selfn.X.(*ast.Ident)
		// iscfn = isidt && selidt.Name == "C"
		if iscsel(te.Fun) {
			// selidt = selfn.Sel.(*ast.Ident)
			// isidt = true
			fca.iscfn = true
		} else {
			selxty := c.info.TypeOf(fca.selfn.X)
			fca.isrcver = !isinvalidty2(selxty)
			fca.ispkgsel = ispackage(c.psctx, fca.selfn.X)

			switch ne := selxty.(type) {
			case *types.Named:
				fca.isifacesel = isiface2(ne.Underlying())
			}
		}
		if idt, ok := fca.selfn.X.(*ast.Ident); ok {
			fca.isbuiltin = idt.Name == "builtin"
		}
		fca.idtfn = fca.selfn.Sel
	} else if idt, ok := te.Fun.(*ast.Ident); ok {
		fca.idtfn = idt
	}
	gotyx := c.info.TypeOf(te.Fun)
	if gotyx == nil {
		log.Println(gotyx != nil, "wtfff", te.Fun, exprstr(te.Fun), exprpos(c.psctx, te))
	}
	if gotyx != nil {
		goty1, ok := gotyx.(*types.Signature)
		if ok {
			fca.fnty = goty1
			fca.isvardic = goty1.Variadic()
			fca.prmty = goty1.Params()
			retlst := goty1.Results()
			for i := 0; retlst != nil && i < retlst.Len(); i++ {
				fldvar := retlst.At(i)
				log.Println(i, fldvar, reftyof(fldvar.Type()))
				if iserrorty2(fldvar.Type()) {
					// log.Println("seems haserrret", te.Fun)
					fca.haserrret = true
				}
			}
		} else {
			log.Println(gotyx, reflect.TypeOf(gotyx), te.Fun)
		}
	}

	// log.Println(te.Args, te.Fun, gotyx, reflect.TypeOf(gotyx), goty.Variadic())
	lexpr := c.psctx.kvpairs[te]
	fca.haslval = lexpr != nil
	if lexpr != nil {
		fca.lexpr = lexpr.(ast.Expr)
	}
	if !fca.haslval {
		lobj := scope.Lookup("varname")
		fca.haslval = lobj != nil
		if fca.haslval {
			fca.lexpr = lobj.Data.(ast.Expr)
		}
	}

	_, fca.isclos = te.Fun.(*ast.FuncLit)
	var fnobj types.Object
	switch fe := te.Fun.(type) {
	case *ast.SelectorExpr:
		fnobj = c.info.ObjectOf(fe.Sel)
	case *ast.Ident:
		fnobj = c.info.ObjectOf(fe)
	}
	if fnobj != nil {
		_, fca.isfnvar = fnobj.(*types.Var)
	}
	// log.Println(te.Fun, reftyof(te.Fun), fnobj, reftyof(fnobj), fca.isfnvar)

	return fca
}

func (c *g2nc) genCallExprNorm(scope *ast.Scope, te *ast.CallExpr) {
	// funame := te.Fun.(*ast.Ident).Name
	fca := c.getCallExprAttr(scope, te)
	// gopp.Assert(!fca.isvardic, "moved", te.Fun)

	idt := newIdent(tmpvarname()) // store variadic args in array
	if fca.isvardic && fca.haslval {
		c.out(cuzero).outfh().outnl()
	}
	if fca.isvardic {
		c.outf("builtin__cxarray3* %s=cxarray3_new(0,sizeof(voidptr))", idt.Name).outfh().outnl()
		prmty := fca.fnty.Params()
		elemty := prmty.At(prmty.Len() - 1).Type().(*types.Slice).Elem()
		// log.Println(te.Fun, prmty, reftyof(prmty), prmty.Len(), elemty, reftyof(elemty))
		for idx, e1 := range te.Args {
			if idx < fca.fnty.Params().Len()-1 {
				continue // non variadic arg
			}
			switch elty := elemty.(type) {
			case *types.Interface:
				e1tyx := c.info.TypeOf(e1)
				if e1ty, ok1 := e1tyx.(*types.Slice); ok1 {
					if _, ok2 := e1ty.Elem().(*types.Interface); ok2 {
						c.outf("%s", idt.Name).outeq()
						c.genExpr(scope, e1)
						break
					}
				}
				tyname := c.exprTypeName(scope, e1)
				if tyname == "cxstring*" {
					tyname = "string"
				}
				tvar := tmpvarname()
				c.outf("voidptr %s= cxrt_type2eface((voidptr)&%s_metatype, (voidptr)&", tvar, tyname)
				c.genExpr(scope, e1)
				c.out(")").outfh().outnl()
				c.outf("cxarray3_append(%s, &%s)", idt.Name, tvar)
			default:
				_ = elty
				c.outf("cxarray3_append(%s, (voidptr)&", idt.Name)
				c.genExpr(scope, e1)
				c.out(")")
			}
			c.outfh().outnl()
		}
	}
	if fca.isvardic && fca.haslval {
		c.out("/*222*/")
		c.genExpr(scope, fca.lexpr)
		c.outeq()
		c.out("/*111*/")
	}

	if fca.isselfn {
		if fca.iscfn {
			c.out("(")
			c.genExpr(scope, fca.selfn.Sel)
			c.out(")")
		} else if fca.isifacesel {
			c.genExpr(scope, fca.selfn.X)
			c.out("->")
			c.genExpr(scope, fca.selfn.Sel)
		} else if fca.ispkgsel {
			c.genExpr(scope, fca.selfn.X)
			c.out(pkgsep)
			c.out(fca.selfn.Sel.Name)
		} else {
			// log.Println(selfn.X, reftyof(selfn.X), c.info.TypeOf(selfn.X))
			vartystr := c.exprTypeName(scope, fca.selfn.X)
			vartystr = strings.TrimRight(vartystr, "*")
			c.out(vartystr + "_" + fca.selfn.Sel.Name)
		}
	} else {
		fnidt := te.Fun.(*ast.Ident)
		fnobj := c.info.ObjectOf(fnidt)
		pkgo := fnobj.Pkg()
		if pkgo != nil && fnobj.Pkg() != nil {
			// c.out(fnobj.Pkg().Name(), "_")
		}
		if funk.Contains([]string{"assert", "sizeof", "alignof"}, fnidt.Name) {
			c.out(fnidt.Name)
		} else {
			_, issig := fnobj.Type().(*types.Signature)
			log.Println(fnidt.Name, fnidt.Obj, fnobj.Type(), issig)
			if fnidt.Obj != nil && fnidt.Obj.Kind == ast.Var && issig {
				c.genCallExprClosure2(scope, te)
				return
			} else {
				c.genExpr(scope, te.Fun)
			}
		}
	}
	if fca.isselfn && fca.isbuiltin && fca.selfn.Sel.Name == "typeof" {
		a0ty := c.info.TypeOf(te.Args[0])
		tystr := c.exprTypeNameImpl2(scope, a0ty, te.Args[0])
		tystr = strings.Trim(tystr, "*")
		c.outf("_goimpl((voidptr)(&%s_metatype))", tystr)
		return
	}

	c.out("(")
	// reciever this
	if fca.isselfn && !fca.iscfn && !fca.ispkgsel && fca.isrcver {
		selx := fca.selfn.X
		c.genExpr(scope, selx)
		c.out(gopp.IfElseStr(fca.isifacesel, "->thisptr", ""))
		c.out(gopp.IfElseStr(len(te.Args) > 0, ",", ""))
	}

	for idx, e1 := range te.Args {
		if fca.isvardic && idx == fca.fnty.Params().Len()-1 {
			c.out(idt.Name)
			break
		}

		if fca.prmty != nil {
			prmn := fca.prmty.At(idx).Type()
			if _, ok := prmn.(*types.Interface); ok {
				c.out("cxrt_type2eface((voidptr)&")
				tyname := c.exprTypeName(scope, e1)
				if strings.Contains(tyname, "cxstring") {
					c.out("string")
				} else {
					c.out(tyname)
				}
				c.out("_metatype, (voidptr)&")
				c.genExpr(scope, e1)
				c.out(")")
			} else {
				c.genExpr(scope, e1)
			}
		} else {
			c.genExpr(scope, e1)
		}
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")")

}
func (c *g2nc) genCallExprExceptionJump(scope *ast.Scope, te *ast.CallExpr) {
	fca := c.getCallExprAttr(scope, te)
	// gopp.Assert(!fca.isvardic, "moved", te.Fun)

	if fca.haserrret {
		c.outfh().outnl()

		upfd := upfindFuncDeclNode(c.psctx, te, 0)
		gopp.Assert(upfd != nil, "wtfff", te.Fun)
		fnexc, ok := c.fnexcepts[upfd]
		gopp.Assert(ok, "wtfff", upfd.Name, te.Fun)
		var exi *ExceptionInfo
		for _, ex := range fnexc.callexes {
			if ex.callexpr == te {
				exi = ex
				break
			}
		}
		gopp.Assert(exi != nil, "wtfff", upfd.Name, te.Fun)

		c.out(gopp.IfElseStr(fca.haslval, "", "//"))
		c.outf("gxtvtoperr =")
		if fca.haslval {
			c.genExpr(scope, fca.lexpr)
		}
		c.outfh().outnl()
		// 如果是创建新的error对象，则放行, .New() error,
		// 但太hacky了，需要更好的方式
		if fca.fnty.Results().Len() == 1 &&
			funk.Contains([]string{"New", "Wrap", "Errorf"}, fca.idtfn.Name) {
			c.outf("gxtvtoperr =nilptr").outfh().outnl()
		}

		tmphaslval := tmptyname() + "lval"
		c.outf("bool %v = %v", tmphaslval, fca.haslval).outfh().outnl()
		c.outf("if (%v && gxtvtoperr != nilptr) {", tmphaslval).outnl()
		c.outf("     gxtvtoperr_lineno = __LINE__").outfh().outnl()
		c.outf("//   goto %v  %v???", fnexc.gotolab, exi.index).outfh().outnl()
		c.outf("  gxjmpfromidx = %v", exi.index).outfh().outnl()
		c.outf("  goto %v", fnexc.gotolab).outfh().outnl()
		c.outf("}").outnl()
		c.outf("%v:; // %v noerr or jmpbak from %v",
			exi.gobaklab, exi.index, fnexc.gotolab).outfh().outnl()
		c.outf("if (gxtvtoperr != nilptr) {").outnl()
		c.outf(" // return now").outfh().outnl()
		// 当前函数无返回，或者无error返回值，则panic
		// 当前函数单返回值，直接返回err
		// 当前函数多返回值，则给其中的err字段赋值，并返回
		fnrety := upfd.Type
		if fnrety.Results == nil || len(fnrety.Results.List) == 0 {
			c.outf(" // panic").outfh().outnl()
			c.outf("panic((voidptr)0x1)").outfh().outnl()
		} else if len(fnrety.Results.List) == 1 {
			retfld := fnrety.Results.List[0]
			if fmt.Sprintf("%v", retfld.Type) == "error" {
				c.outf("return").outsp()
				c.genExpr(scope, fca.lexpr)
				c.outfh().outnl()
			} else {
				c.outf(" // panic").outfh().outnl()
			}
		} else {
			reterridx := -1
			for i := len(fnrety.Results.List) - 1; i >= 0; i++ {
				retfld := fnrety.Results.List[i]
				if fmt.Sprintf("%v", retfld.Type) == "error" {
					reterridx = i
					break
				}
			}
			if reterridx < 0 {
				c.outf(" // panic").outfh().outnl()
			} else {
				rtvname := c.multirets[upfd]
				c.outf("%v->%v = gxtvtoperr", rtvname, tmpvarname2(reterridx))
				c.outfh().outnl()
				c.outf("return %v", rtvname).outfh().outnl()
			}
		}
		c.outf("}").outnl()
	}
}

func (c *g2nc) genCallExprVaridic(scope *ast.Scope, te *ast.CallExpr) {
	// funame := te.Fun.(*ast.Ident).Name
	fca := c.getCallExprAttr(scope, te)
	gopp.Assert(fca.isvardic, "must", te.Fun)
	c.genCallExprNorm(scope, te)
}
func (c *g2nc) genTypeCtor(scope *ast.Scope, te *ast.CallExpr) {
	switch be := te.Fun.(type) {
	case *ast.Ident:
		switch be.Name {
		case "string":
			arg0 := te.Args[0]
			switch ce := arg0.(type) {
			case *ast.BasicLit:
				c.outf("cxstring_new_char(%v)", ce.Value)
			case *ast.Ident:
				varty := c.info.TypeOf(arg0)
				if isslicety2(varty) {
					c.outf("cxstring_new_cstr2((%v)->ptr, (%v)->len)", ce.Name, ce.Name)
				} else if iscstrty2(varty) {
					c.outf("cxstring_new_cstr(%v)", ce.Name)
				} else if funk.Contains(
					[]string{"voidptr", "charptr", "byteptr"}, varty.String()) {
					c.outf("cxstring_new_cstr(%v)", ce.Name)
				} else {
					c.outf("cxstring_new_char(%v)", ce.Name)
				}
			default:
				log.Println("todo", te.Fun, ce)
			}
		default:
			// log.Println("todo", te.Fun)
			c.outf("(%s)(", c.exprstr(te.Fun))
			c.genFuncArgs(scope, te.Args)
			c.outf(")")
		}
	case *ast.SelectorExpr:
		c.out("(")
		c.genExpr(scope, te.Fun)
		c.out(")")
		c.out("(")
		c.genFuncArgs(scope, te.Args)
		c.out(")")
	case *ast.ArrayType:
		c.out("cxstring_dup(")
		c.genExpr(scope, te.Args[0])
		c.out(")")
	default:
		log.Println("todo", te.Fun, be)
	}
}
func (c *g2nc) genFuncArgs(scope *ast.Scope, args []ast.Expr) {
	for idx, arg := range args {
		c.genExpr(scope, arg)
		if idx+1 < len(args) {
			c.out(",")
		}
	}
}
func (c *g2nc) genCallExprClosure(scope *ast.Scope, te *ast.CallExpr, fnlit *ast.FuncLit) {
	// funame := te.Fun.(*ast.Ident).Name
	lefte := c.valnames[te]
	selfn, isselfn := te.Fun.(*ast.SelectorExpr)
	_, isfnlit := te.Fun.(*ast.FuncLit)
	isidt := false
	iscfn := false
	ispkgsel := false
	if isselfn {
		var selidt *ast.Ident
		selidt, isidt = selfn.X.(*ast.Ident)
		iscfn = isidt && selidt.Name == "C"
		selty := c.info.TypeOf(selfn.X)
		ispkgsel = isinvalidty2(selty)
	}
	if !isselfn && isidt {
	}

	if lefte != nil {
		// {0} 只能用于初始化
		c.out("0").outfh().outnl()
	}

	closi := c.getclosinfo(fnlit)
	argtv := tmpvarname()
	if false {
		c.outf("%s%s", c.pkgpfx(), closi.argtyname).outstar().outsp().out(argtv).outeq()
		c.outf("(%s%s*)cxmalloc(sizeof(%s%s))",
			c.pkgpfx(), closi.argtyname, c.pkgpfx(), closi.argtyname).outfh().outnl()
		for _, ido := range closi.idents {
			c.out(argtv, "->", ido.Name).outeq()
			c.out(ido.Name).outfh().outnl()
		}
	}

	if lefte != nil {
		c.genExpr(scope, lefte)
		c.outeq()
	}

	_ = isselfn
	_ = iscfn
	_ = ispkgsel
	_ = isfnlit

	fnidt := te.Fun.(*ast.Ident)
	c.outf("((%s%s)(%s->fnptr))(%s->obj",
		c.pkgpfx(), closi.fntype, fnidt.Name, fnidt.Name)
	c.out(gopp.IfElseStr(len(te.Args) > 0, ",", ""))
	for idx, e1 := range te.Args {
		c.genExpr(scope, e1)
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")").outfh().outnl()
	// assign back in case modified
	if true {
		c.outf("%s%s* %s = %s->obj", c.pkgpfx(), closi.argtyname, argtv, fnidt.Name)
		c.outfh().outnl()
		for _, ido := range closi.idents {
			c.outf("if (%s->%s != %s) {", argtv, ido.Name, ido.Name)
			c.out(ido.Name).outeq()
			c.out(argtv, "->", ido.Name)
			c.outfh().outnl()
			c.out("}").outnl()
		}
	}

	// check if real need, ;\n
	cs := c.psctx.cursors[te]
	if cs.Name() != "Args" {
		// c.outfh().outnl()
	}
}

// inparam
func (c *g2nc) genCallExprClosure2(scope *ast.Scope, te *ast.CallExpr /*fnlit *ast.FuncLit*/) {
	// funame := te.Fun.(*ast.Ident).Name
	lefte := c.valnames[te]
	selfn, isselfn := te.Fun.(*ast.SelectorExpr)
	_, isfnlit := te.Fun.(*ast.FuncLit)
	isidt := false
	iscfn := false
	ispkgsel := false
	if isselfn {
		var selidt *ast.Ident
		selidt, isidt = selfn.X.(*ast.Ident)
		iscfn = isidt && selidt.Name == "C"
		selty := c.info.TypeOf(selfn.X)
		ispkgsel = isinvalidty2(selty)
	}
	if !isselfn && isidt {
	}

	if lefte != nil {
		// {0} 只能用于初始化
		c.out("0").outfh().outnl()
	}

	_ = isselfn
	_ = iscfn
	_ = ispkgsel
	_ = isfnlit
	fnidt := te.Fun.(*ast.Ident)
	tmpvname := tmpvarname()
	c.outf("gxcallable* %s =(gxcallable*)%s", tmpvname, fnidt.Name).outfh().outnl()
	if lefte != nil {
		c.genExpr(scope, lefte)
		c.outeq()
	}
	c.outf("((%s%s)(%s->fnptr))(%s->obj", "void*", "(*)()", tmpvname, tmpvname)
	c.out(gopp.IfElseStr(len(te.Args) > 0, ",", ""))
	for idx, e1 := range te.Args {
		c.genExpr(scope, e1)
		c.out(gopp.IfElseStr(idx == len(te.Args)-1, "", ", "))
	}
	c.out(")").outfh().outnl()
	// assign back in case modified
}

// chan structure args
func (c *g2nc) genChanStargs(scope *ast.Scope, e ast.Expr) {
	var elemtyname = c.chanElemTypeName(e, false)
	var elemtyname2 = c.chanElemTypeName(e, true)
	// typedef struct { int  elem; } chan_arg_int;
	c.out("typedef struct {", elemtyname, " elem;} chan_arg_"+elemtyname2).outfh().outnl()
}
func (c *g2nc) genSendStmt(scope *ast.Scope, s *ast.SendStmt) {
	// var elemtyname = c.chanElemTypeName(s.Chan, false)
	var elemtyname2 = c.chanElemTypeName(s.Chan, true)
	var chanargname = "chan_arg_" + elemtyname2
	c.out("{").outnl()
	c.outf("%s* args = (%s*)cxmalloc(sizeof(%s))",
		chanargname, chanargname, chanargname).outfh().outnl()
	c.out("args->elem = ")
	c.genExpr(scope, s.Value)
	c.outfh().outnl()
	c.outf("cxrt_chan_send(")
	c.genExpr(scope, s.Chan)
	c.out(", args)").outfh().outnl()
	c.out("}").outnl()
}
func (c *g2nc) genRecvStmt(scope *ast.Scope, e ast.Expr) {
	var elemtyname = c.chanElemTypeName(e, false)
	var elemtyname2 = c.chanElemTypeName(e, true)
	var chanargname = "chan_arg_" + elemtyname2

	varobj := scope.Lookup("varname")
	if varobj != nil {
		c.outeq().out(cuzero).outfh().outnl()
	}

	c.out("{")
	c.out("voidptr rvx = cxrt_chan_recv(")
	c.genExpr(scope, e)
	c.out(")").outfh().outnl()
	c.out(" // c = rv->v").outfh().outnl()
	c.outf("%s rvp = ((%s*)rvx)->elem", elemtyname, chanargname).outfh().outnl()

	if varobj != nil {
		c.genExpr(scope, varobj.Data.(ast.Expr)) // left
		c.out("= rvp").outfh().outnl()
	}

	c.out("}").outnl()
}
func (c *g2nc) chanElemTypeName(e ast.Expr, trimstar bool) string {
	var elemtyname = ""
	chtyx := c.info.TypeOf(e)
	switch t := chtyx.(type) {
	case *types.Chan:
		switch te := t.Elem().(type) {
		case *types.Basic:
			switch te.Kind() {
			case types.Int:
				elemtyname = "int"
			default:
				log.Println("unknown", te, te.Kind())
			}
		case *types.Pointer:
			tystr := c.exprTypeNameImpl2(nil, te, e)
			if trimstar {
				tystr = strings.Replace(tystr, "*", "p", -1)
			}
			return tystr
		default:
			log.Println("unknown", t, reflect.TypeOf(t.Elem()))
		}
	default:
		log.Println("unknown", chtyx)
	}
	if elemtyname == "" {
		log.Println("cannot resolve chan element typename", e, reflect.TypeOf(e))
	}
	return elemtyname
}
func (c *g2nc) genReturnStmt(scope *ast.Scope, e *ast.ReturnStmt) {
	fd := upfindFuncDeclNode(c.psctx, e, 0)
	ismret := fd.Type.Results.NumFields() >= 2

	if ismret {
		rtvname := c.multirets[fd]
		names := []*ast.Ident{}
		for _, fld := range fd.Type.Results.List {
			for _, name := range fld.Names {
				names = append(names, name)
			}
		}
		for idx, re := range e.Results {
			c.outf("%s->%s", rtvname.Name, tmpvarname2(idx))
			c.outeq()
			c.genExpr(scope, re)
			c.outfh().outnl()
		}
		if len(e.Results) == 0 {
			for idx, name := range names {
				c.outf("%s->%s", rtvname.Name, tmpvarname2(idx))
				c.outeq().out(name.Name)
				c.outfh().outnl()
			}
		}
		c.out("goto labmret").outfh().outnl()
	} else {
		c.genDeferStmt(scope, e)
		reses := []ast.Expr{}
		for idx, ae := range e.Results {
			if fd.Type.Results == nil {
				reses = append(reses, ae)
				continue
			}

			sigty := c.info.TypeOf(fd.Type.Results.List[idx].Type)
			resty := c.info.TypeOf(ae)
			reset := false

			switch ne := sigty.(type) {
			case *types.Named:
				if sigty != resty && isiface2(ne.Underlying()) {
					reset = true
					idt := newIdent(tmpvarname())
					reses = append(reses, idt)
					tystr := c.exprTypeName(scope, fd.Type.Results.List[idx].Type)
					c.out(tystr).outsp()
					c.genExpr(scope, idt)
					c.outeq()
					c.outf("%s_new_zero()", strings.Trim(tystr, "*")).outfh().outnl()
					undty := ne.Underlying().(*types.Interface)
					c.outf("%s->thisptr =", idt.Name)
					c.genExpr(scope, ae)
					c.outfh().outnl()
					if isnilident(ae) {
						c.out(idt.Name).outeq().out("nilptr").outfh().outnl()
						break
					}
					for i := 0; i < undty.NumMethods(); i++ {
						c.outf("%s->%s = (__typeof__(%s->%s))%s_%s", idt.Name, undty.Method(i).Name(),
							idt.Name, undty.Method(i).Name(),
							strings.Trim(c.exprTypeName(scope, ae), "*"), undty.Method(i).Name())
						c.outfh().outnl()
					}
				}
			default:
				log.Println("todo", reflect.TypeOf(sigty))
			}
			if reset {
				// reses = append(reses, ae)
			} else {
				reses = append(reses, ae)
			}
		}
		if len(reses) < len(e.Results) {
			log.Println("todo", len(reses), len(e.Results), e.Results[0])
		}
		c.out("return").outsp()
		// log.Println(len(reses), len(e.Results), e.Results[0])
		for idx, _ := range e.Results {
			if idx >= len(reses) {
				break
			}

			c.genExpr(scope, reses[idx])
			c.out(gopp.IfElseStr(idx < len(e.Results)-1, ",", ""))
		}
	}
	// c.outfh().outnl().outnl()
}

// defer 也许可以用 goto label实现
func (c *g2nc) genDeferStmtSet(scope *ast.Scope, e *ast.DeferStmt) {
	deferi := c.getdeferinfo(e)
	tvname := tmpvarname()
	c.outf("int %s = %v", tvname, deferi.idx).outfh().outnl()
	c.outf("cxarray3_append(deferarr, (voidptr)&%s)", tvname)
}
func (c *g2nc) genDeferStmt(scope *ast.Scope, e ast.Stmt) {
	dstfd := upfindFuncDeclNode(c.psctx, e, 0)
	defers := []*ast.DeferStmt{}
	for _, defero := range c.psctx.defers {
		tmpfd := upfindFuncDeclNode(c.psctx, defero, 0)
		if tmpfd != dstfd {
			continue
		}
		defers = append(defers, defero)
	}
	// log.Println("got defered return", len(defers))
	c.outf("// defer section %v", len(defers)).outnl()
	if len(defers) == 0 {
		return
	}

	c.out("{").outnl()
	c.out("int deferarrsz = cxarray3_size(deferarr)").outfh().outnl()
	c.out("for (int deferarri = deferarrsz-1; deferarri>=0; deferarri--)")
	c.out("{").outnl()
	c.out("uintptr deferarrn = 0").outfh().outnl()
	c.out("*(uintptr*)cxarray3_get_at(deferarr, deferarri)").outfh().outnl()
	for i := 0; i < len(defers); i++ {
		defero := defers[i]
		c.out(gopp.IfElseStr(i > 0, "else", "")).outsp()
		c.outf("if (deferarrn == %d)", i)
		c.out("{").outnl()
		c.genExpr(scope, defero.Call)
		c.outfh().outnl()
		c.out("}").outnl()
	}
	c.out("}").outnl()
	c.out("}").outnl()
}

func (c *g2nc) genExceptionStmt(scope *ast.Scope, e ast.Stmt) {
	dstfd := upfindFuncDeclNode(c.psctx, e, 0)
	// log.Println("got excepts return", len(defers))
	fnexc, ok := c.fnexcepts[dstfd]
	excposcnt := 0
	if ok {
		excposcnt = len(fnexc.callexes)
	}
	c.outf("// exception section %v", excposcnt).outnl()
	if !ok {
		log.Println(dstfd.Name, ok, reftyof(e))
		return
	}

	c.outf("%s:;", fnexc.gotolab).outnl()
	if len(fnexc.catchexprs) > 0 {
		c.genCatchStmt(scope, fnexc.catchexprs[0])
	}
	c.out("switch (gxjmpfromidx) {")
	for idx, exi := range fnexc.callexes {
		tmplab := exi.gobaklab
		c.outf("case %d: ", idx).outnl()
		c.outf("goto %s; break;", tmplab).outnl()
	}
	c.out("}")
}

// keepvoid
// skiplast 作用于linebrk
func (this *g2nc) genFieldList(scope *ast.Scope, flds *ast.FieldList,
	keepvoid bool, withname bool, linebrk string, skiplast bool) {

	if keepvoid && (flds == nil || flds.NumFields() == 0) {
		this.out("void")
		return
	}
	if flds == nil {
		return
	}

	for idx, fld := range flds.List {
		_, _ = idx, fld
		log.Println(fld.Type, this.exprTypeName(scope, fld.Type))
		if tyname, ok := this.psctx.functypes[fld.Type]; ok {
			this.out(tyname)
		} else {
			// this.outf("/* %v */", exprstr(fld.Type))
			this.genTypeExpr(scope, fld.Type)
		}
		this.outsp()
		if withname && len(fld.Names) > 0 {
			this.genExpr(scope, fld.Names[0])
		}
		outskip := skiplast && (idx == len(flds.List)-1)
		this.out(gopp.IfElseStr(outskip, "", linebrk))
	}
}

func (c *g2nc) genStructZeroFields(scope *ast.Scope) {
	log.Println("zero struct fields")
}

func (this *g2nc) genTypeExpr(scope *ast.Scope, e ast.Expr) {
	this.out(this.exprTypeName(scope, e))
}

func (c *g2nc) genExpr(scope *ast.Scope, e ast.Expr) {
	varname := scope.Lookup("varname")
	if varname != nil {
		vartyp := c.info.TypeOf(varname.Data.(ast.Expr))
		log.Println(vartyp, varname)
		if iseface2(vartyp) {
			_, iscallexpr := e.(*ast.CallExpr)
			_, isidt := e.(*ast.Ident)
			// _, lisidt := varname.Data.(ast.Expr).(*ast.Ident)
			if !iscallexpr && !isidt {
				// vartyp2 := reflect.TypeOf(varname.Data.(ast.Expr))
				c.outf("(cxeface*)%v", cuzero).outfh().outnl()

				tmpvar := tmpvarname()
				c.out(c.exprTypeName(scope, e), tmpvar, "=")
				ns := putscope(scope, ast.Var, "varname", newIdent(tmpvar))
				c.genExpr2(ns, e)
				c.outfh().outnl()
				c.genExpr2(scope, varname.Data.(ast.Expr))
				c.outeq()
				ety := c.info.TypeOf(e)
				switch ety.(type) {
				case *types.Interface:
					c.out(tmpvar)
				default: // convert
					c.outf("cxeface_new_of2((voidptr)&%s, sizeof(%s))", tmpvar, tmpvar)
				}
				return
			}
		}
	}
	c.genExpr2(scope, e)
}
func (this *g2nc) genExpr2(scope *ast.Scope, e ast.Expr) {
	// log.Println(reftyof(e), e)
	switch te := e.(type) {
	case *ast.Ident:
		idname := te.Name
		idname = gopp.IfElseStr(idname == "nil", "nilptr", idname)
		idname = gopp.IfElseStr(idname == "string", "cxstring*", idname)
		eobj := this.info.ObjectOf(te)
		log.Println(e, eobj, isglobalid(this.psctx, te))
		if eobj != nil {
			pkgo := eobj.Pkg()
			if pkgo != nil {
				// this.out(pkgo.Name())
			}
		}
		// TODO 要查看是否有上级,否则无法判断包前缀
		if isglobalid(this.psctx, te) {
			eobj := this.info.ObjectOf(te)
			if eobj != nil && eobj.Pkg().Name() == "C" {
			} else {
				this.out(this.pkgpfx())
			}
		}
		this.out(idname, "")
	case *ast.ArrayType:
		tystr := this.exprstr(te)
		if tystr == "[0]byte" {
			this.out("void")
			break
		}
		log.Println("todo", te, reflect.TypeOf(e), e.Pos())
		this.out(tystr)
	case *ast.StructType:
		this.genFieldList(scope, te.Fields, false, true, ";\n", false)
	case *ast.UnaryExpr:
		// log.Println(te.Op.String(), te.X)
		switch te.Op {
		case token.ARROW:
			this.genRecvStmt(scope, te.X)
			return
		default:
			// log.Println("unknown", te.Op.String())
		}
		keepop := true
		switch t2 := te.X.(type) {
		case *ast.CompositeLit:
			if iscsel(t2.Type) {
				ste := t2.Type.(*ast.SelectorExpr)
				// this.outf("// c struct ctor %s", ste.Sel.Name)
				this.outf("cxmalloc(sizeof(%s))", ste.Sel.Name)
				keepop = false
				break
			}

			tystr := this.exprTypeName(scope, t2.Type)
			this.outf("%s_new_zero()", tystr) //.outnl()
			this.outfh().outnl()
			keepop = false
			varname := scope.Lookup("varname")

			goty := this.info.TypeOf(t2).(*types.Named).Underlying().(*types.Struct)
			for idx, elmx := range t2.Elts {
				// log.Println(elmx, goty, goty.Field(idx), reflect.TypeOf(elmx))
				switch elme := elmx.(type) {
				case *ast.KeyValueExpr:
					this.outf("%s->%s", varname.Data, elme.Key)
					this.outeq()
					this.genExpr(scope, elme.Value)
					this.outfh().outnl()
				default:
					fld := goty.Field(idx)
					this.outf("%s->%s", varname.Data, fld.Name())
					this.outeq()
					this.genExpr(scope, elmx)
					this.outfh().outnl()
				}
			}
		case *ast.UnaryExpr:
			log.Println(t2, t2.X, t2.Op)
		default:
			log.Println(reflect.TypeOf(te), t2, reflect.TypeOf(te.X), te.Pos())
		}
		if keepop {
			this.outf("%v", te.Op.String())
			this.genExpr(scope, te.X)
		}
	case *ast.CompositeLit:
		switch be := te.Type.(type) {
		case *ast.MapType:
			this.outf("cxhashtable_new()").outfh().outnl()
			var vo = scope.Lookup("varname")
			for idx, ex := range te.Elts {
				switch be := ex.(type) {
				case *ast.KeyValueExpr:
					this.genCxmapAddkv(scope, vo.Data, be.Key, be.Value)
					this.outfh().outnl()
				default:
					log.Println("unknown", idx, reflect.TypeOf(ex))
				}
			}
		case *ast.ArrayType:
			var vo = scope.Lookup("varname")
			if vo == nil {
				gotyval := this.info.Types[te]
				log.Println("temp var?", vo, this.exprpos(te), gotyval)
			}
			bety := this.info.TypeOf(be.Elt)
			etystr := this.exprTypeNameImpl2(scope, bety, be.Elt)
			this.outf("cxarray3_new(0, sizeof(%s))", etystr).outfh().outnl()
			for idx, ex := range te.Elts {
				log.Println(vo == nil, ex, idx, this.exprpos(ex))
				this.genCxarrAdd(scope, vo.Data, ex, idx)
				this.outfh().outnl()
			}
			if be == nil {
			}
		case *ast.Ident: // TODO
			var vo = scope.Lookup("varname")
			this.outf("%v_new_zero()", this.exprTypeName(scope, be)).outfh().outnl()
			for _, ex := range te.Elts {
				this.outf("%v->%v = %v", vo, "todoaaa", ex)
				this.outfh().outnl()
			}
		default:
			log.Println("todo", te.Type, reflect.TypeOf(te.Type))
		}

	case *ast.CallExpr:
		if idto, ok := te.Fun.(*ast.Ident); ok {
			funame := idto.Name
			if funk.Contains([]string{"_cgoCheckPointer"}, funame) {
				this.out("//").outsp()
			}
		}
		this.genCallExpr(scope, te)
	case *ast.BasicLit:
		// log.Println(e, exprstr(e), reftyof(e), te.Value)
		ety := this.info.TypeOf(e)
		if ety == nil { // we created
			this.out(te.Value)
			break
		}
		switch t := ety.Underlying().(type) {
		case *types.Basic:
			switch t.Kind() {
			case types.Int, types.UntypedInt, types.UntypedRune,
				types.Uint, types.Int64, types.Uint64:
				this.out(te.Value)
			case types.String, types.UntypedString:
				this.out(fmt.Sprintf("cxstring_new_cstr(%s)", te.Value))
			case types.Float64, types.Float32, types.UntypedFloat:
				this.out(te.Value)
			case types.Uint8, types.Int8, types.Uint32, types.Int32:
				this.out(te.Value)
			case types.Uintptr, types.Voidptr:
				this.out(te.Value)
			default:
				if isctydeftype2(t) {
					this.out(te.Value)
				} else {
					this.outf("unknown %v", e)
					log.Println("unknown", t.String())
				}
			}
		default:
			this.outf("unknown %v", e)
			log.Println("unknown", e, t, reflect.TypeOf(t))
		}
	case *ast.BinaryExpr:
		opty := this.info.TypeOf(te.X)
		if isstrty2(opty) {
			switch te.Op {
			case token.EQL:
				this.out("cxstring_eq(")
			case token.NEQ:
				this.out("cxstring_ne(")
			case token.ADD:
				this.out("cxstring_add(")
			default:
				log.Println("todo", te.Op)
			}
			this.genExpr(scope, te.X)
			this.out(",")
			this.genExpr(scope, te.Y)
			this.out(")")
		} else {
			this.genExpr(scope, te.X)
			this.out(te.Op.String())
			this.genExpr(scope, te.Y)
		}
	case *ast.ChanType:
		this.out("voidptr")
	case *ast.IndexExpr:
		varty := this.info.TypeOf(te.X)
		vo := scope.Lookup("varval")
		if varty == nil { // c type ???
			// gopp.Assert(1 == 2, "waitdep", te)
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if isctydeftype2(varty) {
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if ismapty(varty.String()) {
			if vo == nil {
				this.genCxmapGetkv(scope, te.X, te.Index, nil)
			} else {
				this.genCxmapAddkv(scope, te.X, te.Index, vo.Data)
			}
		} else if isslicety(varty.String()) {
			// get or set?
			if vo == nil { // right value
				this.genCxarrGet(scope, te.X, te.Index, varty)
			} else { // left value
				this.genCxarrSet(scope, te.X, te.Index, vo.Data)
			}
		} else if isstrty(varty.String()) {
			if vo == nil { // right value
				this.out("((cxstring*)")
				this.genExpr(scope, te.X)
				this.out(")->ptr[")
				this.genExpr(scope, te.Index)
				this.out("]")
			} else { // left value
				this.out("((cxstring*)")
				this.genExpr(scope.Outer, te.X) // temporarily left value
				this.out(")->ptr[")
				this.genExpr(scope, te.Index)
				this.out("]")
				this.out("=")
				this.genExpr(scope, vo.Data.(ast.Expr))
			}
		} else if isinvalidty2(varty) { // index of c type???
			gopp.Assert(1 == 2, "waitdep", varty)
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if varty.String() == "byte" { // multiple dimission index of c type???
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if varty.String() == "*byteptr" { // multiple dimission index of c type???
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if realty, ok := varty.(*types.Basic); ok &&
			realty.Kind() == types.Byteptr { // multiple dimission index of c type???
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else if _, ok := varty.(*types.Pointer); ok {
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("]")
		} else {
			this.genExpr(scope, te.X)
			this.out("[")
			this.genExpr(scope, te.Index)
			this.out("] /*warn?*/")
			log.Println("todo", te.X, te.Index, exprstr(te))
			log.Println("todo", reftyof(te.X), varty, reftyof(varty), this.exprpos(te.X))
		}
	case *ast.SliceExpr:
		varty := this.info.TypeOf(te.X)
		lowe := te.Low
		highe := te.High
		if lowe == nil {
			lowe = newLitInt(0)
		}
		if isstrty2(varty) {
			this.outf("cxstring_sub(%v, ", te.X)
			this.genExpr(scope, lowe)
			this.out(",")

			if highe == nil {
				this.outf("(%v)->len", te.X)
			} else {
				this.genExpr(scope, te.High)
			}
			this.out(")")
		} else if isslicety2(varty) {
			this.outf("cxarray3_slice(")
			this.genExpr(scope, te.X)
			this.out(",")
			this.genExpr(scope, lowe)
			this.out(",")

			if highe == nil {
				this.outf("cxarray3_size(")
				this.genExpr(scope, te.X)
				this.out(")")
			} else {
				this.genExpr(scope, te.High)
			}
			this.out(")")
		} else {
			log.Println("todo", varty, te)
		}
	case *ast.SelectorExpr:
		if iscsel(te) {
		} else {
			this.genExpr(scope, te.X)
			selxty := this.info.TypeOf(te.X)
			log.Println(selxty, reflect.TypeOf(selxty), te.X, te.Sel)
			if selxty == nil {
				gopp.Assert(1 == 2, "waitdep", te)
				// c type?
				this.out(". /* c struct selctorexpr */")
			} else if isinvalidty2(selxty) && ispackage(this.psctx, te.X) { // package
				this.out(pkgsep)
			} else {
				switch selxty.(type) {
				case *types.Named:
					this.out(".")
				case *types.Pointer:
					this.out("->")
				default:
					if isctydeftype2(selxty) {
						this.out(".")
					} else {
						this.out("->")
					}
				}
			}
		}
		// this.genExpr(scope, te.Sel)
		this.out(te.Sel.Name)
		this.outsp()
	case *ast.StarExpr:
		idt, isidt := te.X.(*ast.Ident)
		if isidt {
			varobj := this.psctx.info.ObjectOf(idt)
			if istypety(varobj.String()) {
				this.genExpr(scope, te.X)
				this.out("*")
			} else if isvarty(varobj.String()) {
				this.out("*")
				this.genExpr(scope, te.X)
			} else {
				log.Println("todo", varobj.Type(), varobj.String())
			}
		} else {
			this.out("*")
			this.genExpr(scope, te.X)
		}
	case *ast.InterfaceType:
		if te.Methods != nil && te.Methods.NumFields() > 0 {
			this.out("cxiface")
		} else {
			this.out("cxeface")
		}
	case *ast.TypeAssertExpr:
		tystr := this.exprTypeName(scope, te.Type)
		this.outf("(%s)(", tystr)
		this.genExpr(scope, te.X)
		this.out("->thisptr)")
	case *ast.ParenExpr:
		this.out("(")
		this.genExpr(scope, te.X)
		this.out(")")
	case *ast.FuncLit:
		closi := this.getclosinfo(te)
		this.outf("%s%s", this.pkgpfx(), closi.fnname).outfh().outnl()
	default:
		this.outf("unknown %v", e)
		log.Println("unknown", reflect.TypeOf(e), e, te)
	}
}
func (c *g2nc) genCxmapAddkv(scope *ast.Scope, vnamex interface{}, ke ast.Expr, vei interface{}) {
	// vei == nil, then get
	keystr := ""
	switch be := ke.(type) {
	case *ast.BasicLit:
		switch be.Kind {
		case token.STRING:
			keystr = fmt.Sprintf("cxstring_new_cstr(%s)", be.Value)
		default:
			// log.Println("unknown index key kind", be.Kind)
			keystr = fmt.Sprintf("%v", be.Value)
		}
	case *ast.Ident:
		isglob := isglobalid(c.psctx, be)
		pkgpfx := pkgpfxof(c.psctx, ke)
		pkgpfx = gopp.IfElseStr(isglob, pkgpfx, "")
		varty := c.info.TypeOf(ke)
		switch varty.String() {
		case "string":
			keystr = pkgpfx + be.Name
		default:
			keystr = pkgpfx + be.Name
			// log.Println("unknown", varty, ke)
		}
	case *ast.SelectorExpr:
		varty := c.info.TypeOf(ke)
		switch varty.String() {
		case "string":
			sym := fmt.Sprintf("%v->%v", be.X, be.Sel)
			keystr = sym
		default:
			log.Println("unknown", varty, ke)
		}
	default:
		log.Println("unknown index key", ke, reflect.TypeOf(ke))
	}

	valstr := ""
	switch be := vei.(type) {
	case *ast.BasicLit:
		valstr = be.Value
	case *ast.Ident:
		valstr = be.Name
	default:
		log.Println("unknown", vei, reflect.TypeOf(ke), reflect.TypeOf(vei))
	}

	c.outf("hashtable_add(")
	c.genExpr(scope, vnamex.(ast.Expr))
	c.outf(", (voidptr)(uintptr)%v,", keystr)
	c.out("(voidptr)(uintptr)(")
	switch ve := vei.(type) {
	case ast.Expr:
		c.genExpr(scope, ve)
	default:
		if valstr == "" {
			c.outf("%v", vei)
		} else {
			c.out(valstr)
		}
	}
	c.outf(")) /* %v */", valstr) // .outfh().outnl()
}
func (c *g2nc) genCxmapGetkv(scope *ast.Scope, vnamex interface{}, ke ast.Expr, vei interface{}) {
	// vei == nil, then get
	gopp.Assert(vei == nil, "wtfff", vei)

	keystr := ""
	switch be := ke.(type) {
	case *ast.BasicLit:
		switch be.Kind {
		case token.STRING:
			keystr = fmt.Sprintf("cxstring_new_cstr(%s)", be.Value)
		default:
			// log.Println("unknown index key kind", be.Kind)
			keystr = fmt.Sprintf("%v", be.Value)
		}
	case *ast.Ident:
		varty := c.info.TypeOf(ke)
		switch varty.String() {
		case "string":
			keystr = be.Name
		default:
			log.Println("unknown", varty, ke)
		}
	case *ast.SelectorExpr:
		varty := c.info.TypeOf(ke)
		switch varty.String() {
		case "string":
			sym := fmt.Sprintf("%v->%v", be.X, be.Sel)
			keystr = sym
		default:
			log.Println("unknown", varty, ke)
		}
	default:
		log.Println("unknown index key", ke, reflect.TypeOf(ke))
	}

	varobj := scope.Lookup("varname")
	tmpname := tmpvarname()
	if varobj == nil {
		c.outf("voidptr %v =", tmpname).outsp()
	}
	c.out(cuzero).outfh().outnl()

	c.outf("int %v =", tmpvarname()).outsp()
	c.outf("hashtable_get(")
	c.genExpr(scope, vnamex.(ast.Expr))
	if false { // waitdep
		c.outf(", (voidptr)(uintptr)%v,", keystr)
	}
	c.out(",")
	c.genExpr(scope, ke)
	c.out(", (voidptr*)(&(")
	if varobj == nil {
		c.outf("%v", tmpname)
	} else {
		c.outf("%v", varobj.Data)
	}
	c.outf(")))") // .outfh().outnl()
}

func (c *g2nc) genCxarrAdd(scope *ast.Scope, vnamex interface{}, ve ast.Expr, idx int) {
	// log.Println(vnamex, ve, idx)
	tyname := c.exprTypeName(scope, ve)
	tmpname := tmpvarname()
	c.outf("%v %v = ", tyname, tmpname)
	c.genExpr(scope, ve)
	c.outfh().outnl()

	varobj := c.info.ObjectOf(vnamex.(*ast.Ident))
	pkgpfx := gopp.IfElseStr(isglobalid(c.psctx, vnamex.(*ast.Ident)), varobj.Pkg().Name(), "")
	pkgpfx = gopp.IfElseStr(pkgpfx == "", "", pkgpfx+pkgsep)
	c.outf("cxarray3_append(%s%v, ", pkgpfx, vnamex.(*ast.Ident).Name)
	c.outf("(voidptr)&%v)", tmpname) // .outfh().outnl()
}
func (c *g2nc) genCxarrSet(scope *ast.Scope, vname ast.Expr, vidx ast.Expr, elem interface{}) {
	tname := tmpvarname()
	c.out(c.exprTypeName(scope, elem.(ast.Expr))).outsp()
	c.out(tname).outeq()
	c.genExpr(scope, elem.(ast.Expr))
	c.outfh().outnl()
	c.outf("cxarray3_replace_at(")
	c.genExpr(scope, vname)
	c.outf(", (voidptr)(uintptr)&")
	c.out(tname)
	c.out(",")
	c.genExpr(scope, vidx)
	c.outf(", nilptr)").outfh().outnl()
}
func (c *g2nc) genCxarrGet(scope *ast.Scope, vname ast.Expr, vidx ast.Expr, varty types.Type) {
	var elemty types.Type
	switch arrty := varty.(type) {
	case *types.Slice:
		elemty = arrty.Elem()
	case *types.Array:
		elemty = arrty.Elem()
	}

	if false { //TODO
		asobj := scope.Lookup("varname")
		if asobj != nil {
			c.out(cuzero).outfh().outnl()
		}
		// insert bound check code
		c.out("if ((")
		c.genExpr(scope, vidx)
		c.out(")>")
		c.out("cxarray3_size(")
		c.genExpr(scope, vname)
		c.out(")) {").outnl()
		c.out(" // out of array bound").outnl()
		c.out("}").outnl()

		if asobj != nil {
			c.outf("%v", asobj.Data).outeq()
		}
	}
	tystr := c.exprTypeName(scope, vname)
	tystr = c.exprTypeNameImpl2(scope, elemty, nil)
	c.outf("*(%v*)", tystr)
	c.outf("cxarray3_get_at(")
	c.genExpr(scope, vname)
	c.out(",")
	c.genExpr(scope, vidx)
	c.out(")").outnl()
}
func (this *g2nc) exprTypeName(scope *ast.Scope, e ast.Expr) string {
	// log.Println(e, reflect.TypeOf(e))
	tyname := this.exprTypeNameImpl(scope, e)
	// log.Println(exprstr(e), reftyof(e), tyname)
	if tyname == "unknownty" {
		// log.Panicln(tyname, e, reflect.TypeOf(e), this.exprpos(e))
	}
	if strings.Contains(tyname, "literal)") {
		log.Panicln(tyname, e, reflect.TypeOf(e), this.exprpos(e))
	}
	return tyname
}
func (this *g2nc) exprTypeNameImpl(scope *ast.Scope, e ast.Expr) string {

	{
		// return "unknownty"
	}

	goty := this.info.TypeOf(e)
	if goty == nil {
		if ie, ok := e.(*ast.CallExpr); ok {
			log.Println(ie.Fun, reftyof(ie.Fun), this.info.TypeOf(ie.Fun))
			if _, ok2 := ie.Fun.(*ast.ParenExpr); ok2 {
				goty = this.info.TypeOf(ie.Fun)
			}
		}
	}
	if goty == nil {
		if te, ok := e.(*ast.StarExpr); ok {
			goty = this.info.TypeOf(te.X)
			log.Println(e, exprstr(e), reftyof(e), this.exprpos(e), goty, reftyof(goty))
		}
		log.Println(e, exprstr(e), reftyof(e), this.exprpos(e))
		if exprstr(e) == "(bad expr)" {
			return "int"
		}
		if exprstr(e) == mthsep {
			return "int"
		}
		log.Panicln(e, exprstr(e), reftyof(e), this.exprpos(e), goty)
	}
	val := this.exprTypeNameImpl2(scope, goty, e)
	if isinvalidty(val) {
		// log.Panicln("unreachable")
		val = this.exprstr(e)
		val = strings.Replace(val, "C.", "", 1)
		log.Println(val, exprstr(e))
		// log.Panicln(e, iscsel(e), this.exprpos(e), this.exprstr(e), sign2rety(val))
		return sign2rety(val)
	}
	return val
}
func (this *g2nc) exprTypeNameImpl2(scope *ast.Scope, ety types.Type, e ast.Expr) string {

	{
		// return "unknownty"
	}

	goty := ety
	tyval, isudty := this.strtypes[goty.String()]
	log.Println(goty, reftyof(goty), e, reftyof(e), exprstr(e))

	switch te := goty.(type) {
	case *types.Basic:
		if te.Kind() == types.Invalid {
			return "typpp_" + strings.ReplaceAll(te.String(), " ", "_")
		}
		if isstrty(te.Name()) {
			return "cxstring*"
		} else {
			if strings.Contains(te.Name(), "string") {
				log.Println(te.Name())
			}
			// log.Println(te, reftyof(e), te.Info(), te.Name(), te.Underlying(), reftyof(te.Underlying()))
			tystr := strings.Replace(te.String(), ".", pkgsep, 1)
			if strings.HasPrefix(tystr, "untyped ") {
				tystr = tystr[8:]
			}
			return tystr // + "/*jjj*/"
			// return te.Name()
		}
	case *types.Named:
		teobj := te.Obj()
		pkgo := teobj.Pkg()
		undty := te.Underlying()
		log.Println(teobj, pkgo, undty, reftyof(undty))
		switch ne := undty.(type) {
		case *types.Interface:
			if pkgo == nil { // builtin???
				return teobj.Name() + "*"
			}
			return fmt.Sprintf("%s%s%s*", pkgo.Name(), pkgsep, teobj.Name())
		case *types.Struct:
			tyname := teobj.Name()
			if pkgo.Name() == "C" {
				return fmt.Sprintf("%s", tyname)
			}
			return fmt.Sprintf("%s%s%s", pkgo.Name(), pkgsep, teobj.Name())
		case *types.Basic:
			if pkgo.Name() == "C" {
				return undty.String()
			}
			return fmt.Sprintf("%s%s%s", pkgo.Name(), pkgsep, teobj.Name())
		case *types.Array:
			// tyname := teobj.Name()
		default:
			gopp.G_USED(ne)
		}
		log.Println("todo", teobj.Name(), reflect.TypeOf(undty), goty)
		return "/*todo*/" + teobj.Name()
		// return sign2rety(te.String())
	case *types.Pointer:
		tystr := this.exprTypeNameImpl2(scope, te.Elem(), e)
		tystr += "*"
		// log.Println(tystr, reftyof(te.Elem()))
		return tystr
	case *types.Slice:
		tystr := te.String()
		if tystr == "[0]byte" {
			return "void"
		}
		return "builtin__cxarray3*"
	case *types.Array:
		tystr := te.String()
		if tystr == "[0]byte" {
			return "void"
		}
		return "builtin__cxarray3*"
	case *types.Chan:
		return "voidptr"
	case *types.Map:
		return "HashTable*"
	case *types.Signature:
		switch fe := e.(type) {
		case *ast.FuncLit:
			if closi, ok := this.closidx[fe]; ok {
				return this.pkgpfx() + closi.fntype
			} else {
				log.Println("todo", goty, reflect.TypeOf(goty), isudty, tyval, te)
			}
		case *ast.Ident:
			return te.String()
		case *ast.FuncType:
			if tyname, ok := this.psctx.functypes[fe]; ok {
				return tyname + "/*111*/"
			}
		}
		for fntyx, tyname := range this.psctx.functypes {
			fnty := this.info.TypeOf(fntyx)
			// log.Println(te.String(), fmt.Sprintf("%v", fnty), te == fnty, tyname)
			if te == fnty {
				return tyname + fmt.Sprintf("/*222 %v*/", te.String())
			}
		}

		// log.Println(reftyof(e), exprstr(e), reftyof(te))
		return te.String() + "/* 333 */"
	case *types.Interface:
		return "cxeface*"
	case *types.Tuple:
		// log.Println(e, reflect.TypeOf(e), te.String(), pkgpfxof(this.psctx, e))
		switch ce := e.(type) {
		case *ast.CallExpr:
			exstr := exprstr(ce.Fun)
			if ipos := strings.Index(exstr, "."); ipos > 0 {
				exstr = exstr[ipos+1:]
			}
			pkgpfx := pkgpfxof(this.psctx, e)
			return fmt.Sprintf("%s%v_multiret_arg*", pkgpfx, exstr)
		case *ast.TypeAssertExpr:
			return fmt.Sprintf("%v_multiret_arg*", "todoaaa")
		default:
			log.Println("todo", goty, reflect.TypeOf(goty), isudty, tyval, te, this.exprpos(e))
		}
	default:
		log.Println("todo", goty, exprstr(e), reftyof(goty), isudty, tyval, te, this.exprpos(e))
		return te.String() + "/*todo*/"
	}

	panic("unreachable")
}
func (this *g2nc) exprTypeFmt(scope *ast.Scope, e ast.Expr) string {
	goty := this.info.TypeOf(e)
	if goty == nil {
		// maybe not exist func? like c function?
		return "d-nilty"
	}
	tyval, isudty := this.strtypes[goty.String()]
	// log.Println(goty, reflect.TypeOf(goty), tyval, isudty, e, reflect.TypeOf(e))

	switch te := goty.(type) {
	case *types.Basic:
		if isstrty(te.Name()) {
			return ".*s"
		} else {
			switch te.Kind() {
			case types.Float32, types.Float64:
				return "g" // wow
				// return "f"
			case types.Byteptr:
				return "s"
			case types.Voidptr:
				return "p"
			default:
				log.Println(exprstr(e), te, te.Kind(), goty)
				return "d"
			}
		}
	case *types.Named:
		return "p"
	case *types.Pointer:
		return "p"
	case *types.Slice, *types.Array:
		return "p"
	case *types.Map:
		return "p"
	default:
		log.Println(goty, reflect.TypeOf(goty), isudty, tyval, te)
		return "d-wt"
	}

	panic("unreachable")
}

func (c *g2nc) genPredefTypeDecl(scope *ast.Scope, d *ast.GenDecl) {
	for _, spec := range d.Specs {
		switch tspec := spec.(type) {
		case *ast.TypeSpec:
			switch tspec.Type.(type) {
			case *ast.StructType:
				c.outf("// %v", exprpos(c.psctx, tspec)).outnl()
				specname := tspec.Name.Name
				c.outf("typedef struct %s%s %s%s /*hhh*/",
					c.pkgpfx(), specname, c.pkgpfx(), specname).outfh().outnl()
			case *ast.Ident:
				log.Println(tspec.Type, reflect.TypeOf(tspec.Type))
				tystr := c.exprTypeName(scope, tspec.Type)
				specname := trimCtype(tspec.Name.Name)
				c.outf("typedef %v %s%v/*eee*/", tystr, c.pkgpfx(), specname).outfh().outnl()
				// this.outf("typedef %v %s%v", spec.Type, this.pkgpfx(), spec.Name.Name).outfh().outnl()
			}
		}
	}
}
func (c *g2nc) genFunctypesDecl(scope *ast.Scope) {
	c.outf("// functypes %d in %v", len(c.psctx.functypes), c.pkgpfx()).outnl()
	for fntyx, name := range c.psctx.functypes {
		fnty := fntyx.(*ast.FuncType)
		c.outf("// %v %v", exprstr(fnty), name).outnl()
		retcnt := 0
		if fnty.Results != nil {
			retcnt = len(fnty.Results.List)
		}
		if retcnt > 1 {
			log.Fatalln("not support multirets functypes", exprpos(c.psctx, fntyx))
			c.outf("// notimpl multirets").outnl().out("//").outsp()
		}
		c.outf("typedef /*ddd*/").outsp()
		if fnty.Results != nil {
			for idx, fldo := range fnty.Results.List {
				tystr := c.exprTypeName(scope, fldo.Type)
				c.outf("%v%s", tystr, gopp.IfElseStr(idx == retcnt-1, "", ","))
			}
		} else {
			c.outf("void").outsp()
		}
		c.outf("(*%v)(", name)
		prmcnt := len(fnty.Params.List)
		for idx, fldo := range fnty.Params.List {
			tystr := c.exprTypeName(scope, fldo.Type)
			c.outf("%v%s", tystr, gopp.IfElseStr(idx == prmcnt-1, "", ","))
		}
		c.out(")")
		c.outfh().outnl()
	}
	c.outnl()
}
func (this *g2nc) genGenDecl(scope *ast.Scope, d *ast.GenDecl) {
	// log.Println(d.Tok, d.Specs, len(d.Specs), d.Tok.IsKeyword(), d.Tok.IsLiteral(), d.Tok.IsOperator())
	for idx, spec := range d.Specs {
		switch tspec := spec.(type) {
		case *ast.TypeSpec:
			this.genTypeSpec(scope, tspec)
		case *ast.ValueSpec:
			this.genValueSpec(scope, tspec, idx)
		case *ast.ImportSpec:
			// log.Println("todo", reflect.TypeOf(d), reflect.TypeOf(spec), tspec.Path, tspec.Name)
			this.outf("// import %v by %s", tspec.Path, this.exprpos(tspec)).outnl().outnl()
			// log.Println(tspec.Comment)
		default:
			log.Println("unknown", reflect.TypeOf(d), reflect.TypeOf(spec))
		}
	}
}
func (this *g2nc) genTypeSpec(scope *ast.Scope, spec *ast.TypeSpec) {
	log.Println(spec.Type, reflect.TypeOf(spec.Type), spec.Name)
	this.outf("// %s", this.exprpos(spec).String()).outnl()
	switch te := spec.Type.(type) {
	case *ast.StructType:
		specname := trimCtype(spec.Name.Name)
		this.outf("typedef struct %s%s %s%s",
			this.pkgpfx(), specname, this.pkgpfx(), specname).outfh().outnl()
		this.outf("struct %s%s {", this.pkgpfx(), specname)
		this.outnl()
		this.genFieldList(scope, te.Fields, false, true, ";", false)
		this.out("}").outfh().outnl()
		this.outnl()
		this.outf("static const _metatype %s%s_metatype = {", this.pkgpfx(), specname)
		this.outnl()
		this.outf(".kind = %d,", reflect.Struct).outnl()
		this.outf(".size = sizeof(%s%s),", this.pkgpfx(), specname).outnl()
		this.outf(".align = alignof(%s%s),", this.pkgpfx(), specname).outnl()
		this.outf(".tystr = \"%s%s\"", this.pkgpfx(), specname).outnl()
		this.out("}").outfh().outnl()
		this.outnl()
		// this.out("static").outsp()
		this.outf("%s%s* %s%s_new_zero() {",
			this.pkgpfx(), specname, this.pkgpfx(), specname).outnl()
		this.outf("  %s%s* obj = (%s%s*)cxmalloc(sizeof(%s%s))",
			this.pkgpfx(), specname, this.pkgpfx(), specname,
			this.pkgpfx(), specname).outfh().outnl()
		for _, fld := range te.Fields.List {
			fldty := this.info.TypeOf(fld.Type)
			for _, fldname := range fld.Names {
				if isstrty2(fldty) {
					this.outf("obj->%s = cxstring_new()", fldname.Name).outfh().outnl()
				} else if isslicety2(fldty) {
					elemty := fldty.(*types.Slice).Elem()
					tystr := this.exprTypeNameImpl2(scope, elemty, nil)
					this.outf("obj->%s = cxarray3_new(0, sizeof(%s))", fldname.Name, tystr).outfh().outnl()
				} else if ismapty2(fldty) {
					this.outf("obj->%s = cxhashtable_new()", fldname.Name).outfh().outnl()
				} else if ischanty2(fldty) {
					log.Println("how to", fld.Type.(*ast.ChanType).Value)
					this.outf("obj->%s = cxrt_chan_new(0)", fldname.Name).outfh().outnl()
				} else if isstructty2(fldty) {
					tystr := this.exprTypeNameImpl2(scope, fldty, nil)
					tystr = strings.TrimRight(tystr, "*")
					// TODO function semantic order
					this.outf("extern %s* %s_new_zero()", tystr, tystr).outfh().outnl()
					// 像有些地方使用结构体字段是否nil做判断,
					// 并且即使像python也不会自动初始化结构体成员的，默认是None
					this.out("//").outsp()
					this.outf("obj->%s = (voidptr) %s_new_zero()", fldname.Name, tystr).outfh().outnl()
				}
			}
		}
		this.out("  return obj").outfh().outnl()
		this.out("}").outnl()
		this.outnl()
	case *ast.Ident:
		log.Println(spec.Type, reflect.TypeOf(spec.Type), te)
		tystr := this.exprTypeName(scope, spec.Type)
		specname := trimCtype(spec.Name.Name)
		this.outf("typedef %v %s%v/*111*/", tystr, this.pkgpfx(), specname).outfh().outnl()
		// this.outf("typedef %v %s%v", spec.Type, this.pkgpfx(), spec.Name.Name).outfh().outnl()
	case *ast.StarExpr:
		log.Println(spec.Type, reflect.TypeOf(spec.Type), te.X, reflect.TypeOf(te.X), spec.Name)
		this.out("typedef").outsp()
		// this.out(this.pkgpfx())
		// this.genExpr(scope, te.X)
		this.out(this.exprTypeName(scope, te.X))
		this.out("*").outsp()
		specname := trimCtype(spec.Name.Name)
		this.out(this.pkgpfx() + specname)
		this.outfh().outnl()
	case *ast.SelectorExpr:
		this.out("typedef").outsp()
		this.genExpr(scope, te.X)
		this.out("_")
		this.genExpr(scope, te.Sel)
		this.outsp()
		specname := trimCtype(spec.Name.Name)
		this.out(this.pkgpfx() + specname)
		this.outfh().outnl()
	case *ast.InterfaceType:
		this.outf("typedef struct %s%s %s%s", this.pkgpfx(), spec.Name,
			this.pkgpfx(), spec.Name).outfh().outnl()
		this.outf("struct %s%s {", this.pkgpfx(), spec.Name).outnl()
		this.out("voidptr thisptr").outfh().outnl()
		for _, fld := range te.Methods.List {
			switch fldty := fld.Type.(type) {
			case *ast.FuncType:
				for _, name := range fld.Names {
					this.genFieldList(scope, fldty.Results, true, false, "", true)
					this.outf("(*%s)(voidptr", name.Name)
					this.out(gopp.IfElseStr(fldty.Params.NumFields() > 0, ",", ""))
					this.genFieldList(scope, fldty.Params, false, false, "", true)
					this.out(")").outfh().outnl()
				}
			default:
				log.Println("todo", fld.Type, reflect.TypeOf(fld.Type))
			}
		}
		this.out("}").outfh().outnl()
		this.outf("%s%s* %s%s_new_zero() {", this.pkgpfx(), spec.Name.Name, this.pkgpfx(), spec.Name.Name)
		this.outf("return").outsp()
		this.outf("(%s%s*)cxmalloc(sizeof(%s%s))", this.pkgpfx(), spec.Name.Name, this.pkgpfx(), spec.Name.Name)
		this.outfh().outnl()
		this.out("}").outnl().outnl()
	case *ast.ArrayType:
		log.Println("todo", spec.Name, spec.Type, reflect.TypeOf(spec.Type), te)
		log.Println("todo", te.Elt, te.Len, this.exprstr(te))
		tystr := this.exprstr(te)
		if tystr == "[0]byte" {
			log.Println("todo", "hehe")
			// this.out("void...")
			break
		}
		this.out("todo", tystr)
	default:
		log.Println("todo", spec.Name, spec.Type, reflect.TypeOf(spec.Type), te)
	}
}
func putscope(scope *ast.Scope, k ast.ObjKind, name string, value interface{}) *ast.Scope {
	var pscope = ast.NewScope(scope)
	var varobj = ast.NewObj(k, name)
	varobj.Data = value
	pscope.Insert(varobj)
	return pscope
}

// TODO depcreate
var vp1stval ast.Expr
var vp1stty types.Type
var vp1stidx int

func (c *g2nc) genValueSpec(scope *ast.Scope, spec *ast.ValueSpec, validx int) {
	cs := c.psctx.cursors[spec]
	pcs := cs.Parent()
	isconst := false
	if d, ok := pcs.(*ast.GenDecl); ok {
		isconst = d.Tok == token.CONST
	}
	isglobvar := c.psctx.isglobal(spec)

	varcnt := len(spec.Names)
	for idx, varname := range spec.Names {
		varty := c.info.TypeOf(spec.Type)
		if varty == nil && idx < len(spec.Values) {
			varty = c.info.TypeOf(spec.Values[idx])
		}
		if varty == nil && validx > 0 {
			varty = vp1stty
		}
		if varty == nil && idx < len(spec.Values) &&
			strings.HasPrefix(types.ExprString(spec.Values[0]), "C.") {
			varty = types.Typ[types.UntypedInt]
		}
		if varty == nil {
			varty = types.Typ[types.UntypedInt]
			log.Println("todo", spec.Values[idx], types.ExprString(spec.Values[idx]))
		}
		if varty == nil {
			panic("ddd")
		}
		if validx == 0 {
			vp1stty = varty
		}
		if len(spec.Values) > 0 {
			vp1stval = spec.Values[0]
			vp1stidx = validx
		}

		log.Println(varty, varname, reflect.TypeOf(varty))
		c.clinema(spec)
		vartystr := c.exprTypeNameImpl2(scope, varty, varname)
		c.out(gopp.IfElseStr(isglobvar, "static", "")).outsp()
		// comment for less warning of const qualify
		c.out(gopp.IfElseStr(isconst, "/*const*/", "")).outsp()
		if strings.HasPrefix(varty.String(), "untyped ") {
			c.out(sign2rety(varty.String())).outsp()
		} else {
			c.out("/*var*/").outsp()
			if isstrty2(varty) {
				c.out("cxstring*")
			} else if isarrayty2(varty) || isslicety2(varty) {
				c.out("builtin__cxarray3*")
			} else if ismapty2(varty) {
				c.out("HashTable*")
			} else if ischanty2(varty) {
				c.out("voidptr")
			} else if strings.Contains(vartystr, "func(") {
				tyname := tmpvarname()
				c.outf("typedef voidptr (*%s)()", tyname).outfh().outnl()
				c.out(tyname)
			} else {
				if isinvalidty(vartystr) {
					if len(spec.Values) == 1 {
						val0 := spec.Values[0]
						if idt, ok := val0.(*ast.Ident); ok {
							if strings.HasPrefix(idt.Name, "gxtv") {
								c.outf("__typeof__(%v)", idt.Name)
							} else {
								log.Panicln("unexpected")
							}
						} else {
							log.Panicln("unexpected")
						}
					} else {
						// log.Panicln("unexpected", len(spec.Values))
						c.out(vartystr)
					}
				} else {
					c.out(vartystr)
				}
			}
			c.outsp()
		}
		c.out(gopp.IfElseStr(isglobvar, c.pkgpfx(), ""))
		c.out(varname.Name)
		c.outsp().outeq().outsp()

		if idx < len(spec.Values) {
			c.valnames[spec.Values[idx]] = varname
			scope = putscope(scope, ast.Var, "varname", varname)
			if isglobvar && (isstrty2(varty) || isslicety2(varty) ||
				isarrayty2(varty) || isstructty2(varty) || ismapty2(varty)) {
				c.out(cuzero)
			} else {
				c.genExpr(scope, spec.Values[idx])
			}
		} else {
			if isconst {
				c.out("(")
				c.genExpr(scope, vp1stval)
				c.out(")")
				c.outf("+%d", validx)
				c.outf("-%d", vp1stidx)
			} else if isglobvar {
				c.outf("%v /* 111 */", cuzero) // must constant for c
			} else if isstrty2(varty) {
				c.out("cxstring_new()")
			} else if isslicety2(varty) {
				elemty := varty.(*types.Slice).Elem()
				tystr := c.exprTypeNameImpl2(scope, elemty, nil)
				c.outf("cxarray3_new(0, sizeof(%s))", tystr)
			} else if ismapty2(varty) {
				c.out("cxhashtable_new()")
			} else if isstructty2(varty) {
				if ok := ispointer2(varty); ok {
					tystr := c.exprTypeNameImpl2(scope, varty, varname)
					tystr = strings.Trim(tystr, "*")
					c.outf("%s_new_zero()", tystr)
				} else {
					c.out(stzero)
				}
			} else {
				c.outf("%v /* 222 */", cuzero)
			}
		}

		if isglobvar || strings.HasPrefix(varname.Name, "gxtv") ||
			(varcnt > 1 && idx < (varcnt-1)) {
			c.outfh().outnl()
		}
		// c.out("/*333*/").outnl()
	}

}

func (c *g2nc) genInitGlobvars(scope *ast.Scope, pkg *ast.Package) {
	c.outf("void %sglobvars_init() {", c.pkgpfx()).outnl()
	for _, varx := range c.psctx.globvars {
		varo := varx.(*ast.ValueSpec)
		log.Println(varo.Type, reflect.TypeOf(varo.Type))
		for idx, name := range varo.Names {
			_ = idx
			keepon := false
			gotyx := c.info.TypeOf(name)
			log.Println(gotyx, name)
			switch goty := gotyx.(type) {
			case *types.Basic:
				if isstrty2(goty) {
					keepon = true
				}
			case *types.Slice:
				keepon = true
			case *types.Array, *types.Map:
				keepon = true
			default:
				gopp.G_USED(goty)
			}
			if !keepon {
				c.out("// soon ", exprstr(name), " ", c.exprpos(name).String()).outnl()
				continue
			}
			c.out("//", c.exprpos(name).String()).outnl()
			assigno := &ast.AssignStmt{}
			assigno.Tok = token.ASSIGN
			assigno.Lhs = []ast.Expr{name}
			if idx < len(varo.Values) {
				assigno.Rhs = []ast.Expr{varo.Values[idx]}
			}
			c.genAssignStmt(scope, assigno)
			c.outfh().outnl()
		}
	}
	c.out("}").outnl()
}

func (this *g2nc) outsp() *g2nc   { return this.out(" ") }
func (this *g2nc) outeq() *g2nc   { return this.out("=") }
func (this *g2nc) outstar() *g2nc { return this.out("*") }
func (this *g2nc) outfh() *g2nc   { return this.out(";") }
func (this *g2nc) outnl() *g2nc   { return this.out("\n") }
func (this *g2nc) out(ss ...string) *g2nc {
	for _, s := range ss {
		// fmt.Print(s, " ")
		this.sb.WriteString(s + "")
	}
	return this
}
func (this *g2nc) outf(format string, args ...interface{}) *g2nc {
	s := fmt.Sprintf(format, args...)
	this.out(s)
	return this
}
func (this *g2nc) clinema(e ast.Node) *g2nc {
	poso := this.exprpos(e) // file:row:col
	fields := strings.Split(poso.String(), ":")
	if len(fields) > 1 {
		this.outf("// #line %s %s", fields[1], fields[0]).outnl()
	} else {
		this.outf("// #line %v", fields).outnl()
	}
	return this
}

// TODO fix by typedef order
func (this *g2nc) genPrecgodefs() string {
	precgodefs := `
typedef uint32_t u32;
typedef int32_t i32;
typedef uint16_t u16;
typedef int16_t i16;
typedef float f32;
typedef double f64;
typedef uint64_t u64;
typedef int64_t i64;
typedef uintptr_t usize;
typedef uintptr_t uintptr;
// typedef void* error;
typedef void* voidptr;
typedef char* byteptr;
typedef void voidty;
#ifndef have_gxcallbale
#define have_gxcallbale
typedef struct gxcallable gxcallable;
struct gxcallable {voidptr obj; voidptr fnptr; };
extern voidptr gxcallable_new(voidptr fnptr, voidptr obj);
#endif
`
	return precgodefs
}

func (c *g2nc) genBuiltinTypesMetatype() string {
	s := "#include <stdalign.h>\n"
	s += "#include <cxrtbase.h>\n"
	bitypes := append(types.Typ, types.TypeAlias()...)
	for idx, bityp := range bitypes {
		tyname := bityp.Name()
		if strings.Contains(tyname, " ") {
			// untyped
			continue
		}
		if strings.HasPrefix(tyname, "complex") {
			continue
		}
		if tyname == "Pointer" {
			continue
		}

		// tyname = gopp.IfElseStr(tyname == "string", "charptr", tyname)
		s += fmt.Sprintf("static const _metatype %s_metatype = {", tyname)
		s += fmt.Sprintf(".kind = %d,\n", bityp.Kind())
		if tyname == "string" {
			s += fmt.Sprintf(".size = sizeof(%s),\n", "charptr")
			s += fmt.Sprintf(".align = alignof(%s),\n", "charptr")
		} else {
			s += fmt.Sprintf(".size = sizeof(%s),\n", tyname)
			s += fmt.Sprintf(".align = alignof(%s),\n", tyname)
		}
		s += fmt.Sprintf(".tystr = \"%s\"\n", tyname)
		s += fmt.Sprintf("}; // %d\n", idx)
	}

	s += "\n"
	return s
}

func (this *g2nc) code() (string, string) {
	code := ""
	code += fmt.Sprintf("// %s of %s\n", this.psctx.bdpkgs.Dir, this.psctx.wkdir)
	code += this.psctx.ccode
	code += "/* fake cdefs for " + this.psctx.bdpkgs.Dir + "\n" +
		this.psctx.fcdefscc + "\n*/\n\n"
	code += "#include <stddef.h>\n"
	code += "#include <stdalign.h>\n"
	code += "#include <cxrtbase.h>\n\n"
	code += this.genPrecgodefs() + "\n"
	code += this.sb.String()
	return code, "c"
}
