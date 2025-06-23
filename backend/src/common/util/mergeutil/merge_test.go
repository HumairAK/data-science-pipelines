package mergeutil

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Test for mergeByKey generic function
func TestMergeByKey(t *testing.T) {
	type testItem struct {
		ID   string
		Data string
	}

	tests := []struct {
		name     string
		base     []testItem
		overlay  []testItem
		expected []testItem
	}{
		{
			name: "merge with overlapping items",
			base: []testItem{
				{ID: "1", Data: "base1"},
				{ID: "2", Data: "base2"},
			},
			overlay: []testItem{
				{ID: "2", Data: "overlay2"},
				{ID: "3", Data: "overlay3"},
			},
			expected: []testItem{
				{ID: "1", Data: "base1"},
				{ID: "2", Data: "overlay2"},
				{ID: "3", Data: "overlay3"},
			},
		},
		{
			name:     "empty base",
			base:     []testItem{},
			overlay:  []testItem{{ID: "1", Data: "overlay1"}},
			expected: []testItem{{ID: "1", Data: "overlay1"}},
		},
		{
			name:     "empty overlay",
			base:     []testItem{{ID: "1", Data: "base1"}},
			overlay:  []testItem{},
			expected: []testItem{{ID: "1", Data: "base1"}},
		},
		{
			name:     "both empty",
			base:     []testItem{},
			overlay:  []testItem{},
			expected: []testItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeByKey(tt.base, tt.overlay,
				func(item testItem) string { return item.ID },
				func(base, overlay testItem) testItem { return overlay })

			// Sort the results for consistent comparison
			expected := make(map[string]bool)
			for _, item := range tt.expected {
				expected[item.ID] = true
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
			}

			for _, item := range result {
				found := false
				for _, expectedItem := range tt.expected {
					if item.ID == expectedItem.ID && item.Data == expectedItem.Data {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected item in result: %+v", item)
				}
			}
		})
	}
}

// Test for mergeEnvVars function
func TestMergeEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		base     []corev1.EnvVar
		overlay  []corev1.EnvVar
		expected []corev1.EnvVar
	}{
		{
			name: "merge with overlapping env vars",
			base: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
			overlay: []corev1.EnvVar{
				{Name: "VAR2", Value: "new-value2"},
				{Name: "VAR3", Value: "value3"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "new-value2"},
				{Name: "VAR3", Value: "value3"},
			},
		},
		{
			name:     "empty base",
			base:     []corev1.EnvVar{},
			overlay:  []corev1.EnvVar{{Name: "VAR1", Value: "value1"}},
			expected: []corev1.EnvVar{{Name: "VAR1", Value: "value1"}},
		},
		{
			name:     "empty overlay",
			base:     []corev1.EnvVar{{Name: "VAR1", Value: "value1"}},
			overlay:  []corev1.EnvVar{},
			expected: []corev1.EnvVar{{Name: "VAR1", Value: "value1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeEnvVars(tt.base, tt.overlay)

			// Create maps for easier comparison
			resultMap := make(map[string]string)
			for _, env := range result {
				resultMap[env.Name] = env.Value
			}

			expectedMap := make(map[string]string)
			for _, env := range tt.expected {
				expectedMap[env.Name] = env.Value
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d env vars, got %d", len(tt.expected), len(result))
			}

			for name, value := range expectedMap {
				if resultMap[name] != value {
					t.Errorf("Expected env var %s=%s, got %s", name, value, resultMap[name])
				}
			}
		})
	}
}

