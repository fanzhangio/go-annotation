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
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/fanzhangio/go-annotation/pkg/annotation"
	"github.com/fanzhangio/go-annotation/pkg/codegen"
	"github.com/markbates/inflect"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	b.parseAPIResource()
	b.parseGroupNames()
	b.parseAPIs()
	if len(b.Domain) == 0 {
		b.parseDomain()
	}
	return b
}

func (b *APIs) parseAPIResource() {

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
		b.parseResources(resource, ann)
		b.parseNamespace(ann)
		b.parsePrintColumn(t, ann)

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
				log.Fatalf("api resource module not set meta correctly")
			}
			nonNamespaced, ok := ann.GetModule("nonNamespaced").Meta.(*bool)
			if !ok {
				log.Fatalf("nonNamespaced module not set meta correctly")
			}
			r := &codegen.APIResource{Type: t}
			r.NonNamespaced = *nonNamespaced
			r.Group = GetGroup(t)
			r.Version = GetVersion(t, r.Group)
			r.Kind = GetKind(t, r.Group)
			r.Domain = b.Domain

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

			// parse CRD for APIResource
			r.CRD = v1beta1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1beta1",
					Kind:       "CustomResourceDefinition",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   fmt.Sprintf("%s.%s.%s", r.Resource, r.Group, r.Domain),
					Labels: map[string]string{"controller-tools.k8s.io": "1.0"},
				},
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group:   fmt.Sprintf("%s.%s", r.Group, r.Domain),
					Version: resource.Version,
					Names: v1beta1.CustomResourceDefinitionNames{
						Kind:   r.Kind,
						Plural: r.Resource,
					},
					Validation: &v1beta1.CustomResourceValidation{
						OpenAPIV3Schema: &r.JSONSchemaProps,
					},
				},
			}

			// parse AdditionalPrintColumn
			result, ok := ann.GetModule("printcolumn").Meta.(*[]v1beta1.CustomResourceColumnDefinition)
			if !ok {
				log.Fatalf("printcolumn module not set meta correctly")
			}

			r.CRD.Spec.AdditionalPrinterColumns = *result

			if r.NonNamespaced {
				r.CRD.Spec.Scope = "Cluster"
			} else {
				r.CRD.Spec.Scope = "Namespaced"
			}

			if len(resource.ShortName) > 0 {
				r.CRD.Spec.Names.ShortNames = []string{r.ShortName}
			}

			// TODO(fanz): parse Categories

			// TODO(fanz): parse subresource:status and scale

			// parse JSONSchemaProps and Validation
			r.JSONSchemaProps, r.Validation = b.typeToJSONSchemaProps(t, sets.NewString(), []string{}, true)
			r.JSONSchemaProps.Type = ""
			j, err := json.MarshalIndent(r.JSONSchemaProps, "", "    ")
			if err != nil {
				log.Fatalf("Could not Marshall validation %v\n", err)
			}
			r.ValidationComments = string(j)
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

