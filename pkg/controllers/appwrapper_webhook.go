/*
Copyright 2024.

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

package controllers

import (
	"context"
	"fmt"

	awv1beta2 "github.com/project-codeflare/appwrapper/api/v1beta2"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// webhook configuration
//+kubebuilder:webhook:path=/mutate-workload-codeflare-dev-v1beta2-appwrapper,mutating=true,failurePolicy=fail,sideEffects=None,groups=workload.codeflare.dev,resources=appwrappers,verbs=create,versions=v1beta2,name=mappwrapper.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-workload-codeflare-dev-v1beta2-appwrapper,mutating=false,failurePolicy=fail,sideEffects=None,groups=workload.codeflare.dev,resources=appwrappers,verbs=create;update,versions=v1beta2,name=vappwrapper.kb.io,admissionReviewVersions=v1

// permissions needed by the "real" Webhook in the appwrapper project to enable SubjectAccessReview
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=list

type appwrapperWebhook struct {
	externalController bool
}

var _ webhook.CustomDefaulter = &appwrapperWebhook{}

func (w *appwrapperWebhook) Default(ctx context.Context, obj runtime.Object) error {
	return nil
}

var _ webhook.CustomValidator = &appwrapperWebhook{}

func (w *appwrapperWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	if w.externalController {
		return nil, nil
	} else {
		return nil, fmt.Errorf("AppWrappers disabled by CodeFlare operator configuration")
	}
}

func (w *appwrapperWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (w *appwrapperWebhook) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func SetupMockAppWrapperWebhooks(mgr ctrl.Manager, externalController bool) error {
	wh := &appwrapperWebhook{externalController: externalController}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&awv1beta2.AppWrapper{}).
		WithDefaulter(wh).
		WithValidator(wh).
		Complete()
}