// Test for mergeVolumeMounts function
func TestMergeVolumeMounts(t *testing.T) {
	tests := []struct {
		name     string
		base     []corev1.VolumeMount
		overlay  []corev1.VolumeMount
		expected []corev1.VolumeMount
	}{
		{
			name: "merge with overlapping volume mounts",
			base: []corev1.VolumeMount{
				{Name: "vol1", MountPath: "/path1"},
				{Name: "vol2", MountPath: "/path2"},
			},
			overlay: []corev1.VolumeMount{
				{Name: "vol3", MountPath: "/path2"},
				{Name: "vol4", MountPath: "/path3"},
			},
			expected: []corev1.VolumeMount{
				{Name: "vol1", MountPath: "/path1"},
				{Name: "vol3", MountPath: "/path2"},
				{Name: "vol4", MountPath: "/path3"},
			},
		},
		{
			name:     "empty base",
			base:     []corev1.VolumeMount{},
			overlay:  []corev1.VolumeMount{{Name: "vol1", MountPath: "/path1"}},
			expected: []corev1.VolumeMount{{Name: "vol1", MountPath: "/path1"}},
		},
		{
			name:     "empty overlay",
			base:     []corev1.VolumeMount{{Name: "vol1", MountPath: "/path1"}},
			overlay:  []corev1.VolumeMount{},
			expected: []corev1.VolumeMount{{Name: "vol1", MountPath: "/path1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeVolumeMounts(tt.base, tt.overlay)

			// Create maps for easier comparison
			resultMap := make(map[string]string)
			for _, vm := range result {
				resultMap[vm.MountPath] = vm.Name
			}

			expectedMap := make(map[string]string)
			for _, vm := range tt.expected {
				expectedMap[vm.MountPath] = vm.Name
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d volume mounts, got %d", len(tt.expected), len(result))
			}

			for path, name := range expectedMap {
				if resultMap[path] != name {
					t.Errorf("Expected volume mount %s at path %s, got %s", name, path, resultMap[path])
				}
			}
		})
	}
}

// Test for mergeContainerPorts function
func TestMergeContainerPorts(t *testing.T) {
	tests := []struct {
		name     string
		base     []corev1.ContainerPort
		overlay  []corev1.ContainerPort
		expected []corev1.ContainerPort
	}{
		{
			name: "merge with overlapping ports",
			base: []corev1.ContainerPort{
				{ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
				{ContainerPort: 8443, Protocol: corev1.ProtocolTCP},
			},
			overlay: []corev1.ContainerPort{
				{ContainerPort: 8080, Protocol: corev1.ProtocolUDP},
				{ContainerPort: 9090, Protocol: corev1.ProtocolTCP},
			},
			expected: []corev1.ContainerPort{
				{ContainerPort: 8080, Protocol: corev1.ProtocolUDP},
				{ContainerPort: 8443, Protocol: corev1.ProtocolTCP},
				{ContainerPort: 9090, Protocol: corev1.ProtocolTCP},
			},
		},
		{
			name:     "empty base",
			base:     []corev1.ContainerPort{},
			overlay:  []corev1.ContainerPort{{ContainerPort: 8080}},
			expected: []corev1.ContainerPort{{ContainerPort: 8080}},
		},
		{
			name:     "empty overlay",
			base:     []corev1.ContainerPort{{ContainerPort: 8080}},
			overlay:  []corev1.ContainerPort{},
			expected: []corev1.ContainerPort{{ContainerPort: 8080}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeContainerPorts(tt.base, tt.overlay)

			// Create maps for easier comparison
			resultMap := make(map[int32]corev1.Protocol)
			for _, port := range result {
				resultMap[port.ContainerPort] = port.Protocol
			}

			expectedMap := make(map[int32]corev1.Protocol)
			for _, port := range tt.expected {
				expectedMap[port.ContainerPort] = port.Protocol
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d ports, got %d", len(tt.expected), len(result))
			}

			for portNum, protocol := range expectedMap {
				if resultMap[portNum] != protocol {
					t.Errorf("Expected port %d with protocol %s, got protocol %s",
						portNum, protocol, resultMap[portNum])
				}
			}
		})
	}
}

// Test for mergeContainer function
func TestMergeContainer(t *testing.T) {
	tests := []struct {
		name     string
		base     corev1.Container
		overlay  corev1.Container
		expected corev1.Container
	}{
		{
			name: "merge with overlay values",
			base: corev1.Container{
				Name:  "container1",
				Image: "base-image:v1",
				Env: []corev1.EnvVar{
					{Name: "VAR1", Value: "value1"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "vol1", MountPath: "/path1"},
				},
			},
			overlay: corev1.Container{
				Name:    "container1",
				Image:   "overlay-image:v2",
				Command: []string{"/bin/sh"},
				Args:    []string{"-c", "echo hello"},
				Env: []corev1.EnvVar{
					{Name: "VAR2", Value: "value2"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "vol2", MountPath: "/path2"},
				},
				SecurityContext: &corev1.SecurityContext{
					RunAsUser: ptr(int64(1000)),
				},
			},
			expected: corev1.Container{
				Name:    "container1",
				Image:   "overlay-image:v2",
				Command: []string{"/bin/sh"},
				Args:    []string{"-c", "echo hello"},
				Env: []corev1.EnvVar{
					{Name: "VAR1", Value: "value1"},
					{Name: "VAR2", Value: "value2"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "vol1", MountPath: "/path1"},
					{Name: "vol2", MountPath: "/path2"},
				},
				SecurityContext: &corev1.SecurityContext{
					RunAsUser: ptr(int64(1000)),
				},
			},
		},
		{
			name: "merge with resources",
			base: corev1.Container{
				Name: "container1",
			},
			overlay: corev1.Container{
				Name: "container1",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					},
				},
			},
			expected: corev1.Container{
				Name: "container1",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeContainer(tt.base, tt.overlay)

			// Check scalar fields
			if result.Image != tt.expected.Image {
				t.Errorf("Expected Image %s, got %s", tt.expected.Image, result.Image)
			}

			if !reflect.DeepEqual(result.Command, tt.expected.Command) {
				t.Errorf("Expected Command %v, got %v", tt.expected.Command, result.Command)
			}

			if !reflect.DeepEqual(result.Args, tt.expected.Args) {
				t.Errorf("Expected Args %v, got %v", tt.expected.Args, result.Args)
			}

			// Check complex fields (simplified check)
			if len(result.Env) != len(tt.expected.Env) {
				t.Errorf("Expected %d env vars, got %d", len(tt.expected.Env), len(result.Env))
			}

			if len(result.VolumeMounts) != len(tt.expected.VolumeMounts) {
				t.Errorf("Expected %d volume mounts, got %d",
					len(tt.expected.VolumeMounts), len(result.VolumeMounts))
			}

			// Check pointer fields
			if (result.SecurityContext == nil) != (tt.expected.SecurityContext == nil) {
				t.Errorf("SecurityContext mismatch: expected %v, got %v",
					tt.expected.SecurityContext != nil, result.SecurityContext != nil)
			}
		})
	}
}

