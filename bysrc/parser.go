package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"gopp"
	"log"
	"os"
	"reflect"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"

	"github.com/thoas/go-funk"
	"github.com/twmb/algoimpl/go/graph"
)

type ParserContext struct {
	path      string
	wkdir     string // for cgo
	pkgrename string
	fset      *token.FileSet
	files     []*ast.File
	info      *types.Info
	cursors   map[ast.Node]*astutil.Cursor
	grstargs  map[string]bool // goroutines packed arguments structure

	cfg2   *packages.Config
	bdpkgs *packages.Package
	pkgs   map[string]*packages.Package // flatten bdpkgs2 import tree

	typeDeclsm    map[string]*ast.TypeSpec
	typeDeclsv    []*ast.TypeSpec
	funcDeclsm    map[string]*ast.FuncDecl
	funcDeclsv    []*ast.FuncDecl
	funcdeclNodes map[string]graph.Node
	initFuncs     []*ast.FuncDecl

	tmpvars   map[ast.Stmt][]ast.Node // => value node
	gostmts   []*ast.GoStmt
	chanops   []ast.Expr // *ast.SendStmt
	closures  []*ast.FuncLit
	multirets []*ast.FuncDecl
	defers    []*ast.DeferStmt
	globvars  []ast.Node            // => ValueSpec node
	kvpairs   map[ast.Node]ast.Node // left <=> value

	gb    *graph.Graph
	ccode string
}

func NewParserContext(path string, pkgrename string) *ParserContext {
	this := &ParserContext{}
	this.path = path
	this.pkgrename = pkgrename
	this.cursors = make(map[ast.Node]*astutil.Cursor)
	this.grstargs = make(map[string]bool)
	this.typeDeclsm = make(map[string]*ast.TypeSpec)
	this.funcDeclsm = make(map[string]*ast.FuncDecl)
	this.funcdeclNodes = make(map[string]graph.Node)
	this.initFuncs = make([]*ast.FuncDecl, 0)
	this.gb = graph.New(graph.Directed)

	return this
}

func (this *ParserContext) Init() error {
	cfg := &packages.Config{Mode: packages.LoadFiles | packages.LoadImports | packages.LoadTypes | packages.LoadAllSyntax}
	// cfg.Env = append([]string{"CGO_ENABLED=1"}, os.Environ()...)
	// why could not import C (no metadata for C) ?
	// if C side has some compile error, this would happend
	// not really need go 1.13.5+, but CGO_ENABLED=1
	pkgs, err := packages.Load(cfg, flag.Args()...)
	gopp.ErrPrint(err)
	if err != nil {
		return err
	}
	cnt := packages.PrintErrors(pkgs)
	if cnt > 0 {
		pkg := pkgs[0]
		alldeclnouse := true
		for _, e := range pkg.Errors {
			if strings.Contains(fmt.Sprintf("%v", e), "declared but not used") {
			} else {
				alldeclnouse = false
				break
			}
		}
		if alldeclnouse {
			log.Println("all error is `declared but not used`, ignored", cnt)
		} else {
			return fmt.Errorf("load error %d %v", cnt, pkgs[0].Errors)
		}
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no package found?")
	} else if len(pkgs) > 1 {
		log.Println("Owt", len(pkgs))
	}

	this.cfg2 = cfg
	this.bdpkgs = pkgs[0]
	gopp.Assert(len(pkgs) == 1, "wtt", len(pkgs))
	this.info = this.bdpkgs.TypesInfo

	return err
}

