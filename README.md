# Go Annotation
Go Annotation introduces a [Modularized Annotation Pattern](./Modularized-Annotation-Pattern.md). This pattern is to modularize annotation and register fine-grained feature modules (or submodules) with corresponding handler functions for dynamic meta data injection and feature hooks in runtime of go code.

Basically, from a developer's perspective, the ideal user's expereince of tools should be like "All you have to do is focusing on writting your code, and just put instructions with parameters as annotations as what you want. Run tool cli, everything will be handled by the tool based on your annotations."

The codes in this repo demos how `Annotation-based Pattern` can be used for [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) project. By this approach, an easy-to sacle, develop and maintain annotation mechanism is provided to unify annotation schemas and usages in kubebuilder project. It is easily to extend features in kubebuilder.

## Go Annotation Syntax:
Go Annotation has a series of tokens separated by colons into groups from left to right. Each **Token** is a string identifier in an annotation instance. It has meaning by its position in token slice, in the form of
**+[header]:[module]:[submodule]:[key-value elements]** 
Go Annotation starts with `+` (e.g. `// +k8s`) to differentiate from regular go comments. 

## Token types:
- **header** is the identifier of a group of annotations. It helps user know which project provides this annotation. For example, in Kubernetes project, headers like `kubebuilder`, `k8s`, `genclient`, etc. are all project identifiers. A header is required for all annotations, since you may use multiple annotations from different projects in the same codebase.

- **module** is the identifier of functional module in an annotation. An annotation may have a group of modules, each of which performs a particular function.

- **submodule** (optional) In some cases, the module has a big functional scope, split into fine-grained sub modules, which provide the flexibility of extending module functionality. For example: **module:submodule1:submodule2:submodule3** submodule can be multiple following one by one.

## Levels of symboles:
Delimiter symbols are distinguished to work in different levels from top-down for splitting values string in tokens, which provides readability and efficiency.

- **Colon**

  Colon `:` is the 1st level delimiter (to annotation) only for separate tokens. Tokens on different sides of the colon should refer to different token types.

- **Comma**

  Comma `,` is the 2nd level delimiter (to annotation) for splitting key-value pairs in **key-value elements** which is normally the last token in annotation. e.g. `+kubebuilder:printcolumn:name=<name>,type=<type>,description=<desc>,JSONPath:<.spec.Name>,priority=<int32>,format=<format>` It works within token which is the 2nd level of annotation, so it is called "2nd level delimiter"

- **Equal sign**

  Equal sign `=` is the 3rd level delimiter (to annotation) for identifying key and value. Since the `key=value` parts are splitted from single token (2nd level), its inner delimiter `=` works for next level (3rd level)

- **Pipe sign or Vertical bar**

  Pipe sign `|` is the 4th level delimiter, which works inside the `key=value` part (3rd level) indicating key and value.


Examples of annotation signs:
`// +kubebuilder:webhook:serveroption:port=7890,cert-dir=/tmp/test-cert,service=test-system|webhook-service,selector=app|webhook-server,secret=test-system|webhook-secret,mutating-webhook-config-name=test-mutating-webhook-cfg,validating-webhook-config-name=test-validating-webhook-cfg`


## Packages Illustration
This repo takes `controller-tool` as example to illustrate how to develop and use `annotation-based pattern`  
For demo, two headers (`kubebuilder` and `genclient`) and a couple of modules are registered in default annotation.
`webhook` and `rbac` reside in `./pkg/webhook` and `./pkg/rbac` separately. `CRD` and `code-gen` parser and moduels are in `./pkg/codegen/parse`

### Webhook
[header] is `kubebuilder`,
[module] is `webhook`,
[submodule] is `admission` or `serveroption`

- Annotation examples:
```golang
// +kubebuilder:webhook:admission:groups=apps,resources=deployments,verbs=CREATE;UPDATE,name=bar-webhook,path=/bar,type=mutating,failure-policy=Fail

// +kubebuilder:webhook:serveroption:port=7890,cert-dir=/tmp/test-cert,service=test-system|webhook-service,selector=app|webhook-server,secret=test-system|webhook-secret,mutating-webhook-config-name=test-mutating-webhook-cfg,validating-webhook-config-name=test-validating-webhook-cfg
```
- Notes:
1. Separate two `submodule` (categories) under `webhook`: 1) `admission`and 2) `serveroption`, handling webhookTags and serverTags separately.
2. For each submodule, all key-values should put in the same comment line.
3. using `|` instead of `:` for lables

### RBAC
[header] is `kubebuilder`
[module] is `rbac`
No submodule at this moment, support annotations like : `// +rbac`, `// +kubebuilder:rbac`

- Annotation examples:
```golang
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;delete

// +rbac:groups=apps,resources=deployments,verbs=get;list;watch;delete
```

### Core code-gen parser and parse CRD
CRD
   - API Resource
      -  **header** is `kubebuilder`, **module** is `resource`
      - Example: `// +kubebuilder:resource:path=services,shortName=ty`
   - SubresourceRequest
     -  **header** is `kubebuilder`, **module** is `subresource-request`
      - example `// +subresource-request`, or `// +kubebuilder:subresource-request`
   - Namespace
      - **header** is `genclient`, **module** is `nonNamespaced`
      - example `// +genclient:nonNamespaced`.
   - AdditionalPrintColumn
      - **header** is `kubebuilder`, **module** is `printcolumn`
      - example: `// +kubebuilder:printcolumn:name="toy",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description="descr1",format="date",priority=3`
    - SubResource
       - **header** is `kubebuilder`, **module** is `sacle` or  `status`
       - example: `// +kubebuilder:subresource:status` and `// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=`
    - Categories
       - **header** is `kubebuilder`, **module** is `categories`
       - example: `// +kubebuilder:categories:foo,bar,hoo`


Implemented Modules:
- **parseAPIAnnotation**
- **parseCRD**. Implement generating CRD in just one parse by `parseAPIResource`. Optimize CRD generation and parsing api resources.
- **parseAPIResource**, support `// +kubebuilder:resource: ...`, `// +resource: ...`. 
- **parseSubreousrceRequest**, support `// +subresource-request`
- **parseNamespace**, support `// +genclient:nonNamespaced`. Currently, it is implemented as module("nonNamespaced") of header("genclient").
- **parsePrintColumn**, support `// +printcolumn`, and `// +kubebuilder:printcolumn`
- **parseSubresource**, support CRD Subreosurce **sacle** and **status**
- **parseCategories**, support like `// +kubebuilder:categories:foo,bar,hoo`
