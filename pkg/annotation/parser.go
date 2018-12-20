package annotation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func ParseAnnotation(dir string, ann Annotation) error {
	fset := token.NewFileSet()

	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if !isGoFile(info) {
				return nil
			}
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				fmt.Printf("error from parse.ParseFile: %v", err)
				return err
			}
			cmap := ast.NewCommentMap(fset, f, f.Comments)
			for _, commentGroup := range cmap.Comments() {
				err = ann.Parse(commentGroup.Text())
				if err != nil {
					fmt.Print("error when parsing annotation")
					return err
				}
			}
			return nil
			// return ParseFile(fset, path, nil, ann)
		})
	return err
}

/************************************************************************************/

// isGoFile filters files from parsing.
func isGoFile(f os.FileInfo) bool {
	// ignore non-Go or Go test files
	name := f.Name()
	return !f.IsDir() &&
		!strings.HasPrefix(name, ".") &&
		!strings.HasSuffix(name, "_test.go") &&
		strings.HasSuffix(name, ".go")
}