/*
func (this *ParserContext) Initdep() error {
	bdpkgs, err := build.ImportDir(this.path, build.ImportComment)
	gopp.ErrPrint(err)
	this.bdpkgs = bdpkgs
	if len(bdpkgs.InvalidGoFiles) > 0 {
		log.Fatalln("Have InvalidGoFiles", bdpkgs.InvalidGoFiles)
	}
	// use go-clang to resolve c function signature
	// extract c code from bdpkgs.CgoFiles
	// parser step 1, got raw cgo c code
	{
		this.fset = token.NewFileSet()
		pkgs, err := parser.ParseDir(this.fset, this.path, this.dirFilter, 0|parser.AllErrors|parser.ParseComments)
		gopp.ErrPrint(err)
		this.pkgs = pkgs
		this.ccode = this.pickCCode()
	}

	this.walkpass_cgo_processor()
	// replace codebase dir
	this.path, this.wkdir = this.wkdir, this.path
	bdpkgs, err = build.ImportDir(this.path, build.ImportComment)
	gopp.ErrPrint(err)
	this.bdpkgs = bdpkgs
	if len(bdpkgs.InvalidGoFiles) > 0 {
		log.Fatalln("Have InvalidGoFiles", bdpkgs.InvalidGoFiles)
	}
	log.Println(bdpkgs.GoFiles)

	// parser step 2, got ast/types
	this.fset = token.NewFileSet()
	pkgs, err := parser.ParseDir(this.fset, this.path, this.dirFilter, 0|parser.AllErrors|parser.ParseComments)
	gopp.ErrPrint(err)
	this.pkgs = pkgs

	// this.ccode = this.pickCCode()
	this.walkpass_valid_files()

	this.conf.DisableUnusedImportCheck = true
	this.conf.Error = func(err error) {
		if !strings.Contains(err.Error(), "declared but not used") {
			log.Println(err)
		}
	}

	this.walkpass_check()

	this.walkpass_clean_cgodecl()
	this.walkpass_flat_cursors()
	this.walkpass_func_deps()
	log.Println("pkgs", this.typkgs.Name(), "types:", len(this.info.Types),
		"typedefs", len(this.typeDeclsm), "funcdefs", len(this.funcDeclsm))

	this.walkpass_tmpvars()
	this.walkpass_kvpairs()
	this.walkpass_gostmt()
	this.walkpass_chan_send_recv()
	this.walkpass_closures()
	this.walkpass_multiret()
	this.walkpass_defers()
	this.walkpass_globvars()

	return err
}
*/

func (pc *ParserContext) walkpass() {
	pc.walkpass_flatten_packages()
	pc.fset = pc.bdpkgs.Fset
	log.Println("typesinfo cnt ", len(pc.info.Types))
	pc.ccode = pc.pickCCode()

	this := pc
	this.walkpass_valid_files()
	this.walkpass_check()

	this.walkpass_clean_cgodecl()
	this.walkpass_flat_cursors()
	this.walkpass_func_deps()
	log.Println("pkgs", "types:", len(this.info.Types),
		"implicits", len(this.info.Implicits),
		"???defs", len(this.info.Defs),
		"???uses", len(this.info.Uses),
		"typedefs", len(this.typeDeclsm),
		"funcdefs", len(this.funcDeclsm))

	this.walkpass_tmpvars()
	this.walkpass_kvpairs()
	this.walkpass_gostmt()
	this.walkpass_chan_send_recv()
	this.walkpass_closures()
	this.walkpass_multiret()
	this.walkpass_defers()
	this.walkpass_globvars()
}

func (pc *ParserContext) walkpass_flatten_packages() {
	imps := map[string]*packages.Package{}
	var flatten_proc func(pkgs map[string]*packages.Package)
	flatten_proc = func(pkgs map[string]*packages.Package) {
		for pkgpath, pkg := range pkgs {
			pkgname := pkg.Name
			if isgointernpkg(pkgpath, pkgname) {
				log.Println(pkgpath, pkgname, "ignored")
			} else {
				imps[pkgpath] = pkg
			}
		}
		for _, pkg := range pkgs {
			flatten_proc(pkg.Imports)
		}
	}
	tmps := map[string]*packages.Package{pc.bdpkgs.PkgPath: pc.bdpkgs}
	flatten_proc(tmps)
	pc.pkgs = imps
}
func (pc *ParserContext) walkpass_flatten_fset() {
	pc.fset = pc.bdpkgs.Fset
}