func (b *APIs) parseResources(r *codegen.APIResource, a annotation.Annotation) annotation.Annotation {
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

// printcolumn requires name,type,JSONPath fields and rest of the field are optional
// +kubebuilder:printcolumn:name=<name>,type=<type>,description=<desc>,JSONPath:<.spec.Name>,priority=<int32>,format=<format>
func (b *APIs) parsePrintColumn(t *types.Type, a annotation.Annotation) annotation.Annotation {
	result := []v1beta1.CustomResourceColumnDefinition{}
	a.Module(&annotation.Module{
		Name: "printcolumn",
		Meta: &result,
		Do: func(commentText string) error {
			config := v1beta1.CustomResourceColumnDefinition{}
			var count int
			part := strings.Split(commentText, ",")
			if len(part) < 3 {
				return fmt.Errorf(printColumnError)
			}
			for _, elem := range strings.Split(commentText, ",") {
				key, value, err := annotation.ParseKV(elem)
				if err != nil {
					return fmt.Errorf("//+kubebuilder:printcolumn: tags must be key value pairs.Expected "+
						"keys [name=<name>,type=<type>,description=<descr>,format=<format>] "+
						"Got string: [%s]", commentText)
				}
				if key == printColumnName || key == printColumnType || key == printColumnPath {
					count++
				}
				switch key {
				case printColumnName:
					config.Name = value
				case printColumnType:
					if value == "integer" || value == "number" || value == "string" || value == "boolean" || value == "date" {
						config.Type = value
					} else {
						return fmt.Errorf("invalid value for %s printcolumn", printColumnType)
					}
				case printColumnFormat:
					if config.Type == "integer" && (value == "int32" || value == "int64") {
						config.Format = value
					} else if config.Type == "number" && (value == "float" || value == "double") {
						config.Format = value
					} else if config.Type == "string" && (value == "byte" || value == "date" || value == "date-time" || value == "password") {
						config.Format = value
					} else {
						return fmt.Errorf("invalid value for %s printcolumn", printColumnFormat)
					}
				case printColumnPath:
					config.JSONPath = value
				case printColumnPri:
					i, err := strconv.Atoi(value)
					v := int32(i)
					if err != nil {
						return fmt.Errorf("invalid value for %s printcolumn", printColumnPri)
					}
					config.Priority = v
				case printColumnDescr:
					config.Description = value
				default:
					return fmt.Errorf(printColumnError)
				}
			}
			result = append(result, config)
			return nil
		},
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

// typeToJSONSchemaProps returns a JSONSchemaProps object and its serialization
// in Go that describe the JSONSchema validations for the given type.
func (b *APIs) typeToJSONSchemaProps(t *types.Type, found sets.String, comments []string, isRoot bool) (v1beta1.JSONSchemaProps, string) {
	// Special cases
	time := types.Name{Name: "Time", Package: "k8s.io/apimachinery/pkg/apis/meta/v1"}
	meta := types.Name{Name: "ObjectMeta", Package: "k8s.io/apimachinery/pkg/apis/meta/v1"}
	unstructured := types.Name{Name: "Unstructured", Package: "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"}
	intOrString := types.Name{Name: "IntOrString", Package: "k8s.io/apimachinery/pkg/util/intstr"}
	switch t.Name {
	case time:
		return v1beta1.JSONSchemaProps{
			Type:        "string",
			Format:      "date-time",
			Description: parseDescription(comments),
		}, b.getTime()
	case meta:
		return v1beta1.JSONSchemaProps{
			Type:        "object",
			Description: parseDescription(comments),
		}, b.objSchema()
	case unstructured:
		return v1beta1.JSONSchemaProps{
			Type:        "object",
			Description: parseDescription(comments),
		}, b.objSchema()
	case intOrString:
		return v1beta1.JSONSchemaProps{
			OneOf: []v1beta1.JSONSchemaProps{
				{
					Type: "string",
				},
				{
					Type: "integer",
				},
			},
			Description: parseDescription(comments),
		}, b.objSchema()
	}

	var v v1beta1.JSONSchemaProps
	var s string
	switch t.Kind {
	case types.Builtin:
		v, s = b.parsePrimitiveValidation(t, found, comments)
	case types.Struct:
		v, s = b.parseObjectValidation(t, found, comments, isRoot)
	case types.Map:
		v, s = b.parseMapValidation(t, found, comments)
	case types.Slice:
		v, s = b.parseArrayValidation(t, found, comments)
	case types.Array:
		v, s = b.parseArrayValidation(t, found, comments)
	case types.Pointer:
		v, s = b.typeToJSONSchemaProps(t.Elem, found, comments, false)
	case types.Alias:
		v, s = b.typeToJSONSchemaProps(t.Underlying, found, comments, false)
	default:
		log.Fatalf("Unknown supported Kind %v\n", t.Kind)
	}

	return v, s
}

var jsonRegex = regexp.MustCompile("json:\"([a-zA-Z,]+)\"")

type primitiveTemplateArgs struct {
	v1beta1.JSONSchemaProps
	Value       string
	Format      string
	EnumValue   string // TODO check type of enum value to match the type of field
	Description string
}

var primitiveTemplate = template.Must(template.New("map-template").Parse(
	`v1beta1.JSONSchemaProps{
    {{ if .Pattern -}}
    Pattern: "{{ .Pattern }}",
    {{ end -}}
    {{ if .Maximum -}}
    Maximum: getFloat({{ .Maximum }}),
    {{ end -}}
    {{ if .ExclusiveMaximum -}}
    ExclusiveMaximum: {{ .ExclusiveMaximum }},
    {{ end -}}
    {{ if .Minimum -}}
    Minimum: getFloat({{ .Minimum }}),
    {{ end -}}
    {{ if .ExclusiveMinimum -}}
    ExclusiveMinimum: {{ .ExclusiveMinimum }},
    {{ end -}}
    Type: "{{ .Value }}",
    {{ if .Format -}}
    Format: "{{ .Format }}",
    {{ end -}}
    {{ if .EnumValue -}}
    Enum: {{ .EnumValue }},
    {{ end -}}
    {{ if .MaxLength -}}
    MaxLength: getInt({{ .MaxLength }}),
    {{ end -}}
    {{ if .MinLength -}}
    MinLength: getInt({{ .MinLength }}),
    {{ end -}}
}`))

// parsePrimitiveValidation returns a JSONSchemaProps object and its
// serialization in Go that describe the validations for the given primitive
// type.
func (b *APIs) parsePrimitiveValidation(t *types.Type, found sets.String, comments []string) (v1beta1.JSONSchemaProps, string) {
	props := v1beta1.JSONSchemaProps{Type: string(t.Name.Name)}

	for _, l := range comments {
		getValidation(l, &props)
	}

	buff := &bytes.Buffer{}

	var n, f, s, d string
	switch t.Name.Name {
	case "int", "int64", "uint64":
		n = "integer"
		f = "int64"
	case "int32", "uint32":
		n = "integer"
		f = "int32"
	case "float", "float32":
		n = "number"
		f = "float"
	case "float64":
		n = "number"
		f = "double"
	case "bool":
		n = "boolean"
	case "string":
		n = "string"
	default:
		n = t.Name.Name
	}
	if props.Enum != nil {
		s = parseEnumToString(props.Enum)
	}
	d = parseDescription(comments)
	if err := primitiveTemplate.Execute(buff, primitiveTemplateArgs{props, n, f, s, d}); err != nil {
		log.Fatalf("%v", err)
	}
	props.Type = n
	props.Format = f
	props.Description = d
	return props, buff.String()
}

type mapTempateArgs struct {
	Result            string
	SkipMapValidation bool
}

var mapTemplate = template.Must(template.New("map-template").Parse(
	`v1beta1.JSONSchemaProps{
    Type:                 "object",
    {{if not .SkipMapValidation}}AdditionalProperties: &v1beta1.JSONSchemaPropsOrBool{
        Allows: true,
        Schema: &{{.Result}},
    },{{end}}
}`))

// parseMapValidation returns a JSONSchemaProps object and its serialization in
// Go that describe the validations for the given map type.
func (b *APIs) parseMapValidation(t *types.Type, found sets.String, comments []string) (v1beta1.JSONSchemaProps, string) {
	additionalProps, result := b.typeToJSONSchemaProps(t.Elem, found, comments, false)
	additionalProps.Description = ""
	props := v1beta1.JSONSchemaProps{
		Type:        "object",
		Description: parseDescription(comments),
	}
	parseOption := b.arguments.CustomArgs.(*Options)
	if !parseOption.SkipMapValidation {
		props.AdditionalProperties = &v1beta1.JSONSchemaPropsOrBool{
			Allows: true,
			Schema: &additionalProps}
	}
	buff := &bytes.Buffer{}
	if err := mapTemplate.Execute(buff, mapTempateArgs{Result: result, SkipMapValidation: parseOption.SkipMapValidation}); err != nil {
		log.Fatalf("%v", err)
	}
	return props, buff.String()
}

var arrayTemplate = template.Must(template.New("array-template").Parse(
	`v1beta1.JSONSchemaProps{
    Type:                 "{{.Type}}",
    {{ if .Format -}}
    Format: "{{.Format}}",
    {{ end -}}
    {{ if .MaxItems -}}
    MaxItems: getInt({{ .MaxItems }}),
    {{ end -}}
    {{ if .MinItems -}}
    MinItems: getInt({{ .MinItems }}),
    {{ end -}}
    {{ if .UniqueItems -}}
    UniqueItems: {{ .UniqueItems }},
    {{ end -}}
    {{ if .Items -}}
    Items: &v1beta1.JSONSchemaPropsOrArray{
        Schema: &{{.ItemsSchema}},
    },
    {{ end -}}
}`))

type arrayTemplateArgs struct {
	v1beta1.JSONSchemaProps
	ItemsSchema string
}

// parseArrayValidation returns a JSONSchemaProps object and its serialization in
// Go that describe the validations for the given array type.
func (b *APIs) parseArrayValidation(t *types.Type, found sets.String, comments []string) (v1beta1.JSONSchemaProps, string) {
	items, result := b.typeToJSONSchemaProps(t.Elem, found, comments, false)
	items.Description = ""
	props := v1beta1.JSONSchemaProps{
		Type:        "array",
		Items:       &v1beta1.JSONSchemaPropsOrArray{Schema: &items},
		Description: parseDescription(comments),
	}
	// To represent byte arrays in the generated code, the property of the OpenAPI definition
	// should have string as its type and byte as its format.
	if t.Name.Name == "[]byte" {
		props.Type = "string"
		props.Format = "byte"
		props.Items = nil
		props.Description = parseDescription(comments)
	}
	for _, l := range comments {
		getValidation(l, &props)
	}
	buff := &bytes.Buffer{}
	if err := arrayTemplate.Execute(buff, arrayTemplateArgs{props, result}); err != nil {
		log.Fatalf("%v", err)
	}
	return props, buff.String()
}

type objectTemplateArgs struct {
	v1beta1.JSONSchemaProps
	Fields   map[string]string
	Required []string
	IsRoot   bool
}

var objectTemplate = template.Must(template.New("object-template").Parse(
	`v1beta1.JSONSchemaProps{
	{{ if not .IsRoot -}}
    Type:                 "object",
	{{ end -}}
    Properties: map[string]v1beta1.JSONSchemaProps{
        {{ range $k, $v := .Fields -}}
        "{{ $k }}": {{ $v }},
        {{ end -}}
    },
    {{if .Required}}Required: []string{
        {{ range $k, $v := .Required -}}
        "{{ $v }}", 
        {{ end -}}
    },{{ end -}}
}`))

// parseObjectValidation returns a JSONSchemaProps object and its serialization in
// Go that describe the validations for the given object type.
func (b *APIs) parseObjectValidation(t *types.Type, found sets.String, comments []string, isRoot bool) (v1beta1.JSONSchemaProps, string) {
	buff := &bytes.Buffer{}
	props := v1beta1.JSONSchemaProps{
		Type:        "object",
		Description: parseDescription(comments),
	}

	if strings.HasPrefix(t.Name.String(), "k8s.io/api") {
		if err := objectTemplate.Execute(buff, objectTemplateArgs{props, nil, nil, false}); err != nil {
			log.Fatalf("%v", err)
		}
	} else {
		m, result, required := b.getMembers(t, found)
		props.Properties = m
		props.Required = required

		// Only add field validation for non-inlined fields
		for _, l := range comments {
			getValidation(l, &props)
		}

		if err := objectTemplate.Execute(buff, objectTemplateArgs{props, result, required, isRoot}); err != nil {
			log.Fatalf("%v", err)
		}
	}
	return props, buff.String()
}

// getValidation parses the validation tags from the comment and sets the
// validation rules on the given JSONSchemaProps.
func getValidation(comment string, props *v1beta1.JSONSchemaProps) {
	comment = strings.TrimLeft(comment, " ")
	if !strings.HasPrefix(comment, "+kubebuilder:validation:") {
		return
	}
	c := strings.Replace(comment, "+kubebuilder:validation:", "", -1)
	parts := strings.Split(c, "=")
	if len(parts) != 2 {
		log.Fatalf("Expected +kubebuilder:validation:<key>=<value> actual: %s", comment)
		return
	}
	switch parts[0] {
	case "Maximum":
		f, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Fatalf("Could not parse float from %s: %v", comment, err)
			return
		}
		props.Maximum = &f
	case "ExclusiveMaximum":
		b, err := strconv.ParseBool(parts[1])
		if err != nil {
			log.Fatalf("Could not parse bool from %s: %v", comment, err)
			return
		}
		props.ExclusiveMaximum = b
	case "Minimum":
		f, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Fatalf("Could not parse float from %s: %v", comment, err)
			return
		}
		props.Minimum = &f
	case "ExclusiveMinimum":
		b, err := strconv.ParseBool(parts[1])
		if err != nil {
			log.Fatalf("Could not parse bool from %s: %v", comment, err)
			return
		}
		props.ExclusiveMinimum = b
	case "MaxLength":
		i, err := strconv.Atoi(parts[1])
		v := int64(i)
		if err != nil {
			log.Fatalf("Could not parse int from %s: %v", comment, err)
			return
		}
		props.MaxLength = &v
	case "MinLength":
		i, err := strconv.Atoi(parts[1])
		v := int64(i)
		if err != nil {
			log.Fatalf("Could not parse int from %s: %v", comment, err)
			return
		}
		props.MinLength = &v
	case "Pattern":
		props.Pattern = parts[1]
	case "MaxItems":
		if props.Type == "array" {
			i, err := strconv.Atoi(parts[1])
			v := int64(i)
			if err != nil {
				log.Fatalf("Could not parse int from %s: %v", comment, err)
				return
			}
			props.MaxItems = &v
		}
	case "MinItems":
		if props.Type == "array" {
			i, err := strconv.Atoi(parts[1])
			v := int64(i)
			if err != nil {
				log.Fatalf("Could not parse int from %s: %v", comment, err)
				return
			}
			props.MinItems = &v
		}
	case "UniqueItems":
		if props.Type == "array" {
			b, err := strconv.ParseBool(parts[1])
			if err != nil {
				log.Fatalf("Could not parse bool from %s: %v", comment, err)
				return
			}
			props.UniqueItems = b
		}
	case "MultipleOf":
		f, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Fatalf("Could not parse float from %s: %v", comment, err)
			return
		}
		props.MultipleOf = &f
	case "Enum":
		if props.Type != "array" {
			value := strings.Split(parts[1], ",")
			enums := []v1beta1.JSON{}
			for _, s := range value {
				checkType(props, s, &enums)
			}
			props.Enum = enums
		}
	case "Format":
		props.Format = parts[1]
	default:
		log.Fatalf("Unsupport validation: %s", comment)
	}
}

// getMembers builds maps by field name of the JSONSchemaProps and their Go
// serializations.
func (b *APIs) getMembers(t *types.Type, found sets.String) (map[string]v1beta1.JSONSchemaProps, map[string]string, []string) {
	members := map[string]v1beta1.JSONSchemaProps{}
	result := map[string]string{}
	required := []string{}

	// Don't allow recursion until we support it through refs
	// TODO: Support recursion
	if found.Has(t.Name.String()) {
		fmt.Printf("Breaking recursion for type %s", t.Name.String())
		return members, result, required
	}
	found.Insert(t.Name.String())

	for _, member := range t.Members {
		tags := jsonRegex.FindStringSubmatch(member.Tags)
		if len(tags) == 0 {
			// Skip fields without json tags
			//fmt.Printf("Skipping member %s %s\n", member.Name, member.Type.Name.String())
			continue
		}
		ts := strings.Split(tags[1], ",")
		name := member.Name
		strat := ""
		if len(ts) > 0 && len(ts[0]) > 0 {
			name = ts[0]
		}
		if len(ts) > 1 {
			strat = ts[1]
		}

		// Inline "inline" structs
		if strat == "inline" {
			m, r, re := b.getMembers(member.Type, found)
			for n, v := range m {
				members[n] = v
			}
			for n, v := range r {
				result[n] = v
			}
			required = append(required, re...)
		} else {
			m, r := b.typeToJSONSchemaProps(member.Type, found, member.CommentLines, false)
			members[name] = m
			result[name] = r
			if !strings.HasSuffix(strat, "omitempty") {
				required = append(required, name)
			}
		}
	}

	defer found.Delete(t.Name.String())
	return members, result, required
}

func (b *APIs) getTime() string {
	return `v1beta1.JSONSchemaProps{
    Type:   "string",
    Format: "date-time",
}`
}

func (b *APIs) objSchema() string {
	return `v1beta1.JSONSchemaProps{
    Type:   "object",
}`
}
