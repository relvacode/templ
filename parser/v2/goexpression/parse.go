package goexpression

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var (
	ErrContainerFuncNotFound = errors.New("parser error: templ container function not found")
	ErrExpectedNodeNotFound  = errors.New("parser error: expected node not found")
)

var defaultRegexp = regexp.MustCompile(`^default\s*:`)

func Case(content string) (start, end int, err error) {
	if !(strings.HasPrefix(content, "case ") || defaultRegexp.MatchString(content)) {
		return 0, 0, ErrExpectedNodeNotFound
	}
	prefix := "switch {\n"
	src := prefix + content
	start, end, err = extract(src, func(body []ast.Stmt) (start, end int, err error) {
		sw, ok := body[0].(*ast.SwitchStmt)
		if !ok {
			return 0, 0, ErrExpectedNodeNotFound
		}
		if sw.Body == nil || len(sw.Body.List) == 0 {
			return 0, 0, ErrExpectedNodeNotFound
		}
		stmt, ok := sw.Body.List[0].(*ast.CaseClause)
		if !ok {
			return 0, 0, ErrExpectedNodeNotFound
		}
		start = int(stmt.Case) - 1
		end = int(stmt.Colon)
		return start, end, nil
	})
	if err != nil {
		return 0, 0, err
	}
	// Since we added a `switch {` prefix, we need to remove it.
	start -= len(prefix)
	end -= len(prefix)
	return start, end, nil
}

func If(content string) (start, end int, err error) {
	if !strings.HasPrefix(content, "if") {
		return 0, 0, ErrExpectedNodeNotFound
	}
	return extract(content, func(body []ast.Stmt) (start, end int, err error) {
		stmt, ok := body[0].(*ast.IfStmt)
		if !ok {
			return 0, 0, ErrExpectedNodeNotFound
		}
		start = int(stmt.If) + len("if")
		end = latestEnd(start, stmt.Init, stmt.Cond)
		return start, end, nil
	})
}

func For(content string) (start, end int, err error) {
	if !strings.HasPrefix(content, "for") {
		return 0, 0, ErrExpectedNodeNotFound
	}
	return extract(content, func(body []ast.Stmt) (start, end int, err error) {
		stmt := body[0]
		switch stmt := stmt.(type) {
		case *ast.ForStmt:
			start = int(stmt.For) + len("for")
			end = latestEnd(start, stmt.Init, stmt.Cond, stmt.Post)
			return start, end, nil
		case *ast.RangeStmt:
			start = int(stmt.For) + len("for")
			end = latestEnd(start, stmt.Key, stmt.Value, stmt.X)
			return start, end, nil
		}
		return 0, 0, ErrExpectedNodeNotFound
	})
}

func Switch(content string) (start, end int, err error) {
	if !strings.HasPrefix(content, "switch") {
		return 0, 0, ErrExpectedNodeNotFound
	}
	return extract(content, func(body []ast.Stmt) (start, end int, err error) {
		stmt := body[0]
		switch stmt := stmt.(type) {
		case *ast.SwitchStmt:
			start = int(stmt.Switch) + len("switch")
			end = latestEnd(start, stmt.Init, stmt.Tag)
			return start, end, nil
		case *ast.TypeSwitchStmt:
			start = int(stmt.Switch) + len("switch")
			end = latestEnd(start, stmt.Init, stmt.Assign)
			return start, end, nil
		}
		return 0, 0, ErrExpectedNodeNotFound
	})
}

var boundaryTokens = map[rune]any{
	')':  nil,
	'}':  nil,
	']':  nil,
	' ':  nil,
	'\t': nil,
	'\n': nil,
	'@':  nil,
	'<':  nil,
	'+':  nil,
	'.':  nil,
}

func allBoundaries(content string) (boundaries []int) {
	for i, r := range content {
		if _, ok := boundaryTokens[r]; ok {
			boundaries = append(boundaries, i)
			boundaries = append(boundaries, i+1)
		}
	}
	boundaries = append(boundaries, len(content))
	return
}

