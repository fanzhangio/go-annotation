package main

import (
	"fmt"
	"log"
	"path/filepath"

	rbac "github.com/fanzhangio/go-annotation/pkg/rbac"
	// webhook "github.com/fanzhangio/go-annotation/pkg/webhook"
)

func main() {
	fmt.Println("....Starging go annotation example")
	RBACCmd()
	// WebhookCmd()
}

// func WebhookCmd() error {
// 	o := &webhook.ManifestOptions{}
// 	if err := webhook.Generate(o); err != nil {
// 		log.Fatal(err)
// 	}
// 	return nil
// }

func RBACCmd() error {
	o := &rbac.ManifestOptions{}
	o.Name = "manager"
	o.InputDir = filepath.Join(".", "example", "project")
	o.OutputDir = filepath.Join(".", "example", "project", "config", "rbac")
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
