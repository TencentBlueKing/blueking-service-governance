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
  <div
    class="service-item rounded-[2px] overflow-hidden bg-[#FFF] shadow-[0_2px_4px_0_#0000001a,0_2px_4px_0_#1919290d]"
  >
    <div class="service-header px-[16px] pt-[12px] pb-[16px] flex items-center">
      <div class="flex items-center gap-[12px]">
        <span
          class="cursor-pointer inline-flex items-center justify-center"
          @click="toggleExpand"
        >
          <RightShape
            class="text-[#63656E] transition-transform duration-300 origin-center"
            :class="{ 'rotate-90': isExpanded }"
            :height="14"
            :width="14"
          />
          <span class="text-[14px] font-bold text-[#313238] ml-[6px]">{{ service.name }}</span>
          <!-- 是否为泳道服务 -->
          <Tag
            v-if="service.trafficLaneEnabled"
            class="ml-[8px]"
            theme="info"
            >{{ $t('泳道服务') }}</Tag
          >
        </span>
        <!-- 编辑 -->
        <EditLine
          v-bk-tooltips="$t('编辑')"
          class="cursor-pointer text-[#979BA5] hover:text-[#3A84FF] p-[3px]"
          @click="handleEdit"
        ></EditLine>
        <!-- 删除 -->
        <PopConfirm
          :title="$t('确认删除该服务？')"
          trigger="click"
          width="288"
          @confirm="handleDelete"
        >
          <template #content>
            <div class="mb-[16px] text-[12px]">
              <div>
                {{ $t('服务名称') }}：<span class="text-[#313238]">{{ service.name }}</span>
              </div>
              <div>{{ $t('删除后，不可恢复，请谨慎操作！') }}</div>
            </div>
          </template>
          <Del
            v-bk-tooltips="$t('删除')"
            class="cursor-pointer text-[#979BA5] hover:text-[#EA3636] p-[3px]"
          ></Del>
        </PopConfirm>
      </div>
    </div>

    <!-- 展开内容 -->
    <div
      v-if="isExpanded"
      class="service-content bg-[#FFF] px-[36px] pb-[16px]"
    >
      <!-- 选择器信息 -->
      <div class="mb-[24px]">
        <div class="flex items-center gap-[8px] mb-[8px]">
          <span class="text-[12px] font-bold text-[#313238]">{{ $t('选择器') }}</span>
        </div>
        <ul class="flex flex-wrap items-center gap-y-[8px] gap-x-[80px] max-w-[1000px]">
          <li
            v-for="(value, key) in service.selector"
            :key="key"
            v-overflow-title
            class="flex items-center text-[12px] max-w-[300px]"
          >
            <OverflowTitle
              class="max-w-[300px]"
              :content="`${key}：${value}`"
              type="tips"
            >
              <span class="text-[#4D4F56]">{{ key }}：</span>{{ value }}
            </OverflowTitle>
          </li>
        </ul>
      </div>

      <!-- 端口配置 -->
      <div>
        <div class="text-[12px] font-bold text-[#313238] mb-[8px]">{{ $t('端口配置') }}</div>
        <Table
          class="max-w-[1000px]"
          :data="service.ports"
          :show-overflow-tooltip="true"
        >
          <template #empty>
            <TableException />
          </template>
          <TableColumn
            field="name"
            :label="$t('端口名称')"
            :min-width="120"
          />
          <TableColumn
            field="port"
            :label="$t('监听端口')"
            :min-width="120"
          />
          <TableColumn
            field="protocol"
            :label="$t('协议')"
            :min-width="120"
          />
          <TableColumn
            field="targetPort"
            :label="$t('目标端口')"
            :min-width="120"
          />
        </Table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { OverflowTitle, PopConfirm, Tag } from 'bkui-vue';
  import { Del, EditLine, RightShape } from 'bkui-vue/lib/icon';
  import { AppServiceOutput } from '~/@types/v1/app-networking';

  interface Emits {
    (e: 'edit', service: AppServiceOutput): void;
    (e: 'delete', serviceName: string): void;
  }

  interface Props {
    service: AppServiceOutput;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();

  const isExpanded = ref(true);

  const toggleExpand = () => {
    isExpanded.value = !isExpanded.value;
  };

  const handleEdit = () => {
    emit('edit', props.service);
  };

  const handleDelete = () => {
    emit('delete', props.service?.name || '');
  };
</script>
