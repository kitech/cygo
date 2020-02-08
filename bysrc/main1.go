package main

import (
	"gopp"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/twmb/algoimpl/go/graph"
)

var fname string

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("must specify a go source file to tranpiler")
	}
	fname = os.Args[1]
	fio, err := os.Lstat(fname)
	gopp.ErrPrint(err)
	if err != nil {
		return
	}
	if !fio.IsDir() {
		log.Fatalln("Not a dir", fname)
	}

	gopaths := gopp.Gopaths()
	builtin_imppath := "xgo/builtin"
	builtin_pkgpath := ""
	for _, gopath := range gopaths {
		pkgpath := gopath + "/src/" + builtin_imppath
		if gopp.FileExist(pkgpath) {
			builtin_pkgpath = pkgpath
			break
		}
	}
	gopp.Assert(builtin_pkgpath != "", "not found", builtin_imppath)

	pkgpaths := []string{builtin_pkgpath, fname}
	psctxs := []*ParserContext{}
	comps := []*g2nc{}
	pkgrenames := map[string]string{} // path => rename
	dedups := map[string]bool{}       // pkgpath =>

	var builtin_psctx *ParserContext
	gopaths = append(gopaths, runtime.GOROOT())

	for len(pkgpaths) > 0 {
		fname := pkgpaths[0]
		pkgpaths = pkgpaths[1:]
		pkgrename := ""
		segs := strings.Split(fname, ":")
		if len(segs) == 2 {
			fname = segs[0]
			pkgrename = segs[1]
		}
		if _, ok := dedups[fname]; ok {
			log.Println("already gened", fname)
			continue
		}

		psctx, g2n := dogen(fname, pkgrename, builtin_psctx)
		psctxs = append(psctxs, psctx)
		comps = append(comps, g2n)
		if fname == builtin_pkgpath {
			builtin_psctx = psctx
		}

		imprenames := psctx.getImportNameMap()
		for path, rename := range imprenames {
			pkgrenames[path] = rename
			log.Println("pkgimp", path, rename)
		}

		for _, imppath := range psctx.bdpkgs.Imports {
			log.Println("pkgimp", imppath, psctx.bdpkgs.Dir)
			if imppath == "runtime" ||
				imppath == "atomic" ||
				imppath == "runtime/cgo" ||
				imppath == "syscall" || imppath == "syscall/js" ||
				imppath == "internal/race" {
				continue
			}
			for _, gopath1 := range gopaths {
				impdir := gopath1 + "/src/" + imppath
				if gopp.FileExist(impdir) {
					log.Println("got", impdir)
					pkgpaths = append(pkgpaths, impdir+":"+pkgrenames[imppath])
					break
				}
			}
		}
		log.Println("=================", fname)
		dedups[fname] = true
		if strings.Contains(fname, "xnet") {
			// os.Exit(-1)
		}
	}

	// packages  depgraph order
	pkgdepg := graph.New(graph.Directed)
	pkgnodeg := map[string]graph.Node{}
	for i := len(comps) - 1; i >= 0; i-- {
		psctx := comps[i].psctx
		curpkg := psctx.path
		curpkg = trimgopath(curpkg)
		curpkg = gopp.IfElseStr(curpkg == ".", "main", curpkg)
		na, ok := pkgnodeg[curpkg]
		if !ok {
			na = pkgdepg.MakeNode()
			*na.Value = curpkg
			pkgnodeg[curpkg] = na
		}
		for _, imppkg := range psctx.bdpkgs.Imports {
			if imppkg == "C" {
				continue
			}
			if false {
				log.Println("dep", curpkg, "<-", imppkg)
			}
			nb, ok := pkgnodeg[imppkg]
			if !ok {
				nb = pkgdepg.MakeNode()
				*nb.Value = imppkg
				pkgnodeg[imppkg] = nb
			}
			err := pkgdepg.MakeEdge(nb, na)
			gopp.ErrPrint(err)
		}
	}
	nodes := pkgdepg.TopologicalSort()
	// log.Println(nodes)
	for idx, nodeg := range nodes {
		valx := nodeg.Value
		if false {
			log.Println(idx, *valx)
		}
	}
	comps2 := []*g2nc{} // order by depgraph
	for _, nodeg := range nodes {
		valx := *nodeg.Value
		val := valx.(string)

		for i := len(comps) - 1; i >= 0; i-- {
			psctx := comps[i].psctx
			curpkg := psctx.path
			curpkg = trimgopath(curpkg)
			curpkg = gopp.IfElseStr(curpkg == ".", "main", curpkg)
			if curpkg == val {
				comps2 = append(comps2, comps[i])
				break
			}
		}
	}
	gopp.Assert(len(comps2) == len(comps), "wtfff", len(comps2), len(comps))
	comps = comps2

	code := ""
	extname := ""
	pkgclts := []string{}
	for i := 0; i < len(comps); i++ {
		psctx := comps[i].psctx
		if psctx.bdpkgs.Name != "main" {
			pkgclts = append(pkgclts, psctx.bdpkgs.Name)
		}
	}
	log.Println("pkg order", pkgclts, "main")

	mainpkgcode := ""
	for i := 0; i < len(comps); i++ {
		psctx := comps[i].psctx
		if psctx.bdpkgs.Name == "main" {
			comps[i].genCallPkgGlobvarsInits(pkgclts)
			comps[i].genCallPkgInits(pkgclts)
		}

		str, ext := comps[i].code()
		if psctx.bdpkgs.Name == "main" {
			mainpkgcode = str
		} else {
			code += str
			extname = ext
		}
	}

	code = comps[0].genBuiltinTypesMetatype() + code + mainpkgcode
	fname := "opkgs/foo." + extname
	ioutil.WriteFile(fname, []byte(code), 0644)
	log.Println("clangfmt ...", fname, len(code))
	btime := time.Now()
	clangfmt(fname)
	linecnt := strings.Count(code, "\n")
	log.Println("gencode lines", linecnt, len(code), time.Since(btime))
}
func clangfmt(fname string) {
	exepath, err := exec.LookPath("clang-format")
	gopp.ErrPrint(err)
	if err != nil {
		return
	}
	cmdo := exec.Command(exepath, "--sort-includes=0", "-i", fname, "--style", "WebKit")
	err = cmdo.Run()
	gopp.ErrPrint(err, fname)
}
func dogen(fname string, pkgrename string, builtin_psctx *ParserContext) (*ParserContext, *g2nc) {
	psctx := NewParserContext(fname, pkgrename, builtin_psctx)
	err := psctx.Init()
	if err != nil && !strings.Contains(err.Error(), "declared but not used") {
		gopp.ErrPrint(err)
		println()
		println()
		time.Sleep(1 * time.Second)
	}

	// g2n := g2nim{}
	g2n := g2nc{}
	g2n.basecomp = newbasecomp(psctx)
	g2n.genpkgs()
	code, ext := g2n.code()
	dstfile := psctx.bdpkgs.Name + ".go." + ext
	ioutil.WriteFile("opkgs/"+dstfile, []byte(code), 0644)
	return psctx, &g2n
}
