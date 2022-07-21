package handler

import (
	"context"

	cachev1alpha1 "github.com/parthdhanjal/sample-controller/api/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ownerLabelKey = "ownedby"
)

type SampleHandler struct {
	client.Client
	Scheme *runtime.Scheme
}

var log = log.log.WithName("controllers").WithName("SampleKind")

func (sh *SampleHandler) createOrUpdateReconciler(ctx context.Context, instance *cachev1alpha1.SampleKind) (ctrl.Result, error) {
	reqNumPods := instance.Spec.size
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
				instance.Spec.label)
			err := sh.addPod(ctx, instance, pod)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Delete Pods
		log.Info("Deleting existing pods")
		numOfPods := currentPods - reqNumPods
		for i := 0; i < numOfPods; i++ {
			err := sh.deletePod(ctx, instance, pod)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
}

func (sh *SampleHandler) CreatePodDef(prefix, namespace, ownerLabelKey, ownerLabelValue, optionalLabel string) *core.Pod {
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

func (sh *SampleHandler) addPod(ctx context.Context, instance *cachev1alpha1, pod *core.pod) error {
	err = ctrl.SetControllerReference(instance, pod, sh.Scheme)
	if err != nil {
		return err
	}

	// Create pod
	err = sh.Create(ctx, pod)
	if err != nil {
		return err
	}
	log.Info("Started Pod", "name", pod.Name)
}

func (sh *SampleHandler) deletePod(ctx context.Context, instance *cachev1alpha1.Mykind, pod *core.pod) error {
	podName := pod.Name
	log.Info("deleting pod", "name", podName)
	// reflect state in status
	err = sh.Delete(ctx, pod)
	if err != nil {
		log.Error(err, "error deleting pod")
		return err
	}

	return nil
}
