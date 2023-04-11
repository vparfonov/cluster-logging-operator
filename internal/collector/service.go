package collector

import (
	logging "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	"github.com/openshift/cluster-logging-operator/internal/factory"
	"github.com/openshift/cluster-logging-operator/internal/reconcile"
	"github.com/openshift/cluster-logging-operator/internal/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileService reconciles the service specifically for the collector that exposes the collector metrics
func ReconcileService(er record.EventRecorder, k8sClient client.Client, instance *logging.ClusterLogging, name string, owner metav1.OwnerReference) error {
	desired := factory.NewService(
		constants.CollectorName,
		instance.Namespace,
		constants.CollectorName,
		[]v1.ServicePort{
			{
				Port:       MetricsPort,
				TargetPort: intstr.FromString(MetricsPortName),
				Name:       MetricsPortName,
			},
			{
				Port:       ExporterPort,
				TargetPort: intstr.FromString(ExporterPortName),
				Name:       ExporterPortName,
			},
		},
	)

	desired.Annotations = map[string]string{
		constants.AnnotationServingCertSecretName: constants.CollectorMetricSecretName,
	}

	utils.AddOwnerRefToObject(desired, owner)
	utils.SetCommonLabels(desired, instance, "support")
	return reconcile.Service(er, k8sClient, desired)
}
