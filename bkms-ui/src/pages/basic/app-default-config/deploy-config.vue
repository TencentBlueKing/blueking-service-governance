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
  <div class="h-full">
    <div
      v-if="isLoading"
      class="flex flex-col gap-[16px] pb-[8px]"
    >
      <div
        v-for="index in 2"
        :key="index"
        class="overflow-hidden rounded-[2px] bg-[#fff] shadow-[0_2px_4px_0_#1919290d]"
      >
        <div class="flex h-[42px] items-center justify-between bg-[#f0f1f5] px-[16px]">
          <Layout.shape
            class="shrink-0"
            height="20px"
            type="rect"
            width="96px"
          />
          <Layout.shape
            height="22px"
            type="rect"
            width="74px"
          />
        </div>
        <div class="bg-[#fff] pl-[16px] pr-[24px]">
          <div
            class="grid min-h-[44px] items-center gap-[24px] border-b border-[#f0f1f5]"
            :style="skeletonTableStyle"
          >
            <Layout.shape
              v-for="(width, columnIndex) in skeletonHeaderWidths"
              :key="`header-${index}-${columnIndex}`"
              height="16px"
              type="rect"
              :width="`${width}px`"
            />
          </div>
          <div
            v-for="(row, rowIndex) in skeletonRowWidths"
            :key="`row-${index}-${rowIndex}`"
            class="grid min-h-[42px] items-center gap-[24px] border-b border-[#f0f1f5]"
            :style="skeletonTableStyle"
          >
            <Layout.shape
              v-for="(width, columnIndex) in row"
              :key="`row-${index}-${rowIndex}-${columnIndex}`"
              height="16px"
              type="rect"
              :width="`${width}px`"
            />
          </div>
        </div>
      </div>

      <div class="add-config-item pointer-events-none">
        <Layout.shape
          height="18px"
          type="rect"
          width="108px"
        />
      </div>
    </div>

    <div
      v-show="!isLoading"
      class="h-full overflow-auto flex flex-col gap-[16px]"
    >
      <ResourcesForm
        ref="resourcesFormRef"
        @has-data-change="hasResources = $event"
        @loading-change="resourcesLoading = $event"
      />
      <DevModeForm
        ref="devModeFormRef"
        @has-data-change="hasDevMode = $event"
        @loading-change="devModeLoading = $event"
      />

      <!-- 新增配置项：整行可点击打开菜单；全部配齐后禁用并在整行展示 tips -->
      <Popover
        ref="addConfigPopoverRef"
        class="w-full"
        :disabled="allConfigsAdded"
        ext-cls="add-config-popover"
        placement="top"
        theme="light"
        trigger="click"
        :width="420"
      >
        <div
          v-bk-tooltips="{
            content: $t('暂无配置可添加'),
            disabled: !allConfigsAdded,
          }"
          class="add-config-item"
          :class="{ 'is-disabled': allConfigsAdded }"
        >
          <span class="bkms-icon bkms-icon-plus-circle-shape mr-[5px]"></span>
          <span>{{ $t('新增配置项') }}</span>
        </div>
        <template #content>
          <div class="add-config-menu">
            <div class="add-config-menu__title">{{ $t('请选择要配置的项') }}</div>
            <div
              v-for="item in availableConfigItems"
              :key="item.key"
              class="add-config-menu__row-wrapper"
              @click="handleAddConfig(item.key)"
            >
              <div class="add-config-menu__row">
                <div class="add-config-menu__info">
                  <div class="add-config-menu__label">{{ item.label }}</div>
                  <div class="add-config-menu__desc">{{ item.description }}</div>
                </div>
                <div class="add-config-menu__action">
                  <i class="bkms-icon bkms-icon-jiahao text-[14px] mr-[6px]"></i>
                  <span>{{ $t('添加') }}</span>
                </div>
              </div>
            </div>
          </div>
        </template>
      </Popover>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref } from 'vue';

  import { Popover } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import Layout from '~/components/skeleton/skeleton-layout';

  import DevModeForm from './components/dev-mode-form.vue';
  import ResourcesForm from './components/resources-form.vue';

  /** 本期可新增的配置项类型 */
  type ConfigItemKey = 'devMode' | 'resources';

  /** 菜单中展示的配置项 */
  interface ConfigItemOption {
    description: string;
    key: ConfigItemKey;
    label: string;
  }

  const { t } = useI18n();

  const resourcesFormRef = ref<InstanceType<typeof ResourcesForm>>();
  const devModeFormRef = ref<InstanceType<typeof DevModeForm>>();
  const addConfigPopoverRef = ref<InstanceType<typeof Popover>>();

  /** 各配置项是否已有规则数据（有数据才展示对应卡片，菜单中不再出现） */
  const hasResources = ref(false);
  const hasDevMode = ref(false);

  /** 子组件查询状态：内容始终挂载，父层根据状态切换骨架 */
  const resourcesLoading = ref(true);
  const devModeLoading = ref(true);
  const isLoading = computed(() => resourcesLoading.value || devModeLoading.value);

  const skeletonTableStyle = {
    gridTemplateColumns: 'repeat(3, minmax(160px, 1fr)) minmax(88px, 0.8fr)',
  };

  const skeletonHeaderWidths = [88, 52, 76, 52];

  const skeletonRowWidths = [
    [176, 42, 58, 52],
    [152, 36, 76, 52],
  ];

  /** 本期仅开放资源规格、开发模式 */
  const allConfigItems = computed<ConfigItemOption[]>(() => [
    {
      key: 'resources',
      label: t('资源规格'),
      description: t('实例数、CPU、内存等配置'),
    },
    {
      key: 'devMode',
      label: t('开发模式'),
      description: t('支持通过 bkms-cli 上传二进制的方式热更新服务'),
    },
  ]);

  const hasConfigMap = computed<Record<ConfigItemKey, boolean>>(() => ({
    resources: hasResources.value,
    devMode: hasDevMode.value,
  }));

  /** 尚未添加的配置项，已添加的不再出现在菜单中 */
  const availableConfigItems = computed(() => allConfigItems.value.filter(item => !hasConfigMap.value[item.key]));

  /** 全部可配配置项都已添加 */
  const allConfigsAdded = computed(() => availableConfigItems.value.length === 0);

  /** 点击「添加」后打开对应类型侧滑，保存成功后卡片才会出现 */
  async function handleAddConfig(key: ConfigItemKey) {
    addConfigPopoverRef.value?.hide();
    await nextTick();
    if (key === 'resources') {
      resourcesFormRef.value?.openCreate();
      return;
    }
    devModeFormRef.value?.openCreate();
  }
