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
  <Select
    v-model="modelValue"
    class="mr-[16px] min-w-[240px]"
    :clearable="false"
  >
    <template #prefix>
      <FormPrefix :label="$t('泳道')" />
    </template>
    <template v-if="type === 'helm'">
      <Select.Option
        v-for="item in list as TrafficLaneOutputObj[]"
        :id="item.name"
        :key="item.name"
        :name="item.name"
      >
        <FlexRow class="w-full">
          <template #left>{{ item.name }}</template>
          <template #right>
            <bk-tag
              v-if="item.type === 'base'"
              class="mr-[2px]"
            >
              {{ $t('基线') }}
            </bk-tag>
            <bk-tag v-if="item?.serviceVersionLabels?.version">
              {{ item.serviceVersionLabels.version }}
            </bk-tag>
            <span
              v-else
              class="text-[#979BA5]"
              >{{ $t('未部署') }}</span
            >
          </template>
        </FlexRow>
      </Select.Option>
    </template>
  </Select>
</template>

<script lang="ts" setup>
  import { Select } from 'bkui-vue';
  import { TrafficLaneOutputObj } from '~/@types/env';

  type AppTypeLaneMap = {
    helm: TrafficLaneOutputObj[];
  };

  interface IProps {
    list: AppTypeLaneMap[keyof AppTypeLaneMap];
    type: keyof AppTypeLaneMap;
  }

  const modelValue = defineModel('modelValue');
  defineProps<IProps>();
</script>
