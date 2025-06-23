package mergeutil

import corev1 "k8s.io/api/core/v1"

// MergeStrategy defines how to merge two values
type MergeStrategy[T any] func(base, overlay T) T

// mergeByKey merges two slices using a key extractor and merge strategy
func mergeByKey[T any, K comparable](base, overlay []T, keyFn func(T) K, mergeFn MergeStrategy[T]) []T {
	itemMap := make(map[K]T)
	for _, item := range base {
		itemMap[keyFn(item)] = item
	}
	for _, item := range overlay {
		key := keyFn(item)
		if baseItem, exists := itemMap[key]; exists {
			itemMap[key] = mergeFn(baseItem, item)
		} else {
			itemMap[key] = item
		}
	}

	result := make([]T, 0, len(itemMap))
	for _, item := range itemMap {
		result = append(result, item)
	}
	return result
}

// mergeEnvVars merges EnvVar slices by name
func mergeEnvVars(base, overlay []corev1.EnvVar) []corev1.EnvVar {
	return mergeByKey(base, overlay,
		func(e corev1.EnvVar) string { return e.Name },
		func(base, overlay corev1.EnvVar) corev1.EnvVar { return overlay })
}

// mergeVolumeMounts merges VolumeMounts by MountPath
func mergeVolumeMounts(base, overlay []corev1.VolumeMount) []corev1.VolumeMount {
	return mergeByKey(base, overlay,
		func(v corev1.VolumeMount) string { return v.MountPath },
		func(base, overlay corev1.VolumeMount) corev1.VolumeMount { return overlay })
}

// mergeContainerPorts merges ContainerPort slices by port number
func mergeContainerPorts(base, overlay []corev1.ContainerPort) []corev1.ContainerPort {
	return mergeByKey(base, overlay,
		func(p corev1.ContainerPort) int32 { return p.ContainerPort },
		func(base, overlay corev1.ContainerPort) corev1.ContainerPort { return overlay })
}

// mergeContainer merges overlay into base container
func mergeContainer(base, overlay corev1.Container) corev1.Container {
	merged := base

	// Override non-empty scalar fields
	if overlay.Image != "" {
		merged.Image = overlay.Image
	}
	if len(overlay.Command) > 0 {
		merged.Command = overlay.Command
	}
	if len(overlay.Args) > 0 {
		merged.Args = overlay.Args
	}

	// Merge complex fields
	merged.Env = mergeEnvVars(base.Env, overlay.Env)
	merged.VolumeMounts = mergeVolumeMounts(base.VolumeMounts, overlay.VolumeMounts)
	merged.Ports = mergeContainerPorts(base.Ports, overlay.Ports)

	// Override non-nil pointer fields
	if overlay.Resources.Requests != nil || overlay.Resources.Limits != nil {
		merged.Resources = overlay.Resources
	}
	if overlay.SecurityContext != nil {
		merged.SecurityContext = overlay.SecurityContext
	}
	if overlay.ReadinessProbe != nil {
		merged.ReadinessProbe = overlay.ReadinessProbe
	}
	if overlay.LivenessProbe != nil {
		merged.LivenessProbe = overlay.LivenessProbe
	}
	if overlay.StartupProbe != nil {
		merged.StartupProbe = overlay.StartupProbe
	}
	if overlay.Lifecycle != nil {
		merged.Lifecycle = overlay.Lifecycle
	}

	return merged
}

// mergeContainers merges container slices by name
func mergeContainers(base, overlay []corev1.Container) []corev1.Container {
	return mergeByKey(base, overlay, func(c corev1.Container) string { return c.Name }, mergeContainer)
}

// MergePodSpecs merges overlay into base PodSpec
func MergePodSpecs(base, overlay *corev1.PodSpec) *corev1.PodSpec {
	result := base.DeepCopy()

	// Merge container lists
	result.Containers = mergeContainers(base.Containers, overlay.Containers)
	result.InitContainers = mergeContainers(base.InitContainers, overlay.InitContainers)

	// Append overlay volumes
	if len(overlay.Volumes) > 0 {
		result.Volumes = append(result.Volumes, overlay.Volumes...)
	}

	// Initialize and merge NodeSelector
	if result.NodeSelector == nil {
		result.NodeSelector = make(map[string]string)
	}
	for k, v := range overlay.NodeSelector {
		result.NodeSelector[k] = v
	}

	// Append tolerations
	if len(overlay.Tolerations) > 0 {
		result.Tolerations = append(result.Tolerations, overlay.Tolerations...)
	}

	// Override non-nil/non-empty fields
	if overlay.Affinity != nil {
		result.Affinity = overlay.Affinity
	}
	if overlay.SecurityContext != nil {
		result.SecurityContext = overlay.SecurityContext
	}
	if overlay.DNSPolicy != "" {
		result.DNSPolicy = overlay.DNSPolicy
	}
	if overlay.RestartPolicy != "" {
		result.RestartPolicy = overlay.RestartPolicy
	}
	if overlay.ServiceAccountName != "" {
		result.ServiceAccountName = overlay.ServiceAccountName
	}
	if overlay.Hostname != "" {
		result.Hostname = overlay.Hostname
	}
	if overlay.Priority != nil {
		result.Priority = overlay.Priority
	}
	if overlay.SchedulerName != "" {
		result.SchedulerName = overlay.SchedulerName
	}
	if overlay.TerminationGracePeriodSeconds != nil {
		result.TerminationGracePeriodSeconds = overlay.TerminationGracePeriodSeconds
	}
	result.HostNetwork = result.HostNetwork || overlay.HostNetwork

	return result
}
