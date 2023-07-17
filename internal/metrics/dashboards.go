package metrics

import (
	"context"
	"path"

	log "github.com/ViaQ/logerr/v2/log/static"
	logging "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/internal/reconcile"
	"github.com/openshift/cluster-logging-operator/internal/runtime"
	"github.com/openshift/cluster-logging-operator/internal/utils"
	"github.com/openshift/cluster-logging-operator/internal/utils/comparators/configmaps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CollectorDashboardFile = "dashboards/collector/openshift-logging-dashboard.json"
	DashboardName          = "grafana-dashboard-cluster-logging"
	DashboardNS            = "openshift-config-managed"
	DashboardFileName      = "openshift-logging.json"
	DashboardHashName      = "contentHash"
)

func newDashboardConfigMap() *corev1.ConfigMap {
	var spec = string(utils.GetFileContents(path.Join(utils.GetShareDir(), CollectorDashboardFile)))
	hash, err := utils.CalculateMD5Hash(spec)
	if err != nil {
		log.Error(err, "Error calculated hash for metrics dashboard")
	}
	cm := runtime.NewConfigMap(DashboardNS,
		DashboardName,
		map[string]string{
			DashboardFileName: spec,
		},
	)
	runtime.NewConfigMapBuilder(cm).
		AddLabel("console.openshift.io/dashboard", "true").
		AddLabel(DashboardHashName, hash)

	return cm
}

func ReconcileDashboards(writer client.Writer, reader client.Reader, collection *logging.CollectionSpec) (err error) {
	cm := newDashboardConfigMap()
	if err := reconcile.Configmap(writer, reader, cm, configmaps.CompareLabels); err != nil {
		return err
	}

	return nil
}

// RemoveDashboardConfigMap removes the config map in the grafana dashboard
func RemoveDashboardConfigMap(c client.Client) (err error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DashboardName,
			Namespace: DashboardNS,
		},
	}
	return c.Delete(context.TODO(), cm)
}
