package config

import (
	"github.com/go-logr/logr"
	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Manifest(cl client.Client, templatePath string, context interface{}, name string, logger logr.Logger) (mf.Manifest, error) {
	m, err := mf.ManifestFrom(PathTemplateSource(templatePath, context, name), mf.UseLogger(logger))
	if err != nil {
		return mf.Manifest{}, err
	}
	m.Client = mfc.NewClient(cl)
	return m, err
}
