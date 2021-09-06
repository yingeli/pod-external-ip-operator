/*
Copyright 2021.

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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var podexternaliplog = logf.Log.WithName("podexternalip-resource")

func (r *PodExternalIP) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-podexternalip-yglab-eu-org-v1alpha1-podexternalip,mutating=true,failurePolicy=fail,sideEffects=None,groups=podexternalip.yglab.eu.org,resources=podexternalips,verbs=create;update,versions=v1alpha1,name=mpodexternalip.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &PodExternalIP{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PodExternalIP) Default() {
	podexternaliplog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-podexternalip-yglab-eu-org-v1alpha1-podexternalip,mutating=false,failurePolicy=fail,sideEffects=None,groups=podexternalip.yglab.eu.org,resources=podexternalips,verbs=create;update,versions=v1alpha1,name=vpodexternalip.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &PodExternalIP{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PodExternalIP) ValidateCreate() error {
	podexternaliplog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PodExternalIP) ValidateUpdate(old runtime.Object) error {
	podexternaliplog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PodExternalIP) ValidateDelete() error {
	podexternaliplog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
