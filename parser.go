/*
parser inspired by Rob Pikes lexer
*/

package parser

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// last State must return ErrEOF
type State func(p *Parser) (next State, err error)

var ErrEOF = errors.New("End of File")

var EOF = rune('∎')

type ASTNode interface {
	AddChild(ASTNode)
}

type Parser struct {
	astQueue    []ASTNode
	input       string // the string being scanned
	start       int    // start position of this item
	pos         int    // current position in the input
	width       int    // width of the last rune read
	line        int
	linepos     int
	linePrev    int
	lineposPrev int
}

func New(input string, root ASTNode) *Parser {
	return &Parser{
		astQueue: []ASTNode{root},
		input:    input,
	}
}

func (p *Parser) Root() ASTNode {
	return p.astQueue[0]
}

func (p *Parser) currentNode() ASTNode {
	return p.astQueue[len(p.astQueue)-1]
}

func (p *Parser) AddNode(n ASTNode) {
	p.currentNode().AddChild(n)
	p.astQueue = append(p.astQueue, n)
}

func (p *Parser) PopNode() {
	if len(p.astQueue) < 2 {
		return
	}
	p.astQueue = p.astQueue[:len(p.astQueue)-1]
}

func (p *Parser) Next() (rune_ rune) {
	if p.pos >= len(p.input) {
		p.width = 0
		return EOF
	}
	rune_, p.width = utf8.DecodeRuneInString(p.input[p.pos:])
	p.pos += p.width
	p.linePrev = p.line
	p.lineposPrev = p.linepos
	if rune_ == '\n' {
		p.line++
		p.linepos = 0
	} else {
		p.linepos++
	}

	return
}

// emit passes an item back to the client
func (p *Parser) Emit() string {
	s := p.input[p.start:p.pos]
	p.start = p.pos
	return s
}

func (p *Parser) Ignore() {
	p.start = p.pos
}

// backup steps back one rune
// can be called only once per call of next
func (p *Parser) Backup() {
	rune_, _ := utf8.DecodeRuneInString(p.input[p.pos:])
	if rune_ == '\n' {
		p.line--
	}
	p.linepos = p.lineposPrev
	p.pos -= p.width
}

func (p *Parser) Peek() rune {
	r := p.Next()
	p.Backup()
	return r
}

func (p *Parser) Accept(valid string) bool {
	if strings.IndexRune(valid, p.Next()) >= 0 {
		return true
	}
	p.Backup()
	return false
}

func (p *Parser) AcceptRun(valid string) {
	for strings.IndexRune(valid, p.Next()) >= 0 {
	}
	p.Backup()
}

func (p *Parser) Errorf(format string, args ...interface{}) error {
	start := p.pos - 5
	if start < 0 {
		start = 0
	}

	end := p.pos + 5

	if end > len(p.input) {
		end = len(p.input)
	}

	return errors.New(fmt.Sprintf(
		"Error in line %d at position %d: %s\ncontext:\n%s\n",
		p.line+1,
		p.linepos+1,
		fmt.Sprintf(format, args...),
		p.input[start:end],
	))
}

func (p *Parser) Run(fn State) (err error) {
	for err == nil {
		fn, err = fn(p)
	}
	if err == ErrEOF {
		return nil
	}

	return err
}
