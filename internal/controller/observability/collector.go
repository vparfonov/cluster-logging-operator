package observability

import (
	"strings"
	"time"

	log "github.com/ViaQ/logerr/v2/log/static"
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	internalcontext "github.com/openshift/cluster-logging-operator/internal/api/context"
	internalobs "github.com/openshift/cluster-logging-operator/internal/api/observability"
	"github.com/openshift/cluster-logging-operator/internal/auth"
	"github.com/openshift/cluster-logging-operator/internal/collector"
	"github.com/openshift/cluster-logging-operator/internal/collector/cloudwatch"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	"github.com/openshift/cluster-logging-operator/internal/factory"
	forwardergenerator "github.com/openshift/cluster-logging-operator/internal/generator/forwarder"
	"github.com/openshift/cluster-logging-operator/internal/generator/framework"
	generatorhelpers "github.com/openshift/cluster-logging-operator/internal/generator/helpers"
	"github.com/openshift/cluster-logging-operator/internal/metrics"
	"github.com/openshift/cluster-logging-operator/internal/network"
	"github.com/openshift/cluster-logging-operator/internal/reconcile"
	"github.com/openshift/cluster-logging-operator/internal/runtime/serviceaccount"
	"github.com/openshift/cluster-logging-operator/internal/tls"
	"github.com/openshift/cluster-logging-operator/internal/utils"
	"github.com/openshift/cluster-logging-operator/internal/validations/observability"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ReconcileCollector(context internalcontext.ForwarderContext, pollInterval, timeout time.Duration) (err error) {

	if err = reconcile.SecurityContextConstraints(context.Client, context.Reader, auth.NewSCC()); err != nil {
		log.V(3).Error(err, "reconcile.SecurityContextConstraints")
		return err
	}

	ownerRef := utils.AsOwner(context.Forwarder)
	resourceNames := factory.ResourceNames(*context.Forwarder)

	options := framework.Options{}

	// Set options to any options added during initialization of CLF
	if context.AdditionalContext != nil {
		options = context.AdditionalContext
	}

	// Set kubeapi and rollout options based on annotation (LOG-7196)
	// TODO: replace with API fields
	SetKubeCacheOption(context.Forwarder.Annotations, options)
	SetMaxUnavailableRolloutOption(context.Forwarder.Annotations, options)

	if internalobs.Outputs(context.Forwarder.Spec.Outputs).NeedServiceAccountToken() {
		// temporarily create SA token until collector is capable of dynamically reloading a projected serviceaccount token
		var sa *corev1.ServiceAccount
		sa, err = serviceaccount.Get(context.Client, context.Forwarder.Namespace, context.Forwarder.Spec.ServiceAccount.Name)
		if err != nil {
			log.V(3).Error(err, "serviceaccount.Get")
			return err
		}
		var saTokenSecret *corev1.Secret
		if saTokenSecret, err = auth.ReconcileServiceAccountTokenSecret(sa, context.Client, context.Forwarder.Namespace, resourceNames.ServiceAccountTokenSecret, ownerRef); err != nil {
			log.V(3).Error(err, "auth.ReconcileServiceAccountTokenSecret")
			return err
		}
		context.Secrets[saTokenSecret.Name] = saTokenSecret
		options[framework.OptionServiceAccountTokenSecretName] = resourceNames.ServiceAccountTokenSecret
	}

	// Add roles to ServiceAccount to allow the collector to read from the node
	if err = auth.ReconcileRBAC(context.Client, context.Forwarder.Name, context.Forwarder.Namespace, context.Forwarder.Spec.ServiceAccount.Name, ownerRef); err != nil {
		log.V(3).Error(err, "auth.ReconcileRBAC")
		return
	}

	// TODO: This can be the same per NS but what is the ownerref?  Multiple CLFs will clash
	if err = collector.ReconcileTrustedCABundleConfigMap(context.Client, context.Forwarder.Namespace, resourceNames.CaTrustBundle, ownerRef); err != nil {
		log.Error(err, "collector.ReconcileTrustedCABundleConfigMap")
		return err
	}
	trustedCABundle := collector.WaitForTrustedCAToBePopulated(context.Client, context.Forwarder.Namespace, resourceNames.CaTrustBundle, pollInterval, timeout)

	credCm, err := cloudwatch.ReconcileAWSCredentialsConfigMap(context.Client, context.Reader, context.Forwarder.Namespace, resourceNames.AwsCredentialsFile, context.Forwarder.Spec.Outputs, context.Secrets, context.ConfigMaps, ownerRef)
	if err != nil {
		log.V(9).Error(err, "collector.ReconcileAWSProfileConfig")
		return err
	}

	// Add generated credentials configmap to contexts to be mounted in pod
	if credCm != nil {
		context.ConfigMaps[credCm.Name] = credCm
	}

	var collectorConfig string
	if collectorConfig, err = GenerateConfig(context.Client, *context.Forwarder, *resourceNames, context.Secrets, options); err != nil {
		log.V(9).Error(err, "collector.GenerateConfig")
		return err
	}
	log.V(3).Info("Generated collector config", "config", collectorConfig)
	var collectorConfHash string
	collectorConfHash, err = utils.CalculateMD5Hash(collectorConfig)
	if err != nil {
		log.Error(err, "unable to calculate MD5 hash")
		log.V(9).Error(err, "Returning from unable to calculate MD5 hash")
		return
	}

	isDaemonSet := !internalobs.DeployAsDeployment(*context.Forwarder)
	log.V(3).Info("Deploying as DaemonSet", "isDaemonSet", isDaemonSet)
	collectorFactory := collector.New(
		collectorConfHash,
		context.ClusterID,
		context.Forwarder.Spec.Collector,
		context.Secrets, context.ConfigMaps,
		context.Forwarder.Spec,
		resourceNames,
		isDaemonSet,
		LogLevel(context.Forwarder.Annotations),
		factory.IncludesKubeCacheOption(options),
		factory.GetMaxUnavailableValue(options),
	)

	if err = collectorFactory.ReconcileCollectorConfig(context.Client, context.Reader, context.Forwarder.Namespace, collectorConfig, ownerRef); err != nil {
		log.Error(err, "collector.ReconcileCollectorConfig")
		return
	}

	reconcileWorkload := collectorFactory.ReconcileDaemonset
	if !isDaemonSet {
		reconcileWorkload = collectorFactory.ReconcileDeployment
	}

	if err := reconcileWorkload(context.Client, context.Forwarder.Namespace, trustedCABundle, ownerRef); err != nil {
		log.Error(err, "Error reconciling the deployment of the collector")
		return err
	}

	if err := collectorFactory.ReconcileInputServices(context.Client, context.Reader, context.Forwarder.Namespace, ownerRef, collectorFactory.CommonLabelInitializer); err != nil {
		log.Error(err, "collector.ReconcileInputServices")
		return err
	}

	// Reconcile resources to support metrics gathering
	if err := network.ReconcileService(context.Client, context.Forwarder.Namespace, resourceNames.CommonName, context.Forwarder.Name, constants.CollectorName, collector.MetricsPortName, resourceNames.SecretMetrics, collector.MetricsPort, ownerRef, collectorFactory.CommonLabelInitializer); err != nil {
		log.Error(err, "collector.ReconcileService")
		return err
	}
	metricsSelector := metrics.BuildSelector(constants.CollectorName, resourceNames.CommonName)
	if err := metrics.ReconcileServiceMonitor(context.Client, context.Forwarder.Namespace, resourceNames.CommonName, ownerRef, metricsSelector, collector.MetricsPortName); err != nil {
		log.Error(err, "collector.ReconcileServiceMonitor")
		return err
	}

	return nil
}

func GenerateConfig(k8Client client.Client, clf obs.ClusterLogForwarder, resourceNames factory.ForwarderResourceNames, secrets internalobs.Secrets, op framework.Options) (config string, err error) {
	tlsProfile, _ := tls.FetchAPIServerTlsProfile(k8Client)
	op[framework.ClusterTLSProfileSpec] = tls.GetClusterTLSProfileSpec(tlsProfile)
	//EvaluateAnnotationsForEnabledCapabilities(clusterRequest.Forwarder, op)
	g := forwardergenerator.New()
	generatedConfig, err := g.GenerateConf(secrets, clf.Spec, clf.Namespace, clf.Name, resourceNames, op)

	if err != nil {
		log.Error(err, "Unable to generate log configuration")
		return "", err
	}

	log.V(3).Info("ClusterLogForwarder generated config", generatedConfig)
	return generatedConfig, err
}

// EvaluateAnnotationsForEnabledCapabilities populates generator options with capabilities enabled by the ClusterLogForwarder
func EvaluateAnnotationsForEnabledCapabilities(annotations map[string]string, options framework.Options) {
	if annotations == nil {
		return
	}
	for key, value := range annotations {
		switch key {
		case constants.AnnotationDebugOutput:
			if strings.ToLower(value) == "true" {
				options[generatorhelpers.EnableDebugOutput] = "true"
			}
		case constants.AnnotationKubeCache:
			// Matching the validate_annotations logic
			if observability.IsEnabledValue(value) {
				options[framework.UseKubeCacheOption] = "true"
			}
		case constants.AnnotationMaxUnavailable:
			// Matching the validate_annotations logic
			if observability.IsPercentOrWholeNumber(value) {
				options[framework.MaxUnavailableOption] = value
			}
		}
	}
}

func LogLevel(annotations map[string]string) string {
	if level, ok := annotations[constants.AnnotationVectorLogLevel]; ok {
		return level
	}
	return "warn"
}

func SetKubeCacheOption(annotations map[string]string, options framework.Options) {
	if value, found := annotations[constants.AnnotationKubeCache]; found {
		if observability.IsEnabledValue(value) {
			log.V(3).Info("Kube cache annotation found")
			options[framework.UseKubeCacheOption] = "true"
		}
	}
}

func SetMaxUnavailableRolloutOption(annotations map[string]string, options framework.Options) {
	if value, found := annotations[constants.AnnotationMaxUnavailable]; found {
		if observability.IsPercentOrWholeNumber(value) {
			log.V(3).Info("Max Unavailable annotation found")
			options[framework.MaxUnavailableOption] = value
		}
	}
}