// CgoFiles merges in GoFiles in xgo/packages
// https://github.com/golang/tools/blob/a7a6caa82ab2c7a235db8f1e47d3a8f2e7a6c054/go/packages/golist.go#L729
func (pc *ParserContext) walkpass_cgo_files_populate() {
	// extract cgo files from packages.Package.GoFiles
}

// cgo preprocessor types
/*
func (pc *ParserContext) walkpass_cgo_processor() {
	pc.wkdir = "_obj"
	{
		err := os.RemoveAll(pc.wkdir + "/")
		gopp.ErrPrint(err, pc.wkdir)
		os.Mkdir(pc.wkdir, 0755)
	}
	cmdfld := []string{"/opt/go/bin/go", "tool", "cgo", "-objdir", pc.wkdir}
	bdpkgs := pc.bdpkgs
	for _, cgofile := range bdpkgs.CgoFiles {
		cgofile = bdpkgs.Dir + "/" + cgofile
		cmdfld = append(cmdfld, cgofile)
	}
	log.Println(cmdfld)
	if len(bdpkgs.CgoFiles) > 0 {
		cmdo := exec.Command(cmdfld[0], cmdfld[1:]...)
		allout, err := cmdo.CombinedOutput()
		gopp.ErrPrint(err, cmdfld)
		allout = bytes.TrimSpace(allout)
		if len(allout) > 0 {
			log.Println(string(allout))
		}
		if err != nil {
			os.Exit(-1)
		}
	}

	// copy orignal source to wkdir
	// remove cgofile from wkdir
	files, err := filepath.Glob(bdpkgs.Dir + "/*")
	gopp.ErrPrint(err)
	for _, file := range files {
		if funk.Contains(bdpkgs.CgoFiles, filepath.Base(file)) {
			continue
		}
		err = gopp.CopyFile(file, pc.wkdir+"/"+filepath.Base(file))
		gopp.ErrPrint(err, file)
	}
	os.Rename(pc.wkdir+"/_cgo_gotypes.go", pc.wkdir+"/cxuse_cgo_gotypes.go")
}
*/

func (pc *ParserContext) walkpass_check() {
	// pc.conf.FakeImportC = true
	// pc.conf.Importer = &mypkgimporter{}

	// files := pc.files
	// var err error
	// pc.info = &types.Info{}
	// pc.typkgs, err = pc.conf.Check(pc.path, pc.fset, files, pc.info)
	// gopp.ErrPrint(err)
}

func (this *ParserContext) nameFilter2(filename string, files []string) bool {
	for _, okfile := range files {
		if filename == okfile {
			return true // keep
		}
	}
	return false
}
func (this *ParserContext) nameFilter(filename string) bool {
	if this.nameFilter2(filename, this.bdpkgs.GoFiles) {
		return true
	}
	/*
		if this.nameFilter2(filename, this.bdpkgs.CgoFiles) {
			return true
		}
	*/
	return false
}
func (this *ParserContext) dirFilter(f os.FileInfo) bool {
	return this.nameFilter(f.Name())
}

type mypkgimporter struct{}

func (this *mypkgimporter) Import(path string) (pkgo *types.Package, err error) {
	if true {
		// go 1.12
		fset := token.NewFileSet()
		pkgo, err = importer.ForCompiler(fset, "source", nil).Import(path)
	} else {
		pkgo, err = importer.Default().Import(path)
	}
	gopp.ErrPrint(err, path)
	return pkgo, err
}

func trimgopath(filename string) string {
	cachepath := os.Getenv("GOCACHE")
	if cachepath == "" {
		cachepath = os.Getenv("HOME") + "/.cache/go-build"
	}
	if cachepath != "" && strings.HasPrefix(filename, cachepath) {
		filename = "@GOCACHE/" + filename[len(cachepath):]
		return filename
	}

	gopath := os.Getenv("GOPATH")
	gopaths := strings.Split(gopath, ":")

	for _, pfx := range gopaths {
		if strings.HasPrefix(filename, pfx) {
			return "@GOPATH/" + filename[len(pfx)+5:]
		}
	}

	return filename
}
func exprpos(pc *ParserContext, e ast.Node) token.Position {
	if e == nil {
		return token.Position{}
	}
	poso := pc.fset.Position(e.Pos())
	poso.Filename = trimgopath(poso.Filename)
	return poso
}

