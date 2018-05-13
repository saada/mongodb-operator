package stub

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/saada/mongodb-operator/pkg/apis/saada/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"github.com/operator-framework/operator-sdk/pkg/sdk/handler"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"
	"github.com/operator-framework/operator-sdk/pkg/sdk/types"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.MongoService:
		mongo := o

		// Ignore the delete event since the garbage collector will clean up all secondary resources for the CR
		// All secondary resources must have the CR set as their OwnerReference for this to be the case
		if event.Deleted {
			return nil
		}

		// Create the statefulset if it doesn't exist
		statefulset := statefulsetForMongo(mongo)

		err := action.Create(statefulset)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create statefulset: %v", err)
		}

		// Ensure the statefulset replicas is the same as the spec
		err = query.Get(statefulset)
		if err != nil {
			return fmt.Errorf("failed to get statefulset: %v", err)
		}
		replicas := mongo.Spec.Replicas
		if *statefulset.Spec.Replicas != replicas {
			statefulset.Spec.Replicas = &replicas
			err = action.Update(statefulset)
			if err != nil {
				return fmt.Errorf("failed to update statefulset: %v", err)
			}
		}

		// Update the Mongo status with the pod names
		podList := podList()
		labelSelector := labels.SelectorFromSet(labelsForMongo(mongo.Name)).String()
		listOps := &metav1.ListOptions{LabelSelector: labelSelector}
		err = query.List(mongo.Namespace, podList, query.WithListOptions(listOps))
		if err != nil {
			return fmt.Errorf("failed to list pods: %v", err)
		}
		podNames := getPodNames(podList.Items)
		if !reflect.DeepEqual(podNames, mongo.Status.Nodes) {
			mongo.Status.Nodes = podNames
			err := action.Update(mongo)
			if err != nil {
				return fmt.Errorf("failed to update mongo status: %v", err)
			}
		}

		// create service
		service := serviceForMongo(mongo)
		err = action.Create(service)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create service: %v", err)
		}

		// setup replicaset
		if replicas > 1 {
			var errs []string
			cmd := "/usr/bin/mongo --eval 'printjson(db.serverStatus())'"
			for _, pod := range podList.Items {
				err := execCommandInContainer(pod, cmd)
				if err != nil {
					errs = append(errs, fmt.Sprintf("Failed to exec command in container: %v", err))
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("%v", strings.Join(errs, "\n"))
			}
		}
	}

	return nil
}

// statefulsetForMongo returns a mongo StatefulSet object
func serviceForMongo(m *v1alpha1.MongoService) *v1.Service {
	ls := labelsForMongo(m.Name)

	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Port:       27017,
				TargetPort: intstr.FromInt(27017),
			}},
			ClusterIP: "None",
			Selector:  ls,
		},
	}
	addOwnerRefToObject(service, asOwner(m))
	return service
}

// statefulsetForMongo returns a mongo StatefulSet object
func statefulsetForMongo(m *v1alpha1.MongoService) *appsv1.StatefulSet {
	ls := labelsForMongo(m.Name)
	replicas := m.Spec.Replicas

	statefulSet := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						// prefer to run mongo instances on separate nodes
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{{
								Weight: 100,
								PodAffinityTerm: v1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: ls,
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							}},
						},
					},
					// Read more about this here: https://stackoverflow.com/questions/30716354/how-do-i-do-a-literal-int64-in-go
					TerminationGracePeriodSeconds: &(&struct{ x int64 }{15}).x,
					Containers: []v1.Container{{
						Image:   "mongo:3.6.4",
						Name:    "mongo",
						Command: []string{"mongod", "--bind_ip", "0.0.0.0", "--replSet", m.Name, "--smallfiles", "--noprealloc"},
						Ports: []v1.ContainerPort{{
							ContainerPort: 27017,
							Name:          "mongo",
						}},
					}},
				},
			},
			// VolumeClaimTemplates: []v1.PersistentVolumeClaim{{
			// 	ObjectMeta: metav1.ObjectMeta{
			// 		Labels: ls,
			// 	},
			// }}
		},
	}
	addOwnerRefToObject(statefulSet, asOwner(m))
	return statefulSet
}

// labelsForMongo returns the labels for selecting the resources
// belonging to the given mongo CR name.
func labelsForMongo(name string) map[string]string {
	return map[string]string{"app": "mongo", "mongo_cr": name}
}

// addOwnerRefToObject appends the desired OwnerReference to the object
func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}

// asOwner returns an OwnerReference set as the mongo CR
func asOwner(m *v1alpha1.MongoService) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}

// podList returns a v1.PodList object
func podList() *v1.PodList {
	return &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []v1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func execCommandInContainer(pod v1.Pod, cmd ...string) error {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %v", err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %v", err)
	}
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")
	req.VersionedParams(&v1.PodExecOptions{
		Container: pod.Spec.Containers[0].Name,
		Command:   cmd,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to exec: %v", err)
	}

	var (
		stdOut bytes.Buffer
		stdErr bytes.Buffer
	)

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdOut,
		Stderr: &stdErr,
	})

	fmt.Printf("pod name: %s container name: %s\n", pod.Name, pod.Spec.Containers[0].Name)
	fmt.Printf("stdout: %s\n", stdOut.String())
	fmt.Printf("stderr: %s\n", stdErr.String())
	if err != nil {
		return fmt.Errorf("could not execute: %v", err)
	}

	return nil
}