// Test for mergeContainers function
func TestMergeContainers(t *testing.T) {
	tests := []struct {
		name     string
		base     []corev1.Container
		overlay  []corev1.Container
		expected []corev1.Container
	}{
		{
			name: "merge with overlapping containers",
			base: []corev1.Container{
				{Name: "container1", Image: "image1:v1"},
				{Name: "container2", Image: "image2:v1"},
			},
			overlay: []corev1.Container{
				{Name: "container2", Image: "image2:v2"},
				{Name: "container3", Image: "image3:v1"},
			},
			expected: []corev1.Container{
				{Name: "container1", Image: "image1:v1"},
				{Name: "container2", Image: "image2:v2"},
				{Name: "container3", Image: "image3:v1"},
			},
		},
		{
			name:     "empty base",
			base:     []corev1.Container{},
			overlay:  []corev1.Container{{Name: "container1", Image: "image1:v1"}},
			expected: []corev1.Container{{Name: "container1", Image: "image1:v1"}},
		},
		{
			name:     "empty overlay",
			base:     []corev1.Container{{Name: "container1", Image: "image1:v1"}},
			overlay:  []corev1.Container{},
			expected: []corev1.Container{{Name: "container1", Image: "image1:v1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeContainers(tt.base, tt.overlay)

			// Create maps for easier comparison
			resultMap := make(map[string]string)
			for _, container := range result {
				resultMap[container.Name] = container.Image
			}

			expectedMap := make(map[string]string)
			for _, container := range tt.expected {
				expectedMap[container.Name] = container.Image
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d containers, got %d", len(tt.expected), len(result))
			}

			for name, image := range expectedMap {
				if resultMap[name] != image {
					t.Errorf("Expected container %s with image %s, got image %s",
						name, image, resultMap[name])
				}
			}
		})
	}
}

// Test for MergePodSpecs function
func TestMergePodSpecs(t *testing.T) {
	gracePeriod := int64(30)
	priority := int32(100)

	tests := []struct {
		name     string
		base     *corev1.PodSpec
		overlay  *corev1.PodSpec
		expected *corev1.PodSpec
	}{
		{
			name: "merge with overlapping fields",
			base: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1", Image: "image1:v1"},
				},
				Volumes: []corev1.Volume{
					{Name: "volume1"},
				},
				NodeSelector: map[string]string{
					"key1": "value1",
				},
				ServiceAccountName: "service-account-1",
			},
			overlay: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container2", Image: "image2:v1"},
				},
				Volumes: []corev1.Volume{
					{Name: "volume2"},
				},
				NodeSelector: map[string]string{
					"key2": "value2",
				},
				ServiceAccountName:            "service-account-2",
				TerminationGracePeriodSeconds: &gracePeriod,
				Priority:                      &priority,
				HostNetwork:                   true,
			},
			expected: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1", Image: "image1:v1"},
					{Name: "container2", Image: "image2:v1"},
				},
				Volumes: []corev1.Volume{
					{Name: "volume1"},
					{Name: "volume2"},
				},
				NodeSelector: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				ServiceAccountName:            "service-account-2",
				TerminationGracePeriodSeconds: &gracePeriod,
				Priority:                      &priority,
				HostNetwork:                   true,
			},
		},
		{
			name: "merge with overlapping fields",
			base: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1", Image: "image1:v1"},
				},
				Volumes: []corev1.Volume{
					{Name: "volume1"},
				},
				NodeSelector: map[string]string{
					"key1": "value1",
				},
				ServiceAccountName: "service-account-1",
			},
			overlay: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container2", Image: "image2:v1"},
				},
				Volumes: []corev1.Volume{
					{Name: "volume2"},
				},
				NodeSelector: map[string]string{
					"key2": "value2",
				},
				ServiceAccountName:            "service-account-2",
				TerminationGracePeriodSeconds: &gracePeriod,
				Priority:                      &priority,
				HostNetwork:                   true,
			},
			expected: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1", Image: "image1:v1"},
					{Name: "container2", Image: "image2:v1"},
				},
				Volumes: []corev1.Volume{
					{Name: "volume1"},
					{Name: "volume2"},
				},
				NodeSelector: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				ServiceAccountName:            "service-account-2",
				TerminationGracePeriodSeconds: &gracePeriod,
				Priority:                      &priority,
				HostNetwork:                   true,
			},
		},
		{
			name: "merge with nil fields",
			base: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1", Image: "image1:v1"},
				},
			},
			overlay: &corev1.PodSpec{
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser: ptr(int64(1000)),
				},
			},
			expected: &corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "container1", Image: "image1:v1"},
				},
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser: ptr(int64(1000)),
				},
				NodeSelector: map[string]string{},
			},
		},

		{
			name: "merge with overlapping container names",
			base: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "container1",
						Image:   "image1:v1",
						Command: []string{"/bin/bash", "-c"},
						Env: []corev1.EnvVar{
							{Name: "VAR1", Value: "base-value1"},
						},
					},
				},
			},
			overlay: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container1",
						Image: "image1:v2",
						Env: []corev1.EnvVar{
							{Name: "VAR2", Value: "overlay-value2"},
						},
					},
				},
			},
			expected: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "container1",
						Image:   "image1:v2",
						Command: []string{"/bin/bash", "-c"},
						Env: []corev1.EnvVar{
							{Name: "VAR1", Value: "base-value1"},
							{Name: "VAR2", Value: "overlay-value2"},
						},
					},
				},
				NodeSelector: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergePodSpecs(tt.base, tt.overlay)

			// Check containers
			if len(result.Containers) != len(tt.expected.Containers) {
				t.Errorf("Expected %d containers, got %d",
					len(tt.expected.Containers), len(result.Containers))
			}

			// Check volumes
			if len(result.Volumes) != len(tt.expected.Volumes) {
				t.Errorf("Expected %d volumes, got %d",
					len(tt.expected.Volumes), len(result.Volumes))
			}

			// Check node selector
			if len(result.NodeSelector) != len(tt.expected.NodeSelector) {
				t.Errorf("Expected %d node selector entries, got %d",
					len(tt.expected.NodeSelector), len(result.NodeSelector))
			}

			// Check service account
			if result.ServiceAccountName != tt.expected.ServiceAccountName {
				t.Errorf("Expected service account %s, got %s",
					tt.expected.ServiceAccountName, result.ServiceAccountName)
			}

			// Check host network
			if result.HostNetwork != tt.expected.HostNetwork {
				t.Errorf("Expected host network %v, got %v",
					tt.expected.HostNetwork, result.HostNetwork)
			}

			// Check security context
			if (result.SecurityContext == nil) != (tt.expected.SecurityContext == nil) {
				t.Errorf("SecurityContext mismatch: expected %v, got %v",
					tt.expected.SecurityContext != nil, result.SecurityContext != nil)
			}
		})
	}
}

// Helper function to create int64 pointer
func ptr(i int64) *int64 {
	return &i
}
