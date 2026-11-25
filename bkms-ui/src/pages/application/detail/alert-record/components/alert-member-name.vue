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
  <UserDisplay
    v-if="isUnresolvedUser"
    :value="member.id"
  />
  <template v-else>{{ member.display_name || member.id || '--' }}</template>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import type { UserGroupUser } from '~/@types/v1/bkintegrations-bkmonitor';

  const props = defineProps<{ member: UserGroupUser }>();

  /** 告警组角色等非用户成员保留原名；用户成员仅在缺少可信姓名时查名 */
  const isUnresolvedUser = computed(
    () => props.member.type === 'user' && (!props.member.display_name || props.member.display_name === props.member.id),
  );
</script>
