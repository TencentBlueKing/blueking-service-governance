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
  <div class="inline-flex items-center">
    <Button
      :class="{ 'mr-[1px]': showFeatureDeploy }"
      :style="{ borderRadius: showFeatureDeploy ? '2px 0 0 2px' : '2px' }"
      theme="primary"
      @click="emits('deploy')"
    >
      <slot name="label">{{ label }}</slot>
    </Button>
    <Dropdown
      v-if="showFeatureDeploy"
      placement="bottom-end"
      :popover-options="{ boundary: 'body', clickContentAutoHide: true }"
      trigger="click"
    >
      <template #default="{ popoverShow }">
        <Button
          class="!min-w-[32px] !px-0"
          style="border-radius: 0 2px 2px 0"
          theme="primary"
        >
          <AngleDownLine
            class="text-[12px] transition-transform duration-240"
            :class="{ 'rotate-180': popoverShow }"
          />
        </Button>
      </template>
      <template #content>
        <Dropdown.DropdownMenu>
          <Dropdown.DropdownItem @click="emits('feature-deploy')">
            {{ $t('特性部署') }}
          </Dropdown.DropdownItem>
        </Dropdown.DropdownMenu>
      </template>
    </Dropdown>
  </div>
</template>

<script lang="ts" setup>
  import { Button, Dropdown } from 'bkui-vue';
  import { AngleDownLine } from 'bkui-vue/lib/icon';

  withDefaults(
    defineProps<{
      label: string;
      showFeatureDeploy?: boolean;
    }>(),
    {
      showFeatureDeploy: false,
    },
  );

  const emits = defineEmits<{
    (e: 'deploy'): void;
    (e: 'feature-deploy'): void;
  }>();
</script>
