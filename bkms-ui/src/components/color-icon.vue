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
  <svg :style="iconSize">
    <use :xlink:href="`${iconPrefix}${icon}`"></use>
  </svg>
</template>

<script lang="ts" setup>
  import { computed } from 'vue';

  interface IProps {
    icon: string;
    iconPrefix?: string;
    size?: number;
  }
  /**
   * @description 用于彩色图标展示，请事先在静态资源添加好彩色图标(example: #bkms-icon-exmpale, #other-icon-prefix-example)
   */
  const props = withDefaults(defineProps<IProps>(), {
    iconPrefix: '#bkms-icon-',
    size: 16,
  });

  const iconSize = computed(() => ({
    height: `${props.size}px`,
    width: `${props.size}px`,
  }));
</script>
