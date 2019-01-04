/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package rbac contain libraries for generating RBAC manifests from RBAC
// annotations in Go source files.
package rbac

import (
	"log"
	"strings"

	"github.com/fanzhangio/go-annotation/pkg/annotation"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	rbac = "rbac"
	tags = sets.NewString([]string{"groups", "resources", "verbs", "urls"}...)
)

type parserOptions struct {
	rules []rbacv1.PolicyRule
}

func (o *parserOptions) AddToAnnotation(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name: "rbac",
		Do:   o.ParseRBAC,
	})
	return a
}

// parseRBACTag parses the given RBAC annotation in to an RBAC PolicyRule.
// This is copied from Kubebuilder code.
func (o *parserOptions) ParseRBAC(tag string) (err error) {
	result := rbacv1.PolicyRule{}
	for _, elem := range strings.Split(tag, ",") {
		key, value, err := annotation.ParseKV(elem)
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
	o.rules = append(o.rules, result)
	return
}
