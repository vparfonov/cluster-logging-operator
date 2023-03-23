package k8shandler

import (
	"github.com/ViaQ/logerr/v2/kverrors"
	log "github.com/ViaQ/logerr/v2/log/static"
)

func (clusterRequest *ClusterLoggingRequest) appendFinalizer(identifier string) error {
	instance, err := clusterRequest.getClusterLogging(true)
	if err != nil {
		return kverrors.Wrap(err, "Error getting ClusterLogging for appending finalizer.")
	}

	for _, f := range instance.GetFinalizers() {
		if f == identifier {
			// Skip if finalizer already exists
			return nil
		}
	}

	instance.Finalizers = append(instance.GetFinalizers(), identifier)
	if err := clusterRequest.Update(instance); err != nil {
		return kverrors.Wrap(err, "Can not update ClusterLogging finalizers.")
	}

	return nil
}

func (clusterRequest *ClusterLoggingRequest) removeFinalizer(identifier string) error {
	log.V(0).Info("\n\n removeFinalizer.\n\n")
	instance, err := clusterRequest.getClusterLogging(true)
	if err != nil {
		return kverrors.Wrap(err, "Error getting ClusterLogging for removing finalizer.")
	}

	found := false
	finalizers := []string{}
	for _, f := range instance.GetFinalizers() {
		if f == identifier {
			found = true
			continue
		}

		finalizers = append(finalizers, f)
	}

	if !found {
		// Finalizer is not in list anymore
		return nil
	}

	instance.Finalizers = finalizers
	if err := clusterRequest.Update(instance); err != nil {
		return kverrors.Wrap(err, "Failed to remove finalizer from ClusterLogging.")
	}
	log.V(0).Info("\n\n Failed to remove finalizer from ClusterLogging.\n\n")
	return nil
}
