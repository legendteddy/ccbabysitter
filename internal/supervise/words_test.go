package supervise

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// messageCalls are the calls whose string arguments reach the activity
// feed word for word.
var messageCalls = map[string]bool{"logInfo": true, "logError": true, "logAuto": true, "Info": true, "Error": true, "Auto": true}

// sayings collects, from one Go source file, every string literal that is
// shown to a person as it is: the arguments of a log call, the value given
// to a Message field, and every string constant.
func sayings(t *testing.T, path string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	collect := func(n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil {
					out = append(out, s)
				}
			}
			return true
		})
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			name := ""
			switch fn := x.Fun.(type) {
			case *ast.Ident:
				name = fn.Name
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			}
			if messageCalls[name] {
				for _, arg := range x.Args {
					collect(arg)
				}
			}
		case *ast.KeyValueExpr:
			if key, ok := x.Key.(*ast.Ident); ok && key.Name == "Message" {
				collect(x.Value)
			}
		case *ast.GenDecl:
			if x.Tok == token.CONST {
				collect(x)
			}
		}
		return true
	})
	return out
}

// Every result and every activity line is one line, with no parentheses
// and no semicolons. The packages that write them are read from source:
// this one, the web layer with its demonstration engine, the observer and
// the saved state.
func TestMessagesAreOneLineWithoutParenthesesOrSemicolons(t *testing.T) {
	var files []string
	for _, pattern := range []string{"*.go", "../web/*.go", "../observe/*.go", "../state/*.go"} {
		found, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, found...)
	}
	checked := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
		for _, s := range sayings(t, path) {
			checked++
			if strings.ContainsAny(s, "();\n") {
				t.Errorf("%s says %q", path, s)
			}
		}
	}
	if checked < 50 {
		t.Fatalf("only %d sayings found, so this is not checking what it thinks it is", checked)
	}
}
