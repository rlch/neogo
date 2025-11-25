// Package parser provides utilities for parsing Cypher query strings using
// an ANTLR-generated grammar. It includes error handling and parse tree metadata.
package parser

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	cypher "github.com/rlch/neogo/grammar"
)

// ParseCypherQuery parses a Cypher query string and returns the parse tree
func ParseCypherQuery(query string) (antlr.ParseTree, error) {
	input := antlr.NewInputStream(query)
	lexer := cypher.NewCypherLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := cypher.NewCypherParser(stream)

	// Set error listeners to track parsing errors
	errorListener := &ParseErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)

	// Parse the script
	tree := parser.Script()

	if len(errorListener.Errors) > 0 {
		return nil, fmt.Errorf("parsing failed with errors: %v", errorListener.Errors)
	}

	return tree, nil
}

// ParseErrorListener tracks parse errors
type ParseErrorListener struct {
	Errors []string
}

func (pel *ParseErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	pel.Errors = append(pel.Errors, fmt.Sprintf("line %d:%d %s", line, column, msg))
}

func (pel *ParseErrorListener) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
}

func (pel *ParseErrorListener) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
}

func (pel *ParseErrorListener) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, prediction int, configs *antlr.ATNConfigSet) {
}

// ParsedQuery holds the parse tree and metadata
type ParsedQuery struct {
	Query     string
	Tree      antlr.ParseTree
	RuleNames []string
}

// ParseWithMetadata parses a Cypher query and returns metadata
func ParseWithMetadata(query string) (*ParsedQuery, error) {
	tree, err := ParseCypherQuery(query)
	if err != nil {
		return nil, err
	}

	input := antlr.NewInputStream(query)
	lexer := cypher.NewCypherLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := cypher.NewCypherParser(stream)

	return &ParsedQuery{
		Query:     query,
		Tree:      tree,
		RuleNames: parser.GetRuleNames(),
	}, nil
}
