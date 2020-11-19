package main

import (
	"fmt"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

type requests struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Path string

	Requests corev1.ResourceList
}

func main() {
	if len(os.Args) < 2 {
		log.Printf("Usage: requests [FILE|DIRECTORY]")
		return
	}

	var lists []corev1.ResourceList
	filepath.Walk(os.Args[1], func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		reqs, err := parseFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse %q: %v", path, err)
		}
		for _, req := range reqs {
			log.Printf(
				"Kind: %s, Object: %s/%s CPU: %s, Memory: %dM\n",
				req.Kind, req.Namespace, req.Name, req.Requests.Cpu(),
				req.Requests.Memory().ScaledValue(resource.Mega))
			lists = append(lists, req.Requests)
		}

		return nil
	})

	total := SumOfResourceList(lists)
	log.Printf("Total\n\tCPU: %s\n\tMemory: %dM\n", total.Cpu(), total.Memory().ScaledValue(resource.Mega))
}

func parseFile(path string) ([]requests, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}

	objs := strings.Split(string(bytes), "---")

	var reqs []requests
	for _, obj := range objs {
		var req requests
		err = yaml.Unmarshal([]byte(obj), &req)
		if err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %v. Object:\n %s", err, obj)
		}
		switch req.Kind {
		case "Deployment":
			var dep appsv1.Deployment
			err = yaml.Unmarshal([]byte(obj), &dep)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Deployment: %v. Object:\n %s", err, obj)
			}
			replicas := 1
			if dep.Spec.Replicas != nil {
				replicas = int(*dep.Spec.Replicas)
			}
			req.Requests = MultiplyResourceList(RequestsFromPodSpec(&dep.Spec.Template.Spec), replicas)
			break
		case "Pod":
			var pod corev1.Pod
			err = yaml.Unmarshal([]byte(obj), &pod)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Pod: %v. Object:\n %s", err, obj)
			}
			req.Requests = RequestsFromPodSpec(&pod.Spec)
			break
		case "Job":
			var job batchv1.Job
			err = yaml.Unmarshal([]byte(obj), &job)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Job: %v. Object:\n %s", err, obj)
			}
			parallelism := 1
			if job.Spec.Parallelism != nil {
				parallelism = int(*job.Spec.Parallelism)
			}
			req.Requests = MultiplyResourceList(RequestsFromPodSpec(&job.Spec.Template.Spec), parallelism)
			break
		default:
			continue
		}

		if len(req.Requests) == 0 {
			continue
		}
		req.Path = path
		reqs = append(reqs, req)
	}
	return reqs, nil
}

// RequestsFromPodSpec returns a ResourceList representing the sum of all the
// resource requests specified in the PodSpec.
func RequestsFromPodSpec(spec *corev1.PodSpec) corev1.ResourceList {
	var requests []corev1.ResourceList
	for _, container := range spec.Containers {
		if container.Resources.Requests != nil {
			requests = append(requests, container.Resources.Requests)
		}
	}

	return SumOfResourceList(requests)
}

// SumOfResourceList returns a ResourceList where each ResourceName is the sum
// of the same ResourceName for each provided ResourceList.
func SumOfResourceList(lists []corev1.ResourceList) corev1.ResourceList {
	tmp := make(map[corev1.ResourceName]*resource.Quantity)
	for _, list := range lists {
		for k, v := range list {
			if _, ok := tmp[k]; ok {
				tmp[k].Add(v)
			} else {
				c := v.DeepCopy()
				tmp[k] = &c
			}
		}
	}

	out := make(corev1.ResourceList)
	for k, v := range tmp {
		out[k] = *v
	}
	return out
}

// MultiplyResourceList multiplies a ResourceList by the provided factor.
func MultiplyResourceList(list corev1.ResourceList, factor int) corev1.ResourceList {
	out := make(corev1.ResourceList, len(list))
	for k, v := range list {
		acc := resource.Quantity{}
		// TODO: is this necessary?
		for i := 0; i < factor; i++ {
			acc.Add(v)
		}
		out[k] = acc
	}
	return out
}
