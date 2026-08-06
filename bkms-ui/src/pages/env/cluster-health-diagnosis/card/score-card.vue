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
  <div class="px-[32px] py-[12px] min-w-[520px] h-[252px] bg-[#fff] flex justify-between">
    <div class="py-[12px]">
      <span class="text-[#313238] text-[14px] font-bold">
        {{ $t('集群健康分数') }}
      </span>
      <div class="text-[#979BA5] text-[12px] mt-[4px]">
        {{ $t('基于 {0} 个未恢复问题的综合评估', [unRecoveredCount]) }}
      </div>
      <ClusterHealthScore
        class="mt-[18px] ml-[40px]"
        :count="{
          RISK: count.RISK,
          WARN: count.WARN,
        }"
        :radius="60"
        :show-legend="false"
      />
    </div>
    <div>
      <ul class="grid gap-y-[12px]">
        <li
          class="min-w-[186px] h-[68px] bg-[#F5F7FA] rounded-[2px] cursor-pointer flex flex-col items-center p-[12px] hover:bg-[#F0F5FF]"
          :class="activeCard === LEVEL_VALUE.RISK ? 'outline outline-[1px] outline-[#3A84FF] !bg-[#E1ECFF]' : ''"
          @click="handleActiveChange(LEVEL_VALUE.RISK)"
        >
          <span class="text-[24px] leading-[24px] text-[#EA3636] font-bold">
            {{ count.RISK }}
          </span>
          <span class="text-[#4D4F56]">
            {{ `${$t('致命问题')}(${$t('未恢复')})` }}
          </span>
        </li>
        <li
          class="min-w-[186px] h-[68px] bg-[#F5F7FA] rounded-[2px] cursor-pointer flex flex-col items-center p-[12px] hover:bg-[#F0F5FF]"
          :class="activeCard === LEVEL_VALUE.WARN ? 'outline outline-[1px] outline-[#3A84FF] !bg-[#E1ECFF]' : ''"
          @click="handleActiveChange(LEVEL_VALUE.WARN)"
        >
          <span class="text-[24px] leading-[24px] text-[#F59500] font-bold">
            {{ count.WARN }}
          </span>
          <span class="text-[#4D4F56]">
            {{ `${$t('预警问题')}(${$t('未恢复')})` }}
          </span>
        </li>
        <li
          class="min-w-[186px] h-[68px] bg-[#F5F7FA] rounded-[2px] cursor-pointer flex flex-col items-center p-[12px] hover:bg-[#F0F5FF]"
          :class="activeCard === LEVEL_VALUE.RECOVERED ? 'outline outline-[1px] outline-[#3A84FF] !bg-[#E1ECFF]' : ''"
          @click="handleActiveChange(LEVEL_VALUE.RECOVERED)"
        >
          <span class="text-[24px] leading-[24px] text-[#2CAF5E] font-bold">
            {{ count.RECOVERED }}
          </span>
          <span class="text-[#4D4F56]">
            {{ $t('已恢复问题') }}
          </span>
        </li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { LEVEL_VALUE, LevelType } from '../levelMap';
  interface IProps {
    active: LevelType;
    value: number;
    count: {
      RECOVERED: number;
      RISK: number;
      WARN: number;
    };
  }
  const props = defineProps<IProps>();
  const emit = defineEmits<{
    (e: 'change', level: LevelType, isRecovered: boolean): void;
  }>();
  const activeCard = ref<LevelType>();

  const unRecoveredCount = computed(() => {
    return (props.count?.RISK || 0) + (props.count?.WARN || 0);
  });

  function handleActiveChange(level: LevelType) {
    let curLevel = '';
    if (activeCard.value !== level) {
      curLevel = level;
    }
    activeCard.value = level;
    emit('change', curLevel, curLevel === LEVEL_VALUE.RECOVERED);
  }

  watch(
    () => props.active,
    val => {
      activeCard.value = val;
    },
  );
</script>
