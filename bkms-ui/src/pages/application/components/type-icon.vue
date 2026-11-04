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
    <span
      :class="['inline-flex shrink-0 items-center justify-center', classes]"
      :title="$t(curTypeIconData.label)"
    >
      <span
        v-if="displayKind === 'cpp'"
        class="text-[12px] font-bold"
        >C++</span
      >
      <!-- Helm 的遮罩在 sprite use 中会丢失，使用独立 SVG。 -->
      <img
        v-else-if="displayKind === 'helm'"
        alt=""
        class="shrink-0"
        :height="size"
        :src="helmIconUrl"
        :width="size"
      />
      <ColorIcon
        v-else
        class="shrink-0"
        :fill="displayKind === 'bkmsApp' ? '#3A84FF' : undefined"
        :icon="curTypeIconData.icon"
        :size="size"
      />
    </span>
    <span v-if="!$slots.label && showLabel">{{ $t(curTypeIconData.label) }}</span>
    <slot
      v-else
      :label="$t(curTypeIconData.label)"
      name="label"
    ></slot>
  </div>
  <template v-else>
    {{ emptyPlaceholder }}
  </template>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  import helmIconUrl from '~/assets/lang-icons/helm.svg?url';
  import ColorIcon from '~/components/color-icon.vue';
  import { getAppDisplayKind, getAppDisplayLabel } from '~/composables/app-type';

  import type { AppType } from '~/composables/app-type';

  /**
   * @description 若使用slot自定义label，props.type建议直接给出类型而非变量 否则可能会出现type与label不一致的情况
   *
   */

  interface IProps {
    classes?: string;
    emptyPlaceholder?: string;
    framework?: string;
    language?: string;
    showLabel?: boolean;
    size?: number;
    type?: AppType | string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    classes: '',
    showLabel: true,
    size: 20,
    emptyPlaceholder: '--',
  });

  const typeIconMap: Record<string, string> = {
    trpc: 'trpc',
    taf: 'taf',
    helm: 'helm',
    agones: 'agones',
    bkmsApp: 'chuangjianyingyong',
    go: 'go',
    python: 'python',
    nodejs: 'node',
    java: 'java',
  };

  const displayKind = computed(() => getAppDisplayKind(props));
  const curTypeIconData = computed(() => {
    const kind = displayKind.value;
    const icon = typeIconMap[kind];
    if (!icon && kind !== 'cpp') return null;
    return { icon: icon || '', label: getAppDisplayLabel(kind) };
  });
</script>
