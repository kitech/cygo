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

	builtin_pkgpath := "../xgo/builtin"
	pkgpaths := []string{fname, builtin_pkgpath}
	psctxs := []*ParserContext{}
	comps := []*g2nc{}
	pkgrenames := map[string]string{} // path => rename
	dedups := map[string]bool{}       // pkgpath =>

	var builtin_psctx *ParserContext
	gopaths := gopp.Gopaths()
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

	code := ""
	extname := ""
	pkgclts := []string{}
	for i := len(comps) - 1; i >= 0; i-- {
		psctx := comps[i].psctx
		if psctx.bdpkgs.Name == "main" {
			comps[i].genCallPkgGlobvarsInits(pkgclts)
			comps[i].genCallPkgInits(pkgclts)
		} else {
			pkgclts = append(pkgclts, psctx.bdpkgs.Name)
		}

		str, ext := comps[i].code()
		if psctx.path == builtin_pkgpath {
			code = str + code // 最前置
		} else {
			code += str
		}
		extname = ext
	}
	code = comps[0].genBuiltinTypesMetatype() + code
	fname := "opkgs/foo." + extname
	ioutil.WriteFile(fname, []byte(code), 0644)
	clangfmt(fname)
}
func clangfmt(fname string) {
	cmdo := exec.Command("clang-format", "-i", fname, "--style", "WebKit")
	err := cmdo.Run()
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
