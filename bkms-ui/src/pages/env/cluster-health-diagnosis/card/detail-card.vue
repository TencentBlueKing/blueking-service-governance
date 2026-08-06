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
  <div class="px-[24px] py-[16px] grid grid-cols-3 gap-y-[16px]">
    <div class="col-span-3">
      <Tag
        v-if="data.level"
        :theme="levelMap[data.level].theme"
      >
        {{ levelMap[data.level].label }}
      </Tag>
      <span class="text-[#313238] text-[14px] font-bold ml-[16px] mr-[12px]">
        {{ data.description }}
      </span>
    </div>
    <FieldItem
      v-if="data.level !== 'INFO'"
      container-height="auto"
      field-color="#4D4F56"
      :field-value="$t('告警资源')"
      :field-width="70"
      :value="data.resourceKey"
      value-color="#313238"
      value-max-width="85%"
    >
    </FieldItem>
    <FieldItem
      v-else
      container-height="auto"
      field-color="#4D4F56"
      :field-value="$t('节点')"
      :field-width="70"
      :value="data.resourceKey"
      value-color="#313238"
      value-max-width="85%"
    >
    </FieldItem>
    <FieldItem
      class="col-span-2"
      container-height="auto"
      field-color="#4D4F56"
      :field-value="$t('类型')"
      :field-width="70"
      :value="data.resourceType"
      value-color="#313238"
    >
    </FieldItem>
    <FieldItem
      container-height="auto"
      field-color="#4D4F56"
      :field-value="$t('开始时间')"
      :field-width="70"
      :value="formatTimeByTimezone(data.timestamp!)"
      value-color="#313238"
    >
    </FieldItem>
    <FieldItem
      class="col-span-2"
      container-height="auto"
      field-color="#4D4F56"
      :field-value="$t('持续时长')"
      :field-width="70"
      :value="durationTime"
      value-color="#313238"
    >
    </FieldItem>
    <FieldItem
      class="col-span-3"
      container-height="auto"
      field-color="#4D4F56"
      :field-value="$t('记录数')"
      :field-width="70"
      :value="data.recordCount"
      value-color="#313238"
    >
    </FieldItem>
    <FieldItem
      class="col-span-3"
      container-height="auto"
      field-color="#4D4F56"
      :field-value="$t('告警内容')"
      :field-width="70"
      :value="data.contextMsg"
      value-color="#313238"
    >
    </FieldItem>
    <FieldItem
      class="col-span-3 !items-start"
      container-height="auto"
      field-color="#4D4F56"
      :field-value="$t('解决方案')"
      :field-width="70"
      value-color="#313238"
    >
      <template #value>
        <MarkdownViewer
          class="p-0 text-[12px] text-[#313238]"
          :value="data.solutions"
        />
      </template>
    </FieldItem>
  </div>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import { Tag } from 'bkui-vue';
  import { TagThemeEnum } from 'bkui-vue/lib/shared';
  import { useI18n } from 'vue-i18n';
  import { CheckItemOutput } from '~/@types/v1/bkintegrations-kubeinsight';
  import { formatTimeByTimezone, getDurationTime } from '~/common/util';
  interface IProps {
    data: CheckItemOutput;
  }

  const props = defineProps<IProps>();
  const { t } = useI18n();
  const levelMap: Record<
    string,
    {
      label: string;
      theme: TagThemeEnum;
    }
  > = {
    RISK: {
      label: t('致命'),
      theme: TagThemeEnum.DANGER,
    },
    WARN: {
      label: t('预警'),
      theme: TagThemeEnum.WARNING,
    },
    INFO: {
      label: t('已恢复'),
      theme: TagThemeEnum.SUCCESS,
    },
  };

  const durationTime = computed(() => {
    if (!props.data.lastUpdateTimestamp || !props.data.timestamp) return '--';
    return getDurationTime(props.data.lastUpdateTimestamp, props.data.timestamp);
  });
</script>

<style lang="postcss" scoped>
  :deep(.markdown-body) {
    ol {
      padding-left: 0px;
    }
  }
</style>
