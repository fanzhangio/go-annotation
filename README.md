# Go Annotation
Go Annotation is an implementation for [Generic Annotation Spec](https://github.com/kubernetes-sigs/kubebuilder/issues/554).
It introduces an `Annotation-based Pattern`.

The codes in this repo demos how `Annotation-based Pattern` can be used for [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) project. By this approach, an easy-to sacle, develop and maintain annotaion mechanism is provided to unify annotation schemas and usages in kubebuilder project. It is easily to extend features in kubebuilder.

The annotation spec is like 
## [header]:[module]:[submodule]:[key-value elements]

[header] is like `// +kubebuilder`, `// +k8s`, `// +genclient`, etc. For forward-compatibility, [header] could be emitted, in this case, [module] must be the first token

[module] is like rbac, webhook, doc, etc. [submodule] is optional, for example: subresource or something need to extend prior module. [submodule] could be append if necessary, for example: [module]:[submodule1]:[submodule2]:[submodule3] for fine-grained annotation control

[key-value elements] is bunch of meta data key-values pairs, separate by ,. Inner value delimiter within each pair is ;. Inner delimiter of label within value of key is marked by |, like selector=app|webhook-server

The same module or submodule annotation should be lay in the same comment line.

## Packages Illustration
This repo takes `controller-tool` as example to illustrate how to develop and use `annotaion-based pattern`  
For demo, two headers (`kubebuilder` and `genclient`) and a couple of modules are registered in default annotation.
`webhook` and `rbac` reside in `./pkg/webhook` and `./pkg/rbac` separately. The core parser and codegen are `WIP`

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
No submodule at this moment, support annotaions like : `// +rbac`, `// +kubebuilder:rbac`

- Annotation examples:
```golang
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;delete

// +rbac:groups=apps,resources=deployments,verbs=get;list;watch;delete
```

### Core code-gen parser and parse CRD

Implemented Modules:
- parseAPIAnnotation
- parseAPIResource
  - support `// +kubebuilder:resource: ...`, `// +resource: ...`. 
  - Example: `// +kubebuilder:resource:path=services,shortName=ty`
- parseSubreousrceRequest
  - support `// +subresource-request`
- parseNamespace
  - support `// +genclient:nonNamespaced`. Currently, it is implemented as module("nonNamespaced") of header("genclient").
- parsePrintColumn
  - support `// +printcolumn`, and `// +kubebuilder:printcolumn`
  - example: `// +kubebuilder:printcolumn:name="toy",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description="descr1",format="date",priority=3`
- parseCRD
  - Implement generating CRD in just one parse by `parseAPIResource`. Optimize CRD generation and parsing api resources.
- parseSubresource
  - support CRD Subreosurce **sacle** and **status**
  - example: `// +kubebuilder:subresource:status` and `// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=`
- parseCategories
  - example: `// +kubebuilder:categories:foo,bar,hoo`
