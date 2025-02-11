package main

import (
	"context"
	"kube-sqs-autoscaler/scale"
	kubesqs "kube-sqs-autoscaler/sqs"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"
)

func TestRunReachMinReplicas(t *testing.T) {
	ctx := context.Background()
	// override default vars for testing
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	maxPods = 50
	minPods = 4
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 5
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods, 1, 1)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{
		"ApproximateNumberOfMessages":           aws.String("1"),
		"ApproximateNumberOfMessagesDelayed":    aws.String("1"),
		"ApproximateNumberOfMessagesNotVisible": aws.String("1"),
	}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	_, _ = s.Client.SetQueueAttributes(input)

	time.Sleep(10 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(minPods), *deployment.Spec.Replicas, "Number of replicas should be the min")
}

func TestRunReachMaxReplicas(t *testing.T) {
	ctx := context.Background()
	// override default vars for testing
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods, 1, 1)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{
		"ApproximateNumberOfMessages":           aws.String("100"),
		"ApproximateNumberOfMessagesDelayed":    aws.String("100"),
		"ApproximateNumberOfMessagesNotVisible": aws.String("100"),
	}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	_, _ = s.Client.SetQueueAttributes(input)

	time.Sleep(10 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(maxPods), *deployment.Spec.Replicas, "Number of replicas should be the max")
}

func TestRunShouldReachTarget(t *testing.T) {
	ctx := context.Background()
	// override default vars for testing
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	maxPods = 500
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods, 1, 1)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{
		"ApproximateNumberOfMessages":           aws.String("100"),
		"ApproximateNumberOfMessagesDelayed":    aws.String("100"),
		"ApproximateNumberOfMessagesNotVisible": aws.String("100"),
	}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	_, _ = s.Client.SetQueueAttributes(input)

	time.Sleep(10 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(300), *deployment.Spec.Replicas, "Number of replicas should be the max")
}

func TestRunScaleUpCoolDown(t *testing.T) {
	ctx := context.Background()
	pollInterval = 5 * time.Second
	scaleDownCoolPeriod = 10 * time.Second
	scaleUpCoolPeriod = 10 * time.Second
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods, 1, 1)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{
		"ApproximateNumberOfMessages":           aws.String("100"),
		"ApproximateNumberOfMessagesDelayed":    aws.String("100"),
		"ApproximateNumberOfMessagesNotVisible": aws.String("100"),
	}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	_, _ = s.Client.SetQueueAttributes(input)

	time.Sleep(15 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(maxPods), *deployment.Spec.Replicas, "Number of replicas should be 5 (maxPods) if cool down for scaling up was obeyed")
}

func TestRunScaleDownCoolDown(t *testing.T) {
	ctx := context.Background()
	pollInterval = 5 * time.Second
	scaleDownCoolPeriod = 10 * time.Second
	scaleUpCoolPeriod = 10 * time.Second
	maxPods = 50
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods, 1, 1)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{
		"ApproximateNumberOfMessages":           aws.String("1"),
		"ApproximateNumberOfMessagesDelayed":    aws.String("1"),
		"ApproximateNumberOfMessagesNotVisible": aws.String("1"),
	}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	_, _ = s.Client.SetQueueAttributes(input)

	time.Sleep(15 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(3), *deployment.Spec.Replicas, "Number of replicas should be 3 if cool down for scaling down was obeyed")
}

func TestRunReachMinReplicasWithScaleingPodNum(t *testing.T) {
	ctx := context.Background()
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	maxPods = 100
	minPods = 10
	scaleUpPods = 100
	scaleDownPods = 100
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 100
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods, scaleUpPods, scaleDownPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{
		"ApproximateNumberOfMessages":           aws.String("1"),
		"ApproximateNumberOfMessagesDelayed":    aws.String("1"),
		"ApproximateNumberOfMessagesNotVisible": aws.String("1"),
	}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	_, _ = s.Client.SetQueueAttributes(input)

	time.Sleep(3 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(minPods), *deployment.Spec.Replicas, "Number of replicas should be the min")
}

func TestRunReachMaxReplicasWithScaleingPodNum(t *testing.T) {
	ctx := context.Background()
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	maxPods = 100
	minPods = 1
	scaleUpPods = 100
	scaleDownPods = 100
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "deploy"
	kubernetesNamespace = "namespace"
	initPods := 3
	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods, initPods, scaleUpPods, scaleDownPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{
		"ApproximateNumberOfMessages":           aws.String("100"),
		"ApproximateNumberOfMessagesDelayed":    aws.String("100"),
		"ApproximateNumberOfMessagesNotVisible": aws.String("100"),
	}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	_, _ = s.Client.SetQueueAttributes(input)

	time.Sleep(3 * time.Second)
	deployment, _ := p.Client.Get(ctx, "deploy", metav1.GetOptions{})
	assert.Equal(t, int32(maxPods), *deployment.Spec.Replicas, "Number of replicas should be the max")
}

func NewMockPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max, min, init, upPods, downPods int) *scale.PodAutoScaler {
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
	return &scale.PodAutoScaler{
		Client:        mock.AppsV1().Deployments(kubernetesNamespace),
		Min:           min,
		Max:           max,
		ScaleDownPods: downPods,
		ScaleUpPods:   upPods,
		Deployment:    kubernetesDeploymentName,
		Namespace:     kubernetesNamespace,
	}
}

type MockSQS struct {
	QueueAttributes *sqs.GetQueueAttributesOutput
}

func (m *MockSQS) GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	return m.QueueAttributes, nil
}

func (m *MockSQS) SetQueueAttributes(input *sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error) {
	m.QueueAttributes = &sqs.GetQueueAttributesOutput{
		Attributes: input.Attributes,
	}
	return &sqs.SetQueueAttributesOutput{}, nil
}

func NewMockSqsClient() *kubesqs.SqsClient {
	Attributes := map[string]*string{
		"ApproximateNumberOfMessages":           aws.String("100"),
		"ApproximateNumberOfMessagesDelayed":    aws.String("100"),
		"ApproximateNumberOfMessagesNotVisible": aws.String("100"),
	}

	return &kubesqs.SqsClient{
		Client: &MockSQS{
			QueueAttributes: &sqs.GetQueueAttributesOutput{
				Attributes: Attributes,
			},
		},
		QueueUrl: "example.com",
	}
}