// return notrimmed filename
func exprpos2(pc *ParserContext, e ast.Node) token.Position {
	if e == nil {
		return token.Position{}
	}
	poso := pc.fset.Position(e.Pos())
	// poso.Filename = trimgopath(poso.Filename)
	return poso
}

func (this *ParserContext) pickCCode() string {
	rawcode := this.pickCCode2()
	lines := strings.Split(rawcode, "\n")
	rawcode = ""
	for _, line := range lines {
		if !strings.HasPrefix(line, "#cgo ") {
			rawcode += line + "\n"
		} else {
			rawcode += "// " + line + "\n"
		}
	}
	// log.Println("got c code", rawcode)
	return rawcode
}
func (this *ParserContext) pickCCode2() string {
	ccode := ""
	// CgoFiles is in packages.Package.GoFiles
	for _, f := range this.bdpkgs.GoFiles {
		var fo *ast.File = this.findFileobj(f)
		if fo == nil {
			continue
		}
		ccode += this.pickCCode3(fo)
	}
	/*
		for _, f := range this.bdpkgs.CgoFiles {
			var fo *ast.File = this.findFileobj(f)
			ccode += this.pickCCode3(fo)
		}
	*/
	return ccode
}
func (this *ParserContext) pickCCode3(fo *ast.File) string {
	for idx, cmto := range fo.Comments {
		// isimpcblock(cmto)???
		for idx2, impo := range fo.Imports {
			gopp.G_USED(idx, idx2)
			if impo.Pos()-token.Pos(len("\nimport ")) == cmto.End() {
				// log.Println("got c code", cmto.Text())
				lineposi := exprpos(this, cmto)
				codeblk := "// embeded C code block begin\n"
				codeblk += fmt.Sprintf("#line %d \"%s\"\n%s\n",
					lineposi.Line, lineposi.Filename, cmto.Text())
				codeblk += "// embeded C code block end\n"
				return codeblk
			}
		}
	}
	return ""
}

// origname full path name of go file
func (this *ParserContext) findFileobj(origname string) *ast.File {
	for _, pkgo := range this.pkgs {
		for _, fileo := range pkgo.Syntax {
			if fileo == nil {
			}
			fullname := exprpos2(this, fileo).Filename
			if fullname == origname {
				return fileo
			}
		}
		/*
			for filename, fileo := range pkgo.Files {
				name := filepath.Base(filename)
				if name == fbname {
					return fileo
				}
			}
		*/
	}
	return nil
}

func (pc *ParserContext) pkgmap2arr() (arr []*packages.Package) {
	for _, pkgo := range pc.pkgs {
		arr = append(arr, pkgo)
	}
	return
}

func (pc *ParserContext) walkpass_valid_files() {
	this := pc
	pkgs := pc.pkgs

	var files []*ast.File
	for _, pkg := range pkgs {
		for _, fileo := range pkg.Syntax {
			if strings.HasSuffix(fileo.Name.Name, "_test") {
				continue
			}
			files = append(files, fileo)
		}
	}
	this.files = files
}

// astutil.Apply for all packages, files
func (pc *ParserContext) Apply(pre func(c *astutil.Cursor) bool, post func(c *astutil.Cursor) bool) {
	for _, pkg := range pc.pkgs {
		for _, fileo := range pkg.Syntax {
			astutil.Apply(fileo, pre, post)
		}
	}
}