func Expression(content string) (start, end int, err error) {
	var candidates []int
	for _, to := range allBoundaries(content) {
		expr, err := parser.ParseExpr(content[:to])
		if err != nil {
			continue
		}
		switch expr := expr.(type) {
		case *ast.CallExpr:
			end = int(expr.Rparen)
		default:
			end = int(expr.End()) - 1
		}
		// If the expression ends with `...` then it's a child spread expression.
		if end < len(content) {
			if strings.HasPrefix(content[end:], "...") {
				end += len("...")
			}
		}
		candidates = append(candidates, end)
	}
	if len(candidates) == 0 {
		return 0, 0, ErrExpectedNodeNotFound
	}
	slices.Sort(candidates)
	return 0, candidates[len(candidates)-1], nil
}

func SliceArgs(content string) (expr string, err error) {
	prefix := "package main\nvar templ_args = []any{"
	src := prefix + content + "}"

	node, parseErr := parser.ParseFile(token.NewFileSet(), "", src, parser.AllErrors)
	if node == nil {
		return expr, parseErr
	}

	var from, to int
	inspectFirstNode(node, func(n ast.Node) bool {
		decl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		from = int(decl.Lbrace)
		to = int(decl.Rbrace) - 1
		for _, e := range decl.Elts {
			to = int(e.End()) - 1
		}
		if to > int(decl.Rbrace)-1 {
			to = int(decl.Rbrace) - 1
		}
		betweenEndAndBrace := src[to : decl.Rbrace-1]
		var hasCodeBetweenEndAndBrace bool
		for _, r := range betweenEndAndBrace {
			if !unicode.IsSpace(r) {
				hasCodeBetweenEndAndBrace = true
				break
			}
		}
		if hasCodeBetweenEndAndBrace {
			to = int(decl.Rbrace) - 1
		}
		return false
	})

	return src[from:to], err
}

// Func returns the Go code up to the opening brace of the function body.
func Func(content string) (name, expr string, err error) {
	prefix := "package main\n"
	src := prefix + content

	node, parseErr := parser.ParseFile(token.NewFileSet(), "", src, parser.AllErrors)
	if node == nil {
		return name, expr, parseErr
	}

	inspectFirstNode(node, func(n ast.Node) bool {
		// Find the first function declaration.
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		start := int(fn.Pos()) + len("func")
		end := fn.Type.Params.End() - 1
		if len(src) < int(end) {
			err = errors.New("parser error: function identifier")
			return false
		}
		expr = src[start:end]
		name = fn.Name.Name
		return false
	})

	return name, expr, err
}

func latestEnd(start int, nodes ...ast.Node) (end int) {
	end = start
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if int(n.End())-1 > end {
			end = int(n.End()) - 1
		}
	}
	return end
}

func inspectFirstNode(node ast.Node, f func(ast.Node) bool) {
	var stop bool
	ast.Inspect(node, func(n ast.Node) bool {
		if stop {
			return true
		}
		if f(n) {
			return true
		}
		stop = true
		return false
	})
}

// Extract a Go expression from the content.
// The Go expression starts at "start" and ends at "end".
// The reader should skip until "length" to pass over the expression and into the next
// logical block.
type Extractor func(body []ast.Stmt) (start, end int, err error)

func extract(content string, extractor Extractor) (start, end int, err error) {
	prefix := "package main\nfunc templ_container() {\n"
	src := prefix + content

	node, parseErr := parser.ParseFile(token.NewFileSet(), "", src, parser.AllErrors)
	if node == nil {
		return 0, 0, parseErr
	}

	var found bool
	inspectFirstNode(node, func(n ast.Node) bool {
		// Find the "templ_container" function.
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if fn.Name.Name != "templ_container" {
			err = ErrContainerFuncNotFound
			return false
		}
		if fn.Body.List == nil || len(fn.Body.List) == 0 {
			err = ErrExpectedNodeNotFound
			return false
		}
		found = true
		start, end, err = extractor(fn.Body.List)
		return false
	})
	if !found {
		return 0, 0, ErrExpectedNodeNotFound
	}

	start -= len(prefix)
	end -= len(prefix)

	if end > len(content) {
		end = len(content)
	}
	if start > end {
		start = end
	}

	return start, end, err
}
