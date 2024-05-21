package buildkit

import (
	"context"
	buildkitv1alpha1 "cops-buildkit/api/v1alpha1"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

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
	Arch         []buildkitv1alpha1.Arch
	Image        string
	NodeSelector map[string]string
	Rootless     bool
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
		"app":     b.Name,
		"service": "buildkit",
	}
	args := []string{
		"--addr",
		"unix:///run/buildkit/buildkitd.sock",
		"--addr",
		"tcp://0.0.0.0:1234",
		"--debug",
		"--tlscacert",
		"/certs/ca.pem",
		"--tlscert",
		"/certs/cert.pem",
		"--tlskey",
		"/certs/key.pem",
	}

	if b.Rootless {
		args = []string{
			"--addr",
			"unix:///run/user/1000/buildkit/buildkitd.sock",
			"--addr",
			"tcp://0.0.0.0:1234",
			"--oci-worker-no-process-sandbox",
			"--debug",
			"--tlscacert",
			"/certs/ca.pem",
			"--tlscert",
			"/certs/cert.pem",
			"--tlskey",
			"/certs/key.pem",
		}

	}
	var privileged bool = true
	var user int64 = 1000
	sc := corev1.SecurityContext{
		Privileged: &privileged,
	}

	if b.Rootless {
		sc = corev1.SecurityContext{
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeUnconfined,
			},
			RunAsUser:  &user,
			RunAsGroup: &user,
		}
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
					Annotations: map[string]string{
						"container.apparmor.security.beta.kubernetes.io/buildkitd": "unconfined",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "buildkitd",
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
							Name: "certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: b.Name,
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
				},
			},
		},
	}

	return deployment, nil
}

func (b *Buildkit) secret() (*corev1.Secret, error) {
	certs, key, ca, err := generateCertificate()
	if err != nil {
		return nil, err
	}
	labels := map[string]string{
		"app": b.Name,
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Namespace:   b.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Data: map[string][]byte{
			"cert.pem": certs,
			"key.pem":  key,
			"ca.pem":   ca,
		},
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
			Namespace:   b.Namespace,
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
			Namespace:   b.Namespace,
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

func (b *Buildkit) CreateOrUpdateSecret(ctx context.Context) error {

	secret, err := b.secret()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &corev1.Secret{})

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

func generateCertificate() (certs []byte, key []byte, ca []byte, err error) {
	// Generate a new private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create a self-signed certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Your Organization"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// Encode the certificate and private key to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	// Generate a CA certificate (self-signed)
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Your CA Organization"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // Valid for 10 years
		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, nil, err
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes})

	return certPEM, keyPEM, caPEM, nil
}
