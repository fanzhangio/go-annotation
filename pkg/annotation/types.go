package annotation

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// meata schema is [Module]:[Tokens]
//  e.g.  // +kubebuilder:webhook:groups=apps,resources=deployments,verbs=CREATE;UPDATE
type metaSchema map[string]map[string][]string

type DefaultAnnotation struct {

	// Header is string set, containing all annotation prefixes: e.g.  +kuberbuilder, +rbac
	Headers   sets.String
	Modules   sets.String
	ModuleMap map[string]Module
	Meta      metaSchema
}

func (a *DefaultAnnotation) Header(header string) {
	a.Headers.Insert(prefixName(header))
}

// module name will be added to Headers, e.g. "+rbac", "+resource"
func (a *DefaultAnnotation) Module(m Module) {
	//a.Headers.Insert(prefixName(m.Name))
	a.Modules.Insert(prefixName(m.Name))
	a.ModuleMap[m.Name] = m
}

func (a *DefaultAnnotation) HasModule(name string) bool {
	return a.Modules.Has(name)
}

func (a *DefaultAnnotation) GetModule(name string) Module {
	for _, m := range a.ModuleMap {
		if m.Name == name {
			return m
		}
	}
	return Module{}
}
func (a *DefaultAnnotation) GetModuleManifest(name string) interface{} {
	// TODO
	return nil
}

func (a *DefaultAnnotation) Parse(comments string) error {
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
			tokens := convert(list, a.Headers, a.Modules)
			// tokens has information for annotaions

		}
	}

	return nil
}

type Annotation interface {
	Header(string)
	Module(Module)
	HasModule(string) bool
	//HasAnnotation(string) bool // (TODO: for some true/false values)
	GetModule(string) Module
	GetModuleManifest(string) interface{}
	Parse(string) error
	// (TODO:) Need a plugable parsing func
}

type Tag string
type Value string

// Module is
type Module struct {
	Name     string
	Manifest interface{}
	Tags     []Tag
	// function maps by key, how to use Token
	// e.g.   verbs=get;list;delete
	FuncMap map[Tag]func(Value) error
}

func (m *Module) HasTag(t string) bool {
	for _, v := range m.Tags {
		if string(v) == t {
			return true
		}
	}
	return false
}

func NewAnnotation() Annotation {
	return &DefaultAnnotation{}
}

// register annotaiton schema into Annotaion struct

// Encoding (TODO:) move to separate file
/**********************************************************/

// Utils (TODO:) move to separate file
/**********************************************************/
func prefixName(name string) string {
	return "+" + name
}

// GetAnnotation extracts the annotation from comment text.
// It will return "foo" for comment "+kubebuilder:webhook:foo" .
func getAnnotation(c, name string) string {
	prefix := fmt.Sprintf("+%s:", name)
	if strings.HasPrefix(c, prefix) {
		return strings.TrimPrefix(c, prefix)
	}
	return ""
}

// Node is for storing parsed annotation tokens
type Node struct {
	depth int
	end   bool
	kind  string
	next  *Node
	val   string
	elem  map[string]*Node
}

type Tokens struct {
	size int
	root *Node
}

func (t *Tokens) Root() *Node {
	return t.root
}

func (t *Tokens) Add(node *Node) {
	if t.size == 0 {
		t.root = node
	} else {
		curr := t.root
		for curr.next != nil {
			curr = curr.next
		}
		curr.next = node
	}
	t.size++
}

func convert(tokens []string, h, m sets.String) *Tokens {
	t := &Tokens{size: 0}
	for k, v := range tokens {
		node := Node{val: v, depth: k}
		if h.Has(v) {
			node.kind = "Header"
		}
		if m.Has(v) {
			node.kind = "Module"
		}
		if k == len(tokens)-1 {
			node.end = true
		}
		t.Add(&node)
	}
	return t
}
