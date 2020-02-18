package main

// try parse C use modernc.org/cc

import (
	"fmt"
	"go/token"
	"go/types"
	"gopp"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/thoas/go-funk"
	cc1x "github.com/xlab/c-for-go/parser"
	cc1 "modernc.org/cc"
	cc2 "modernc.org/cc/v2"
	"modernc.org/cc/v3"
	"modernc.org/xc"
)

// 突然发现不支持 C11 atomic
type cparser2 struct {
	name    string
	predefs string // like -DGC_THREADS
	ctu     *cc.AST
	cfg     *cc.Config
	ctu2    *cc2.TranslationUnit
	ctu1    *cc1.TranslationUnit
	syms    map[string]*csymdata2 // identity/struct/type name =>

}
type csymdata2 struct {
	name string
	kind int
	typ  cc1.Type
}

func newcsymdata2(name string, kind int, typ cc1.Type) *csymdata2 {
	csi := &csymdata2{}
	csi.name = name
	csi.kind = kind
	csi.typ = typ
	return csi
}

func newcparser2(name string) *cparser2 {
	cp := &cparser2{}
	cp.name = name
	cp.syms = map[string]*csymdata2{}
	return cp
}

func newfilesource(filename string) *cc.Source {
	srco := &cc.Source{}
	srco.Name = filename
	return srco
}
func newstrsource(code string) *cc.Source {
	srco := &cc.Source{}
	srco.Name = "flycode"
	srco.Value = code
	return srco
}

const codepfx = "#include <stdio.h>\n" +
	"#include <stdlib.h>\n" +
	"#include <string.h>\n" +
	"#include <errno.h>\n" +
	"#include <pthread.h>\n" +
	"#include <time.h>\n" +
	"#include <cxrtbase.h>\n" +
	"\n"

var (
	cxrtroot = "/home/me/oss/cxrt"
)

// 使用单独init函数名
func init() { init_cxrtroot() }
func init_cxrtroot() {
	if !gopp.FileExist(cxrtroot) {
		gopaths := gopp.Gopaths()
		for _, gopath := range gopaths {
			d := gopath + "/src/cxrt" // github actions runner
			if gopp.FileExist(d) {
				cxrtroot = d
				break
			}
		}
	}
	for _, item := range []string{"src", "3rdparty/cltc/src", "3rdparty/cltc/include"} {
		d := cxrtroot + "/" + item
		if funk.Contains(preincdirs, d) {
			continue
		}
		preincdirs = append(preincdirs, d)
	}
}

var preincdirs = []string{"/home/me/oss/src/cxrt/src",
	"/home/me/oss/src/cxrt/3rdparty/cltc/src",
	"/home/me/oss/src/cxrt/3rdparty/cltc/src/include",
	//	"/home/me/oss/src/cxrt/3rdparty/tcc",
	"/usr/include/gc",
	"/usr/include/curl",
}
var presysincs = []string{"/usr/include", "/usr/local/include",
	"/usr/include/x86_64-linux-gnu/", // ubuntu
	"/usr/lib/gcc/x86_64-pc-linux-gnu/9.2.1/include"}

// https://github.com/gcc-mirror/gcc/blob/master/gcc/memmodel.h
var c11_builtin_atomic_defs = `
#define __ATOMIC_RELAXED  0
#define __ATOMIC_CONSUME  1
#define __ATOMIC_ACQUIRE  2
#define __ATOMIC_RELEASE  3
#define __ATOMIC_ACQ_REL  4
#define __ATOMIC_SEQ_CST  5
#define __ATOMIC_LAST  6
`

// "-DFOO=1 -DBAR -DBAZ=fff"
func cp2_split_predefs(predefs string) map[string]interface{} {
	items := strings.Split(predefs, " ")
	res := map[string]interface{}{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		gopp.Assert(strings.HasPrefix(item, "-D"), "wtfff", item)
		item = item[2:]
		kv := strings.Split(item, "=")
		if len(kv) == 1 {
			res[item] = 1
		} else {
			if gopp.IsInteger(kv[1]) {
				res[kv[0]] = gopp.MustInt(kv[1])
			} else {
				res[kv[0]] = kv[1]
			}
		}
	}
	for k, v := range res {
		log.Println("predefsm", k, v, reftyof(v))
	}

	return res
}

