package providers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type Associater interface {
	Initialize(ctx context.Context, localNetworks []string) error
	Associate(ctx context.Context, pod *corev1.Pod, localIP string, externalIP string) (bool, error)
	Dissociate(ctx context.Context, pod *corev1.Pod, localIP string, externalIP string) error
}

type Finalizer interface {
	Initialize(ctx context.Context) error
	Finalize(ctx context.Context, pod *corev1.Pod, localIP string, externalIP string) error
}
