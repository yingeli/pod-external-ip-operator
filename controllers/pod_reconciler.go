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
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"

	"github.com/yingeli/pod-external-ip-operator/providers"
)

const (
	externalipAnnotation      = "podexternalip.yglab.eu.org/externalip"
	externalipReadyAnnotation = "podexternalip.yglab.eu.org/ready"
	externalipFinalizer       = "podexternalip.yglab.eu.org/finalizer"
	externalipDissociater     = "podexternalip.yglab.eu.org/dissociater"
)

type PodAssociater struct {
	client   *client.Client
	assoMap  map[string]string
	provider providers.Associater
	log      logr.Logger
}

func newPodAssociater(client *client.Client, provider providers.Associater) PodAssociater {
	return PodAssociater{
		client:   client,
		assoMap:  make(map[string]string),
		provider: provider,
		log:      ctrl.Log.WithName("pod-associater"),
	}
}

func (r *PodAssociater) setup(localNetworks []string) error {
	return r.provider.Initialize(context.Background(), localNetworks)
}

func (r *PodAssociater) reconcile(ctx context.Context, pod *corev1.Pod) (ctrl.Result, error) {
	if pod.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.associate(ctx, pod); err != nil {
			result := ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 5,
			}
			return result, err
		}
		return ctrl.Result{}, nil
	} else {
		return ctrl.Result{}, r.dissociate(ctx, pod)
	}
}

func (r *PodAssociater) associate(ctx context.Context, pod *corev1.Pod) error {
	externalIP := getExternalIP(pod)
	if externalIP == "" {
		return nil
	}

	if _, ok := r.assoMap[pod.Namespace+"/"+pod.Name]; ok && isExternalIPReady(pod) {
		return nil
	}

	addFinalizers(pod)
	if err := (*r.client).Update(ctx, pod); err != nil {
		return err
	}

	ok, err := r.provider.Associate(ctx, pod, externalIP)
	if err != nil {
		return err
	}
	if ok {
		setExternalIPReady(pod)
		if err := (*r.client).Update(ctx, pod); err != nil {
			return err
		}
		r.assoMap[pod.Namespace+"/"+pod.Name] = externalIP
		r.log.Info("associate pod with external IP", "pod.Name", pod.Name, "pod.Status.PodIP", pod.Status.PodIP, "externalIP", externalIP)
	}

	return nil
}

func (r *PodAssociater) dissociate(ctx context.Context, pod *corev1.Pod) error {
	if !hasDissociater(pod) {
		return nil
	}

	if err := r.provider.Dissociate(ctx, pod, pod.Annotations[externalipAnnotation]); err != nil {
		return err
	}

	if err := r.removeDissociater(ctx, pod); err != nil {
		return err
	}

	r.log.Info("dissociated pod with external IP", "pod.Name", pod.Name)
	return nil
}

func (r *PodAssociater) removeDissociater(ctx context.Context, pod *corev1.Pod) error {
	original := pod.DeepCopy()
	controllerutil.RemoveFinalizer(pod, externalipDissociater)
	return (*r.client).Patch(ctx, pod, client.StrategicMergeFrom(original))
}

type PodFinalizer struct {
	client   *client.Client
	provider providers.Finalizer
	log      logr.Logger
}

func newPodFinalizer(client *client.Client, provider providers.Finalizer) PodFinalizer {
	return PodFinalizer{
		client:   client,
		provider: provider,
		log:      ctrl.Log.WithName("pod-associater"),
	}
}

func (r *PodFinalizer) finalize(ctx context.Context, pod *corev1.Pod) error {
	if pod.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	if !hasFinalizer(pod) {
		return nil
	}

	if err := r.provider.Finalize(ctx, pod, pod.Annotations[externalipAnnotation]); err != nil {
		return err
	}

	if err := r.removeFinalizer(ctx, pod); err != nil {
		return err
	}

	r.log.Info("finalized pod with external IP", "pod.Name", pod.Name)
	return nil
}

func (r *PodFinalizer) removeFinalizer(ctx context.Context, pod *corev1.Pod) error {
	original := pod.DeepCopy()
	controllerutil.RemoveFinalizer(pod, externalipFinalizer)
	return (*r.client).Patch(ctx, pod, client.StrategicMergeFrom(original))
}

