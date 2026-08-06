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

/*
Package testutil provides utility functions for testing.

deps.go provides default URLs/Configs for test dependencies, the default values can be
overridden via environment variables.
*/
package testutil

import (
	"context"
	"encoding/base64"
	"os"
	"strings"

	"k8s.io/client-go/rest"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

const (
	// envGitServerURLVar overrides the Git server URL used in tests.
	envGitServerURLVar = "FOR_TEST_GIT_SERVER_URL"
	// envHelmRegistryURLVar overrides the Helm registry URL used in tests.
	envHelmRegistryURLVar = "FOR_TEST_HELM_REGISTRY_URL"
	// envContainerRegistryURLVar overrides the container registry URL used in tests.
	envContainerRegistryURLVar = "FOR_TEST_CONTAINER_REGISTRY_URL"

	// Test Kubernetes cluster related:

	// envKubeConfigPathVar points to the kubeconfig file used in tests and takes precedence over
	// other Kube* env vars.
	envKubeConfigPathVar = "FOR_TEST_KUBE_CONFIG_PATH"
	// envKubeAPIServerURLVar sets the apiserver URL for the test Kubernetes cluster.
	envKubeAPIServerURLVar = "FOR_TEST_KUBE_APISERVER_URL"
	// envKubeCADataVar sets the CA data for the test Kubernetes cluster (base64 or plain string).
	envKubeCADataVar = "FOR_TEST_KUBE_CA_DATA"
	// envKubeTokenValueVar provides the bearer token for authenticating to the test Kubernetes cluster.
	envKubeTokenValueVar = "FOR_TEST_KUBE_TOKEN_VALUE"
)

const (
	// The default URLs for various test dependencies
	defaultGitServerURL         = "http://localhost:28010"
	defaultHelmRegistryURL      = "http://localhost:28020"
	defaultContainerRegistryURL = "http://localhost:28030"
)

// GitServerURL 代码库服务 URL
func GitServerURL() string {
	if val := os.Getenv(envGitServerURLVar); val != "" {
		return val
	}
	return defaultGitServerURL
}

// HelmRegistryURL helm repo 仓库 URL
func HelmRegistryURL() string {
	if val := os.Getenv(envHelmRegistryURLVar); val != "" {
		return val
	}
	return defaultHelmRegistryURL
}

// ContainerRegistryURL 容器镜像仓库 URL
func ContainerRegistryURL() string {
	if val := os.Getenv(envContainerRegistryURLVar); val != "" {
		return val
	}
	return defaultContainerRegistryURL
}

// KubeConfigPath 包含有效集群配置的 kubeconfig 文件地址
func KubeConfigPath() string {
	return os.Getenv(envKubeConfigPathVar)
}

// KubeConfigFromEnv returns rest config built from env vars if provided, otherwise nil.
func KubeConfigFromEnv() *rest.Config {
	apiServerURL := strings.TrimSpace(os.Getenv(envKubeAPIServerURLVar))
	if apiServerURL == "" {
		return nil
	}

	caData := strings.TrimSpace(os.Getenv(envKubeCADataVar))
	token := os.Getenv(envKubeTokenValueVar)
	// try to decode caData from base64, if fails, use it as plain string
	caBytes, err := base64.StdEncoding.DecodeString(caData)
	if err != nil {
		log.Warnf(context.Background(), "failed to decode CA data from base64, using it as plain string: %v", err)
		caBytes = []byte(caData)
	}

	return &rest.Config{
		Host:            apiServerURL,
		BearerToken:     token,
		TLSClientConfig: rest.TLSClientConfig{CAData: caBytes},
	}
}
