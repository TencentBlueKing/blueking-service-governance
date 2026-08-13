<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<!-- 部署总览指标卡使用的轻量线性图标，颜色由卡片统一控制。 -->
<template>
  <svg
    fill="none"
    :height="size"
    stroke="currentColor"
    stroke-linecap="round"
    stroke-linejoin="round"
    stroke-width="1.5"
    viewBox="0 0 24 24"
    :width="size"
  >
    <!-- 部署环境：上下两层机架。 -->
    <template v-if="name === 'total'">
      <rect
        height="6"
        rx="2"
        width="17"
        x="3.5"
        y="3.5"
      />
      <rect
        height="6"
        rx="2"
        width="17"
        x="3.5"
        y="14.5"
      />
      <path d="M7 6.5h.01M7 17.5h.01" />
    </template>

    <!-- 部署中：带开口的进度圆环。 -->
    <template v-else-if="name === 'deploying'">
      <path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3" />
    </template>

    <!-- 部署失败：圆形叉号。 -->
    <template v-else-if="name === 'failed'">
      <circle
        cx="12"
        cy="12"
        r="8.5"
      />
      <path d="m9.3 9.3 5.4 5.4M14.7 9.3l-5.4 5.4" />
    </template>

    <!-- 异常实例：实例方块内的感叹号。 -->
    <template v-else>
      <rect
        height="17"
        rx="4"
        width="17"
        x="3.5"
        y="3.5"
      />
      <path d="M12 7.75v5M12 16.25h.01" />
    </template>
  </svg>
</template>

<script lang="ts" setup>
  import type { DeployOverviewStatKey } from './use-deploy-overview';

  withDefaults(
    defineProps<{
      name: DeployOverviewStatKey;
      size?: number;
    }>(),
    { size: 22 },
  );
</script>
