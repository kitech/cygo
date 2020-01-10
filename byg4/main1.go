package main

import "github.com/antlr/antlr4/runtime/Go/antlr"
import "cxrt/byg4/parser"
import "fmt"

func main() {
	// Setup the input
	is := antlr.NewInputStream("1 + 2 * 3")

	// Create the Lexer
	lexer := parser.NewCalcLexer(is)

	// Read all tokens
	for {
		t := lexer.NextToken()
		if t.GetTokenType() == antlr.TokenEOF {
			break
		}
		fmt.Printf("%s (%q)\n",
			lexer.SymbolicNames[t.GetTokenType()], t.GetText())
	}
}
