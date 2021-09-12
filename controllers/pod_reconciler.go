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

	"github.com/go-logr/logr"

	"github.com/yingeli/pod-external-ip-operator/providers"
)

type PodAssociater struct {
	client     *client.Client
	associater providers.Associater
	finalizer  providers.Finalizer
	log        logr.Logger
	assoMap    map[string]string
}

func newPodAssociater(client *client.Client, associater providers.Associater, finalizer providers.Finalizer) PodAssociater {
	return PodAssociater{
		client:     client,
		associater: associater,
		finalizer:  finalizer,
		log:        ctrl.Log.WithName("pod-associater"),
		assoMap:    make(map[string]string),
	}
}

func (r *PodAssociater) setup(localNetworks []string) error {
	ctx := context.Background()
	if err := r.associater.Initialize(ctx, localNetworks); err != nil {
		return err
	}
	return r.finalizer.Initialize(ctx)
}

func (r *PodAssociater) reconcile(ctx context.Context, pod *corev1.Pod) (ctrl.Result, error) {
	externalIP := parseExternalIP(pod)
	if externalIP == "" {
		return ctrl.Result{}, nil
	}

	podIP := pod.Status.PodIP
	if pod.ObjectMeta.DeletionTimestamp.IsZero() && podIP != "" {
		if err := r.associateOrUpdate(ctx, pod, externalIP); err != nil {
			result := ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 10,
			}
			return result, err
		}
		return ctrl.Result{}, nil
	} else {
		return ctrl.Result{}, r.dissociate(ctx, pod, externalIP)
	}
}

func (r *PodAssociater) associateOrUpdate(ctx context.Context, pod *corev1.Pod, externalIP string) error {
	podIP := pod.Status.PodIP
	if podIP == r.assoMap[namespacedName(pod)] {
		return nil
	}

	delete(r.assoMap, namespacedName(pod))

	associatedPodIP := parseAssociatedPodIP(pod)
	if podIP != associatedPodIP {
		removeAssociatedPodIP(pod)
		if err := (*r.client).Update(ctx, pod); err != nil {
			return err
		}
	}

	original := pod.DeepCopy()
	if podIP != parseDissociater(pod) {
		if err := dissociate(ctx, r.associater, pod, externalIP); err != nil {
			return err
		}
		r.log.Info("dissociated pod with external IP", "pod.Name", pod.Name, "externalIP", externalIP)
		addDissociater(pod, podIP)
	}

	if podIP != parseFinalizer(pod) {
		if err := finalize(ctx, r.finalizer, pod, externalIP); err != nil {
			return err
		}
		r.log.Info("finalized pod with external IP", "pod.Name", pod.Name, "externalIP", externalIP)
		addFinalizer(pod, podIP)
	}
	if err := (*r.client).Patch(ctx, pod, client.StrategicMergeFrom(original)); err != nil {
		return err
	}

	if err := r.associater.Associate(ctx, pod, podIP, externalIP); err != nil {
		return err
	}

	original = pod.DeepCopy()
	setAssociatedPodIP(pod, podIP)
	if err := (*r.client).Patch(ctx, pod, client.StrategicMergeFrom(original)); err != nil {
		return err
	}

	r.log.Info("associated pod with external IP", "pod.Name", pod.Name, "externalIP", externalIP)
	r.assoMap[namespacedName(pod)] = podIP
	return nil
}

func (r *PodAssociater) dissociate(ctx context.Context, pod *corev1.Pod, externalIP string) error {
	delete(r.assoMap, namespacedName(pod))
	original := pod.DeepCopy()
	if err := dissociate(ctx, r.associater, pod, externalIP); err != nil {
		return err
	}
	if err := (*r.client).Patch(ctx, pod, client.StrategicMergeFrom(original)); err != nil {
		return err
	}
	r.log.Info("dissociated pod with external IP", "pod.Name", pod.Name, "externalIP", externalIP)
	return nil
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

func (r *PodFinalizer) reconcile(ctx context.Context, pod *corev1.Pod) error {
	if pod.ObjectMeta.DeletionTimestamp.IsZero() && pod.Status.PodIP != "" {
		return nil
	}

	externalIP := parseExternalIP(pod)
	if externalIP == "" {
		return nil
	}

	original := pod.DeepCopy()
	if err := finalize(ctx, r.provider, pod, externalIP); err != nil {
		return err
	}
	if err := (*r.client).Patch(ctx, pod, client.StrategicMergeFrom(original)); err != nil {
		return err
	}
	r.log.Info("finalized pod with external IP", "pod.Name", pod.Name, "externalIP", externalIP)
	return nil
}

func dissociate(ctx context.Context, provider providers.Associater, pod *corev1.Pod, externalIP string) error {
	localIP := parseDissociater(pod)
	if localIP != "" {
		if err := provider.Dissociate(ctx, pod, localIP, externalIP); err != nil {
			return err
		}
	}
	removeDissociater(pod)
	return nil
}

func finalize(ctx context.Context, provider providers.Finalizer, pod *corev1.Pod, externalIP string) error {
	localIP := parseFinalizer(pod)
	if localIP != "" {
		if err := provider.Finalize(ctx, pod, localIP, externalIP); err != nil {
			return err
		}
	}
	removeFinalizer(pod)
	return nil
}
