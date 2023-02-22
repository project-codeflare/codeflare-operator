package config

import (
	"bytes"
	"io"
	"os"
	"text/template"

	mf "github.com/manifestival/manifestival"
)

// PathPrefix is the file system path which template paths will be prefixed with.
// Default is no prefix, which causes paths to be read relative to process working dir
var PathPrefix string

// PathTemplateSource A templating source read from a file
func PathTemplateSource(path string, context interface{}, name string) mf.Source {
	f, err := os.Open(prefixedPath(path))
	if err != nil {
		panic(err)
	}
	return templateSource(f, context, name)
}

func prefixedPath(p string) string {
	if PathPrefix != "" {
		return PathPrefix + "/" + p
	}
	return p
}

// A templating manifest source
func templateSource(r io.Reader, context interface{}, name string) mf.Source {
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	t, err := template.New(name).Parse(string(b))
	if err != nil {
		panic(err)
	}
	var b2 bytes.Buffer
	err = t.Execute(&b2, context)
	if err != nil {
		panic(err)
	}
	return mf.Reader(&b2)
}
