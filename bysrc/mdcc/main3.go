package main

import (
	"gopp"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	// https://github.com/xlab/c-for-go/commit/9a426bcc5e562dfa41d7d122b6c5777b8f84d8f8
	//"github.com/xlab/c-for-go/parser"

	"modernc.org/cc/v3"
)

// demo modernc.org/cc/v3 header parser
func main() {
	os.Setenv("LANG", "C")
	os.Setenv("LC_ALL", "C")
	os.Setenv("LC_CTYPE", "C")
	predefines, incpaths, sysincpaths, err := cc.HostConfig("")
	gopp.ErrPrint(err)
	log.Println(predefines, incpaths, sysincpaths)

	abi, err := cc.NewABI(runtime.GOOS, runtime.GOARCH) // must
	gopp.ErrFatal(err)
	cfg := &cc.Config{ABI: abi}
	incdirs := []string{}
	sysincs := []string{}
	sysincs = append(sysincs, "/usr/include")
	sysincs = append(sysincs, "/usr/lib/gcc/x86_64-pc-linux-gnu/10.2.0/include")
	sysincs = append(sysincs, sysincpaths...)
	srcs := []cc.Source{}
	// srcs = append(srcs, cc.Source{Name: "predefines", Value: predefines})
	bcc, err := ioutil.ReadFile("./testc11atomic.h")
	gopp.ErrFatal(err)
	srcs = append(srcs, cc.Source{Name: "src100", Value: predefines + "\n" +
		builtinBase + "\n" + string(bcc)})

	ast, err := cc.Parse(cfg, incdirs, sysincs, srcs)
	gopp.ErrPrint(err, ast == nil)
	err = ast.Typecheck()
	gopp.ErrFatal(err, ast == nil)
	log.Println(tu)

	//tlcfg := &translator.Config{}
	//tl, err := translator.New(tlcfg)
	//gopp.ErrPrint(err, tl == nil)
	//tl.Learn(ast.TranslationUnit)
}

/////

var builtinBase = `
#define __builtin_va_list void *
#define __asm(x)
#define __inline
#define __inline__
#define __signed
#define __signed__
#define __const const
#define __extension__
#define __attribute__(x)
#define __attribute(x)
#define __restrict
#define __volatile__
#define __builtin_inff() (0)
#define __builtin_infl() (0)
#define __builtin_inf() (0)
#define __builtin_fabsf(x) (0)
#define __builtin_fabsl(x) (0)
#define __builtin_fabs(x) (0)
#define __INTRINSIC_PROLOG(name)
`

var basePredefines = `
#define __STDC_HOSTED__ 1
#define __STDC_VERSION__ 199901L
#define __STDC__ 1
#define __GNUC__ 4
#define __GNUC_PREREQ(maj,min) 0
#define __POSIX_C_DEPRECATED(ver)
#define __has_include_next(...) 1
#define __FLT_MIN__ 0
#define __DBL_MIN__ 0
#define __LDBL_MIN__ 0
void __GO__(char*, ...);
`
