/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package workload

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestValidateVolumeMountPaths_NoConflict(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "vol-a", MountPath: "/etc/app/config.yaml"},
		{Name: "vol-b", MountPath: "/etc/nginx/nginx.conf"},
		{Name: "vol-c", MountPath: "/data/logs"},
	}
	if err := validateVolumeMountPaths("main", mounts); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateVolumeMountPaths_Conflict(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "framework-cfg", MountPath: "/etc/app/config.yaml"},
		{Name: "plain-cfg", MountPath: "/etc/app/config.yaml"},
	}
	err := validateVolumeMountPaths("main", mounts)
	if err == nil {
		t.Fatal("expected error for duplicate MountPath, got nil")
	}
	if want := `duplicate MountPath "/etc/app/config.yaml"`; !contains(err.Error(), want) {
		t.Fatalf("error should mention duplicate path, got: %v", err)
	}
	if want := `"framework-cfg"`; !contains(err.Error(), want) {
		t.Fatalf("error should mention conflicting volume, got: %v", err)
	}
	if want := `"plain-cfg"`; !contains(err.Error(), want) {
		t.Fatalf("error should mention conflicting volume, got: %v", err)
	}
}

func TestValidateVolumeMountPaths_Empty(t *testing.T) {
	if err := validateVolumeMountPaths("init", nil); err != nil {
		t.Fatalf("expected no error for empty mounts, got: %v", err)
	}
}

func TestValidateVolumeMountPaths_SingleElement(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "only-vol", MountPath: "/etc/app/config.yaml"},
	}
	if err := validateVolumeMountPaths("main", mounts); err != nil {
		t.Fatalf("expected no error for single mount, got: %v", err)
	}
}

func TestValidateVolumeMountPaths_MultipleConflicts(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "vol-a", MountPath: "/etc/app/a.yaml"},
		{Name: "vol-b", MountPath: "/etc/app/a.yaml"},
		{Name: "vol-c", MountPath: "/data/b.conf"},
		{Name: "vol-d", MountPath: "/data/b.conf"},
	}
	err := validateVolumeMountPaths("main", mounts)
	if err == nil {
		t.Fatal("expected error for duplicate MountPath, got nil")
	}
	// Should report the first conflict found
	if want := `duplicate MountPath`; !contains(err.Error(), want) {
		t.Fatalf("error should mention duplicate path, got: %v", err)
	}
}

func TestValidateVolumeMountPaths_ContainerNameInError(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "vol-a", MountPath: "/etc/dup"},
		{Name: "vol-b", MountPath: "/etc/dup"},
	}
	err := validateVolumeMountPaths("trpc-init", mounts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if want := `"trpc-init"`; !contains(err.Error(), want) {
		t.Fatalf("error should contain container name, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
