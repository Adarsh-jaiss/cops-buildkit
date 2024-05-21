package buildkit

import (
	"context"
	buildkitv1alpha1 "cops-buildkit/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Buildkit struct {
	Name         string
	Namespace    string
	Labels       map[string]string
	Cloud        buildkitv1alpha1.CloudProvider
	Arch         buildkitv1alpha1.Arch
	Image        string
	NodeSelector map[string]string
	MaxReplica   int64
	Resource     corev1.ResourceRequirements
	client.Client
}

// TODO:// Create Spec of each resource of buildkit
// example https://github.com/andrcuns/charts/blob/main/charts/buildkit-service/templates
func (b *Buildkit) service() (*corev1.Service, error) {
	labels := map[string]string{
		"app": b.Name,
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Namespace:   b.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP", // Replace with your actual service type
			Ports: []corev1.ServicePort{
				{
					Port:       1234, // Replace with your actual port number
					TargetPort: intstr.FromString("tcp"),
					Protocol:   corev1.ProtocolTCP,
					Name:       "tcp",
				},
			},
			Selector: labels,
		},
	}

	return service, nil
}

func (b *Buildkit) deployment() (*appsv1.Deployment, error) {
	labels := map[string]string{
		"app": b.Name,
	}
	args := []string{
		"--addr",
		"unix:///run/user/1000/buildkit/buildkitd.sock",
		"tcp://0.0.0.0:1234",
		"unix:///run/user/1000/buildkit/buildkitd.sock",
		"--oci-worker-no-process-sandbox",
		"--tlscacert",
		"/certs/ca.pem",
		"--tlscert",
		"/certs/cert.pem",
		"--tlskey",
		"/certs/key.pem",
	}

	var user int64 = 1000

	sc := corev1.SecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: "Unconfined",
		},
		RunAsUser:  &user,
		RunAsGroup: &user,
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.Name,
			Namespace: b.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "buildkit-agent",
							Image: b.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "tcp",
									ContainerPort: 1234,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Args: args,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "certs",
									MountPath: "/certs",
									ReadOnly:  true,
								},
								{
									Name:      "buildkitd",
									MountPath: "/home/user/.local/share/buildkit",
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"buildctl", "debug", "workers"},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       30,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"buildctl", "debug", "workers"},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       30,
							},
							Resources:       b.Resource,
							SecurityContext: &sc,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "buildkitd-config",
									},
								},
							},
						},
						{
							Name: "certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "buildkitd-certs",
								},
							},
						},
						{
							Name: "buildkitd",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					NodeSelector: b.NodeSelector,
					// Affinity: &corev1.Affinity{},
					// Tolerations: []corev1.Toleration{},
				},
			},
		},
	}

	return deployment, nil
}

func (b *Buildkit) secret(ca, certs, key string) (*corev1.Secret, error) {
	labels := map[string]string{
		"app": b.Name,
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Namespace:   "default",
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Data: map[string][]byte{
			"cert.pem": []byte(certs),
			"key.pem":  []byte(key),
			// "ca.pem:" []byte(key),
		},
	}, nil
}

func (b *Buildkit) configmap() (*corev1.ConfigMap, error) {
	labels := map[string]string{
		"app": b.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Namespace:   "default",
			Labels:      labels,
			Annotations: map[string]string{},
		},
		// Data: map[string][]byte{},
	}, nil
}

func (b *Buildkit) podDisruptionBudget() (*policyv1.PodDisruptionBudget, error) {
	labels := map[string]string{
		"app": b.Name,
	}
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Labels:      labels,
			Namespace:   "default",
			Annotations: map[string]string{},
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
	}, nil
}

func (b *Buildkit) horizontalPodAutoscalerionBudget() (*autoscalingv2.HorizontalPodAutoscaler, error) {
	labels := map[string]string{
		"app": b.Name,
	}
	var minReplica int32 = 1
	var avg int32 = 80
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Labels:      labels,
			Namespace:   "default",
			Annotations: map[string]string{},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       b.Name,
			},
			MinReplicas: &minReplica,
			MaxReplicas: int32(b.MaxReplica),
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &avg,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "memory",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &avg,
						},
					},
				},
			},
		},
	}, nil
}

func (b *Buildkit) CreateOrUpdateDeployment(ctx context.Context) error {

	deployment, err := b.deployment()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &appsv1.Deployment{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, deployment); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, deployment); err != nil {
		return err
	}
	return nil
}

func (b *Buildkit) CreateOrUpdateService(ctx context.Context) error {

	svc, err := b.service()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &corev1.Service{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, svc); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, svc); err != nil {
		return err
	}
	return nil
}

func (b *Buildkit) CreateOrUpdateConfig(ctx context.Context) error {

	cm, err := b.configmap()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &corev1.ConfigMap{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, cm); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, cm); err != nil {
		return err
	}
	return nil
}

func (b *Buildkit) CreateOrUpdateSecret(ctx context.Context) error {

	secret, err := b.secret("sa", "sa", "sa")
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &corev1.ConfigMap{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, secret); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, secret); err != nil {
		return err
	}
	return nil
}

func (b *Buildkit) CreateOrUpdatePodDisruptionBudget(ctx context.Context) error {

	pdb, err := b.podDisruptionBudget()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &policyv1.PodDisruptionBudget{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, pdb); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, pdb); err != nil {
		return err
	}
	return nil
}

func (b *Buildkit) CreateOrUpdateHorizontalPodAutoscalerionBudget(ctx context.Context) error {

	hpa, err := b.horizontalPodAutoscalerionBudget()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &autoscalingv2.HorizontalPodAutoscaler{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, hpa); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, hpa); err != nil {
		return err
	}
	return nil
}
