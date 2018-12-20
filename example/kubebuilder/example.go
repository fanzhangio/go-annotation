package kubebuilder

// import (
// 	anotation "github.com/fanzhangio/go-annotation/pkg/annotation"
// 	rbac "github.com/fanzhangio/go-annotation/pkg/annotation/modules/rbac"
// 	webhook "github.com/fanzhangio/go-annotation/pkg/annotation/modules/webhook"
// )

// func WebhookCmd() error {

// 	o := &webhook.ManifestOptions{}
// 	// if err := webhook.Generate(o); err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	err := anotation.ParseDir(o.InputDir, o.parseAnnotation)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func RBACCmd() error {
// 	o := &rbac.ManifestOptions{}
// 	// if err := rbac.Generate(o); err != nil {
// 	// 	log.Fatal(err)
// 	// }

// 	err := anotation.ParseDir(o.InputDir, o.parseAnnotation)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func CRDCmd() error {
// 	return nil
// }

// func inti() {
// 	// register schema
// 	annotation.AddToAnnotations = append(annotation.AddToAnnotations, o.AddToAnnotation)
// 	AddToAnnotations = append(annotation.AddToAnnotations, webhook.Annotaion)
// }
