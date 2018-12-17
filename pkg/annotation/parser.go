package annotation

import (
	"fmt"
	"strings"
)

// GetAnnotation extracts the annotation from comment text.
// It will return "foo" for comment "+kubebuilder:webhook:foo" .
func GetAnnotation(c, name string) string {
	prefix := fmt.Sprintf("+%s:", name)
	if strings.HasPrefix(c, prefix) {
		return strings.TrimPrefix(c, prefix)
	}
	return ""
}
