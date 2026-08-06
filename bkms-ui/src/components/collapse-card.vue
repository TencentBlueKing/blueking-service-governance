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
  <Card
    v-model:collapse-status="collapseStatus"
    class="collapse-card"
    is-collapse
  >
    <template #icon>
      <slot name="icon">
        <RightShape
          :class="['mr-[8px] transition duration-300', collapseStatus ? 'rotate-90' : '']"
          fill="#313238"
          :height="12"
          :width="12"
          @click.stop="collapseStatus = !collapseStatus"
        />
      </slot>
    </template>
    <template #header>
      <slot name="header">
        <FlexRow class="w-full">
          <template #left>
            <div
              class="cursor-pointer"
              @click.stop="collapseStatus = !collapseStatus"
            >
              <slot name="header-left" />
            </div>
          </template>
          <template #right>
            <slot name="header-right" />
          </template>
        </FlexRow>
      </slot>
    </template>
    <slot />
  </Card>
</template>

<script lang="ts" setup>
  import { Card } from 'bkui-vue';
  import { RightShape } from 'bkui-vue/lib/icon';

  const collapseStatus = defineModel<boolean>({ default: true });
</script>

<style lang="postcss" scoped>
  .collapse-card {
    & :deep(.bk-card-head) {
      margin: 0 16px;
      padding: unset;
      width: calc(100% - 32px);
      border-color: #dcdee5;
      cursor: auto;
    }
    & :deep(.bk-card-body) {
      padding: 12px 16px;
    }
    padding-bottom: 0px;
  }
</style>
