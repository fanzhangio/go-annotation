package parse

import (
	"errors"

	"github.com/fanzhangio/go-annotation/pkg/annotation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/types"
)

// Add
func (b *APIs) AddParseResource(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "resource",
		Manifest: b,
		Tags:     nil,
		Func:     b.parseResource,
	})
	return a
}

func (b *APIs) parseResource(commentText string, t interface{}) error {
	// 1. Validate resource,
	// 	 if resource Do:
	//
	c, ok := t.(types.Type)
	if !ok {
		return fmt.Errorf("parse resource should be take types.Type as input, but get %T", t)
	}
	// parsePackage()
	versioned := c.Name.Package
	b.VersionedPkgs.Insert(versioned)
	unversioned := filepath.Dir(versioned)
	b.UnversionedPkgs.Insert(unversioned)

	// parseIndex()
	r := &codegen.APIResource{
		Type:          c,
		NonNamespaced: IsNonNamespaced(c), // TODO:(fanz) +genclient:nonNamespaced
	}
	r.Group = GetGroup(c)
	r.Version = GetVersion(c, r.Group)
	r.Kind = GetKind(c, r.Group)
	r.Domain = b.Domain

	// TODO: revisit the part...
	if r.Resource == "" {
		r.Resource = strings.ToLower(inflect.Pluralize(r.Kind))
	}
	rt, err := annotation.ParseKV(c)
	if err != nil {
		return fmt.Errorf("failed to parse resource annotations, error: %v", err.Error())
	}
	if rt.Resource != "" {
		r.Resource = rt.Resource
	}
	r.ShortName = rt.ShortName

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
	r.Type = c

	return nil
}

func (b *APIs) AddParseAPISubResource(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "subresource-request",
		Manifest: b,
		Tags:     nil,
		Func:     b.parseAPISubresource,
	})
	return a
}


func (b *APIs) parseAPISubresource(commentText string, t interface{}) error {
	group := GetGroup(c)
	version := GetVersion(c, group)
	kind := GetKind(c, group)
	if _, f := b.SubByGroupVersionKind[group]; !f {
		b.SubByGroupVersionKind[group] = map[string]map[string]*types.Type{}
	}
	if _, f := b.SubByGroupVersionKind[group][version]; !f {
		b.SubByGroupVersionKind[group][version] = map[string]*types.Type{}
	}
	b.SubByGroupVersionKind[group][version][kind] = c
	return nil
}

/***************************************************************/

// IsNonNamespaced module
func AddGenclient(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "genclient",
		Manifest: nil,
		Tags:     nil,
		Func:     nil,
	})
	return a
}

// IsController module
func AddController(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "controller",
		Manifest: nil,
		Tags:     nil,
		Func:     nil,
	})
	return a
}

//
func Addprintcolumn(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "printcolumn",
		Manifest: nil,
		Tags:     nil,
		Func:     nil,
	})
	return a
}

// hasSubresource - Status, Scale
func Addsubresource(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "subresource",
		Manifest: nil,
		Tags:     sets.NewString([]string{"status", "scale"}),
		Func:     nil,
	})
	return a
}

func (b *APIs) ParseSubresource(commentText string) error {
	// 1. Subresource
}

// hasCategories
func hasCategories(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "categories",
		Manifest: nil,
		Tags:     nil,
		Func:     nil,
	})
	return a
}

// HasDocAnnotation    e.g.  +kubebuilder:doc
func HasDocAnnotation(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "doc",
		Manifest: nil,
		Tags:     nil,
		Func:     nil,
	})
	return a
}

func IsNonNamespaced(a annotation.Annotation) annotation.Annotation {
	a.Module(&annotation.Module{
		Name:     "doc",
		Manifest: nil,
		Tags:     nil,
		Func:     func(b.),
	})
	return a
}