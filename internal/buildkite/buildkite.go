package buildkite

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Buildkite struct {
	Name         string
	Namespace    string
	Labels       map[string]string
	Image        string
	Secret       string
	NodeSelector map[string]string
	Resource     corev1.ResourceRequirements
	client.Client
}

func (b *Buildkite) sa() (*corev1.ServiceAccount, error) {
	labels := map[string]string{
		"app":     b.Name,
		"service": "buildkite",
	}

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Namespace:   b.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
	}, nil
}

func (b *Buildkite) role() (*rbacv1.Role, error) {
	labels := map[string]string{
		"app":     b.Name,
		"service": "buildkite",
	}

	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Namespace:   b.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"batch"},
				Resources: []string{"job"},
				Verbs:     []string{"get", "list", "update", "delete", "watch", "create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "update", "delete", "watch", "create"},
			},
		},
	}, nil
}

func (b *Buildkite) rolebinding() (*rbacv1.RoleBinding, error) {
	labels := map[string]string{
		"app":     b.Name,
		"service": "buildkite",
	}

	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Namespace:   b.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     b.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: b.Namespace,
				Name:      b.Name,
			},
		},
	}, nil
}

func (b *Buildkite) configmap() (*corev1.ConfigMap, error) {
	labels := map[string]string{
		"app":     b.Name,
		"service": "buildkite",
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        b.Name,
			Namespace:   b.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Data: map[string]string{
			"config.yaml": `
namespace: default
agent-token-secret: ssssss
			`,
		},
	}, nil
}

func (b *Buildkite) deployment() (*appsv1.Deployment, error) {
	labels := map[string]string{
		"app":     b.Name,
		"service": "buildkite",
	}

	var bl bool = true

	sc := corev1.SecurityContext{
		AllowPrivilegeEscalation: &bl,
		ReadOnlyRootFilesystem:   &bl,
		RunAsNonRoot:             &bl,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
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
					ServiceAccountName: "",
					NodeSelector:       map[string]string{},
					Containers: []corev1.Container{
						{
							Name:  "controller",
							Image: b.Image,
							Args:  []string{},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/config",
									SubPath:   "config.yaml",
									ReadOnly:  true,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "CONFIG",
									Value: "/etc/config.yaml",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: b.Secret,
										},
									},
								},
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
										Name: b.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return deployment, nil
}

func (b *Buildkite) CreateOrUpdateDeployment(ctx context.Context) error {

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

func (b *Buildkite) CreateOrUpdateConfigMap(ctx context.Context) error {

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

func (b *Buildkite) CreateOrUpdateRole(ctx context.Context) error {

	role, err := b.role()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &rbacv1.Role{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, role); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, role); err != nil {
		return err
	}
	return nil
}

func (b *Buildkite) CreateOrUpdateRoleBinding(ctx context.Context) error {

	rb, err := b.rolebinding()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &rbacv1.RoleBinding{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, rb); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, rb); err != nil {
		return err
	}
	return nil
}

func (b *Buildkite) CreateOrUpdateServiceAccount(ctx context.Context) error {

	sa, err := b.sa()
	if err != nil {
		return err
	}

	err = b.Client.Get(ctx, types.NamespacedName{
		Name:      b.Name,
		Namespace: b.Namespace,
	}, &corev1.ServiceAccount{})

	if err != nil {
		if errors.IsNotFound(err) {

			if err := b.Client.Create(ctx, sa); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if err := b.Client.Update(ctx, sa); err != nil {
		return err
	}
	return nil
}
