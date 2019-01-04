package annotation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ann  Annotation
	once sync.Once
)

// GetAnnotation returns singleton of annotaiton
func GetAnnotation() Annotation {
	once.Do(func() {
		ann = Build()
		ann.Header("kubebuilder")
	})
	return ann
}

// ParseAnnotationByDir parses the Go files under given directory and parses the annotation by
// invoking the Parse function on each comment group (multi-lines comments).
func ParseAnnotationByDir(dir string, ann Annotation) error {
	fset := token.NewFileSet()

	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if !isGoFile(info) {
				return nil
			}
			return ParseAnnotationByFile(fset, path, nil, ann)
		})
	return err
}

// ParseAnnotationByFile parses given filename or content src and parses annotations by
// invoking the parseFn function on each comment group (multi-lines comments).
func ParseAnnotationByFile(fset *token.FileSet, path string, src interface{}, ann Annotation) error {
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		fmt.Printf("error from parse.ParseFile: %v", err)
		return err
	}

	// using commentMaps here because it sanitizes the comment text by removing
	// comment markers, compresses newlines etc.
	cmap := ast.NewCommentMap(fset, f, f.Comments)
	for _, commentGroup := range cmap.Comments() {
		err = ann.Parse(commentGroup.Text())
		if err != nil {
			fmt.Print("error when parsing annotation")
			return err
		}
	}
	return nil
}

// OldGetAnnotation extracts the annotation from comment text.
// It will return "foo" for comment "+kubebuilder:webhook:foo" .
func OldGetAnnotation(c, name string) string {
	prefix := fmt.Sprintf("+%s:", name)
	if strings.HasPrefix(c, prefix) {
		return strings.TrimPrefix(c, prefix)
	}
	return ""
}
