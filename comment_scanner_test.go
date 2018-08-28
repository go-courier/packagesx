package packagesx

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommentScanner(t *testing.T) {
	fset := token.NewFileSet()
	contents, _ := ioutil.ReadFile("./__fixtures__/comments.go")
	file, _ := parser.ParseFile(fset, "./__fixtures__/comments.go", contents, parser.ParseComments)

	commentScanner := NewCommentScanner(fset, file)

	ast.Inspect(file, func(node ast.Node) bool {
		comments := commentScanner.CommentsOf(node)
		require.True(t, len(strings.Split(comments, "\n")) <= 3)
		return true
	})
}
