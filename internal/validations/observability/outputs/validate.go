package outputs

import (
	"fmt"
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	internalcontext "github.com/openshift/cluster-logging-operator/internal/api/context"
	internalobs "github.com/openshift/cluster-logging-operator/internal/api/observability"
	"github.com/openshift/cluster-logging-operator/internal/validations/observability/common"
	"strings"
)

func Validate(context internalcontext.ForwarderContext) {
	for _, out := range context.Forwarder.Spec.Outputs {
		messages := []string{}
		configs := internalobs.SecretReferencesAsValueReferences(out)
		if out.TLS != nil {
			messages = append(messages, validateURLAccordingToTLS(out)...)
			configs = append(configs, internalobs.ValueReferences(out.TLS.TLSSpec)...)
		}
		messages = append(messages, common.ValidateValueReference(configs, context.Secrets, context.ConfigMaps)...)
		// Validate by output type
		switch out.Type {
		case obs.OutputTypeCloudwatch:
			messages = append(messages, ValidateCloudWatchAuth(out, context)...)
		case obs.OutputTypeHTTP:
			messages = append(messages, validateHttpContentTypeHeaders(out)...)
		case obs.OutputTypeOTLP:
			messages = append(messages, ValidateOtlpAnnotation(context)...)
		}
		// Set condition
		if len(messages) > 0 {
			internalobs.SetCondition(&context.Forwarder.Status.Outputs,
				internalobs.NewConditionFromPrefix(obs.ConditionTypeValidOutputPrefix, out.Name, false, obs.ReasonValidationFailure, strings.Join(messages, ",")))
		} else {
			internalobs.SetCondition(&context.Forwarder.Status.Outputs,
				internalobs.NewConditionFromPrefix(obs.ConditionTypeValidOutputPrefix, out.Name, true, obs.ReasonValidationSuccess, fmt.Sprintf("output %q is valid", out.Name)))
		}
	}
}
