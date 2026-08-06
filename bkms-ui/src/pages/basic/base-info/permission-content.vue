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
  <div class="py-[12px] px-[16px] bg-[#F5F7FA]">
    <div class="text-[#313238] font-bold text-[12px] mb-[4px]">{{ $t('权限内容') }}</div>
    <ul v-if="value !== 'admin'">
      <li
        v-for="item in permissionContent"
        :key="item.resource"
        class="text-[#63656E] text-[12px] leading-[20px]"
      >
        <span class="font-bold">{{ item.resource }}</span>
        <span>：{{ item.operations.join(', ') }}</span>
      </li>
    </ul>
    <span
      v-else
      class="text-[#63656E] text-[12px] leading-[20px]"
      >{{ $t('拥有全部权限') }}</span
    >
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { useI18n } from 'vue-i18n';

  import { IRole, PERMISSION_LIST } from './permission-list';
  interface IProps {
    value: IRole;
  }

  const props = defineProps<IProps>();
  const { t } = useI18n();
  const permissionContent = computed(() => {
    if (props.value === 'admin') return [];
    const map = new Map();
    const curList = PERMISSION_LIST.filter(item => item[props.value]);
    for (const item of curList) {
      const groupKey = item.resource;
      if (!map.has(item.resource)) {
        map.set(groupKey, []);
      }
      map.get(groupKey).push(item.operation);
    }
    const allows = [];
    for (const [key, value] of map.entries()) {
      allows.push({
        resource: t(key),
        operations: value.map((item: string) => t(item)),
      });
    }
    return allows;
  });
</script>
