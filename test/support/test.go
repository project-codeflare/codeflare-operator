/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package support

import (
	"context"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
)

type Test interface {
	T() *testing.T
	Ctx() context.Context
	Client() Client
	OutputDir() string

	gomega.Gomega

	NewTestNamespace(...Option[*corev1.Namespace]) *corev1.Namespace
}

type Option[T any] interface {
	applyTo(to T) error
}

type errorOption[T any] func(to T) error

// nolint: unused
// To be removed when the false-positivity is fixed.
func (o errorOption[T]) applyTo(to T) error {
	return o(to)
}

var _ Option[any] = errorOption[any](nil)

func With(t *testing.T) Test {
	t.Helper()
	ctx := context.Background()
	if deadline, ok := t.Deadline(); ok {
		withDeadline, cancel := context.WithDeadline(ctx, deadline)
		t.Cleanup(cancel)
		ctx = withDeadline
	}

	return &T{
		WithT: gomega.NewWithT(t),
		t:     t,
		ctx:   ctx,
	}
}

type T struct {
	*gomega.WithT
	t *testing.T
	// nolint: containedctx
	ctx       context.Context
	client    Client
	outputDir string
	once      struct {
		client    sync.Once
		outputDir sync.Once
	}
}

func (t *T) T() *testing.T {
	return t.t
}

func (t *T) Ctx() context.Context {
	return t.ctx
}

func (t *T) Client() Client {
	t.T().Helper()
	t.once.client.Do(func() {
		c, err := newTestClient()
		if err != nil {
			t.T().Fatalf("Error creating client: %v", err)
		}
		t.client = c
	})
	return t.client
}

func (t *T) OutputDir() string {
	t.T().Helper()
	t.once.outputDir.Do(func() {
		if parent, ok := os.LookupEnv(CodeFlareTestOutputDir); ok {
			if !path.IsAbs(parent) {
				if cwd, err := os.Getwd(); err == nil {
					// best effort to output the parent absolute path
					parent = path.Join(cwd, parent)
				}
			}
			t.T().Logf("Creating output directory in parent directory: %s", parent)
			dir, err := os.MkdirTemp(parent, t.T().Name())
			if err != nil {
				t.T().Fatalf("Error creating output directory: %v", err)
			}
			t.outputDir = dir
		} else {
			t.T().Logf("Creating ephemeral output directory as %s env variable is unset", CodeFlareTestOutputDir)
			t.outputDir = t.T().TempDir()
		}
		t.T().Logf("Output directory has been created at: %s", t.outputDir)
	})
	return t.outputDir
}

func (t *T) NewTestNamespace(options ...Option[*corev1.Namespace]) *corev1.Namespace {
	t.T().Helper()
	namespace := createTestNamespace(t, options...)
	t.T().Cleanup(func() {
		deleteTestNamespace(t, namespace)
	})
	return namespace
}
