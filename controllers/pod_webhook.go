/*
Copyright 2018 The Kubernetes Authors.

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
	"encoding/json"

	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	//eipv1alpha1 "github.com/yingeli/pod-external-ip-operator/api/v1alpha1"
)

//+kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io,sideEffects=none,admissionReviewVersions=v1
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// podAnnotator annotates Pods
type PodWebhook struct {
	Client  client.Client
	decoder *admission.Decoder
}

// PodAnnotator adds an annotation to every incoming pods.
func (a *PodWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if getExternalIP(pod) != "" {
		found := false
		for _, ic := range pod.Spec.InitContainers {
			if ic.Name == "init-external-ip" {
				found = true
				break
			}
		}
		if !found {
			inject(pod)
		}
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func inject(pod *corev1.Pod) {
	arg := "while ! grep -q \"podexternalip.yglab.eu.org/ready\" /etc/podinfo/annotations; do cat /etc/podinfo/annotations; sleep 5; done;"
	init := corev1.Container{
		Name:  "init-external-ip",
		Image: "k8s.gcr.io/busybox",
		Command: []string{
			"sh",
			"-c",
		},
		Args: []string{arg},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "podinfo",
				MountPath: "/etc/podinfo",
			},
		},
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, init)

	volume := corev1.Volume{
		Name: "podinfo",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path: "annotations",
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.annotations",
						},
					},
				},
			},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
}

// PodWebhook implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (a *PodWebhook) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
