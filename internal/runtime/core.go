package runtime

import (
	"github.com/openshift/cluster-logging-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NewNamespace returns a corev1.Namespace with name.
func NewNamespace(name string) *corev1.Namespace {
	ns := &corev1.Namespace{}
	Initialize(ns, "", name)
	return ns
}

// NewConfigMap returns a corev1.ConfigMap with namespace, name and data.
func NewConfigMap(namespace, name string, data map[string]string, visitors ...func(meta metav1.Object)) *corev1.ConfigMap {
	if data == nil {
		data = map[string]string{}
	}
	cm := &corev1.ConfigMap{Data: data}
	Initialize(cm, namespace, name, visitors...)
	return cm
}

// NewPod returns a corev1.Pod with namespace, name, containers.
func NewPod(namespace, name string, containers ...corev1.Container) *corev1.Pod {
	pod := &corev1.Pod{Spec: corev1.PodSpec{Containers: containers}}
	Initialize(pod, namespace, name)
	return pod
}

// NewService returns a corev1.Service with namespace and name.
func NewService(namespace, name string) *corev1.Service {
	svc := &corev1.Service{}
	Initialize(svc, namespace, name)
	return svc
}

// NewServiceAccount returns a corev1.ServiceAccount with namespace and name.
func NewServiceAccount(namespace, name string) *corev1.ServiceAccount {
	obj := &corev1.ServiceAccount{}
	Initialize(obj, namespace, name)
	return obj
}

// NewSecret returns a corev1.Secret with namespace and name.
func NewSecret(namespace, name string, data map[string][]byte, visitors ...func(meta metav1.Object)) *corev1.Secret {
	if data == nil {
		data = map[string][]byte{}
	}
	s := &corev1.Secret{Data: data}
	Initialize(s, namespace, name, visitors...)
	return s
}

func NewDaemonSet(daemonSetName, namespace, component string, podSpec corev1.PodSpec, visitors ...func(meta metav1.Object)) *appsv1.DaemonSet {
	labels := map[string]string{
		"provider":      "openshift",
		"component":     component,
		"logging-infra": component,
	}
	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      daemonSetName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   daemonSetName,
					Labels: labels,
					Annotations: map[string]string{
						"scheduler.alpha.kubernetes.io/critical-pod": "",
						"target.workload.openshift.io/management":    `{"effect": "PreferredDuringScheduling"}`,
					},
				},
				Spec: podSpec,
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "100%",
					},
				},
			},
		},
	}
	Initialize(ds, namespace, daemonSetName, visitors...)
	//TODO: should we keep this labels?
	utils.AddLabels(ds, labels)

	dl := ds.GetLabels()
	ds.Spec.Template.SetLabels(dl)
	ds.Spec.Selector.MatchLabels = dl
	return ds
}
