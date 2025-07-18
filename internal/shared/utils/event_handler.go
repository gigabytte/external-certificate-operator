package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gigabytte/external-certificate-operator/internal/shared/log"
	"github.com/gigabytte/external-certificate-operator/internal/shared/vars"
)

type CertificateObject interface {
	client.Object
	GetConditions() []metav1.Condition
	GetAnnotations() map[string]string
	SetConditions([]metav1.Condition)
	GetRetryCount() int
	SetRetryCount(count int)
}

// Options defines configuration for event handling and retries
type Options struct {
	Recorder       record.EventRecorder
	Client         client.Client
	MaxDuration    time.Duration
	MaxRetries     int
	BaseDelay      time.Duration
	Logger         logr.Logger
	MaxStatusCount int
}

// DefaultOptions returns Options with sensible defaults
func DefaultOptions(recorder record.EventRecorder, client client.Client) Options {
	logger := log.FromContext(context.Background())
	return Options{
		Recorder:       recorder,
		Client:         client,
		MaxDuration:    vars.MaxDuration,
		MaxRetries:     vars.MaxRetries,
		BaseDelay:      vars.BaseDelay,
		MaxStatusCount: vars.MaxStatusCount,
		Logger:         logger,
	}
}

type EventHandler struct {
	opts Options
}

func NewEventHandler(opts Options) *EventHandler {
	return &EventHandler{
		opts: opts,
	}
}

func (h *EventHandler) ReturnEvent(
	ctx context.Context,
	obj CertificateObject,
	currentRetryCount int,
	err error,
	msg string,
) (ctrl.Result, error) {
	freshObj := obj.DeepCopyObject().(CertificateObject)
	key := client.ObjectKeyFromObject(obj)
	if err := h.opts.Client.Get(ctx, key, freshObj); err != nil {
		return ctrl.Result{}, err
	}
	if err != nil {
		if currentRetryCount >= h.opts.MaxRetries {
			return h.handleMaxRetriesExceeded(ctx, freshObj, err)
		}
		return h.scheduleNextRetry(ctx, freshObj, currentRetryCount, err)
	}

	return h.handleSuccess(ctx, freshObj)
}

func (h *EventHandler) handleSuccess(
	ctx context.Context,
	obj CertificateObject,
) (ctrl.Result, error) {
	msg := "Successfully synced secret(s)"
	h.opts.Recorder.Event(obj, corev1.EventTypeNormal, "CertificateProcessingSucceeded", msg)

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "CertificateProcessingSucceeded",
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	}

	if updateErr := h.updateStatus(ctx, obj, condition); updateErr != nil {
		h.opts.Logger.Error(updateErr, "Failed to update status after success")
	}

	return ctrl.Result{}, nil
}

func (h *EventHandler) handleMaxRetriesExceeded(
	ctx context.Context,
	obj CertificateObject,
	err error,
) (ctrl.Result, error) {
	msg := fmt.Sprintf("Maximum retry attempts (%d) reached: %v", h.opts.MaxRetries, err)
	h.opts.Recorder.Event(obj, corev1.EventTypeWarning, "MaxRetriesExceeded", msg)

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "MaxRetriesExceeded",
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	}

	if updateErr := h.updateStatus(ctx, obj, condition); updateErr != nil {
		h.opts.Logger.Error(updateErr, "failed to update status")
	}

	return ctrl.Result{}, err
}

func (h *EventHandler) scheduleNextRetry(
	ctx context.Context,
	obj CertificateObject,
	currentRetryCount int,
	err error,
) (ctrl.Result, error) {
	backoff := h.calculateBackoff(currentRetryCount)
	msg := fmt.Sprintf("Retry %d/%d scheduled in %v: %v",
		currentRetryCount, h.opts.MaxRetries, backoff, err)

	h.opts.Recorder.Event(obj, corev1.EventTypeNormal, "CertificateProcessingFailed", msg)

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "CertificateProcessingFailed",
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	}

	if updateErr := h.updateStatus(ctx, obj, condition); updateErr != nil {
		h.opts.Logger.Error(updateErr, "Failed to update status after retry")
	}

	return ctrl.Result{RequeueAfter: backoff}, nil
}

func (h *EventHandler) calculateBackoff(currentRetryCount int) time.Duration {
	duration := time.Duration(1<<uint(currentRetryCount)) * h.opts.BaseDelay
	if duration > h.opts.MaxDuration {
		duration = h.opts.MaxDuration
	}

	return duration
}

func (h *EventHandler) updateStatus(
	ctx context.Context,
	obj CertificateObject,
	condition metav1.Condition,
) error {
	maxStatusCount := h.opts.MaxStatusCount
	conditions := obj.GetConditions()
	conditions = append([]metav1.Condition{condition}, conditions...)
	// Only keep the last MaxEventCount number of status conditions
	if len(conditions) > maxStatusCount {
		conditions = conditions[:maxStatusCount]
	}

	obj.SetConditions(conditions)

	return h.opts.Client.Status().Update(ctx, obj)
}
