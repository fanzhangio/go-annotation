package annotation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

var ann Annotation

func init() {
	// run once
	ann = Build()
	ann.Header("kubebuilder")
}

func GetAnnotation() Annotation {
	return ann
}

// ParseAnnotation parses the Go files under given directory and parses the annotation by
// invoking the Parse function on each comment group (multi-lines comments).
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
