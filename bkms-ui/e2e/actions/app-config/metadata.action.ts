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
import type AppDetailPage from '../../pages/app-detail.page';

export const METADATA_ANNOTATION_KEY = 'e2e.bkms.io/annotation';
export const METADATA_ANNOTATION_TEXT = `${METADATA_ANNOTATION_KEY}=e2e-annotation-value`;
export const METADATA_ANNOTATION_VALUE = 'e2e-annotation-value';
export const METADATA_ANNOTATIONS_LABEL = '注解（Annotations）';
export const METADATA_CANCEL_LABEL_KEY = 'e2e.bkms.io/cancel-label';
export const METADATA_CANCEL_LABEL_TEXT = `${METADATA_CANCEL_LABEL_KEY}=e2e-cancel-label`;
export const METADATA_LABEL_KEY = 'e2e.bkms.io/label';
export const METADATA_LABEL_TEXT = `${METADATA_LABEL_KEY}=e2e-label`;
export const METADATA_LABEL_VALUE = 'e2e-label';
export const METADATA_LABELS_LABEL = '标签（Labels）';
export const METADATA_RESERVED_LABEL_TEXT = 'app.kubernetes.io/name=e2e-reserved';

/** 编辑 Labels 元数据后取消：用于验证取消不会把草稿带回查看态 */
export async function editLabelsMetadataAndCancel({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickMetadataEdit(METADATA_LABELS_LABEL);
  await appDetailPage.selectMetadataTextMode(METADATA_LABELS_LABEL);
  await appDetailPage.fillMetadataText(METADATA_LABELS_LABEL, METADATA_CANCEL_LABEL_TEXT);
  await appDetailPage.clickMetadataCancel(METADATA_LABELS_LABEL);
}

/** 保存有效 Annotations 元数据配置：用于验证 annotations PUT 保存链路与查看态回显 */
export async function saveValidAnnotationsMetadata({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickMetadataEdit(METADATA_ANNOTATIONS_LABEL);
  await appDetailPage.selectMetadataTextMode(METADATA_ANNOTATIONS_LABEL);
  await appDetailPage.fillMetadataText(METADATA_ANNOTATIONS_LABEL, METADATA_ANNOTATION_TEXT);
  await appDetailPage.clickMetadataSaveAndWait(METADATA_ANNOTATIONS_LABEL, 'annotations');
}

/** 保存有效 Labels 元数据配置：用于验证 labels PUT 保存链路与查看态回显 */
export async function saveValidLabelsMetadata({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickMetadataCancel(METADATA_LABELS_LABEL);
  await appDetailPage.clickMetadataEdit(METADATA_LABELS_LABEL);
  await appDetailPage.selectMetadataTextMode(METADATA_LABELS_LABEL);
  await appDetailPage.fillMetadataText(METADATA_LABELS_LABEL, METADATA_LABEL_TEXT);
  await appDetailPage.clickMetadataSaveAndWait(METADATA_LABELS_LABEL, 'labels');
}

/** 提交系统保留 Labels 元数据：用于验证保留 key 校验 */
export async function submitReservedLabelsMetadata({ appDetailPage }: { appDetailPage: AppDetailPage }) {
  await appDetailPage.clickMetadataEdit(METADATA_LABELS_LABEL);
  await appDetailPage.selectMetadataTextMode(METADATA_LABELS_LABEL);
  await appDetailPage.fillMetadataText(METADATA_LABELS_LABEL, METADATA_RESERVED_LABEL_TEXT);
  await appDetailPage.clickMetadataSave(METADATA_LABELS_LABEL);
}
