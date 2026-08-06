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

<script setup lang="ts">
  import { computed, inject, toRefs } from 'vue';

  import { parseCss } from '~/common/util';

  export type ShapeSize = 'default' | 'large' | 'small';
  export type SkeletonShape = 'circle' | 'rect';

  const props = withDefaults(
    defineProps<{
      height?: number | string;
      loading?: boolean;
      type?: SkeletonShape;
      width?: number | string;
    }>(),
    {
      type: 'rect',
      width: 88,
      height: 32,
      loading: true,
    },
  );

  const { type, width, height, loading } = toRefs(props);
  const theme: 'gray' | 'white' = inject('theme', 'white');
  const defaultShapeClass: Record<SkeletonShape, string[]> = {
    circle: ['rounded-full'],
    rect: ['rounded'],
  };
  const classes = computed(() => [...defaultShapeClass[type.value]]);

  const themeStyleMap: Record<
    string,
    {
      /** 默认颜色 */
      defaultColor: string;
      /** 闪动颜色 */
      flickerColor: string;
    }
  > = {
    white: {
      defaultColor: '#EBECF3',
      flickerColor: '#F6F7FB',
    },
    gray: {
      defaultColor: '#EBECF3',
      flickerColor: '#F6F7FB',
    },
  };

  const themeStyle = computed(() => themeStyleMap[theme]);
  const backgroundImageByTheme = computed(
    () =>
      `linear-gradient(90deg, ${themeStyle.value.defaultColor} 25%, ${themeStyle.value.flickerColor} 37%, ${themeStyle.value.defaultColor} 63%)`,
  );
</script>
<template>
  <template v-if="loading">
    <div
      class="bkms-skeleton inline-block align-top"
      :class="classes"
      :style="{
        width: parseCss(width || 0),
        height: parseCss(height || 0),
        backgroundImage: backgroundImageByTheme,
      }"
    >
      <slot name="content"></slot>
    </div>
  </template>
  <template v-else>
    <slot></slot>
  </template>
</template>
