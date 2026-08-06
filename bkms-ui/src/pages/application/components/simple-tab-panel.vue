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
  <slot v-if="active || parentTab?.activeID === id"></slot>
</template>
<script lang="ts" setup>
  import { getCurrentInstance, inject, onBeforeMount, onBeforeUnmount } from 'vue';

  import SimpleTabPanel from './simple-tab-panel.vue';

  import type { IProvide } from './simple-tab.vue';

  interface IProps {
    active?: boolean;
    id: number | string;
    name: string;
  }

  const props = defineProps<IProps>();
  const parentTab = inject<IProvide>('simple-tab');

  const instance = getCurrentInstance();

  onBeforeMount(() => {
    if (instance) {
      parentTab?.registry?.(props.id, instance.proxy as InstanceType<typeof SimpleTabPanel>);
    }
  });

  onBeforeUnmount(() => {
    parentTab?.unregistry?.(props.id);
  });
</script>
