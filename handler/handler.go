package handler

import (
	"context"
	"time"

	cachev1alpha1 "github.com/parthdhanjal/sample-controller/api/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log1 "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ownerLabelKey = "ownedby"
)

func SampleHandler(client client.Client, scheme *runtime.Scheme) SampleHandlerInterface {
	return &SampleHandlerStructType{
		client,
		scheme,
	}
}

type SampleHandlerInterface interface {
	SampleHandle(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error)
}

type SampleHandlerStructType struct {
	client.Client
	Scheme *runtime.Scheme
}

var log = log1.Log.WithName("controllers").WithName("SampleKind")

func (sh *SampleHandlerStructType) SampleHandle(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	// Create Or Delete Pods
	return sh.createOrDeletePods(ctx, instance)
}

func (sh *SampleHandlerStructType) createOrDeletePods(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	reqNumPods := int(instance.Spec.Size)
	var podList *core.PodList
	currentPods := len(podList.Items)

	if currentPods == reqNumPods {
		// Maintain Status
		log.Info("Pods match. No need to update.")
	} else if currentPods < reqNumPods {
		// Create Pods
		log.Info("Creating new pods")
		numOfPods := reqNumPods - currentPods
		for i := 0; i < numOfPods; i++ {
			prefix := instance.Name
			pod := sh.CreatePodDef(
				prefix,
				instance.Namespace,
				ownerLabelKey,
				instance.Name,
				instance.Spec.Label)
			err := sh.addPod(ctx, instance, pod)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Delete Pods
		log.Info("Deleting existing pods")
		numOfPods := currentPods - reqNumPods
		pod := &core.Pod{}
		for i := 0; i < numOfPods; i++ {
			err := sh.deletePod(ctx, pod)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return sh.success(ctx, instance)
}

func (sh *SampleHandlerStructType) CreatePodDef(prefix, namespace, ownerLabelKey, ownerLabelValue, optionalLabel string) *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix + "-",
			Namespace:    namespace,
			Labels: map[string]string{
				"mykindlabel": optionalLabel,
				ownerLabelKey: ownerLabelValue,
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            "busybox",
					Image:           "busybox",
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"sleep",
						"3600",
					},
				},
			},
		},
	}
}

func (sh *SampleHandlerStructType) addPod(ctx context.Context, instance *cachev1alpha1.SampleKind, pod *core.Pod) error {
	err := ctrl.SetControllerReference(instance, pod, sh.Scheme)
	if err != nil {
		return err
	}

	// Create pod
	err = sh.Create(ctx, pod)
	if err != nil {
		return err
	}
	log.Info("Started Pod", "name", pod.Name)
	return nil
}

func (sh *SampleHandlerStructType) deletePod(ctx context.Context, pod *core.Pod) error {
	podName := pod.Name
	log.Info("deleting pod", "name", podName)
	// reflect state in status
	err := sh.Delete(ctx, pod)
	if err != nil {
		log.Error(err, "error deleting pod")
		return err
	}

	return nil
}

func (sh *SampleHandlerStructType) success(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	instance.Status.LastUpdate = metav1.Now()
	instance.Status.Reason = "Success"
	instance.Status.Status = metav1.StatusSuccess

	updateErr := sh.Status().Update(ctx, instance)
	if updateErr != nil {
		log.Info("Error when updating Status")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, nil
}
