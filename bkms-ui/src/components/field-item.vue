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
    class="flex items-center w-full"
    :style="containerStyle"
  >
    <template v-if="$slots.field">
      <slot name="field"></slot>
    </template>
    <template v-else>
      <div
        class="shrink-0"
        :style="fieldStyle"
      >
        {{ fieldValue }}{{ fieldSplitCode }}
      </div>
    </template>

    <template v-if="$slots.value">
      <slot name="value"></slot>
    </template>
    <template v-else>
      <template v-if="isLabelOverflow">
        <OverflowTitle
          :style="labelStyle"
          type="tips"
        >
          {{ value || emptyPlaceholder }}
        </OverflowTitle>
      </template>
      <template v-else>
        <span :style="labelStyle">
          {{ value || emptyPlaceholder }}
        </span>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
  import type { CSSProperties } from 'vue';
  import { computed } from 'vue';

  import { OverflowTitle } from 'bkui-vue';
  import { parseCss } from '~/common/util';
  import { i18n } from '~/modules/i18n';

  interface IProps {
    /** 组件总高度(默认30) */
    containerHeight?: number | string;
    /** 右侧值(value)为空时展示(默认'--') */
    emptyPlaceholder?: string;
    /** 左侧字段(key)文本颜色(默认#979BA5) */
    fieldColor?: string;
    /** 左侧字段(key)文本方向(默认right) */
    fieldDirection?: 'center' | 'left' | 'right';
    /** 左侧字段(key)文字大小(默认12) */
    fieldSize?: number | string;
    /** 分隔符(默认：) */
    fieldSplitCode?: string;
    /** 左侧字段(key)展示 */
    fieldValue?: number | string;
    /** 左侧字段(key)文本宽度(默认100，英文环境下自动翻倍) */
    fieldWidth?: number | string;
    /** 右侧值(value)是否使用OverflowTitle(默认使用) */
    isLabelOverflow?: boolean;
    /** 右侧值(value)展示 */
    value?: number | string;
    /** 右侧值(value)文字颜色 */
    valueColor?: string;
    /** 右侧值(value)最大宽度(默认220) */
    valueMaxWidth?: number | string;
    /** 右侧值(value)文字大小 */
    valueSize?: number | string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    containerHeight: 30,
    fieldSplitCode: '：',
    fieldDirection: 'right',
    fieldWidth: 100,
    fieldColor: '#979BA5',
    fieldSize: 12,
    isLabelOverflow: true,
    emptyPlaceholder: '--',
    valueMaxWidth: 220,
    valueSize: 12,
    valueColor: '',
  });

  // 根据语言自动调整 fieldWidth（英文翻倍）
  const computedFieldWidth = computed(() => {
    const baseWidth = typeof props.fieldWidth === 'number' ? props.fieldWidth : parseInt(String(props.fieldWidth), 10);
    return i18n.global.locale.value === 'en-US' ? baseWidth * 2 : baseWidth;
  });

  const containerStyle = computed<CSSProperties>(() => ({
    height: parseCss(props.containerHeight),
  }));

  const fieldStyle = computed<CSSProperties>(() => ({
    textAlign: props.fieldDirection,
    width: parseCss(computedFieldWidth.value),
    color: props.fieldColor,
    fontSize: parseCss(props.fieldSize),
  }));

  const labelStyle = computed<CSSProperties>(() => ({
    maxWidth: parseCss(props.valueMaxWidth),
    fontSize: parseCss(props.valueSize),
    color: parseCss(props.valueColor),
  }));
</script>
