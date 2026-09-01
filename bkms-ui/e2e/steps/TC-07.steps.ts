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
/**
 * TC-07 更新策略配置：业务语义步骤。
 */
import {
  editUpdateStrategyAndCancel,
  saveValidUpdateStrategy,
  submitInvalidUpdateStrategy,
  UPDATE_STRATEGY_INVALID_VALUE,
  UPDATE_STRATEGY_SAVED_MAX_SURGE,
  UPDATE_STRATEGY_SAVED_MAX_UNAVAILABLE,
} from '../actions/app-config.action';
import { Then, When } from '../fixtures/fixtures';

When('我编辑更新策略后取消', async ({ pages }) => {
  await editUpdateStrategyAndCancel({ appDetailPage: pages.appDetailPage });
});

When('我保存有效更新策略配置', async ({ pages }) => {
  await saveValidUpdateStrategy({ appDetailPage: pages.appDetailPage });
});

When('我提交无效更新策略配置', async ({ pages }) => {
  await submitInvalidUpdateStrategy({ appDetailPage: pages.appDetailPage });
});

Then('更新策略不应包含取消测试值', async ({ pages }) => {
  await pages.appDetailPage.expectUpdateStrategyTextHidden(UPDATE_STRATEGY_INVALID_VALUE);
});

Then('更新策略区域应展示当前配置', async ({ pages }) => {
  await pages.appDetailPage.expectUpdateStrategyInViewMode();
  await pages.appDetailPage.expectUpdateStrategyContains('最大超出数量');
  await pages.appDetailPage.expectUpdateStrategyContains('最大不可用数量');
});

Then('更新策略应处于查看态', async ({ pages }) => {
  await pages.appDetailPage.expectUpdateStrategyInViewMode();
});

Then('更新策略应展示格式校验提示', async ({ pages }) => {
  await pages.appDetailPage.expectUpdateStrategyValidationVisible('请输入非负整数或百分比');
});

Then('更新策略应展示已保存配置', async ({ pages }) => {
  await pages.appDetailPage.expectUpdateStrategyContains(UPDATE_STRATEGY_SAVED_MAX_SURGE);
  await pages.appDetailPage.expectUpdateStrategyContains(UPDATE_STRATEGY_SAVED_MAX_UNAVAILABLE);
});