func (cp *cparser2) ccHostConfig() (
	predefsm map[string]interface{}, incpaths, sysincs []string, err error) {
	var predefs string
	predefs, incpaths, sysincs, err = cc.HostConfig("")
	gopp.ErrPrint(err, cp.name, "can ignore")
	if err != nil {
		incpaths = append(incpaths, preincdirs...)
		sysincs = append(sysincs, presysincs...)
		err = nil
	}
	predefs = " -D__ATOMIC_RELAXED=0 -D__ATOMIC_CONSUME=1 -D__ATOMIC_ACQUIRE=2 -D__ATOMIC_RELEASE=3 -D__ATOMIC_ACQ_REL=4 -D__ATOMIC_SEQ_CST=5 " + predefs
	predefs += " " + cp.predefs + " -DGC_THREADS "
	predefsm = cp2_split_predefs(predefs)

	pwdir, err := os.Getwd()
	if pwdir != "" {
		incpaths = append(incpaths, pwdir)
	}
	log.Println(predefs, incpaths, sysincs)
	return
}

func (cp *cparser2) parsefile(filename string) error {
	predefs, incpaths, sysincs, err := cp.ccHostConfig()
	cfg := &cc.Config{}
	cfg.ABI, err = cc.NewABI(runtime.GOOS, runtime.GOARCH)
	gopp.ErrPrint(err)
	// log.Println(cfg.ABI)

	if false {
		// not work for bits/types.h
		srco := newfilesource(filename)
		cfg.RejectIncludeNext = false
		ctu, err := cc.Parse(cfg, incpaths, sysincs, []cc.Source{*srco})
		// ctu, err := cc.Translate(cfg, incpaths, sysincs, []cc.Source{*srco})
		gopp.ErrPrint(err, ctu != nil, filename)
		cp.ctu = ctu
		os.Exit(-1)
	}
	if false {
		// not work for stdarg.h
		srco, err := cc2.NewFileSource(filename)
		gopp.ErrPrint(err, filename)
		cfg := &cc2.Tweaks{}
		cfg.EnableImplicitBuiltins = true
		cfg.EnableImplicitDeclarations = true
		ctu, err := cc2.Translate(cfg, incpaths, sysincs, srco)
		gopp.ErrPrint(err)
		cp.ctu2 = ctu
		os.Exit(-1)
	}
	if true {
		// not work for c11 stdatomic.h
		paths := append(incpaths, sysincs...)
		cfg := &cc1x.Config{}
		cfg.IncludePaths = paths
		cfg.SourcesPaths = []string{filename}
		cfg.Defines = predefs
		// cfg.CCDefs = true
		// cfg.CCIncl = true
		// patch cc1x:100: 	model := *models[cfg.archBits]
		// patch cc1x:106:  // cc.EnableIncludeNext(),
		ctu, err1 := cc1x.ParseWith(cfg)
		err = err1
		gopp.ErrPrint(err, filename)
		cp.ctu1 = ctu
	}
	if err == nil {
		cp.collects1()
		// cp.dumpdecls1()
	}
	return err
}

func (cp *cparser2) gotypeof(sym string) types.Type {
	if strings.HasPrefix(sym, "struct_") {
		old := sym
		sym = sym[7:]
		log.Println("mapsymto", old, sym)
	}

	csi, found := cp.syms[sym]
	if found {
		if csi.kind == csym_define {
			return types.Typ[types.UntypedInt]
		}

		// TODO symbol kind
		return cp.gotypeof2(sym, csi.typ, true)
	} else {
		syms := map[string]types.Type{
			"__FILE__": types.Typ[types.String],
			"__LINE__": types.Typ[types.Int],
		}
		if tyobj, ok := syms[sym]; ok {
			return tyobj
		}

		// primitive_type
		amdl := cp.ctu1.Model
		vmdl := reflect.ValueOf(amdl).Elem()
		for tk, _ := range amdl.Items {
			if sym == strings.ToLower(tk.String()) ||
				sym == tk.CString() {
				tyfname := fmt.Sprintf("%sType", tk.String())
				fval := vmdl.FieldByName(tyfname)
				fval2 := fval.Interface().(cc1.Type)
				// log.Println(fval, reftyof(fval2), fval2.String(), fval2.Kind())
				return cp.gotypeof2(sym, fval2, true)
			}
		}
		// log.Fatalln("symbol not found", sym)
	}
	return nil
}