func (pc *ParserContext) walkpass_func_deps() {
	pc.walkpass_func_deps1()
	pc.walkpass_func_deps2()
}
func (pc *ParserContext) walkpass_func_deps1() {
	this := pc

	pc.putFuncCallDependcy("main", "main_go")
	var curfds []string // stack, current func decls
	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.TypeSpec:
			// log.Println("typedef", t.Name.Name)
			this.typeDeclsm[te.Name.Name] = te
		case *ast.FuncDecl:
			if te.Recv != nil && te.Recv.NumFields() > 0 {
				varty := te.Recv.List[0].Type
				if ve, ok := varty.(*ast.StarExpr); ok {
					varty2 := ve.X
					tyname := varty2.(*ast.Ident).Name
					fnfullname := tyname + "_" + te.Name.Name
					this.funcDeclsm[fnfullname] = te
					curfds = append(curfds, fnfullname)
				} else if ve, ok := varty.(*ast.Ident); ok {
					tyname := ve.Name
					fnfullname := tyname + "_" + te.Name.Name
					this.funcDeclsm[fnfullname] = te
					curfds = append(curfds, fnfullname)
				} else {
					log.Println("todo", varty, reflect.TypeOf(te.Recv.List[0]))
				}
			} else {
				if te.Name.Name == "init" {
					this.initFuncs = append(this.initFuncs, te)
				}
				this.funcDeclsm[te.Name.Name] = te
				curfds = append(curfds, te.Name.Name)
			}

		case *ast.CallExpr:
			if len(curfds) == 0 { // global scope call
				switch be := te.Fun.(type) {
				case *ast.SelectorExpr:
					if iscsel(be.X) {
						break
					} else {
						log.Println("wtf", te, te.Fun, reflect.TypeOf(te.Fun))
					}
				default:
					log.Println("wtf", te, te.Fun, reflect.TypeOf(te.Fun))
				}
				// break
			} else {
				var curfd = curfds[len(curfds)-1]
				switch be := te.Fun.(type) {
				case *ast.Ident:
					this.putFuncCallDependcy(curfd, be.Name)
				case *ast.SelectorExpr:
					if iscsel(be.X) {
						break
					}
					varty := this.info.TypeOf(be.X)
					if varty == nil {
						break
					}
					tyname := sign2rety(varty.String())
					tyname = strings.TrimRight(tyname, "*")
					fnfullname := tyname + "_" + be.Sel.Name
					this.putFuncCallDependcy(curfd, fnfullname)
				default:
					log.Println("todo", te.Fun, reflect.TypeOf(te.Fun))
				}
			}
		case *ast.Ident: // func name referenced
			if len(curfds) == 0 {
				break
			}
			var curfd = curfds[len(curfds)-1]
			varobj := this.info.ObjectOf(te)
			switch varobj.(type) {
			case *types.Func:
				this.putFuncCallDependcy(curfd, te.Name)
			}
		case *ast.ReturnStmt:
		case *ast.CompositeLit:
			if len(curfds) == 0 {
				log.Println("todo globvar", exprpos(pc, c.Node()))
				break
			}
			var curfd = curfds[len(curfds)-1]
			goty := pc.info.TypeOf(te.Type)
			for funame, fd := range pc.funcDeclsm {
				if fd.Recv.NumFields() == 0 {
					continue
				}
				rcv0 := fd.Recv.List[0]
				rcvty := pc.info.TypeOf(rcv0.Type)
				samety := rcvty == goty
				if ptrty, ok := rcvty.(*types.Pointer); ok && !samety {
					samety = ptrty.Elem() == goty
				}
				if samety {
					this.putFuncCallDependcy(curfd, funame)
				}
			}
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.FuncDecl:
			if te.Recv != nil && te.Recv.NumFields() > 0 {
				curfds = curfds[:len(curfds)-1]
			} else {
				curfds = curfds[:len(curfds)-1]
			}
		default:
			gopp.G_USED(te)
		}
		return true
	})
}
func (pc *ParserContext) walkpass_func_deps2() {
	nodes := pc.gb.TopologicalSort()
	for _, node := range nodes {
		pc.funcDeclsv = append(pc.funcDeclsv, pc.funcDeclsm[(*node.Value).(string)])
	}
	// unused decls
	for _, d := range pc.funcDeclsm {
		if _, ok := builtinfns[d.Name.Name]; ok {
			continue
		}
		invec := false
		for _, d1 := range pc.funcDeclsv {
			if d1 == d {
				invec = true
				break
			}
		}
		if !invec {
			pc.funcDeclsv = append(pc.funcDeclsv, d)
		}
	}
}

