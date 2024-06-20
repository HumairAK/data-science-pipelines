package k8sresource

import "context"

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Resources struct {
	KubeClient kubernetes.Interface
	Namespace  string
}

func (r Resources) GetSecret(ctx context.Context, name, key string) (string, error) {
	secret, err := r.KubeClient.CoreV1().Secrets(r.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(secret.Data[key]), nil
}

func (r Resources) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	configMap, err := r.KubeClient.CoreV1().ConfigMaps(r.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return configMap.Data[key], nil
}