func isExternalIPReady(pod *corev1.Pod) bool {
	return pod.Annotations[externalipReadyAnnotation] == "true"
}

func setExternalIPReady(pod *corev1.Pod) {
	pod.Annotations[externalipReadyAnnotation] = "true"
}

func getExternalIP(pod *corev1.Pod) string {
	return pod.Annotations[externalipAnnotation]
}

func addFinalizers(pod *corev1.Pod) {
	if !containsString(pod.GetFinalizers(), externalipDissociater) {
		controllerutil.AddFinalizer(pod, externalipDissociater)
	}
	if !containsString(pod.GetFinalizers(), externalipFinalizer) {
		controllerutil.AddFinalizer(pod, externalipFinalizer)
	}
}

func hasFinalizer(pod *corev1.Pod) bool {
	return containsString(pod.GetFinalizers(), externalipFinalizer)
}

func hasDissociater(pod *corev1.Pod) bool {
	return containsString(pod.GetFinalizers(), externalipDissociater)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

/*
func setConditionReady(pod *corev1.Pod) {
	for i, cond := range pod.Status.Conditions {
		if cond.Type == externalipReady {
			pod.Status.Conditions[i].Status = corev1.ConditionTrue
			pod.Status.Conditions[i].LastTransitionTime = metav1.Now()
			return
		}
	}
	cond := corev1.PodCondition{
		Type:               externalipReady,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	}
	pod.Status.Conditions = append(pod.Status.Conditions, cond)
}

func getCondition(pod *corev1.Pod) *corev1.PodCondition {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == externalipReady {
			return &cond
		}
	}
	return nil
}
*/

/*
func addReadinessGate(pod *corev1.Pod) {
	for _, gate := range pod.Spec.ReadinessGates {
		if gate.ConditionType == podConditionType {
			return
		}
	}
	readinessGate := corev1.PodReadinessGate{
		ConditionType: podConditionType,
	}
	pod.Spec.ReadinessGates = append(pod.Spec.ReadinessGates, readinessGate)
}
*/

/*
func (r *PodAssociater) lookupExternalIP(ctx context.Context, pod *corev1.Pod) (eip *eipv1alpha1.PodExternalIP, err error) {
	var eips eipv1alpha1.PodExternalIPList
	if err := (*r.client).List(ctx, &eips); err != nil {
		return eip, err
	}

	for _, ip := range eips.Items {
		selector := labels.SelectorFromSet(ip.Spec.PodSelector.MatchLabels)
		lables := labels.Set(pod.Labels)
		if selector.Matches(lables) {
			return &ip, nil
		}
	}
	return nil, nil
}
*/

/*
func readFinalizer(pod *corev1.Pod) (finalizer string, hostName string, internalIP string, ok bool) {
	for _, fin := range pod.GetFinalizers() {
		if strings.HasPrefix(fin, podFinalizerPrefix) {
			parts := strings.Split(fin, "/")
			if len(parts) == 2 {
				params := strings.Split(parts[1], ".")
				if len(params) == 3 {
					return fin, params[1], strings.ReplaceAll(params[2], "-", "."), true
				}
			}
		}
	}
	return "", "", "", false
}
*/

/*
func (r *GatewayReconciler) insertSNAT(ctx context.Context, podIP string, toSrcIP) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	rule := NewSNATRule(podIP, r.localNetwork, toSrcIP)
	if err := ipt.Insert("nat", "POSTROUTING", 1, rule.Spec()...); err != nil {
		return err
	}

	r.snatMap[podIP] = &rule

	return nil
}

func (r *GatewayReconciler) deleteSNAT(ctx context.Context, srcIP string) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	if rule, ok := r.snatMap[srcIP]; ok {
		if err := ipt.Delete("nat", "POSTROUTING", rule.Spec()...); err != nil {
			return err
		}
		delete(r.snatMap, srcIP)
	}
	return nil
}

func (r *GatewayReconciler) deleteToSourceSNATs(ctx context.Context, toSrcIP string) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	for _, rule := range r.snatMap {
		if rule.ToSource == toSrcIP {
			if err := ipt.Delete("nat", "POSTROUTING", rule.Spec()...); err != nil {
				return err
			}
			delete(r.snatMap, rule.Source)
		}
	}
	return nil
}
*/
