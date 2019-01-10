# Go Annotation
Go Annotation introduces a [Modularized Annotation Pattern](./Modularized-Annotation-Modularized-Annotation-Pattern.md). This pattern is to modularize annotation and register fine-grained feature modules (or submodules) with corresponding handler functions for dynamic meta data injection and feature hooks in runtime of go code.

The codes in this repo demos how `Annotation-based Pattern` can be used for [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) project. By this approach, an easy-to sacle, develop and maintain annotation mechanism is provided to unify annotation schemas and usages in kubebuilder project. It is easily to extend features in kubebuilder.

## Annotation Syntax
### Annotation format
Annotation has a series of tokens seperate by colon. **Token** is a string value in annotation. It has meaning by its position in token slice, in the form of **+[header]:[module]:[submodule]:[key-value elements]**. Annotation starts with `+` (e.g. `// +k8s`) to differ regular go comments.

- **header** is like `kubebuilder`, `k8s`, `genclient`, etc. Header is recommended for all annotaitons, but considering forward-compatibility, header could be omitted, in this case module must be the first token, like `+resource:path=services,shortName=mem`

- **module** is like rbac, webhook, doc, etc. 
- **submodule** is optional, for example: subresource or something need to extend prior module. submodule could be append if necessary, for example: **module:submodule1:submodule2:submodule3** for fine-grained annotation module control

- **key-value elements** is bunch of meta data key-values pairs, separate by `,`. Inner value delimiter within each pair is `;`. Inner delimiter of label within value of key is marked by `|`, like `selector=app|webhook-server`. For some cases, key-value element is omitted.

- The same module or submodule annotation should be lay in the same comment line.

### Allowed annotation symbols
- **Colon**
  - Colon `:` is the 1st level delimiter (to annotation) only for separate tokens. Tokens on the different sides of colon should refer to different
- **Comma**
  - Comma `,` is the 2nd level delimiter (to annotation) for slpitting key-value pairs in **key-value elements** which is normally the last token in annotation. e.g. `+kubebuilder:printcolumn:name=<name>,type=<type>,description=<desc>,JSONPath:<.spec.Name>,priority=<int32>,format=<format>` It works within token which is the 2nd levle of annotation, so it is called "2nd level delimiter"
- **Equals sign**
  - Equals sign `=` is the 3rd level delimiter (to annotation) for identify key and value. Since the `key=value` parts are splitted from single token (2nd level), its inner delimiter `=` works for next level (3rd level)
- **Pipe sign or Vertical bar**
  - Pip sign `|` is the 4th level delimiter, which works inside `key=value` part (3rd level) indicating key and value.

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
