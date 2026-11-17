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

package bcstoken

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
)

var _ = Describe("ErrNoActiveUserToken", func() {
	It("带创建地址时仍可用 errors.Is 判断，并保留地址参数", func() {
		err := errors.Wrap(ErrNoActiveUserToken.With("createURL", "http://bcs.example.com/token"), "fetch token")

		Expect(errors.Is(err, ErrNoActiveUserToken)).To(BeTrue())
		bkErr := bkerrs.FromError(err)
		Expect(bkErr).NotTo(BeNil())
		Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeNotFound))
		Expect(bkErr.Params()).To(Equal(map[string]string{"createURL": "http://bcs.example.com/token"}))
	})

	It("不带参数也能渲染文案", func() {
		msg, ok := i18n.Localize(context.Background(), string(ErrNoActiveUserToken.DetailCode()), nil)
		Expect(ok).To(BeTrue())
		Expect(msg).To(ContainSubstring("BCS"))
	})
})