func (pc *ParserContext) walkpass_flat_cursors() {
	pc.Apply(func(c *astutil.Cursor) bool {
		tc := *c
		pc.cursors[c.Node()] = &tc
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	})
}

func (pc *ParserContext) walkpass_tmpl_proc() {
	pc.Apply(func(c *astutil.Cursor) bool {
		tc := *c
		pc.cursors[c.Node()] = &tc
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	})
}

func (pc *ParserContext) dumpup(cs *astutil.Cursor, no int) {
	if cs == nil {
		return
	}
	log.Println(no, cs.Name(), reflect.TypeOf(cs.Node()))
	pn := cs.Parent()
	pcs := pc.cursors[pn]
	pc.dumpup(pcs, no+1)
}

func upfindstmt(pc *ParserContext, cs *astutil.Cursor, no int) ast.Stmt {
	if cs == nil {
		return nil
	}
	pn := cs.Parent()
	pcs := pc.cursors[pn]
	if stmt, ok := pn.(ast.Stmt); ok {
		return stmt
	} else {
		return upfindstmt(pc, pcs, no+1)
	}
}

func upfindFuncDeclNode(pc *ParserContext, n ast.Node, no int) *ast.FuncDecl {
	cs := pc.cursors[n]
	return upfindFuncDecl(pc, cs, no)
}
func upfindFuncDeclAst(pc *ParserContext, e ast.Expr, no int) *ast.FuncDecl {
	cs := pc.cursors[e]
	return upfindFuncDecl(pc, cs, no)
}
func upfindFuncDecl(pc *ParserContext, cs *astutil.Cursor, no int) *ast.FuncDecl {
	if cs == nil {
		return nil
	}
	pn := cs.Parent()
	pcs := pc.cursors[pn]
	if stmt, ok := pn.(*ast.FuncDecl); ok {
		return stmt
	} else {
		return upfindFuncDecl(pc, pcs, no+1)
	}
}

func (pc *ParserContext) walkpass_multiret() {
	multirets := []*ast.FuncDecl{}
	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.FuncDecl:
			if te.Type.Results.NumFields() < 2 {
				break
			}
			for idx, fld := range te.Type.Results.List {
				if len(fld.Names) == 0 {
					fld.Names = append(fld.Names, newIdent(tmpvarname2(idx)))
				}
			}
			multirets = append(multirets, te)
		default:
			gopp.G_USED(te)
		}
		return true
	})

	log.Println("multirets", len(multirets))
	pc.multirets = multirets
}

