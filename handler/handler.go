package handler

import (
	"context"
	"fmt"
	"reflect"
	"time"

	cachev1alpha1 "github.com/parthdhanjal/sample-controller/api/v1alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	log1 "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ownerLabelKey = "ownedby"
	finalizer     = "grp.example.com/finalizer"
)

func SampleHandler(client client.Client, scheme *runtime.Scheme, cache cache.Cache) SampleHandlerInterface {
	return &SampleHandlerStructType{
		client,
		scheme,
		cache,
	}
}

type SampleHandlerInterface interface {
	SampleHandle(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error)
}

type SampleHandlerStructType struct {
	client.Client
	Scheme *runtime.Scheme
	Cache  cache.Cache
}

var log = log1.Log.WithName("controllers").WithName("SampleKind")

func (sh *SampleHandlerStructType) SampleHandle(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	// Delete Resource based on TimeStamp
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		return sh.deletionReconciler(ctx, instance)
	}

	// Initialize resource
	log.Info("Initializing")
	res, err := sh.initialize(ctx, instance)
	if err != nil {
		return res, err
	}
	// Create Or Delete Pods
	return sh.createOrDeletePods(ctx, instance)
}

// Reconciliation of deleted resources
func (sh *SampleHandlerStructType) deletionReconciler(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	// Remove Finalizer if exists
	if controllerutil.ContainsFinalizer(instance, finalizer) {
		controllerutil.RemoveFinalizer(instance, finalizer)
		err := sh.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Stop reconciliation as the item is being deleted
	return ctrl.Result{}, nil
}

func (sh *SampleHandlerStructType) initialize(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	// Convert Instance in case conversion webhook isn't enabled
	err := instance.ConvertTo()
	if err != nil {
		return sh.failure(ctx, instance, err)
	}

	// Mutate Instance in case mutating webhook isn't enabled
	oldSpec := instance.DeepCopy().Spec
	instance.Default()
	// check if mutation changed anything
	if !reflect.DeepEqual(oldSpec, instance.Spec) {
		log.Info("mutating")
		if err := sh.Update(ctx, instance); err != nil {
			return sh.failure(ctx, instance, err)
		}
	}

	// Validate Instance in case validating webhook isn't enabled
	// validate creation
	if err := instance.ValidateCreate(); err != nil {
		return sh.failure(ctx, instance, err)
	}
	return ctrl.Result{}, nil
}

func (sh *SampleHandlerStructType) createOrDeletePods(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	reqNumPods := int(instance.Spec.Size)

	//var podList *core.PodList
	// Get list from GET request

	podList, err := sh.getRunningPodsOwnedBy(ctx, instance)
	currentPods := int(len(podList.Items))
	fmt.Print("----err----", err)
	fmt.Print("----reqNumPods----", reqNumPods)
	fmt.Print("----current----", currentPods)

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
			err := sh.deletePod(ctx, instance, pod)
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
				"label":       optionalLabel,
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

func (sh *SampleHandlerStructType) deletePod(ctx context.Context, instance *cachev1alpha1.SampleKind, pod *core.Pod) error {
	existingPod := &core.Pod{}
	podName := pod.Name
	err := sh.Get(ctx, types.NamespacedName{Name: podName, Namespace: instance.Namespace}, existingPod)
	if err != nil && !errors.IsNotFound(err) {
		// Pod exists but there is a problem
		return err
	}
	log.Info("deleting pod", "name", podName)
	// reflect state in status
	if err == nil {
		err := sh.Delete(ctx, existingPod)
		if err != nil {
			log.Error(err, "error deleting pod")
			return err
		}
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

func (sh *SampleHandlerStructType) failure(ctx context.Context, instance *cachev1alpha1.SampleKind, err error) (ctrl.Result, error) {
	instance.Status.LastUpdate = metav1.Now()
	instance.Status.Reason = err.Error()
	instance.Status.Status = metav1.StatusFailure

	updateErr := sh.Status().Update(ctx, instance)
	if updateErr != nil {
		log.Info("Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, err
}

func (sh *SampleHandlerStructType) getRunningPodsOwnedBy(ctx context.Context, instance *cachev1alpha1.SampleKind) (*core.PodList, error) {
	podList := &core.PodList{}
	runningPodList := &core.PodList{}
	ownerReq, _ := labels.NewRequirement(ownerLabelKey, selection.Equals, []string{instance.Name})
	listOptions := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(*ownerReq),
		Namespace:     instance.Namespace,
	}
	err := sh.List(ctx, podList, listOptions)
	if err != nil {
		log.Error(err, "error listing pods")
		return nil, err
	}
	// only count pods not scheduled for deletion
	for _, pod := range podList.Items {
		if pod.ObjectMeta.DeletionTimestamp.IsZero() {
			runningPodList.Items = append(runningPodList.Items, pod)
		}
	}
	return runningPodList, nil
}
