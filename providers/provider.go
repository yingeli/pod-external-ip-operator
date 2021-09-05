package providers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type Associater interface {
	Initialize(ctx context.Context, localNetworks []string) error
	Associate(ctx context.Context, pod *corev1.Pod, externalIP string) (bool, error)
	Dissociate(ctx context.Context, pod *corev1.Pod) error
}

type Finalizer interface {
	Initialize(ctx context.Context, localNetworks []string) error
	Finalize(ctx context.Context, pod *corev1.Pod) error
}
