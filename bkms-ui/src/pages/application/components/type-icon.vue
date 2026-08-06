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
    v-if="curTypeIconData"
    class="flex items-center"
  >
    <i :class="[curTypeIconData.icon, classes]"></i>
    <span v-if="!$slots.label && showLabel">{{ curTypeIconData.label }}</span>
    <slot
      v-else
      :label="curTypeIconData?.label || emptyPlaceholder"
      name="label"
    ></slot>
  </div>
  <template v-else>
    {{ emptyPlaceholder }}
  </template>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import type { AppType } from '~/composables/app-type';

  /**
   * @description 若使用slot自定义label，props.type建议直接给出类型而非变量 否则可能会出现type与label不一致的情况
   *
   */

  interface IProps {
    classes?: string;
    emptyPlaceholder?: string;
    showLabel?: boolean;
    type?: AppType | string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    classes: '',
    showLabel: true,
    emptyPlaceholder: '--',
  });

  const typeIconMap: Record<
    AppType,
    {
      icon: string;
      label: string;
    }
  > = {
    trpc: {
      icon: 'bkms-icon bkms-icon-trpc text-[#1B44C8] text-[8px]',
      label: 'trpc',
    },
    taf: {
      icon: 'bkms-icon bkms-icon-taf text-[#1b44c8] text-[20px]',
      label: 'taf',
    },
    helm: {
      icon: 'bkms-icon bkms-icon-HelmCharts text-[#0F1689] text-[16px]',
      label: 'helm',
    },
    agones: {
      icon: 'bkms-icon bkms-icon-agones text-[#0F1689] text-[16px]',
      label: 'agones',
    },
  };

  const curTypeIconData = computed(() => {
    const result = props?.type && typeIconMap?.[props.type as AppType] ? typeIconMap[props.type as AppType] : null;
    return result;
  });
</script>
