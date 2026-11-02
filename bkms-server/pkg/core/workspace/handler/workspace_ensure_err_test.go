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

package handler_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/handler"
)

var _ = Describe("classifyEnsureBkSystemsErr", func() {
	It("maps missing independent biz id to invalid request", func() {
		code := handler.ClassifyEnsureBkSystemsErr(workspace.ErrIndependentWorkspaceBizRequired)
		Expect(code).To(Equal(bkerrs.ErrCodeInvalidRequest))
	})

	It("maps inaccessible biz to invalid request", func() {
		err := fmt.Errorf("%w: validate bkcc biz 398", workspace.ErrIndependentWorkspaceBizInvalid)
		code := handler.ClassifyEnsureBkSystemsErr(err)
		Expect(code).To(Equal(bkerrs.ErrCodeInvalidRequest))
	})

	It("maps existing deterministic bcs project to already exists", func() {
		err := fmt.Errorf("%w: bcs project(%s) already exists", workspace.ErrBCSProjectAlreadyExists, "bkms-ws-1")
		code := handler.ClassifyEnsureBkSystemsErr(err)
		Expect(code).To(Equal(bkerrs.ErrCodeAlreadyExists))
	})

	It("maps bound bcs project to already exists", func() {
		err := fmt.Errorf(
			"%w: bcs project(%s) already bound by workspace %s",
			workspace.ErrBCSProjectAlreadyBound,
			"existing-bcs",
			"other-ws",
		)
		code := handler.ClassifyEnsureBkSystemsErr(err)
		Expect(code).To(Equal(bkerrs.ErrCodeAlreadyExists))
	})

	It("keeps unexpected errors as internal server error", func() {
		code := handler.ClassifyEnsureBkSystemsErr(errors.New("init bkrepo project"))
		Expect(code).To(Equal(bkerrs.ErrCodeInternalServerError))
	})
})
