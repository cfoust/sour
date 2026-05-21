package cscript

import (
	"strings"
)

// Node types for the AST
type Node interface{}

// A sequence of statements
type Block struct {
	Stmts []Node
}

// A command invocation: command arg1 arg2 ...
type Command struct {
	Name Node   // usually a Word, but can be any expression
	Args []Node
}

// Assignment: name = value
type Assignment struct {
	Name  string
	Value Node
}

// A literal word/identifier
type Word struct {
	Val string
}

// A quoted string: "..."
type StringLit struct {
	Val string
}

// A bracket block: [...]
type CodeBlock struct {
	Body string // raw source inside brackets, for lazy evaluation
}

// A subexpression: (...)
type SubExpr struct {
	Body *Block
}

// A variable lookup: $name or $(expr) or $[expr]
type Lookup struct {
	Name Node // Word or SubExpr
}

// An interpolated block: [text @expr more]
type Interpolation struct {
	Parts []Node // mix of StringLit and Lookup/SubExpr
}

// Parse parses CubeScript source into a Block (list of statements).
func Parse(src string) *Block {
	p := &parser{src: src, pos: 0}
	return p.parseBlock('\x00')
}

type parser struct {
	src string
	pos int
}

func (p *parser) eof() bool {
	return p.pos >= len(p.src)
}

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) next() byte {
	if p.eof() {
		return 0
	}
	b := p.src[p.pos]
	p.pos++
	return b
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == ' ' || c == '\t' || c == '\r' {
			p.pos++
		} else if c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/' {
			// line comment
			for p.pos < len(p.src) && p.src[p.pos] != '\n' && p.src[p.pos] != 0 {
				p.pos++
			}
		} else {
			break
		}
	}
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			p.pos++
		} else if c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/' {
			for p.pos < len(p.src) && p.src[p.pos] != '\n' && p.src[p.pos] != 0 {
				p.pos++
			}
		} else {
			break
		}
	}
}

// parseBlock parses statements until EOF or closing bracket.
func (p *parser) parseBlock(endChar byte) *Block {
	block := &Block{}
	for {
		p.skipWhitespace()
		if p.eof() {
			break
		}
		if endChar != 0 && p.peek() == endChar {
			p.pos++
			break
		}
		if p.peek() == ')' || p.peek() == ']' {
			// unexpected closer, skip
			p.pos++
			break
		}
		stmt := p.parseStatement(endChar)
		if stmt != nil {
			block.Stmts = append(block.Stmts, stmt)
		}
	}
	return block
}

// parseStatement parses a single statement (command or assignment).
func (p *parser) parseStatement(endChar byte) Node {
	p.skipSpaces()
	if p.eof() || p.peek() == '\n' || p.peek() == ';' || p.peek() == endChar {
		if !p.eof() && (p.peek() == '\n' || p.peek() == ';') {
			p.pos++
		}
		return nil
	}

	// Parse the first word
	first := p.parseArg()
	if first == nil {
		p.skipToEnd(endChar)
		return nil
	}

	p.skipSpaces()

	// Check for assignment: name = value
	if w, ok := first.(*Word); ok && !p.eof() && p.peek() == '=' {
		// Check the next char after =
		if p.pos+1 < len(p.src) {
			next := p.src[p.pos+1]
			if next == '/' || next == ';' || next == ' ' || next == '\t' || next == '\r' || next == '\n' || next == 0 || p.pos+1 >= len(p.src) {
				p.pos++ // skip =
				p.skipSpaces()
				val := p.parseArg()
				if val == nil {
					val = &StringLit{Val: ""}
				}
				// Consume rest of args (shouldn't be any for assignment)
				p.skipToEnd(endChar)
				return &Assignment{Name: w.Val, Value: val}
			}
		} else {
			p.pos++ // skip = at end
			p.skipToEnd(endChar)
			return &Assignment{Name: w.Val, Value: &StringLit{Val: ""}}
		}
	}

	// It's a command invocation
	cmd := &Command{Name: first}
	for {
		p.skipSpaces()
		if p.eof() || p.peek() == '\n' || p.peek() == ';' {
			if !p.eof() {
				p.pos++
			}
			break
		}
		if p.peek() == endChar && endChar != 0 {
			break
		}
		if p.peek() == ')' || p.peek() == ']' {
			break
		}
		arg := p.parseArg()
		if arg == nil {
			break
		}
		cmd.Args = append(cmd.Args, arg)
	}

	return cmd
}

// parseArg parses a single argument value.
func (p *parser) parseArg() Node {
	p.skipSpaces()
	if p.eof() {
		return nil
	}

	switch p.peek() {
	case '"':
		return p.parseString()
	case '$':
		return p.parseLookup()
	case '(':
		return p.parseSubExpr()
	case '[':
		return p.parseBracket()
	case '\n', ';', 0:
		return nil
	case ')':
		return nil
	case ']':
		return nil
	default:
		return p.parseWord()
	}
}

