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

package parse

import (
	"bufio"
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fanzhangio/go-annotation/pkg/annotation"
	"github.com/fanzhangio/go-annotation/pkg/codegen"
	"github.com/markbates/inflect"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/types"
)

// APIs is the information of a collection of API
type APIs struct {
	context         *generator.Context
	arguments       *args.GeneratorArgs
	Domain          string
	VersionedPkgs   sets.String
	UnversionedPkgs sets.String
	APIsPkg         string
	APIsPkgRaw      *types.Package
	GroupNames      sets.String

	APIs        *codegen.APIs
	Controllers []codegen.Controller

	ByGroupKindVersion    map[string]map[string]map[string]*codegen.APIResource
	ByGroupVersionKind    map[string]map[string]map[string]*codegen.APIResource
	SubByGroupVersionKind map[string]map[string]map[string]*types.Type
	Groups                map[string]types.Package
	Rules                 []rbacv1.PolicyRule
	Informers             map[v1.GroupVersionKind]bool
}

// NewAPIs returns a new APIs instance with given context.
func NewAPIs(context *generator.Context, arguments *args.GeneratorArgs, domain, apisPkg string) *APIs {
	b := &APIs{
		context:   context,
		arguments: arguments,
		Domain:    domain,
		APIsPkg:   apisPkg,
	}

	b.parseAPI()
	b.parseGroupNames()
	b.parseAPIs()
	b.parseCRDs()
	if len(b.Domain) == 0 {
		b.parseDomain()
	}
	return b
}

func (b *APIs) parseAPI() {

	b.VersionedPkgs = sets.NewString()
	b.UnversionedPkgs = sets.NewString()
	b.GroupNames = sets.String{}

	b.ByGroupVersionKind = map[string]map[string]map[string]*codegen.APIResource{}
	b.ByGroupKindVersion = map[string]map[string]map[string]*codegen.APIResource{}
	b.SubByGroupVersionKind = map[string]map[string]map[string]*types.Type{}

	ann := annotation.GetAnnotation()
	for _, t := range b.context.Order {
		// register api annoations

		resource := &codegen.APIResource{}
		b.parseSubresourceRequest(t, ann)
		b.parseAPIResource(resource, ann)
		b.parseNamespace(ann)

		if IsAPIResource(t) {

			// parse packages
			versioned := t.Name.Package
			b.VersionedPkgs.Insert(versioned)
			unversioned := filepath.Dir(versioned)
			b.UnversionedPkgs.Insert(unversioned)

			// parse APIResource by annotations
			parseAPIAnnotation(t, ann)

			// parse APIResource
			res, ok := ann.GetModule("resource").Meta.(*codegen.APIResource)
			if !ok {
				log.Fatalf("api resource is not set in module meta")
			}

			r := &codegen.APIResource{
				Type: t,
			}
			if nonNamespaced, ok := ann.GetModule("nonNamespaced").Meta.(*bool); ok {
				r.NonNamespaced = *nonNamespaced
			}
			r.Group = GetGroup(t)
			r.Version = GetVersion(t, r.Group)
			r.Kind = GetKind(t, r.Group)
			r.Domain = b.Domain

			// TODO: revisit the part...
			if res.Resource != "" {
				r.Resource = res.Resource
			} else {
				r.Resource = strings.ToLower(inflect.Pluralize(r.Kind))
			}
			r.ShortName = res.ShortName

			// Copy the Status strategy to mirror the non-status strategy
			r.StatusStrategy = strings.TrimSuffix(r.Strategy, "Strategy")
			r.StatusStrategy = fmt.Sprintf("%sStatusStrategy", r.StatusStrategy)

			// Initialize the map entries so they aren't nill
			if _, f := b.ByGroupKindVersion[r.Group]; !f {
				b.ByGroupKindVersion[r.Group] = map[string]map[string]*codegen.APIResource{}
			}
			if _, f := b.ByGroupKindVersion[r.Group][r.Kind]; !f {
				b.ByGroupKindVersion[r.Group][r.Kind] = map[string]*codegen.APIResource{}
			}
			if _, f := b.ByGroupVersionKind[r.Group]; !f {
				b.ByGroupVersionKind[r.Group] = map[string]map[string]*codegen.APIResource{}
			}
			if _, f := b.ByGroupVersionKind[r.Group][r.Version]; !f {
				b.ByGroupVersionKind[r.Group][r.Version] = map[string]*codegen.APIResource{}
			}
			// Add the resource to the map
			b.ByGroupKindVersion[r.Group][r.Kind][r.Version] = r
			b.ByGroupVersionKind[r.Group][r.Version][r.Kind] = r
			r.Type = t
		}
	}

}

