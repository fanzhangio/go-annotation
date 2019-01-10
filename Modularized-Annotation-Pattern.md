# Modularized Annotation Pattern

This document is about `Modularized Annotation Pattern`. This pattern is to modularize annotation and register fine-grained feature modules (or sub-modules) with corresponding handler functions for dynamic meta data injection and feature hooks in runtime of go code. 

Annotation consists of a series of components: `Header`, `Module`, `Key-Value Elements`(optional). Each component is represented by token in annotation string, separated by highest level delimiter. `Header` is the prefix in annotation representing a high level group of modules. For example, [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) project denotes its supported project grade annotations by `kubebuilder`. [kubernetes](https://github.com/kubernetes/kubernetes) has its annotation header like `k8s`. Header may contain multiple modules. `Module` defines the actual functional feature for annotation. Module has nested architecture, for example, single module may contain sub-modules. It is represented by token-chains in annotation. Module invokes `Do` function when valid module name is found in parsing annotations. `Do` is the handler function which defines what this module can do. It takes annotation token passed by Module, and might involve context from runtime. If module has sub-modules nested, the final sub-module in the chain will be the actual one performing the behavior for the whole annotation by calling the final sub-module's correlative Do function. `Key-Value Elements` token is optional token in annotation. If it presents, there is only one element token in single annotation instance. It works as parameter for the last module (normally the closed token next to it on the left side). The whole token may consists of a couple of key-value elements. Each element may have nested key-value style format in its value part. Thus, distinguished delimiters are used for level-based token spliter. For example, annotation has highest level delimiter for splitting header, modules or submodules, and element. The second level delimiter is responsible for splitting key-value element array in element token. The third level delimiter identifies key part and value part in single key-value element. In the value part, if nested key-value pairs exist, it requires next-level (distinguished) delimiter for identification. The delimiter should be valid ASCII symbol and not conflict with regular expression symbol.

## Recommend Syntax Schema
`+[Header:]Module[:SubModule][:key-value elements]`
Considering backward compatibility, header may omit in some cases. It is not recommended though.

## Specification

- Annotation interface:

```golang

type Annotation interface {

	// Header register header string without "+" of annotation, e.g. "kubebuilder", "k8s"
	Header(string)

	// Module register functional annotation module, it could be second token after header or first token in annotation
	// e.g. rbac module refers annotation like "+kubebuilder:rbac", or "+rbac"
	Module(*Module)

	// HasModule returns true if given module name is registered
	HasModule(string) bool

	// GetModule returns module by given name
	GetModule(string) *Module

	// Parse takes single comment group and parse registered annotation
	Parse(string) error
}
```

- Annotation Module

```golang
type Module struct {

	// Name of the module. It should match the token string in the annotation
	Name string
	// Meta holds meta data this module will return or impact. It may involve context
	Meta interface{}
	// SubModules represents a recursive architecture of annotation syntax, e.g. [header]:[module]:[submodule1]:[submodule2]:...
	SubModules map[string]*Module
	// Do is handler function which defines what this module can do. It takes annotation token passed by Module, and might involve context from runtime
	Do func(string) error
}
```

- Register Module to Annotation
```golang

	var GlobalAnnotation Annotation
	
	// Initialized global annotation and register headers
	func init() {
		GlobalAnnotation = Build()
		GlobalAnnotation.Header("kubebuilder")
		GlobalAnnotation.Header("k8s")
		// ...
	}

	// Register Module to Annotation
	GlobalAnnotation.Module(
		// functional variable definition within Module
		&Module{
			// name should match the string in module token, it is also the key to valid and retrieve Module
			Name: "module",
		    // Meta holds the reference returned by this Module Do function
			Meta: context,

			SubModules: map[string]*Module{
				"submodule": &Module{
					Name: "submodule",
					Meta: context,
					// SubModules may have nested sub-modules
					Do: func(string) error {
						// Implement what the sub-module do. Take key-value elements tokens string as parameter
					},
				},
			},

			Do: func(string)error{
				// Implement what the module do. If sub-module perform the actual behavior, it could be nil.
			}
		}
	)

	// Call general parse function of annotation, parsing comment groups
	GlobalAnnotation.Parse("comment groups")

```
