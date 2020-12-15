package main

import (
	"fmt"
	"strings"

	v1alpha1 "github.com/mudler/img-controller/pkg/apis/img.k8s.io/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func genGitCommand(foo *v1alpha1.ImageBuild) []string {
	switch foo.Spec.Repository.Checkout {
	case "":
		return []string{fmt.Sprintf(
			"git clone %s /repository",
			foo.Spec.Repository.Url,
		)}

	default:
		return []string{fmt.Sprintf(
			"git clone %s /repository && cd /repository && git checkout -b build %s",
			foo.Spec.Repository.Url,
			foo.Spec.Repository.Checkout,
		)}
	}

}

func genImgCommand(foo *v1alpha1.ImageBuild) []string {

	args := []string{"cd", "/repository", "img", "build", "-t", foo.Spec.ImageName}

	if foo.Spec.Dockerfile != "" {
		args = append(args, "-f", foo.Spec.Dockerfile)
	}

	context := "."
	if foo.Spec.Context != "" {
		context = foo.Spec.Context
	}

	args = append(args, context)

	if foo.Spec.RegistryCredentials.Enabled {
		args = append([]string{
			"img",
			"login",
			"-u",
			"$REGISTRY_USERNAME",
			"-p",
			"$REGISTRY_PASSWORD",
			"$REGISTRY_URI",
			"&&",
		}, args...)
		args = append(args, []string{"&&", "img", "push", foo.Spec.ImageName}...)
	}
	return []string{strings.Join(args, " ")}
}

func genEnvVars(foo *v1alpha1.ImageBuild) []corev1.EnvVar {
	envs := []corev1.EnvVar{}

	addEnvFromSecret := func(name, secretName, secretKey string) {
		envs = append(envs, corev1.EnvVar{
			Name: name,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: secretKey,
				},
			},
		})
	}

	addEnv := func(name, value string) {
		envs = append(envs, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}

	if foo.Spec.RegistryCredentials.FromSecret != "" {
		addEnvFromSecret("REGISTRY_USERNAME", foo.Spec.RegistryCredentials.FromSecret, "registryUsername")
		addEnvFromSecret("REGISTRY_PASSWORD", foo.Spec.RegistryCredentials.FromSecret, "registryPassword")
		addEnvFromSecret("REGISTRY_URI", foo.Spec.RegistryCredentials.FromSecret, "registryUri")
	} else {
		addEnv("REGISTRY_USERNAME", foo.Spec.RegistryCredentials.Username)
		addEnv("REGISTRY_PASSWORD", foo.Spec.RegistryCredentials.Password)
		addEnv("REGISTRY_URI", foo.Spec.RegistryCredentials.Registry)
	}

	return envs
}

func newWorkload(foo *v1alpha1.ImageBuild) *corev1.Pod {
	secUID := int64(1000)
	privileged := false
	serviceAccount := false
	if foo.Spec.Privileged {
		secUID = int64(0)
		privileged = true
	}
	pmount := corev1.UnmaskedProcMount

	podAnnotations := foo.Spec.Annotations
	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}
	// Needed by img
	podAnnotations["container.apparmor.security.beta.kubernetes.io/spec-build"] = "unconfined"
	podAnnotations["container.seccomp.security.alpha.kubernetes.io/spec-build"] = "unconfined"

	envs := genEnvVars(foo)
	envs = append(envs, []corev1.EnvVar{
		{
			Name:  "USER",
			Value: "img",
		},
	}...)

	cloneContainer := corev1.Container{
		ImagePullPolicy: corev1.PullIfNotPresent,

		Name:    "spec-fetch",
		Image:   "quay.io/mudler/img-controller:latest",
		Command: []string{"/bin/bash", "-cxe"},
		Args:    genGitCommand(foo),

		VolumeMounts: []corev1.VolumeMount{{
			Name:      "repository",
			MountPath: "/repository",
		}},
	}

	buildContainer := corev1.Container{
		Resources: foo.Spec.Resources,
		Env:       envs,

		SecurityContext: &corev1.SecurityContext{
			RunAsUser:  &secUID,
			ProcMount:  &pmount,
			Privileged: &privileged,
		},
		ImagePullPolicy: corev1.PullAlways,
		Name:            "spec-build",
		Image:           "quay.io/mudler/img-controller:latest", // https://github.com/genuinetools/img/issues/289#issuecomment-626501410
		Command:         []string{"/bin/bash", "-ce"},
		Args:            genImgCommand(foo),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "repository",
				MountPath: "/repository",
			},
		},
	}

	workloadPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      UUID(foo),
			Namespace: foo.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(foo, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "ImageBuild",
				}),
			},
			Annotations: podAnnotations,
			Labels:      foo.Spec.Labels,
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: &serviceAccount,
			NodeSelector:                 foo.Spec.NodeSelector,
			InitContainers: []corev1.Container{
				cloneContainer,
			},
			Containers: []corev1.Container{
				buildContainer,
			},
			SecurityContext: &corev1.PodSecurityContext{RunAsUser: &secUID},
			RestartPolicy:   corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name:         "repository",
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				},
			},
		},
	}

	return workloadPod
}