// parseAPI annotation
func parseAPIAnnotation(t *types.Type, ann annotation.Annotation) error {
	for _, c := range t.CommentLines {
		if err := ann.Parse(c); err != nil {
			return err
		}
	}
	return nil
}

func (b *APIs) parseAPIResource(r *codegen.APIResource, a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name: "resource",
		Meta: r,
		Do: func(commentText string) error {
			// fmt.Printf("\n[Debug]] ... parseResourceAnnotation() with comment (%s)\n", commentText)
			// indexes all types with the comment "// +resource=RESOURCE" by GroupVersionKind and GroupKindVersion
			for _, elem := range strings.Split(commentText, ",") {
				key, value, err := annotation.ParseKV(elem)
				if err != nil {
					return fmt.Errorf("// +kubebuilder:resource: tags must be key value pairs.  Expected "+
						"keys [path=<resourcepath>] "+
						"Got string: [%s]", commentText)
				}
				switch key {
				case "path":
					r.Resource = value
				case "shortName":
					r.ShortName = value
				default:
					return fmt.Errorf("The given input %s is invalid", value)
				}
			}
			return nil
		},
	})
	return a
}

// subresourceRequest module is for compatibility
func (b *APIs) parseSubresourceRequest(t *types.Type, a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name: "subresource-request",
		Do: func(commentText string) error {
			//fmt.Printf("\n[Debug]] ... parse parseSubresourceRequest() with comment (%s)\n", commentText)
			group := GetGroup(t)
			version := GetVersion(t, group)
			kind := GetKind(t, group)
			if _, f := b.SubByGroupVersionKind[group]; !f {
				b.SubByGroupVersionKind[group] = map[string]map[string]*types.Type{}
			}
			if _, f := b.SubByGroupVersionKind[group][version]; !f {
				b.SubByGroupVersionKind[group][version] = map[string]*types.Type{}
			}
			b.SubByGroupVersionKind[group][version][kind] = t
			return nil
		},
	})
	return a
}

// TODO (fanz) : parseSubResource has two submodules: status, and scale
func (b *APIs) parseSubresource(t *types.Type, a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name: "subresource",
		Do:   nil,
		SubModules: map[string]*annotation.Module{
			"status": &annotation.Module{
				Name:       "status",
				SubModules: map[string]*annotation.Module{},
				Do:         nil,
			},
			"scale": &annotation.Module{
				Name:       "scale",
				SubModules: map[string]*annotation.Module{},
				Do:         nil,
			},
		},
	})

	return a
}

// TODO (fanz): parsePrintColumn
func (b *APIs) parsePrintColumn(t *types.Type, a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name: "printcolumn",
		Do:   nil,
	})

	return a
}

// TODO(fanz): nonNamespaced will be put into submodule of genclient
// Currently, having "nonNamespaced" as module of Header "genclient"
func (b *APIs) parseNamespace(a annotation.Annotation) annotation.Annotation {
	var found bool
	a.Module(&annotation.Module{
		Name: "nonNamespaced",
		Meta: &found,
		Do: func(commentText string) error {
			found = true
			return nil
		},
	})
	return a
}

// parseGroupNames initializes b.GroupNames with the set of all groups
func (b *APIs) parseGroupNames() {
	b.GroupNames = sets.String{}
	for p := range b.UnversionedPkgs {
		pkg := b.context.Universe[p]
		if pkg == nil {
			// If the input had no Go files, for example.
			continue
		}
		b.GroupNames.Insert(filepath.Base(p))
	}
}

// parseDomain parses the domain from the apis/doc.go file comment "// +domain=YOUR_DOMAIN".
func (b *APIs) parseDomain() {
	pkg := b.context.Universe[b.APIsPkg]
	if pkg == nil {
		// If the input had no Go files, for example.
		panic(errors.Errorf("Missing apis package."))
	}
	comments := Comments(pkg.Comments)
	b.Domain = comments.getTag("domain", "=")
	if len(b.Domain) == 0 {
		b.Domain = parseDomainFromFiles(b.context.Inputs)
		if len(b.Domain) == 0 {
			panic("Could not find string matching // +domain=.+ in apis/doc.go")
		}
	}
}

func parseDomainFromFiles(paths []string) string {
	var domain string
	for _, path := range paths {
		if strings.HasSuffix(path, "pkg/apis") {
			filePath := strings.Join([]string{build.Default.GOPATH, "src", path, "doc.go"}, "/")
			lines := []string{}

			file, err := os.Open(filePath)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				if strings.HasPrefix(scanner.Text(), "//") {
					lines = append(lines, strings.Replace(scanner.Text(), "// ", "", 1))
				}
			}
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}

			comments := Comments(lines)
			domain = comments.getTag("domain", "=")
			break
		}
	}
	return domain
}
