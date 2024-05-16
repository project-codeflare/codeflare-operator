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

// webhook configuration
//+kubebuilder:webhook:path=/mutate-workload-codeflare-dev-v1beta2-appwrapper,mutating=true,failurePolicy=fail,sideEffects=None,groups=workload.codeflare.dev,resources=appwrappers,verbs=create,versions=v1beta2,name=mappwrapper.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-workload-codeflare-dev-v1beta2-appwrapper,mutating=false,failurePolicy=fail,sideEffects=None,groups=workload.codeflare.dev,resources=appwrappers,verbs=create;update,versions=v1beta2,name=vappwrapper.kb.io,admissionReviewVersions=v1

// permissions needed by the "real" Webhook in the appwrapper project to enable SubjectAccessReview
//+kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=list
