package kubebuilder

import (
	"log"

	rbac "github.com/fanzhangio/go-annotation/pkg/rbac"
	webhook "github.com/fanzhangio/go-annotation/pkg/webhook"
)

func WebhookCmd() error {

	o := &webhook.ManifestOptions{}
	if err := webhook.Generate(o); err != nil {
		log.Fatal(err)
	}
	return nil
}

func RBACCmd() error {
	o := &rbac.ManifestOptions{}
	if err := rbac.Generate(o); err != nil {
		log.Fatal(err)
	}

	return nil
}

// func CRDCmd() error {
// 	return nil
// }

// func inti() {
// 	// register schema
// 	annotation.AddToAnnotations = append(annotation.AddToAnnotations, o.AddToAnnotation)
// 	AddToAnnotations = append(annotation.AddToAnnotations, webhook.Annotaion)
// }
