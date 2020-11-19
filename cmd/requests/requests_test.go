package main

import (
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestParseFile(t *testing.T) {
	cases := []struct {
		name string
		path string
		want []requests
	}{{
		name: "Ignored object",
		path: "testdata/ignored-object.yaml",
		want: nil,
	}, {
		name: "Single deployment",
		path: "testdata/single-deployment.yaml",
		want: []requests{{
			TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-name",
				Namespace: "some-namespace",
			},
			Path: "testdata/single-deployment.yaml",
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
		}},
	}, {
		name: "Single pod",
		path: "testdata/single-pod.yaml",
		want: []requests{{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "core/v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-name",
				Namespace: "some-namespace",
			},
			Path: "testdata/single-pod.yaml",
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("30m"),
				corev1.ResourceMemory: resource.MustParse("40Mi"),
			},
		}},
	}, {
		name: "Single Job",
		path: "testdata/single-job.yaml",
		want: []requests{{
			TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-name",
				Namespace: "some-namespace",
			},
			Path: "testdata/single-job.yaml",
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("1000Mi"),
			},
		}},
	}, {
		name: "Multiple Objects",
		path: "testdata/multiple-objects.yaml",
		want: []requests{{
			TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-name",
				Namespace: "some-namespace",
			},
			Path: "testdata/multiple-objects.yaml",
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
		}, {
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "core/v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-name",
				Namespace: "some-namespace",
			},
			Path: "testdata/multiple-objects.yaml",
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("30m"),
				corev1.ResourceMemory: resource.MustParse("40Mi"),
			},
		}},
	}, {
		name: "Different units",
		path: "testdata/different-units.yaml",
		want: []requests{{
			TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-name",
				Namespace: "some-namespace",
			},
			Path: "testdata/different-units.yaml",
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    sumOfquantities(resource.MustParse("10m"), resource.MustParse("1")),
				corev1.ResourceMemory: sumOfquantities(resource.MustParse("100Mi"), resource.MustParse("1G")),
			},
		}},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseFile(c.path)
			if err != nil {
				t.Errorf("parseFile failed: %v", err)
			} else if !cmp.Equal(c.want, got) {
				t.Errorf("Unexpected output of parseFile, (-want, +got):\n%s", cmp.Diff(c.want, got))
			}
		})
	}
}

func TestSumOfResourceList(t *testing.T) {
	cases := []struct {
		name string
		list []corev1.ResourceList
		want corev1.ResourceList
	}{{
		name: "Empty list",
		list: []corev1.ResourceList{},
		want: corev1.ResourceList{},
	}, {
		name: "Single entry",
		list: []corev1.ResourceList{{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		}},
		want: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}, {
		name: "Multiple entries",
		list: []corev1.ResourceList{{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1000Mi"),
		}, {
			corev1.ResourceCPU: resource.MustParse("100m"),
		}, {
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		}},
		want: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("1100Mi"),
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SumOfResourceList(c.list)
			if !cmp.Equal(c.want, got) {
				t.Errorf("Unexpected output of SumOfResourceList, (-want, +got):\n%s", cmp.Diff(c.want, got))
			}
		})
	}
}
func TestMultiplyResourceList(t *testing.T) {
	cases := []struct {
		name   string
		list   corev1.ResourceList
		factor int
		want   corev1.ResourceList
	}{{
		name: "Multiply by 1",
		list: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		factor: 1,
		want: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}, {
		name: "Multiply by 5",
		list: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		factor: 5,
		want: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MultiplyResourceList(c.list, c.factor)
			if !cmp.Equal(c.want, got) {
				t.Errorf("Unexpected output of MultiplyResourceList, (-want, +got):\n%s", cmp.Diff(c.want, got))
			}
		})
	}
}

func sumOfquantities(quantities ...resource.Quantity) resource.Quantity {
	acc := resource.Quantity{}
	for _, q := range quantities {
		acc.Add(q)
	}
	return acc
}
