package parse

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fanzhangio/go-annotation/pkg/annotation"
	"github.com/fanzhangio/go-annotation/pkg/codegen"
	"github.com/markbates/inflect"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/types"
)

func (b *APIs) parseAPI() {
	// parse API
	// 1. Parse package
	// 2. Parse group name
	// 3. Parse Index
	// 4. Parse APIs

	// 5. Parse CRD -> for..type -> IsAPIResource

	b.VersionedPkgs = sets.NewString()
	b.UnversionedPkgs = sets.NewString()
	b.GroupNames = sets.String{}

	b.ByGroupVersionKind = map[string]map[string]map[string]*codegen.APIResource{}
	b.ByGroupKindVersion = map[string]map[string]map[string]*codegen.APIResource{}
	b.SubByGroupVersionKind = map[string]map[string]map[string]*types.Type{}

	ann := annotation.GetAnnotation()
	for _, o := range b.context.Order {
		// register api annoations
		b.parseResourceAnnotation(o, ann)
		b.parseSubresourceAnnotation(o, ann)

		if IsAPIResource(o) {
			// parsePackages()
			versioned := o.Name.Package
			b.VersionedPkgs.Insert(versioned)
			unversioned := filepath.Dir(versioned)
			b.UnversionedPkgs.Insert(unversioned)

			// find API Resource
			parseAPIAnnotation(o, ann)
		}
	}

	// TODO:
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

func (b *APIs) parseResourceAnnotation(t *types.Type, a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name: "resource",
		Do: func(commentText string) error {
			// indexes all types with the comment "// +resource=RESOURCE" by GroupVersionKind and GroupKindVersion

			// Parse resource to APIResource
			r := &codegen.APIResource{
				Type:          t,
				NonNamespaced: IsNonNamespaced(t),
			}
			r.Group = GetGroup(t)
			r.Version = GetVersion(t, r.Group)
			r.Kind = GetKind(t, r.Group)
			r.Domain = b.Domain

			// TODO: revisit the part...
			if r.Resource == "" {
				r.Resource = strings.ToLower(inflect.Pluralize(r.Kind))
			}

			// parse resource tags
			// res := resourceTags{}
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
			return nil
		},
	})
	return a
}

func (b *APIs) parseSubresourceAnnotation(t *types.Type, a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name: "subresource-request",
		Do: func(string) error {
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