// parseString parses a quoted string "...".
func (p *parser) parseString() Node {
	p.pos++ // skip opening "
	var buf strings.Builder
	for !p.eof() {
		c := p.next()
		switch c {
		case '"':
			return &StringLit{Val: buf.String()}
		case '^':
			if !p.eof() {
				e := p.next()
				switch e {
				case 'n':
					buf.WriteByte('\n')
				case 't':
					buf.WriteByte('\t')
				case 'f':
					buf.WriteByte('\f')
				case '"':
					buf.WriteByte('"')
				case '^':
					buf.WriteByte('^')
				default:
					buf.WriteByte(e)
				}
			}
		case '\n', '\r':
			// strings don't cross lines in cubescript
			return &StringLit{Val: buf.String()}
		default:
			buf.WriteByte(c)
		}
	}
	return &StringLit{Val: buf.String()}
}

// parseLookup parses $name, $(expr), or $[expr].
func (p *parser) parseLookup() Node {
	p.pos++ // skip $
	if p.eof() {
		return &Lookup{Name: &Word{Val: ""}}
	}
	switch p.peek() {
	case '(':
		sub := p.parseSubExpr()
		return &Lookup{Name: sub}
	case '[':
		sub := p.parseBracket()
		return &Lookup{Name: sub}
	case '$':
		// nested lookup
		inner := p.parseLookup()
		return &Lookup{Name: inner}
	default:
		// read identifier
		start := p.pos
		for !p.eof() {
			c := p.peek()
			if isAlphaNum(c) || c == '_' {
				p.pos++
			} else {
				break
			}
		}
		name := p.src[start:p.pos]
		return &Lookup{Name: &Word{Val: name}}
	}
}

// parseSubExpr parses (...).
func (p *parser) parseSubExpr() Node {
	p.pos++ // skip (
	block := p.parseBlock(')')
	return &SubExpr{Body: block}
}

// parseBracket parses [...], preserving the raw source for lazy evaluation.
// It also handles @-interpolation at the top level.
func (p *parser) parseBracket() Node {
	p.pos++ // skip [
	start := p.pos
	depth := 1
	hasInterp := false

	// Scan to find the matching ]
	scanPos := p.pos
	for scanPos < len(p.src) && depth > 0 {
		c := p.src[scanPos]
		switch c {
		case '[':
			depth++
		case ']':
			depth--
		case '"':
			scanPos++
			for scanPos < len(p.src) && p.src[scanPos] != '"' && p.src[scanPos] != '\n' {
				if p.src[scanPos] == '^' {
					scanPos++
				}
				scanPos++
			}
		case '@':
			if depth == 1 {
				hasInterp = true
			}
		}
		if depth > 0 {
			scanPos++
		}
	}

	body := p.src[start:scanPos]
	p.pos = scanPos
	if p.pos < len(p.src) && p.src[p.pos] == ']' {
		p.pos++ // skip ]
	}

	if hasInterp {
		return p.parseInterpolated(body)
	}

	return &CodeBlock{Body: body}
}

// parseInterpolated handles @-substitution inside [...] blocks.
func (p *parser) parseInterpolated(body string) Node {
	var parts []Node
	var buf strings.Builder
	i := 0
	for i < len(body) {
		if body[i] == '@' {
			// Flush text buffer
			if buf.Len() > 0 {
				parts = append(parts, &StringLit{Val: buf.String()})
				buf.Reset()
			}
			i++ // skip @
			// Parse what follows
			sub := &parser{src: body, pos: i}
			switch {
			case sub.pos < len(body) && body[sub.pos] == '(':
				node := sub.parseSubExpr()
				parts = append(parts, node)
				i = sub.pos
			case sub.pos < len(body) && body[sub.pos] == '[':
				node := sub.parseBracket()
				// Lookup the result
				parts = append(parts, &Lookup{Name: node})
				i = sub.pos
			default:
				// Read identifier
				start := sub.pos
				for sub.pos < len(body) {
					c := body[sub.pos]
					if isAlphaNum(c) || c == '_' {
						sub.pos++
					} else {
						break
					}
				}
				name := body[start:sub.pos]
				if name != "" {
					parts = append(parts, &Lookup{Name: &Word{Val: name}})
				}
				i = sub.pos
			}
		} else {
			buf.WriteByte(body[i])
			i++
		}
	}
	if buf.Len() > 0 {
		parts = append(parts, &StringLit{Val: buf.String()})
	}

	if len(parts) == 1 {
		if s, ok := parts[0].(*StringLit); ok {
			return &CodeBlock{Body: s.Val}
		}
	}

	return &Interpolation{Parts: parts}
}

// parseWord parses an unquoted word.
func (p *parser) parseWord() Node {
	start := p.pos
	depth := 0 // track parens/brackets
	for !p.eof() {
		c := p.peek()
		switch c {
		case '"', ';', ' ', '\t', '\r', '\n':
			if depth == 0 {
				goto done
			}
		case '/':
			if depth == 0 && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/' {
				goto done
			}
		case '(', '[':
			depth++
		case ')':
			if depth <= 0 {
				goto done
			}
			depth--
		case ']':
			if depth <= 0 {
				goto done
			}
			depth--
		case 0:
			goto done
		}
		p.pos++
	}
done:
	word := p.src[start:p.pos]
	if word == "" {
		return nil
	}
	return &Word{Val: word}
}

func (p *parser) skipToEnd(endChar byte) {
	for !p.eof() {
		c := p.peek()
		if c == '\n' || c == ';' {
			p.pos++
			return
		}
		if endChar != 0 && c == endChar {
			return
		}
		if c == ')' || c == ']' {
			return
		}
		p.pos++
	}
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
