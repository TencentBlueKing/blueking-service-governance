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
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// buildProbe converts an appmodel.Probe to a corev1.Probe.
func buildProbe(probe *appmodel.Probe) (*corev1.Probe, error) {
	if probe == nil {
		return nil, nil
	}
	if probe.ProbeHandler == nil {
		return nil, errors.New("probe handler is required")
	}

	handler, err := buildProbeHandler(probe.ProbeHandler)
	if err != nil {
		return nil, err
	}

	return &corev1.Probe{
		ProbeHandler:        *handler,
		InitialDelaySeconds: probe.InitialDelaySeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}, nil
}

// Builds a corev1.ProbeHandler from an appmodel.ProbeHandler.
func buildProbeHandler(handler *appmodel.ProbeHandler) (*corev1.ProbeHandler, error) {
	switch handler.Type {
	case appmodel.ProbeTypeExec:
		if handler.ExecAction == nil {
			return nil, errors.New("probe handler exec action is required")
		}
		if handler.ShCommand != "" && len(handler.Command) > 0 {
			return nil, errors.New("probe handler exec: command and sh command are mutually exclusive")
		}
		if handler.ShCommand != "" {
			return &corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"/bin/sh", "-c", handler.ShCommand},
				},
			}, nil
		}
		if len(handler.Command) > 0 {
			return &corev1.ProbeHandler{
				Exec: &corev1.ExecAction{Command: handler.Command},
			}, nil
		}
		return nil, errors.New("probe handler exec requires non-empty command or sh command")
	case appmodel.ProbeTypeHTTP:
		if handler.HTTPGetAction == nil {
			return nil, errors.New("probe handler httpGet action is required")
		}
		httpGet, err := buildHTTPGetAction(handler.HTTPGetAction)
		if err != nil {
			return nil, err
		}
		return &corev1.ProbeHandler{HTTPGet: httpGet}, nil
	case appmodel.ProbeTypeTCP:
		if handler.TCPSocketAction == nil {
			return nil, errors.New("probe handler tcpSocket action is required")
		}
		return &corev1.ProbeHandler{
			TCPSocket: buildTCPSocketAction(handler.TCPSocketAction),
		}, nil
	default:
		return nil, errors.Errorf("unknown probe handler type %q", handler.Type)
	}
}

// Builds a corev1.HTTPGetAction from an appmodel.HTTPGetAction.
func buildHTTPGetAction(action *appmodel.HTTPGetAction) (*corev1.HTTPGetAction, error) {
	parsed, err := url.Parse(action.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing httpGet url")
	}
	headers := make([]corev1.HTTPHeader, 0, len(action.Headers))
	for name, value := range action.Headers {
		headers = append(headers, corev1.HTTPHeader{Name: name, Value: value})
	}

	return &corev1.HTTPGetAction{
		Path:        parsed.Path,
		Host:        parsed.Hostname(),
		Port:        intstr.FromInt32(action.Port),
		Scheme:      corev1.URIScheme(strings.ToUpper(parsed.Scheme)),
		HTTPHeaders: headers,
	}, nil
}

// buildExecCommand builds commands for an appmodel.ExecAction.
// Shell command and command are mutually exclusive;
// If both Shell command and command is not provided, sleepSeconds is the fallback.
// SleepSeconds generates a "sleep <n>" command.
func buildExecCommand(exec *appmodel.ExecAction) ([]string, error) {
	if exec.ShCommand != "" && len(exec.Command) > 0 {
		return nil, errors.New("exec action: command and sh command are mutually exclusive")
	}
	if exec.ShCommand != "" {
		return []string{"/bin/sh", "-c", exec.ShCommand}, nil
	}
	if len(exec.Command) > 0 {
		return exec.Command, nil
	}
	if exec.SleepSeconds != nil {
		return []string{"sleep", strconv.FormatInt(*exec.SleepSeconds, 10)}, nil
	}
	return nil, errors.New("lifecycle handler exec command or sleepSeconds is required")
}

// buildTCPSocketAction builds a corev1.TCPSocketAction from an appmodel.TCPSocketAction.
func buildTCPSocketAction(action *appmodel.TCPSocketAction) *corev1.TCPSocketAction {
	return &corev1.TCPSocketAction{
		Port: intstr.FromInt32(action.Port),
	}
}
