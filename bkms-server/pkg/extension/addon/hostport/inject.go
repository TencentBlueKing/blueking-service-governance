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

package hostport

import (
	"context"
	"maps"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InjectFromStore loads declared HostPort mappings and:
//  1. merges BCS random HostPort webhook annotations into pod template metadata
//  2. upserts matching containerPorts on the main container
//
// The webhook only allocates HostPort / injects BCS_RANDHOSTPORT_* when the
// container declares the corresponding containerPort (same idea as the former
// Polaris health-check port injection).
//
// Callers decide whether injection applies (e.g. only for federation envs).
// Returns the ports snapshot used for this inject (empty when none),
// so deploy reconcile can record the same applied set without a second ListPorts.
func InjectFromStore(
	ctx context.Context,
	db *mongo.Database,
	appID string,
	meta *metav1.ObjectMeta,
	containers []corev1.Container,
	mainContainerName string,
) ([]int32, error) {
	ports, err := ListPorts(ctx, db, appID)
	if err != nil {
		return nil, errors.Wrap(err, "listing hostport mappings")
	}
	if len(ports) == 0 {
		return ports, nil
	}

	if meta != nil {
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}
		maps.Copy(meta.Annotations, BuildPodAnnotations(ports))
	}

	for i := range containers {
		if containers[i].Name != mainContainerName {
			continue
		}
		injectContainerPorts(ports, &containers[i])
		break
	}
	return ports, nil
}

// injectContainerPorts ensures HostPort-declared ports exist on the container.
// Existing entries with the same ContainerPort are left unchanged.
func injectContainerPorts(ports []int32, container *corev1.Container) {
	for _, port := range NormalizePorts(ports) {
		ensureContainerPort(&container.Ports, port)
	}
}

func ensureContainerPort(items *[]corev1.ContainerPort, port int32) {
	for idx := range *items {
		if (*items)[idx].ContainerPort == port {
			return
		}
	}
	*items = append(*items, corev1.ContainerPort{ContainerPort: port})
}