// 一句表达不了的表达式临时变量
func (pc *ParserContext) walkpass_tmpvars() {
	var tmpvars = map[ast.Stmt][]ast.Node{} // => tmpvarname
	gopp.G_USED(tmpvars)

	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			// log.Println(c.Name(), exprpos(pc, c.Node()))
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.CompositeLit:
			// break
			ce := c.Node().(ast.Expr)
			vsp2 := &ast.AssignStmt{}
			vsp2.Lhs = []ast.Expr{newIdent(tmpvarname())}
			vsp2.Rhs = []ast.Expr{ce}
			vsp2.Tok = token.DEFINE
			anded := false
			cety := pc.info.TypeOf(ce)
			if _, ok1 := cety.(*types.Named); ok1 {
				if _, ok2 := cety.Underlying().(*types.Struct); ok2 {
					anded = true
				}
			}
			if anded {
				xe := &ast.UnaryExpr{}
				xe.Op = token.AND
				xe.OpPos = c.Node().Pos()
				xe.X = ce
				vsp2.Rhs = []ast.Expr{xe}
			} else {
				// vsp2.Rhs = []ast.Expr{ce}
			}
			c.Replace(vsp2.Lhs[0])
			stmt := upfindstmt(pc, c, 0)
			tmpvars[stmt] = append(tmpvars[stmt], vsp2)
			tyval := types.TypeAndValue{}
			tyval.Type = pc.info.TypeOf(ce)
			if anded {
				tyval.Type = types.NewPointer(tyval.Type)
			}
			pc.info.Types[vsp2.Lhs[0]] = tyval
			pc.info.Types[vsp2.Rhs[0]] = tyval
		case *ast.UnaryExpr:
			if te.Op == token.AND {
				stmt := upfindstmt(pc, c, 0)
				if tmpvars[stmt] != nil {
					c.Replace(te.X)
				}
				if _, ok := te.X.(*ast.CompositeLit); ok {
					log.Panicln("not reach")
					vsp2 := &ast.AssignStmt{}
					vsp2.Lhs = []ast.Expr{newIdent(tmpvarname())}
					vsp2.Rhs = []ast.Expr{te}
					vsp2.Tok = token.DEFINE
					vsp2.TokPos = c.Node().Pos()
					c.Replace(vsp2.Lhs[0])
					stmt := upfindstmt(pc, c, 0)
					tmpvars[stmt] = append(tmpvars[stmt], vsp2)
					tyval := types.TypeAndValue{}
					tyval.Type = pc.info.TypeOf(te)
					pc.info.Types[vsp2.Lhs[0]] = tyval
				}
			}
		default:
			gopp.G_USED(te)
		}
		return true
	})

	log.Println("tmpvars", len(tmpvars))
	pc.tmpvars = tmpvars
}

func (pc *ParserContext) walkpass_kvpairs() {
	kvpairs := map[ast.Node]ast.Node{}
	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.AssignStmt:
			for idx, le := range te.Lhs {
				kvpairs[le] = te.Rhs[idx]
				kvpairs[te.Rhs[idx]] = le
			}
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	})

	log.Println("kvpairs", len(kvpairs))
	pc.kvpairs = kvpairs
}

func (pc *ParserContext) walkpass_gostmt() {
	var gostmts = []*ast.GoStmt{}
	_ = gostmts

	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.GoStmt:
			gostmts = append(gostmts, te)
		default:
			gopp.G_USED(te)
		}
		return true
	})

	log.Println("gostmts", len(gostmts))
	pc.gostmts = gostmts
}

func (pc *ParserContext) walkpass_chan_send_recv() {
	var chanops = []ast.Expr{}
	_ = chanops

	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.SendStmt:
			chanops = append(chanops, te.Chan)
		case *ast.UnaryExpr:
			if te.Op == token.ARROW {
				chanops = append(chanops, te.X)
			}
		default:
			gopp.G_USED(te)
		}
		return true
	})

	log.Println("chanops", len(chanops))
	pc.chanops = chanops
}

func (pc *ParserContext) walkpass_closures() {
	var closures = []*ast.FuncLit{}
	_ = closures

	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.FuncLit:
			closures = append(closures, te)
		default:
			gopp.G_USED(te)
		}
		return true
	})

	log.Println("closures", len(closures))
	pc.closures = closures
}

