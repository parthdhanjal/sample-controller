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
	log1 "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ownerLabelKey = "ownedby"
	finalizer     = "grp.sample.com/finalizer"
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

	// Initialize CR
	log.Info("Initializing")
	res, err := sh.Initialize(ctx, instance)
	if err != nil {
		return res, err
	}

	// Create Or Delete Pods
	return sh.createOrDeletePods(ctx, instance)
}

func (sh *SampleHandlerStructType) Initialize(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	oldSpec := instance.DeepCopy().Spec
	if instance.Spec.Label == "" {
		instance.Spec.Label = "sample"
	}
	if !reflect.DeepEqual(oldSpec, instance.Spec) {
		log.Info("Updating Spec to new spec")
		err := sh.Update(ctx, instance)
		if err != nil {
			instance.Status.LastUpdate = metav1.Now()
			instance.Status.Reason = err.Error()
			instance.Status.Status = metav1.StatusFailure

			updateErr := sh.Status().Update(ctx, instance)
			if updateErr != nil {
				log.Info("Error when updating status", updateErr)
				return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
			}
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (sh *SampleHandlerStructType) createOrDeletePods(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	reqNumPods := int(instance.Spec.Size)
	currentPods := 0
	var podList *core.PodList
	var err error

	fmt.Print("------instance------", instance)
	if instance.Status.Pods == nil {
		instance.Status.Pods = make(map[string]core.PodPhase)
	} else {
		podList, err = sh.getRunningPodsOwnedBy(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		currentPods = len(podList.Items)
	}

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
		for _, pod := range podList.Items {
			if numOfPods > 0 {
				err = sh.deletePod(ctx, instance, pod.Name)
				if err != nil {
					return ctrl.Result{}, err
				}
				numOfPods -= 1
			} else {
				// no deletions needed
				break
			}
		}
	}

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
	existingPod := &core.Pod{}
	err := sh.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, existingPod)

	// Pod exists
	if err == nil {
		log.Info("Pod already exists", pod.Name)
		// reflect state in status
		instance.Status.Pods[pod.Name] = pod.Status.Phase
		return nil
	}

	// Creating Pod
	err = ctrl.SetControllerReference(instance, pod, sh.Scheme)
	if err != nil {
		return err
	}

	// Create pod
	err = sh.Create(ctx, pod)
	if err != nil {
		return err
	}
	log.Info("Pod ", pod.Name, " Created")

	// reflect state in status
	instance.Status.Pods[pod.Name] = pod.Status.Phase
	return nil
}

func (sh *SampleHandlerStructType) deletePod(ctx context.Context, instance *cachev1alpha1.SampleKind, podName string) error {
	// Check if pod exists
	existingPod := &core.Pod{}
	err := sh.Get(ctx, types.NamespacedName{Name: podName, Namespace: instance.Namespace}, existingPod)
	if err != nil && !errors.IsNotFound(err) {
		// Pod exists but there is a problem
		return err
	}

	// Pod exists
	if err == nil {
		log.Info("Deleting Pod", podName)
		// reflect state in status
		err = sh.Delete(ctx, existingPod)
		if err != nil {
			log.Error(err, "error deleting pod")
			return err
		}
		instance.Status.Pods[podName] = existingPod.Status.Phase
		return nil
	}

	// Pod does not exist
	delete(instance.Status.Pods, podName)
	return nil
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
