package annotation

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// meata schema is [Module]:[Tokens]
//  e.g.
//  Pattern 1: +kubebuilder:webhook:groups=apps,resources=deployments,verbs=CREATE;UPDATE
//  Pattern 2: +kubebuilder:subresource:scale:specpath=.spec.replica,statuspath=.status.replica,selectorpath=.spec.Label
type metaSchema map[string]map[string][]string

type defaultAnnotation struct {

	// Header is string set, containing all annotation prefixes: e.g.  +kuberbuilder, +rbac
	Headers   sets.String
	Modules   sets.String
	ModuleMap map[string]*Module
	Meta      metaSchema
}

type Annotation interface {
	Header(string)
	Module(*Module)
	HasModule(string) bool
	//HasAnnotation(string) bool // (TODO: for some true/false values)
	GetModule(string) *Module
	Parse(string) error
	// (TODO:) Need a plugable parsing func
}

func (a *defaultAnnotation) Header(header string) {
	a.Headers.Insert(prefixName(header))
}

// module name will be added to Headers, e.g. "+rbac", "+resource"
func (a *defaultAnnotation) Module(m *Module) {
	//a.Headers.Insert(prefixName(m.Name))
	a.Modules.Insert(prefixName(m.Name))
	a.ModuleMap[m.Name] = m
}

func (a *defaultAnnotation) HasModule(name string) bool {
	return a.Modules.Has(name)
}

func (a *defaultAnnotation) GetModule(name string) *Module {
	for _, m := range a.ModuleMap {
		if m.Name == name {
			return m
		}
	}
	return nil
}

func (a *defaultAnnotation) Parse(comments string) (err error) {
	for _, comment := range strings.Split(comments, "\n") {
		comment = strings.TrimSpace(comment)
		// Validate annotations then continue
		// 1. Has valid header
		// 2. Populate meta data
		for k, _ := range a.Headers.Union(a.Modules) {
			if !strings.HasPrefix(comments, k) {
				continue
			}

			// (TODO:) valid annotation pattern
			list := strings.Split(strings.TrimPrefix(comments, "+"), ":")
			if err = a.Complete(list); err != nil {
				return
			}
		}
	}
	return nil
}

type Module struct {
	Name     string
	Manifest interface{}
	Tags     sets.String
	// function maps by key, how to use Token
	// e.g.   verbs=get;list;delete
	Func func(string) error
}

// Complete process annotaion string into Tokens
func (a *defaultAnnotation) Complete(tokens []string) (err error) {
	var module string
	for k, v := range tokens {
		if a.Headers.Has(v) {
			// (TODO:) exceptional validation
			// Ignore header, parsing module and its tokens
			continue
		}
		if a.Modules.Has(v) {
			// Find module, parsing module and calling Func from Map
			module = v
			continue
		}
		// Find Token following Module
		// Pattern 1:   [header]:[module]:[element-values]
		if module != "" && v != "" && k == len(tokens)-1 {
			err = a.GetModule(module).Func(v)
			if err != nil {
				return
			}
		} // (TODO): consider corner case - subresource:scale:<key=value>
	}
	return
}

func (m *Module) HasTag(t string) bool {
	return m.Tags.Has(t)
}

func Build() Annotation {
	return &defaultAnnotation{}
}
