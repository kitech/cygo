package main

import (
	"log"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	"cxrt/golang/parser"
)

func main() {
	fstm, err := antlr.NewFileStream("./examples/function.go")
	if err != nil {
		log.Panicln(err)
	}
	lexer := parser.NewGoLexer(fstm)
	tkstm := antlr.NewCommonTokenStream(lexer, 0)

	// Create the Lexer
	prs := parser.NewGoParser(tkstm)

	log.Println(prs)
}
