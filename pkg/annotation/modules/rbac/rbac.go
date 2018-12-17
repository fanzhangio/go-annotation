package rbac

import (
	"log"
	"strings"

	general "github.com/fanzhangio/go-annotation/pkg/annotation"
	rbacv1 "k8s.io/api/rbac/v1"
)

// Wrapper
type ManifestOptions struct {
	InputDir  string
	OutputDir string
	Name      string
	Labels    map[string]string
	parserOptions
}

type parserOptions struct {
	rules []rbacv1.PolicyRule
}

// parseAnnotation parses RBAC annotations
func (o *parserOptions) parseAnnotation(commentText string) error {
	for _, comment := range strings.Split(commentText, "\n") {
		comment := strings.TrimSpace(comment)
		if strings.HasPrefix(comment, "+rbac") {
			if ann := general.GetAnnotation(comment, "rbac"); ann != "" {
				o.rules = append(o.rules, parseRBACTag(ann))
			}
		}
		if strings.HasPrefix(comment, "+kubebuilder:rbac") {
			if ann := general.GetAnnotation(comment, "kubebuilder:rbac"); ann != "" {
				o.rules = append(o.rules, parseRBACTag(ann))
			}
		}
	}
	return nil
}

// parseRBACTag parses the given RBAC annotation in to an RBAC PolicyRule.
// This is copied from Kubebuilder code.
func parseRBACTag(tag string) rbacv1.PolicyRule {
	result := rbacv1.PolicyRule{}
	for _, elem := range strings.Split(tag, ",") {
		key, value, err := general.ParseKV(elem)
		if err != nil {
			log.Fatalf("// +kubebuilder:rbac: tags must be key value pairs.  Expected "+
				"keys [groups=<group1;group2>,resources=<resource1;resource2>,verbs=<verb1;verb2>] "+
				"Got string: [%s]", tag)
		}
		values := strings.Split(value, ";")
		switch key {
		case "groups":
			normalized := []string{}
			for _, v := range values {
				if v == "core" {
					normalized = append(normalized, "")
				} else {
					normalized = append(normalized, v)
				}
			}
			result.APIGroups = normalized
		case "resources":
			result.Resources = values
		case "verbs":
			result.Verbs = values
		case "urls":
			result.NonResourceURLs = values
		}
	}
	return result
}
