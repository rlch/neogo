package internal

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type (
	token int
	lexer struct {
		r *strings.Reader
	}
	parser struct {
		l   *lexer
		buf struct {
			tok token
			lit string
			n   int
		}
	}
)

const (
	tokenEOF token = iota
	tokenWS
	tokenIllegal

	tokenIdent
	tokenLeftBrace
	tokenRightBrace
	tokenComma
	tokenColon
	tokenDot
)

var (
	eof                = rune(0)
	errUnexpectedToken = func(name string) error { return fmt.Errorf("unexpected token: %s", name) }
)

func newQueryScanner(input string) *lexer {
	return &lexer{
		r: strings.NewReader(input),
	}
}

func newQueryParser(input string) *parser {
	return &parser{
		l: newQueryScanner(input),
	}
}

func (l *lexer) read() rune {
	ch, _, err := l.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

func (l *lexer) unread() {
	_ = l.r.UnreadRune()
}

func (l *lexer) scan() (tok token, lit string) {
	ch := l.read()

	if isWhitespace(ch) {
		l.unread()
		return l.scanWhitespace()
	} else if isIdent(ch) {
		l.unread()
		return l.scanIdent()
	}

	switch ch {
	case eof:
		return tokenEOF, ""
	case '{':
		return tokenLeftBrace, "{"
	case '}':
		return tokenRightBrace, "}"
	case ',':
		return tokenComma, ","
	case ':':
		return tokenColon, ":"
	case '.':
		return tokenDot, "."
	}
	return tokenIllegal, string(ch)
}

func (l *lexer) scanWhitespace() (tok token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(l.read())
	for {
		if ch := l.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			l.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}
	return tokenWS, buf.String()
}

func (l *lexer) scanIdent() (tok token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(l.read())
	for {
		if ch := l.read(); ch == eof {
			break
		} else if !isIdent(ch) {
			l.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return tokenIdent, buf.String()
}

// scan returns the next token from the underlying scanner.
// If a token has been unscanned then read that instead.
func (p *parser) scan() (tok token, lit string) {
	// We have a buffered token, return it
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	tok, lit = p.l.scan()
	p.buf.tok, p.buf.lit = tok, lit
	return
}

func (p *parser) unscan() {
	p.buf.n = 1
}

func (p *parser) scanIgnoreWhitespace() (tok token, lit string) {
	for {
		tok, lit = p.scan()
		if tok != tokenWS {
			break
		}
	}
	return
}

func (p *parser) parse() (query QuerySpec, err error) {
	expectNextField := func() error {
		tok, lit := p.scan()
		p.unscan()
		if tok != tokenWS && tok != tokenEOF {
			return fmt.Errorf("expected next field or EOF after dot, got %s", lit)
		}
		return nil
	}

	parseSelector := func() (*QuerySelector, error) {
		qp := &QuerySelector{}
		tok, lit := p.scanIgnoreWhitespace()
		switch tok {
		case tokenIdent:
			if err := expectNextField(); err == nil {
				qp.Field = lit
				return qp, nil
			}
			qp.Name = lit
			tok, _ = p.scan()
		case tokenDot:
			qp.Field = "."
			return qp, expectNextField()
		case tokenEOF:
			return nil, nil
		}
		if tok == tokenLeftBrace {
			qp.Props, err = p.parseProps()
			if err != nil {
				return nil, fmt.Errorf("failed to parse fields: %w", err)
			}
			tok, _ = p.scan()
		}
		if tok == tokenColon {
			tok, lit = p.scan()
		} else {
			return nil, errors.New("expected colon after qualifier/props")
		}
		switch tok {
		case tokenIdent, tokenDot:
			qp.Field = lit
			return qp, expectNextField()
		default:
			return nil, errUnexpectedToken(lit)
		}
	}

	for {
		qp, err := parseSelector()
		if err != nil {
			return nil, err
		}
		if qp != nil {
			query = append(query, qp)
		} else {
			break
		}
	}
	return
}

func (p *parser) parseProps() ([]string, error) {
	var fields []string
	for {
		tok, lit := p.scanIgnoreWhitespace()
		if tok == tokenEOF {
			break
		}
		switch tok {
		case tokenIdent:
			fields = append(fields, lit)
		case tokenComma:
			continue
		case tokenRightBrace:
			if fields == nil {
				// To differentiate between no brackets and empty brackets.
				fields = []string{}
			}
			return fields, nil
		default:
			return nil, fmt.Errorf("expected field, comma or }, got %s", lit)
		}
	}
	return nil, errors.New("unreachable")
}

func isIdent(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-'
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}
