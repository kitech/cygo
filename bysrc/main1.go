package main

import (
	"gopp"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var fname string

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("must specify a go source file to tranpiler")
	}
	fname = os.Args[1]

	psctx := NewParserContext(fname)
	err := psctx.Init()
	gopp.ErrPrint(err)
	if err != nil && !strings.Contains(err.Error(), "declared but not used") {
		println()
		println()
		time.Sleep(2 * time.Second)
	}

	// g2n := g2nim{}
	g2n := g2nc{}
	g2n.psctx = psctx
	g2n.genpkgs()
	code, ext := g2n.code()
	ioutil.WriteFile("opkgs/foo."+ext, []byte(code), 0644)
}
