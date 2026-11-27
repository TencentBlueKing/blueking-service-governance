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

package apierrs_test

import (
	stderrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envapierrs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/apierrs"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("UpdateEnvCluster rules", func() {
	rules := bkerrs.Rules(
		envapierrs.HasAppsCannotUpdateCluster,
		envapierrs.SharedNamespaceCannotUpdateCluster,
		envapierrs.HasNamespaceSharersCannotUpdateCluster,
		envapierrs.NamespaceOccupied,
	)

	DescribeTable("should map business constraint errors to invalid request",
		func(serviceErr error, wantMsg string) {
			apiErr := rules(errors.Wrap(serviceErr, "update env cluster"))

			Expect(apiErr).NotTo(BeNil())
			Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(apiErr.Error()).To(HavePrefix(wantMsg))
			Expect(stderrors.Is(apiErr, serviceErr)).To(BeTrue())
		},
		Entry(
			"shared namespace",
			errors.Wrapf(
				envmodel.ErrSharedNamespaceUnsupported,
				"cannot change cluster or namespace of env %s",
				"feat-1",
			),
			"environments sharing a namespace cannot change cluster or namespace, recreate the environment instead",
		),
		Entry(
			"namespace sharers",
			errors.Wrapf(
				bkmsenv.ErrEnvHasNamespaceSharers,
				"feature environments [%s] share namespace %s",
				"feat-1",
				"ns",
			),
			"namespace is shared by feature environments, delete them before changing cluster or namespace",
		),
	)

	It("should leave unrelated errors to the error handler", func() {
		Expect(rules(errors.Wrap(stderrors.New("connection refused"), "update env cluster"))).To(BeNil())
	})
})
