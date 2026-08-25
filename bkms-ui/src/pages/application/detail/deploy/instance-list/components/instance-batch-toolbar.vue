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
  <div class="flex items-center gap-[8px]">
    <Button
      class="bg-[#fff]"
      @click="emit('monitor')"
    >
      {{ $t('监控') }}
    </Button>
    <Button
      v-bk-tooltips="{
        content: grayTooltipContent,
        disabled: isGrayTooltipDisabled,
      }"
      class="bg-[#fff]"
      :disabled="isGrayDisabled"
      @click="emit('gray')"
    >
      {{ $t('灰度') }}
    </Button>
    <Popover
      ref="removeDeployPopoverRef"
      :disabled="isDeletePopoverDisabled"
      ext-cls="remove-deploy-shortcut-popover"
      placement="top"
      theme="dark"
      trigger="hover"
    >
      <span class="inline-flex">
        <Button
          class="bg-[#fff]"
          :disabled="isDeleteDisabled"
          @click="emit('delete')"
        >
          {{ $t('删除') }}
        </Button>
      </span>
      <template #content>
        <!-- 单环境、多环境全量选择实例，删除提示 -->
        <div
          v-if="isRemoveDeployShortcutVisible"
          class="flex items-center gap-[12px] whitespace-nowrap text-[12px] leading-[20px]"
        >
          <span>{{ $t('不支持全量删除实例，如需全量删除请使用') }}</span>
          <Button
            text
            theme="primary"
            @click.stop="handleRemoveDeployClick"
          >
            {{ $t('移除部署') }}
          </Button>
        </div>
        <span
          v-else-if="disableDelete"
          class="whitespace-nowrap text-[12px] leading-[20px]"
        >
          {{ $t('跨页全选后暂不支持批量删除实例') }}
        </span>
      </template>
    </Popover>
    <Button
      v-bk-tooltips="{
        content: $t('跨页全选后暂不支持执行管理命令'),
        disabled: !selectedCount || !disableAdminCommand,
      }"
      class="bg-[#fff]"
      :disabled="!selectedCount || disableAdminCommand"
      @click="emit('admin-command')"
    >
      {{ $t('管理命令') }}
    </Button>
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Button, Popover } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  const props = withDefaults(
    defineProps<{
      canGrayDeploy: boolean; // 是否允许灰度部署
      disableAdminCommand?: boolean; // 是否禁用管理命令（跨页全选时）
      disableDelete?: boolean; // 是否禁用删除（跨页全选时）
      disableGray?: boolean; // 是否禁用灰度（跨页全选/多环境选中时）
      grayDisabledTip?: string; // 灰度被能力限制禁用时的优先提示
      isAllInstancesSelected: boolean; // 是否已全选所有实例
      selectedCount: number; // 已选实例数量
      showRemoveDeployShortcut?: boolean; // 是否展示“移除部署”快捷入口
    }>(),
    {
      grayDisabledTip: '',
      showRemoveDeployShortcut: false,
    },
  );

  const emit = defineEmits<{
    'admin-command': [];
    delete: [];
    gray: [];
    monitor: [];
    'remove-deploy': [];
  }>();

  const { t } = useI18n();
  const removeDeployPopoverRef = ref<InstanceType<typeof Popover> | null>(null);

  // 是否有任意实例被选中
  const hasSelection = computed(() => props.selectedCount > 0);

  // 灰度按钮是否禁用：未选中 / 跨页全选禁用 / 不允许灰度部署
  const isGrayDisabled = computed(() => !props.selectedCount || props.disableGray || !props.canGrayDeploy);

  // 灰度按钮 tooltip 文案：优先使用能力受限提示，其次按禁用原因给出对应文案
  const grayTooltipContent = computed(
    () =>
      props.grayDisabledTip ||
      (props.disableGray ? t('多环境跨页全选后暂不支持批量灰度') : t('仅支持实例状态为 Running、Pending 的实例')),
  );

  // 灰度 tooltip 是否禁用：存在自定义提示时始终展示；否则仅在按钮非可用（未选中或被灰度限制禁用）时不展示
  const isGrayTooltipDisabled = computed(
    () => !props.grayDisabledTip && (!props.selectedCount || (!props.disableGray && props.canGrayDeploy)),
  );

  // “移除部署”快捷入口可见条件：开启快捷入口 + 有选中 + 删除未禁用 + 已全选
  const isRemoveDeployShortcutVisible = computed(
    () => props.showRemoveDeployShortcut && hasSelection.value && !props.disableDelete && props.isAllInstancesSelected,
  );
  // 删除按钮禁用：无选中 或 外部禁用 或 已全选
  const isDeleteDisabled = computed(() => !hasSelection.value || props.disableDelete || props.isAllInstancesSelected);

  // 删除 Popover 是否不触发：无选中 或（未禁用删除且未全选，此时不需要提示）
  const isDeletePopoverDisabled = computed(
    () => !hasSelection.value || (!props.disableDelete && !isRemoveDeployShortcutVisible.value),
  );

  // 点击“移除部署”快捷入口：先隐藏 Popover，再触发事件
  function handleRemoveDeployClick() {
    removeDeployPopoverRef.value?.hide();
    emit('remove-deploy');
  }
</script>
