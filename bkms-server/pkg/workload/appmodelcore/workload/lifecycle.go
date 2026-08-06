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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// buildLifecycle builds a Kubernetes Lifecycle object from the given appmodel.Lifecycle.
func buildLifecycle(lifecycle *appmodel.Lifecycle) (*corev1.Lifecycle, error) {
	if lifecycle == nil {
		return nil, nil
	}

	preStop, err := buildLifecycleHandler(lifecycle.PreStop)
	if err != nil {
		return nil, errors.Wrap(err, "building preStop")
	}
	postStart, err := buildLifecycleHandler(lifecycle.PostStart)
	if err != nil {
		return nil, errors.Wrap(err, "building postStart")
	}

	if preStop == nil && postStart == nil {
		return nil, nil
	}

	return &corev1.Lifecycle{
		PreStop:   preStop,
		PostStart: postStart,
	}, nil
}

func buildLifecycleHandler(handler *appmodel.LifecycleHandler) (*corev1.LifecycleHandler, error) {
	if handler == nil {
		return nil, nil
	}

	switch handler.Type {
	case appmodel.LifecycleTypeExec:
		if handler.ExecAction == nil {
			return nil, errors.New("lifecycle handler exec action is required")
		}
		command, err := buildExecCommand(handler.ExecAction)
		if err != nil {
			return nil, err
		}
		return &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{Command: command},
		}, nil
	case appmodel.LifecycleTypeHTTP:
		if handler.HTTPGetAction == nil {
			return nil, errors.New("lifecycle handler httpGet action is required")
		}
		httpGet, err := buildHTTPGetAction(handler.HTTPGetAction)
		if err != nil {
			return nil, err
		}
		return &corev1.LifecycleHandler{HTTPGet: httpGet}, nil
	default:
		return nil, errors.Errorf("unknown lifecycle handler type %q", handler.Type)
	}
}
