package client

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

type FakeSecretClient struct{}

func (FakeSecretClient) Create(ctx context.Context, secret *v1.Secret, opts metav1.CreateOptions) (*v1.Secret, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Update(ctx context.Context, secret *v1.Secret, opts metav1.UpdateOptions) (*v1.Secret, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	glog.Error("This fake method is not yet implemented")
	return nil
}
func (FakeSecretClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return nil
}
func (FakeSecretClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Secret, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) List(ctx context.Context, opts metav1.ListOptions) (*v1.SecretList, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Secret, err error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Apply(ctx context.Context, secret *corev1.SecretApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Secret, err error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}

type FakeBadSecretClient struct {
	FakeSecretClient
}

func (FakeBadSecretClient) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	return errors.New("failed to delete pod")
}
