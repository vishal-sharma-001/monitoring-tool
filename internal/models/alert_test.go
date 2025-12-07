package models_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/monitoring-engine/monitoring-tool/internal/models"
)

var _ = Describe("Alert", func() {
	Describe("NewAlert", func() {
		It("should create a new alert with all fields populated", func() {
			severity := "high"
			message := "Test alert message"
			source := "k8s_pod"
			value := 95.5
			labels := map[string]string{
				"pod":       "test-pod",
				"namespace": "default",
			}

			alert := models.NewAlert(severity, message, source, value, labels)

			Expect(alert).NotTo(BeNil())
			Expect(alert.ID.String()).NotTo(BeEmpty())
			Expect(alert.Status).To(Equal(models.AlertStatusFiring))
			Expect(alert.Severity).To(Equal(severity))
			Expect(alert.Message).To(Equal(message))
			Expect(alert.Source).To(Equal(source))
			Expect(alert.Value).To(Equal(value))
			Expect(alert.Labels).NotTo(BeNil())
			Expect(alert.TriggeredAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(alert.ResolvedAt).To(BeNil())
			Expect(alert.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(alert.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should handle empty labels", func() {
			alert := models.NewAlert("low", "Test", "test", 0, map[string]string{})

			Expect(alert).NotTo(BeNil())
			Expect(alert.Labels).NotTo(BeNil())
		})

		It("should handle nil labels", func() {
			alert := models.NewAlert("low", "Test", "test", 0, nil)

			Expect(alert).NotTo(BeNil())
			Expect(alert.Labels).NotTo(BeNil())
		})
	})

	Describe("TableName", func() {
		It("should return the correct table name", func() {
			alert := models.Alert{}
			Expect(alert.TableName()).To(Equal("alerts"))
		})
	})

	Describe("Severity Levels", func() {
		It("should support all severity levels", func() {
			severities := []string{"critical", "high", "medium", "low"}

			for _, severity := range severities {
				alert := models.NewAlert(severity, "Test", "test", 0, nil)
				Expect(alert.Severity).To(Equal(severity))
			}
		})
	})

	Describe("Sources", func() {
		It("should support all source types", func() {
			sources := []string{"k8s_pod", "k8s_node", "rule"}

			for _, source := range sources {
				alert := models.NewAlert("high", "Test", source, 0, nil)
				Expect(alert.Source).To(Equal(source))
			}
		})
	})


	Describe("Resolve", func() {
		It("should mark alert as resolved and set resolved timestamp", func() {
			alert := models.NewAlert("high", "Test", "k8s_pod", 0, nil)
			Expect(alert.Status).To(Equal(models.AlertStatusFiring))
			Expect(alert.ResolvedAt).To(BeNil())

			alert.Resolve()

			Expect(alert.Status).To(Equal(models.AlertStatusResolved))
			Expect(alert.ResolvedAt).NotTo(BeNil())
			Expect(*alert.ResolvedAt).To(BeTemporally("~", time.Now(), time.Second))
		})
	})

	Describe("IsFiring", func() {
		It("should return true when alert is firing", func() {
			alert := models.NewAlert("high", "Test", "k8s_pod", 0, nil)
			Expect(alert.IsFiring()).To(BeTrue())
		})

		It("should return false when alert is resolved", func() {
			alert := models.NewAlert("high", "Test", "k8s_pod", 0, nil)
			alert.Resolve()
			Expect(alert.IsFiring()).To(BeFalse())
		})
	})

	Describe("GetLabelsMap", func() {
		It("should return labels as a map", func() {
			labels := map[string]string{
				"pod":       "test-pod",
				"namespace": "default",
			}
			alert := models.NewAlert("high", "Test", "k8s_pod", 0, labels)

			result := alert.GetLabelsMap()
			Expect(result).To(HaveKeyWithValue("pod", "test-pod"))
			Expect(result).To(HaveKeyWithValue("namespace", "default"))
		})

		It("should handle empty labels", func() {
			alert := models.NewAlert("high", "Test", "k8s_pod", 0, nil)

			result := alert.GetLabelsMap()
			Expect(result).To(BeEmpty())
		})
	})
})
