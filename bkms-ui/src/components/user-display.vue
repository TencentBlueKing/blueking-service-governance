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

<template>
  <span>
    <template v-if="canDisplayUserNames && !plain && userIds.length">
      <template
        v-for="(id, index) in userIds"
        :key="`${index}:${id}`"
      >
        <template v-if="index > 0">{{ separator }}</template>
        <bk-user-display-name :user-id.attr="id" />
      </template>
    </template>
    <template v-else>{{ userIds.join(separator) || emptyText }}</template>
  </span>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import BkUserDisplayName from '@blueking/bk-user-display-name';

  const props = withDefaults(
    defineProps<{
      emptyText?: string;
      /** 原样展示，不查姓名（如内置组件、系统生成标记） */
      plain?: boolean;
      separator?: string;
      value?: null | readonly (null | string | undefined)[] | string;
    }>(),
    { emptyText: '--', separator: ', ' },
  );

  const canDisplayUserNames = !!(BkUserDisplayName.tenantId && BkUserDisplayName.apiBaseUrl);
  const userIds = computed(() => {
    const values = typeof props.value === 'string' ? [props.value] : (props.value ?? []);
    return values.map(id => id?.trim()).filter((id): id is string => !!id);
  });
</script>
