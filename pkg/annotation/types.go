package annotation

// annotation types defination

type Header string

type Module struct {
	Name      string
	KeyValue  map[string]string
	ParseFunc func(string) error
}

// Schema represents annotation schema: [Header]:[Module]:[Token = Value]
// type Schema map[Header]map[Module]map[Token]Value

type Schema map[Header]Module

type Annotation interface {
	GetSchema() *Schema // map[Header]map[Module]map[Token]Value
	GetComments() Comments
	GetModule(string) Module

	Do() error
	ParseFuncMap(ParseFuncMap)
}

type webhookAnnotation struct {
	schema    *Schema
	coments   Comments // all comments for webhook
	parseFunc ParseFuncMap
}

func (a *webhookAnnotation) ParseFuncMap(fn ParseFuncMap) {
	for k, v := range fn {
		a.parseFunc[k] = v
	}
}

func (a *webhookAnnotation) Do() error {
	for _, f := range a.parseFunc {
		for _, c := range a.GetComments() {
			f(c)
		}
	}
	return nil
}

func (a *webhookAnnotation) GetSchema() *Schema {
	return a.schema
}

func (a *webhookAnnotation) GetComments() Comments {
	return a.coments
}

type ParseFuncMap map[string]func(string) error

// Comments is a structure for using comment tags on go structs and fields
type Comments []string

// // GetTags returns the value for the first comment with a prefix matching "+name="
// // e.g. "+name=foo\n+name=bar" would return "foo"
// func (c Comments) getTag(name, sep string) string {
// 	for _, c := range c {
// 		prefix := fmt.Sprintf("+%s%s", name, sep)
// 		if strings.HasPrefix(c, prefix) {
// 			return strings.Replace(c, prefix, "", 1)
// 		}
// 	}
// 	return ""
// }

// // hasTag returns true if the Comments has a tag with the given name
// func (c Comments) hasTag(name string) bool {
// 	for _, c := range c {
// 		prefix := fmt.Sprintf("+%s", name)
// 		if strings.HasPrefix(c, prefix) {
// 			return true
// 		}
// 	}
// 	return false
// }

// // GetTags returns the value for all comments with a prefix and separator.  E.g. for "name" and "="
// // "+name=foo\n+name=bar" would return []string{"foo", "bar"}
// func (c Comments) getTags(name, sep string) []string {
// 	tags := []string{}
// 	for _, c := range c {
// 		prefix := fmt.Sprintf("+%s%s", name, sep)
// 		if strings.HasPrefix(c, prefix) {
// 			tags = append(tags, strings.Replace(c, prefix, "", 1))
// 		}
// 	}
// 	return tags
// }

// // getCategoriesTag returns the value of the +kubebuilder:categories tags
// func getCategoriesTag(c *types.Type) string {
// 	comments := Comments(c.CommentLines)
// 	resource := comments.getTag("kubebuilder:categories", "=")
// 	if len(resource) == 0 {
// 		panic(errors.Errorf("Must specify +kubebuilder:categories comment for type %v", c.Name))
// 	}
// 	return resource
// }