//
func (pc *ParserContext) walkpass_clean_cgodecl() {
	skipfds := []string{"_cgo_runtime_cgocallback", "_cgoCheckResult", "_cgoCheckPointer",
		"_Cgo_use", "_cgo_runtime_cgocall", "_Cgo_ptr", "_cgo_cmalloc", "runtime_throw",
		"_cgo_runtime_gostringn", "_cgo_runtime_gostring"}

	pc.Apply(func(c *astutil.Cursor) bool {

		switch te := c.Node().(type) {
		case *ast.FuncDecl:

			if funk.Contains(skipfds, te.Name.Name) {
				c.Delete()
			}
		case *ast.ValueSpec:
			name := te.Names[0].Name
			if strings.HasPrefix(name, "__cgofn__cgo_") || strings.HasPrefix(name, "_cgo_") ||
				strings.HasPrefix(name, "_Ciconst_") || strings.HasPrefix(name, "_Cfpvar_") {
				c.Delete()
				break
			}
			tystr := types.ExprString(te.Type)
			if tystr == "syscall.Errno" || te.Names[0].Name == "_" {
				c.Delete()
				break
			}
		case *ast.CallExpr:
			if fe, ok := te.Fun.(*ast.Ident); ok {
				if fe.Name == "_Cgo_ptr" {
					c.Replace(newIdent(te.Args[0].(*ast.Ident).Name[11:]))
					break
				}
				if fe.Name == "_cgoCheckPointer" {
					// panic: Delete node not contained in slice
					// c.Delete()
					// break
				}
			}
		case *ast.Ident:
			if strings.HasPrefix(te.Name, "_Ciconst_") {
				c.Replace(newIdent(te.Name[9:]))
			}
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	})

}

// todo
func (pc *ParserContext) walkpass_nested_func() {
}

// todo
func (pc *ParserContext) walkpass_nested_type() {
}

func (pc *ParserContext) walkpass_defers() {
	defers := []*ast.DeferStmt{}

	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.DeferStmt:
			defers = append(defers, te)
		case *ast.FuncDecl:
			if te.Type.Results.NumFields() == 0 {
				// if len(te.Body.List) == 0 {
				// 	retstmt := &ast.ReturnStmt{}
				// 	retstmt.Results = []ast.Expr{}
				// 	// retstmt.Return = te.Pos()
				// 	te.Body.List = append(te.Body.List, retstmt)
				// } else {
				// 	log.Println("hhh")
				// 	laststmt := te.Body.List[len(te.Body.List)-1]
				// 	log.Println(laststmt)
				// }
			}
		default:
			gopp.G_USED(te)
		}
		return true
	})

	log.Println("defers", len(defers))
	pc.defers = defers
}

func (pc *ParserContext) walkpass_globvars() {
	globvars := []ast.Node{}

	pc.Apply(func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		case *ast.ValueSpec:
			for _, name := range te.Names {
				if isglobalid(pc, name) {
					globvars = append(globvars, te)
				}
			}
		default:
			gopp.G_USED(te)
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch te := c.Node().(type) {
		default:
			gopp.G_USED(te)
		}
		return true
	})

	log.Println("globvars", len(globvars))
	pc.globvars = globvars
}

func (pc *ParserContext) putTyperefDependcy(funame, tyname string) {

}

// name0: caller
// name1: callee
func (pc *ParserContext) putFuncCallDependcy(name0, name1 string) {
	if name0 == name1 {
		return
	}
	if _, ok := builtinfns[name1]; ok {
		return
	}
	n0, ok0 := pc.funcdeclNodes[name0]
	if !ok0 {
		n0 = pc.gb.MakeNode()
		*n0.Value = name0
		pc.funcdeclNodes[name0] = n0
	}
	n1, ok1 := pc.funcdeclNodes[name1]
	if !ok1 {
		n1 = pc.gb.MakeNode()
		*n1.Value = name1
		pc.funcdeclNodes[name1] = n1
	}
	// log.Println("adding", name0, n0.Value, "->", name1, n1.Value)
	err := pc.gb.MakeEdge(n1, n0)
	gopp.ErrPrint(err, name0, name1)
}

func (pc *ParserContext) getImportNameMap() map[string]string {
	pkgrenames := map[string]string{} // path => rename
	for pname, pkgo := range pc.pkgs {
		log.Println(pname, pkgo.Name, pkgo.Imports)
		for _, fileo := range pkgo.Syntax {
			fname := fileo.Name.String()
			log.Println(fname, fileo.Imports)
			for _, declo := range fileo.Decls {
				ad, ok := declo.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, tspec := range ad.Specs {
					id, ok := tspec.(*ast.ImportSpec)
					if ok {
						log.Println(id.Name, id.Path)
						dirp := strings.Trim(id.Path.Value, "\"")
						if id.Name != nil {
							pkgrenames[dirp] = id.Name.Name
						} else {
							pkgrenames[dirp] = ""
						}
					}
				}
			}
		}
	}
	return pkgrenames
}
