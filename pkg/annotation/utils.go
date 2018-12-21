package annotation

import (
	"fmt"
	"os"
	"strings"
)

func prefixName(name string) string {
	return "+" + name
}

// isGoFile filters files from parsing.
func isGoFile(f os.FileInfo) bool {
	// ignore non-Go or Go test files
	name := f.Name()
	return !f.IsDir() &&
		!strings.HasPrefix(name, ".") &&
		!strings.HasSuffix(name, "_test.go") &&
		strings.HasSuffix(name, ".go")
}

// ParseKV parses key-value string formatted as "foo=bar" and returns key and value.
func ParseKV(s string) (key, value string, err error) {
	kv := strings.Split(s, "=")
	if len(kv) != 2 {
		err = fmt.Errorf("invalid key value pair")
		return key, value, err
	}
	key, value = kv[0], kv[1]
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		value = value[1 : len(value)-1]
	}
	return key, value, err
}
