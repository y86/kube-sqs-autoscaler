package scale

import (
	"context"
	"os"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeConfigPath string
)

type PodAutoScaler struct {
	Client        typedappv1.DeploymentInterface
	Max           int
	Min           int
	ScaleUpPods   int
	ScaleDownPods int
	Deployment    string
	Namespace     string
}

func NewPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max, min, scaleUpPods, scaleDownPods int) *PodAutoScaler {
	kubeConfigPath = os.Getenv("KUBE_CONFIG_PATH")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic("Failed to configure incluster or local config")
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic("Failed to configure client")
	}

	return &PodAutoScaler{
		Client:        k8sClient.AppsV1().Deployments(kubernetesNamespace),
		Min:           min,
		Max:           max,
		ScaleUpPods:   scaleUpPods,
		ScaleDownPods: scaleDownPods,
		Deployment:    kubernetesDeploymentName,
		Namespace:     kubernetesNamespace,
	}
}

func (pod *PodAutoScaler) CurrentReplicas(ctx context.Context) (*int32, error) {
	deployment, err := pod.Client.Get(ctx, pod.Deployment, metav1.GetOptions{})
	if err != nil {
		var currentReplicas *int32
		return currentReplicas, errors.Wrap(err, "Failed to get deployment from kube server, no scale up occured")
	}

	return deployment.Spec.Replicas, nil
}

func (p *PodAutoScaler) ScaleTo(ctx context.Context, targetReplicaCount int32) error {
	deployment, err := p.Client.Get(ctx, p.Deployment, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to get deployment from kube server, no scale to occured")
	}

	currentReplicas := deployment.Spec.Replicas

	if targetReplicaCount >= *currentReplicas && *currentReplicas > int32(p.Max) {
		log.Infof("More than max pods running. No scale up. Replicas: %d", *deployment.Spec.Replicas)
		return nil
	}

	if targetReplicaCount <= *currentReplicas && *currentReplicas < int32(p.Min) {
		log.Infof("Less than min pods running. No scale down. Replicas: %d", *deployment.Spec.Replicas)
		return nil
	}

	if targetReplicaCount > int32(p.Max) {
		targetReplicaCount = int32(p.Max)
	}

	if targetReplicaCount < int32(p.Min) {
		targetReplicaCount = int32(p.Min)
	}

	deployment.Spec.Replicas = &targetReplicaCount

	_, err = p.Client.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to scale to desired target")
	}

	log.Infof("Scaled to target successfully. Replicas: %d", *deployment.Spec.Replicas)
	return nil
}