</script>

<style lang="postcss" scoped>
  .add-config-item {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 52px;
    color: #3a84ff;
    font-size: 14px;
    background-color: #ffffff;
    border: 1px dashed #c4c6cc;
    border-radius: 2px;
    box-shadow: 0 2px 4px 0 #1919290d;
    cursor: pointer;

    &:hover {
      background-color: #f6f7fb;
    }

    /* 全部配置项已添加：整行禁用，不可打开菜单 */
    &.is-disabled {
      color: #c4c6cc;
      cursor: not-allowed;
      background-color: #fafbfd;
      border-color: #dcdee5;
      box-shadow: none;
    }
  }
</style>

<!-- Popover 内容会 Teleport 到 body，样式必须写在非 scoped 中才能命中 -->
<style lang="postcss">
  .bk-popover.bk-pop2-content.add-config-popover {
    padding: 0 !important;
    font-size: 12px;
    background: #fff;
    border: none;
    box-shadow: 0 0 10px 0 #0000001a;
  }

  .add-config-menu {
    width: 100%;
    background: #fff;
    border-radius: 8px;
  }

  .add-config-menu__title {
    padding: 16px;
    color: #313238;
    font-size: 14px;
    font-weight: 700;
    line-height: 22px;
  }

  .add-config-menu__row-wrapper {
    padding: 0 16px;
    cursor: pointer;

    &:hover {
      background-color: #f6f7fb;
    }
  }

  .add-config-menu__row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 0;
    border-top: 1px solid #eaebf0;
  }

  .add-config-menu__info {
    min-width: 0;
    flex: 1;
    font-size: 12px;
  }

  .add-config-menu__label {
    color: #4d4f56;
    font-weight: 700;
    line-height: 20px;
  }

  .add-config-menu__desc {
    color: #979ba5;
    line-height: 20px;
  }

  .add-config-menu__action {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    color: #3a84ff;
    font-size: 12px;
    line-height: 20px;
  }
</style>
