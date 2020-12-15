package main

import (
	"context"
	"fmt"
	"strings"

	v1alpha1 "github.com/mudler/img-controller/pkg/apis/img.k8s.io/v1alpha1"
	v1 "github.com/mudler/img-controller/pkg/generated/controllers/core/v1"
	v1alpha1controller "github.com/mudler/img-controller/pkg/generated/controllers/img.k8s.io/v1alpha1"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

const controllerAgentName = "img-controller"

const (
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by img-controller"
)

// Handler is the controller implementation for Foo resources
type Handler struct {
	pods            v1.PodClient
	podsCache       v1.PodCache
	controller      v1alpha1controller.ImageBuildController
	controllerCache v1alpha1controller.ImageBuildCache
	recorder        record.EventRecorder
}

// NewController returns a new sample controller
func Register(
	ctx context.Context,
	events typedcorev1.EventInterface,
	pods v1.PodController,
	ImageBuild v1alpha1controller.ImageBuildController) {

	controller := &Handler{
		pods:            pods,
		podsCache:       pods.Cache(),
		controller:      ImageBuild,
		controllerCache: ImageBuild.Cache(),
		recorder:        buildEventRecorder(events),
	}

	// Register handlers
	pods.OnChange(ctx, "img-handler", controller.OnPodChanged)
	ImageBuild.OnChange(ctx, "img-handler", controller.OnImageBuildChanged)
}

func buildEventRecorder(events typedcorev1.EventInterface) record.EventRecorder {
	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))
	logrus.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logrus.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: events})
	return eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
}

func cleanString(s string) string {
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ToLower(s)
	return s
}

func UUID(foo *v1alpha1.ImageBuild) string {
	return fmt.Sprintf("%s", foo.Name)
	//	return cleanString(fmt.Sprintf("%s-%s", foo.Spec.PackageName, foo.Spec.Repository))
}

func (h *Handler) OnImageBuildChanged(key string, foo *v1alpha1.ImageBuild) (*v1alpha1.ImageBuild, error) {
	// foo will be nil if key is deleted from cache
	if foo == nil {
		return nil, nil
	}
	logrus.Infof("Reconciling '%s' ", foo.Name)

	imageName := foo.Spec.ImageName
	if imageName == "" {
		utilruntime.HandleError(fmt.Errorf("%s: image name must be specified", key))
		return nil, nil
	}

	repository := foo.Spec.Repository.Url
	if repository == "" {
		utilruntime.HandleError(fmt.Errorf("%s: repository url must be specified", key))
		return nil, nil
	}

	logrus.Infof("Getting pod for '%s' ", foo.Name)

	deployment, err := h.podsCache.Get(foo.Namespace, UUID(foo))
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		logrus.Infof("Pod not found for '%s' ", foo.Name)

		deployment, err = h.pods.Create(newWorkload(foo))
	}
	if err != nil {
		return nil, err
	}

	if !metav1.IsControlledBy(deployment, foo) {
		msg := fmt.Sprintf(MessageResourceExists, foo.Spec.ImageName)
		logrus.Infof(msg)

		h.recorder.Event(foo, corev1.EventTypeWarning, ErrResourceExists, msg)
		return nil, nil
	}

	err = h.updateStatus(foo, deployment)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (h *Handler) updateStatus(foo *v1alpha1.ImageBuild, pod *corev1.Pod) error {
	fooCopy := foo.DeepCopy()
	fooCopy.Status.State = string(pod.Status.Phase)
	logrus.Infof("ImageBuild '%s' has status '%s'", foo.Name, fooCopy.Status.State)
	_, err := h.controller.UpdateStatus(fooCopy)
	return err
}

func (h *Handler) OnPodChanged(key string, pod *corev1.Pod) (*corev1.Pod, error) {
	if pod == nil {
		return nil, nil
	}
	logrus.Infof("Pod '%s' has changed", pod.Name)

	if ownerRef := metav1.GetControllerOf(pod); ownerRef != nil {
		if ownerRef.Kind != "ImageBuild" {
			logrus.Infof("Pod '%s' not owned by ImageBuild", pod.Name)

			return nil, nil
		}

		foo, err := h.podsCache.Get(pod.Namespace, pod.Name)
		if err != nil {
			logrus.Infof("ignoring orphaned object '%s' of ImageBuild '%s'", pod.GetSelfLink(), ownerRef.Name)
			return nil, nil
		}
		logrus.Infof("Enqueueing reconcile for %s", pod.Name)

		h.controller.Enqueue(foo.Namespace, foo.Name)
		return nil, nil
	}

	return nil, nil
}
