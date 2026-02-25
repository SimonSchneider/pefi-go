package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseAmount parses a string as either a single number or a simple arithmetic
// expression (+, -, *, /, parentheses). Only the computed result is returned.
// Scientific notation (e.g. 1e-10) is supported and parsed as a single number.
func ParseAmount(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	// Try as single number first so scientific notation (1e-10) is not treated as expression.
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	if !looksLikeExpression(s) {
		return 0, fmt.Errorf("invalid amount: %q", s)
	}
	return evalExpr(s)
}

func looksLikeExpression(s string) bool {
	for i, r := range s {
		if r == '+' || r == '*' || r == '/' || r == '(' || r == ')' {
			return true
		}
		if r == '-' && i > 0 {
			return true
		}
	}
	return false
}

type tokenKind int

const (
	tokNumber tokenKind = iota
	tokAdd
	tokSub
	tokMul
	tokDiv
	tokLparen
	tokRparen
	tokEOF
)

type token struct {
	kind tokenKind
	val  float64
}

type lexer struct {
	s   string
	pos int
}

func (l *lexer) peek() byte {
	if l.pos >= len(l.s) {
		return 0
	}
	return l.s[l.pos]
}

func (l *lexer) advance() byte {
	if l.pos >= len(l.s) {
		return 0
	}
	b := l.s[l.pos]
	l.pos++
	return b
}

func (l *lexer) skipSpaces() {
	for l.pos < len(l.s) && (l.s[l.pos] == ' ' || l.s[l.pos] == '\t') {
		l.pos++
	}
}

func (l *lexer) readNumber() (float64, error) {
	start := l.pos
	// Optional leading minus
	if l.peek() == '-' {
		l.advance()
	}
	// Integer part
	if l.pos >= len(l.s) || !unicode.IsDigit(rune(l.s[l.pos])) {
		return 0, fmt.Errorf("expected number at position %d", l.pos)
	}
	for l.pos < len(l.s) && unicode.IsDigit(rune(l.s[l.pos])) {
		l.advance()
	}
	// Optional decimal part
	if l.peek() == '.' {
		l.advance()
		for l.pos < len(l.s) && unicode.IsDigit(rune(l.s[l.pos])) {
			l.advance()
		}
	}
	// Optional exponent
	if l.pos < len(l.s) && (l.s[l.pos] == 'e' || l.s[l.pos] == 'E') {
		l.advance()
		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}
		for l.pos < len(l.s) && unicode.IsDigit(rune(l.s[l.pos])) {
			l.advance()
		}
	}
	numStr := l.s[start:l.pos]
	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", numStr, err)
	}
	return f, nil
}

func (l *lexer) next() (token, error) {
	l.skipSpaces()
	if l.pos >= len(l.s) {
		return token{kind: tokEOF}, nil
	}
	switch l.peek() {
	case '+':
		l.advance()
		return token{kind: tokAdd}, nil
	case '-':
		l.advance()
		return token{kind: tokSub}, nil
	case '*':
		l.advance()
		return token{kind: tokMul}, nil
	case '/':
		l.advance()
		return token{kind: tokDiv}, nil
	case '(':
		l.advance()
		return token{kind: tokLparen}, nil
	case ')':
		l.advance()
		return token{kind: tokRparen}, nil
	default:
		if unicode.IsDigit(rune(l.peek())) || (l.peek() == '-' && l.pos+1 < len(l.s) && unicode.IsDigit(rune(l.s[l.pos+1]))) {
			val, err := l.readNumber()
			if err != nil {
				return token{}, err
			}
			return token{kind: tokNumber, val: val}, nil
		}
		return token{}, fmt.Errorf("unexpected character %q at position %d", l.peek(), l.pos)
	}
}

type parser struct {
	lex   *lexer
	tok   token
	tokErr error
}

func (p *parser) advance() error {
	if p.tokErr != nil {
		return p.tokErr
	}
	p.tok, p.tokErr = p.lex.next()
	return p.tokErr
}

// parseExpr parses an expression; it advances to read the first token, then parses.
func (p *parser) parseExpr() (float64, error) {
	if err := p.advance(); err != nil {
		return 0, err
	}
	return p.parseExprContent()
}

// parseExprContent parses expression content assuming the first token is already in p.tok.
// Used after consuming "(" so we don't skip the first token of the parenthesized expression.
func (p *parser) parseExprContent() (float64, error) {
	val, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for p.tok.kind == tokAdd || p.tok.kind == tokSub {
		op := p.tok.kind
		if err := p.advance(); err != nil {
			return 0, err
		}
		rhs, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == tokAdd {
			val += rhs
		} else {
			val -= rhs
		}
	}
	return val, nil
}

func (p *parser) parseTerm() (float64, error) {
	val, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for p.tok.kind == tokMul || p.tok.kind == tokDiv {
		op := p.tok.kind
		if err := p.advance(); err != nil {
			return 0, err
		}
		rhs, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		if op == tokMul {
			val *= rhs
		} else {
			if rhs == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			val /= rhs
		}
	}
	return val, nil
}

func (p *parser) parseFactor() (float64, error) {
	switch p.tok.kind {
	case tokNumber:
		val := p.tok.val
		if err := p.advance(); err != nil {
			return 0, err
		}
		return val, nil
	case tokLparen:
		if err := p.advance(); err != nil {
			return 0, err
		}
		// Parse expression content without advancing first (we already have the first token).
		val, err := p.parseExprContent()
		if err != nil {
			return 0, err
		}
		if p.tok.kind != tokRparen {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		if err := p.advance(); err != nil {
			return 0, err
		}
		return val, nil
	case tokSub:
		// Unary minus
		if err := p.advance(); err != nil {
			return 0, err
		}
		val, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		return -val, nil
	default:
		return 0, fmt.Errorf("unexpected token at position %d", p.lex.pos)
	}
}

func evalExpr(s string) (float64, error) {
	p := &parser{lex: &lexer{s: s}}
	val, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if p.tok.kind != tokEOF {
		return 0, fmt.Errorf("unexpected token after expression")
	}
	return val, nil
}