func (cp *cparser2) gotypeof2(sym string, cty cc1.Type, resty bool) types.Type {
	switch cty.Kind() {
	case cc1.Struct:
		fields, iscomplete := cty.Members()
		if !iscomplete {
			log.Println(cty, iscomplete, len(fields), cty.Declarator().Type)
		}
		tyn1 := types.NewTypeName(token.NoPos, fcpkg, cty.String(), nil)
		named1 := types.NewNamed(tyn1, nil, nil)
		var fldvars []*types.Var
		for idx, fldx := range fields {
			fldtyx := fldx.Type
			iscyclerefty2 := fldtyx.String() == cty.String() ||
				fldtyx.String() == cty.Pointer().String()
			var fldty types.Type
			if iscyclerefty2 {
				fldty = named1
			} else {
				if fldtyx.Kind() == cc1.Array {
					fldty = cp.gotypeof2(sym, fldtyx, resty)
				} else {
					fldty = cp.gotypeof2(sym, fldtyx, resty)
				}
			}

			tkval, bindings := fldtyx.Declarator().Identifier()
			gopp.Assert(tkval == fldx.Name, "wtfff", bindings)
			fldidtx := bindings.Lookup(cc1.NSIdentifiers, tkval)
			dirdecl := fldidtx.Node.(*cc1.DirectDeclarator)
			fldname := cp2_token_toname(dirdecl.Token.String())
			log.Println(idx, fldx.Name, fldtyx.Kind(), fldtyx,
				iscyclerefty2, fldname, fldty, cty.Pointer())
		}
		st1 := types.NewStruct(fldvars, nil)
		named1.SetUnderlying(st1)
		return named1
	case cc1.Union: // TODO
		log.Panicln(sym)
	case cc1.Array:
		ety := cty.Element()
		goety := cp.gotypeof2(sym, ety, resty)
		if cty.Elements() == 0 {
			goty := types.NewSlice(goety)
			return goty
		} else {
			goty := types.NewArray(goety, int64(cty.Elements()))
			return goty
		}
	case cc1.Enum:
		return types.Typ[types.Int]
	case cc1.Function:
		ty := cty.Result()
		// log.Println(cp.name, sym, cty.Kind(), ty.Kind(), ty)
		return cp.gotypeof2(sym, ty, resty)
	case cc1.Void:
		return types.NewTuple()
	case cc1.UintPtr:
		return types.Typ[types.Uintptr]
	case cc1.Ptr:
		switch tk2 := cty.Element().Kind(); tk2 {
		case cc1.Void:
			return types.Typ[types.Voidptr]
		case cc1.Char, cc1.SChar, cc1.UChar:
			return types.Typ[types.Byteptr]
		}
		undty := cp.gotypeof2(sym, cty.Element(), resty)
		return types.NewPointer(undty)
	case cc1.Char, cc1.SChar:
		return types.Typ[types.Int8]
	case cc1.UChar:
		return types.Typ[types.Byte]
	case cc1.Bool:
		return types.Typ[types.Bool]
	case cc1.ULong, cc1.ULongLong:
		return types.Typ[types.Uint64]
	case cc1.Long, cc1.LongLong:
		return types.Typ[types.Int64]
	case cc1.UInt:
		return types.Typ[types.Uint]
	case cc1.Int:
		return types.Typ[types.Int]
	case cc1.UShort:
		return types.Typ[types.Uint16]
	case cc1.Short:
		return types.Typ[types.Int16]
	case cc1.Double:
		return types.Typ[types.Float64]
	case cc1.FLOAT:
		return types.Typ[types.Float32]
	default:
		log.Println(cp.name, sym, cty.Kind(), cty)
	}
	return nil
}

// ddl.Token format: 3 parts: file id name
func cp2_token_toname(tkstr string) string {
	idname := strings.Trim(strings.Split(tkstr, " ")[2], "\"")
	return idname
}

