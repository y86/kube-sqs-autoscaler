package scale

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"
)

func TestScaleTo(t *testing.T) {
	ctx := context.Background()
	p := NewMockPodAutoScaler("deploy", "namespace", 500, 50, 3, 1, 1)

	// Scale up replicas until we reach the max (5).
	// Scale up again and assert that we get an error back when trying to scale up replicas pass the max
	err := p.ScaleTo(ctx, 120)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, int32(120), *deployment.Spec.Replicas)

	err = p.ScaleTo(ctx, 600)
	deployment, _ = p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, int32(500), *deployment.Spec.Replicas)

	err = p.ScaleTo(ctx, 60)
	deployment, _ = p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, int32(60), *deployment.Spec.Replicas)

	err = p.ScaleTo(ctx, 1)
	deployment, _ = p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, int32(50), *deployment.Spec.Replicas)
}

func NewMockPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max, min, init, upPods, downPods int) *PodAutoScaler {
	initialReplicas := int32(init)
	mock := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "deploy",
			Namespace:   "namespace",
			Annotations: map[string]string{},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &initialReplicas,
		},
	}, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "deploy-no-scale",
			Namespace:   "namespace",
			Annotations: map[string]string{},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &initialReplicas,
		},
	})
	return &PodAutoScaler{
		Client:        mock.AppsV1().Deployments(kubernetesNamespace),
		Min:           min,
		Max:           max,
		ScaleUpPods:   upPods,
		ScaleDownPods: downPods,
		Deployment:    kubernetesDeploymentName,
		Namespace:     kubernetesNamespace,
	}
}
