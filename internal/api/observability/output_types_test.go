package observability_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	obsv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	. "github.com/openshift/cluster-logging-operator/internal/api/observability"
	"github.com/openshift/cluster-logging-operator/test"
	"strings"
)

var _ = Describe("helpers for output types", func() {

	Context("#HasAtLeastOnceDelivery", func() {
		It("should return false when no outputs have AtLeastOnce delivery", func() {
			outputs := Outputs{
				{Name: "out1", Type: obsv1.OutputTypeSplunk, Splunk: &obsv1.Splunk{}},
			}
			Expect(outputs.HasAtLeastOnceDelivery()).To(BeFalse())
		})

		It("should return true when an output has AtLeastOnce delivery", func() {
			outputs := Outputs{
				{
					Name: "out1",
					Type: obsv1.OutputTypeSplunk,
					Splunk: &obsv1.Splunk{
						Tuning: &obsv1.SplunkTuningSpec{
							BaseOutputTuningSpec: obsv1.BaseOutputTuningSpec{
								DeliveryMode: obsv1.DeliveryModeAtLeastOnce,
							},
						},
					},
				},
			}
			Expect(outputs.HasAtLeastOnceDelivery()).To(BeTrue())
		})

		It("should return false when outputs have AtMostOnce delivery", func() {
			outputs := Outputs{
				{
					Name: "out1",
					Type: obsv1.OutputTypeSplunk,
					Splunk: &obsv1.Splunk{
						Tuning: &obsv1.SplunkTuningSpec{
							BaseOutputTuningSpec: obsv1.BaseOutputTuningSpec{
								DeliveryMode: obsv1.DeliveryModeAtMostOnce,
							},
						},
					},
				},
			}
			Expect(outputs.HasAtLeastOnceDelivery()).To(BeFalse())
		})

		It("should return false for empty outputs", func() {
			outputs := Outputs{}
			Expect(outputs.HasAtLeastOnceDelivery()).To(BeFalse())
		})
	})

	Context("#SecretReferences", func() {

		It("should return an empty set of keys when authentication is not defined for an output", func() {
			for _, t := range obsv1.OutputTypes {

				outputType := strings.TrimPrefix("OutputType", string(t))
				outputType = strings.ToLower(outputType[0:1]) + outputType[1:]
				yaml := test.JSONLine(map[string]interface{}{
					"type":     t,
					outputType: map[string]interface{}{},
				})
				spec := &obsv1.OutputSpec{}
				test.MustUnmarshal(yaml, spec)
				Expect(SecretReferences(*spec)).To(BeEmpty())
			}
		})

	})
})