func (cp *cparser2) collects1() {
	ctu1 := cp.ctu1
	declids1 := ctu1.Declarations.Identifiers
	decltags1 := ctu1.Declarations.Tags // struct/union/enum here
	for ikey, ido := range declids1 {
		_ = ikey
		ddl := ido.Node.(*cc1.DirectDeclarator)
		td := ddl.TopDeclarator()
		// log.Println(ikey, td.Type, "//", ddl.EnumVal, td.Type.Kind(),
		//	ddl.Token, "//", ddl.Token2, "//", ddl.Token3)
		// ddl.Token format: 3 parts: file id name
		idname := strings.Trim(strings.Split(ddl.Token.String(), " ")[2], "\"")
		cp.syms[idname] = newcsymdata2(idname, int(td.Type.Kind()), td.Type)
	}
	for ikey, tagx := range decltags1 {
		_ = ikey
		switch tago := tagx.Node.(type) {
		case *cc1.StructOrUnionSpecifier:
			sty := tago.Declarator().Type
			// members, _ := sty.Members()
			stname := strings.Split(sty.String(), " ")[1]
			// log.Println(ikey, sty.Kind(), stname, sty, "fieldcnt", len(members))
			cp.syms[stname] = newcsymdata2(stname, csym_struct, sty)
		case *cc1.EnumSpecifier:
			// log.Println(ikey, "enum", tago.EnumeratorList.Enumerator.Value)
		case xc.Token:
			// log.Println(ikey, "xc.Token?", tago.Val)
		default:
			log.Println(ikey, reftyof(tagx.Node))
		}
	}
	for ikey, macx := range ctu1.Macros {
		_ = ikey
		idname := strings.Trim(strings.Split(macx.DefTok.String(), " ")[2], "\"")
		// log.Println(ikey, idname, macx.Type, macx.Value)
		cp.syms[idname] = newcsymdata2(idname, csym_define, nil)
	}

	log.Println("macros", len(ctu1.Macros), "syms", len(cp.syms))
}

func (cp *cparser2) dumpdecls1() {
	ctu1 := cp.ctu1
	declids1 := ctu1.Declarations.Identifiers
	decltags1 := ctu1.Declarations.Tags // struct/union/enum here
	for ikey, ido := range declids1 {
		ddl := ido.Node.(*cc1.DirectDeclarator)
		td := ddl.TopDeclarator()
		log.Println(ikey, td.Type, "//", ddl.EnumVal, td.Type.Kind(),
			ddl.Token, "//", ddl.Token2, "//", ddl.Token3)
		prms, varidic := td.Type.Parameters()
		_ = varidic
		for idx, prmo := range prms {
			log.Println(idx, prmo.Name, prmo.Type, prmo.Type.Kind(), prmo.Type.Element())
		}
	}
	for ikey, tagx := range decltags1 {
		switch tago := tagx.Node.(type) {
		case *cc1.StructOrUnionSpecifier:
			sty := tago.Declarator().Type
			members, _ := sty.Members()
			log.Println(ikey, "struct", sty, "fieldcnt", len(members))
		case *cc1.EnumSpecifier:
			log.Println(ikey, "enum", tago.EnumeratorList.Enumerator.Value)
		default:
			log.Println(ikey, reftyof(tagx.Node))
		}

	}

	log.Println("macros", len(ctu1.Macros), "tags", len(ctu1.Declarations.Tags),
		"declids", len(ctu1.Declarations.Identifiers))
}

func cprsavetmp(cpname string, code string) (string, error) {
	rmoldtccppfiles()
	filename := fmt.Sprintf("/tmp/tcctrspp.%s.%d.c", cpname, rand.Intn(10000000)+50000)
	cp1cache.ppfiles[filename] = 1
	err := ioutil.WriteFile(filename, []byte(code), 0644)
	return filename, err
}

func (cp *cparser2) parsestr(code string) error {
	code = c11_builtin_atomic_defs + codepfx + code
	filename, err := cprsavetmp(cp.name, code)
	err = cp.parsefile(filename)
	return err
}

// preprocessor
func (cp *cparser2) cpp() {}

// parser
// check
