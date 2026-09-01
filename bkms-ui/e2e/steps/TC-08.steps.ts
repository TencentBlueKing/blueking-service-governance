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
 * TC-08 元数据配置：业务语义步骤。
 */
import {
  editLabelsMetadataAndCancel,
  METADATA_ANNOTATION_KEY,
  METADATA_ANNOTATION_VALUE,
  METADATA_ANNOTATIONS_LABEL,
  METADATA_CANCEL_LABEL_KEY,
  METADATA_LABEL_KEY,
  METADATA_LABEL_VALUE,
  METADATA_LABELS_LABEL,
  saveValidAnnotationsMetadata,
  saveValidLabelsMetadata,
  submitReservedLabelsMetadata,
} from '../actions/app-config.action';
import { Then, When } from '../fixtures/fixtures';

When('我编辑标签元数据后取消', async ({ pages }) => {
  await editLabelsMetadataAndCancel({ appDetailPage: pages.appDetailPage });
});

When('我保存有效标签元数据配置', async ({ pages }) => {
  await saveValidLabelsMetadata({ appDetailPage: pages.appDetailPage });
});

When('我保存有效注解元数据配置', async ({ pages }) => {
  await saveValidAnnotationsMetadata({ appDetailPage: pages.appDetailPage });
});

When('我提交系统保留标签配置', async ({ pages }) => {
  await submitReservedLabelsMetadata({ appDetailPage: pages.appDetailPage });
});

Then('标签元数据不应包含取消测试值', async ({ pages }) => {
  await pages.appDetailPage.expectMetadataTextHidden(METADATA_LABELS_LABEL, METADATA_CANCEL_LABEL_KEY);
});

Then('标签元数据应展示已保存配置', async ({ pages }) => {
  await pages.appDetailPage.expectMetadataContains(METADATA_LABELS_LABEL, METADATA_LABEL_KEY);
  await pages.appDetailPage.expectMetadataContains(METADATA_LABELS_LABEL, METADATA_LABEL_VALUE);
});

Then('元数据配置区域应展示标签和注解卡片', async ({ pages }) => {
  await pages.appDetailPage.expectMetadataCardsVisible();
});

Then('元数据配置应展示系统保留字段校验提示', async ({ pages }) => {
  await pages.appDetailPage.expectMetadataValidationVisible('system reserved field');
});

Then('注解元数据应展示已保存配置', async ({ pages }) => {
  await pages.appDetailPage.expectMetadataContains(METADATA_ANNOTATIONS_LABEL, METADATA_ANNOTATION_KEY);
  await pages.appDetailPage.expectMetadataContains(METADATA_ANNOTATIONS_LABEL, METADATA_ANNOTATION_VALUE);
});
