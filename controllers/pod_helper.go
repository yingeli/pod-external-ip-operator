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

package controllers

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	externalIPAnnotation      = "podexternalip.yglab.eu.org/externalip"
	associatedPodIPAnnotation = "podexternalip.yglab.eu.org/associatedpodip"

	finalizerPrefix   = "azurecni.podexternalip.yglab.eu.org/finalizer"
	dissociaterPrefix = "azurecni.podexternalip.yglab.eu.org/dissociater"
)

func parseExternalIP(pod *corev1.Pod) string {
	return pod.Annotations[externalIPAnnotation]
}

func parseAssociatedPodIP(pod *corev1.Pod) string {
	return pod.Annotations[associatedPodIPAnnotation]
}

func setAssociatedPodIP(pod *corev1.Pod, podIP string) {
	pod.Annotations[associatedPodIPAnnotation] = podIP
}

func removeAssociatedPodIP(pod *corev1.Pod) {
	delete(pod.Annotations, associatedPodIPAnnotation)
}

func parseFinalizer(pod *corev1.Pod) string {
	for _, f := range pod.GetFinalizers() {
		if strings.HasPrefix(f, finalizerPrefix) {
			return f[len(finalizerPrefix)+1:]
		}
	}
	return ""
}

func parseDissociater(pod *corev1.Pod) string {
	for _, f := range pod.GetFinalizers() {
		if strings.HasPrefix(f, dissociaterPrefix) {
			return f[len(dissociaterPrefix)+1:]
		}
	}
	return ""
}

func addFinalizer(pod *corev1.Pod, localIP string) {
	//removeFinalizer(pod)
	controllerutil.AddFinalizer(pod, finalizerPrefix+"-"+localIP)
}

func addDissociater(pod *corev1.Pod, localIP string) {
	//removeDissociater(pod)
	controllerutil.AddFinalizer(pod, dissociaterPrefix+"-"+localIP)
}

func removeFinalizer(pod *corev1.Pod) {
	for _, f := range pod.GetFinalizers() {
		if strings.HasPrefix(f, finalizerPrefix) {
			controllerutil.RemoveFinalizer(pod, f)
		}
	}
}

func removeDissociater(pod *corev1.Pod) {
	for _, f := range pod.GetFinalizers() {
		if strings.HasPrefix(f, dissociaterPrefix) {
			controllerutil.RemoveFinalizer(pod, f)
		}
	}
}

/*
func containsPrefix(slice []string, s string) bool {
	for _, item := range slice {
		if strings.HasPrefix(item, s) {
			return true
		}
	}
	return false
}
*/

func namespacedName(pod *corev1.Pod) string {
	return pod.Namespace + "/" + pod.Name
}
