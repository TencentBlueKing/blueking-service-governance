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
  <div :class="['relative', { '!border-[#3A84FF]': checked }]">
    <slot></slot>
    <span
      v-show="checked"
      class="is-checked"
    >
      <Done
        class="nc-check-small"
        :height="28"
        :width="28"
      ></Done>
    </span>
  </div>
</template>
<script setup lang="ts">
  import { ref, watch } from 'vue';

  import { Done } from 'bkui-vue/lib/icon';

  type IProps = {
    active?: boolean;
  };

  const props = defineProps<IProps>();

  const checked = ref(props.active);
  watch(
    () => props.active,
    val => {
      checked.value = val;
    },
  );
</script>
<style scoped lang="postcss">
  .is-checked {
    position: absolute;
    top: 0;
    right: 0;
    width: 0;
    height: 0;
    border-style: solid;
    border-width: 0 42px 42px 0;
    border-color: transparent #3a84ff transparent transparent;
    .nc-check-small {
      position: absolute;
      color: #fff;
      top: 0;
      right: -44px;
    }
  }
</style>
