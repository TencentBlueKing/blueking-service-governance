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
  <!-- 无规则时不展示卡片，仅保留侧滑供「新增配置项」首次配置 -->
  <BkmsContent
    v-if="hasData"
    class="dev-mode-form-wrapper"
    :collapsible="true"
  >
    <template #title>
      <span class="text-[#313238]">{{ $t('开发模式') }}</span>
      <span class="ml-[8px] flex items-center font-normal text-[12px] text-[#979BA5]">
        <i class="bkms-icon bkms-icon-circle-info mr-[4px] text-[14px]"></i>
        {{ $t('支持通过 bkms-cli 上传二进制的方式热更新服务，更新过程不会重启容器实例。') }}
      </span>
    </template>
    <template #action>
      <span
        v-bk-tooltips="{
          content: $t('已无可新增的适用范围'),
          disabled: !isAddDisabled,
        }"
      >
        <Button
          class="!h-[22px] !px-[8px] !text-[12px]"
          :disabled="isAddDisabled"
          outline
          theme="primary"
          @click.stop="handleCreate"
        >
          <i class="bkms-icon bkms-icon-jiahao text-[14px] mr-[2px]"></i>
          {{ $t('新增规则') }}
        </Button>
      </span>
    </template>
    <div class="bg-[#FFF]">
      <Table
        v-bkloading="{ loading }"
        class="dev-mode-table"
        :data="ruleList"
        :row-config="{ isHover: true }"
        :show-overflow="false"
      >
        <TableColumn
          field="enabled"
          :label="$t('开发模式')"
          min-width="180"
        >
          <template #default="{ row }: { row: DevModeRuleOutputObj }">
            <Tag
              class="h-[20px]"
              :theme="row.spec?.enabled ? 'success' : 'danger'"
            >
              {{ row.spec?.enabled ? $t('开启') : $t('关闭') }}
            </Tag>
          </template>
        </TableColumn>
        <TableColumn
          field="envTypes"
          :label="$t('适用范围')"
          min-width="220"
        >
          <template #default="{ row }: { row: DevModeRuleOutputObj }">
            <div
              v-if="row.envTypes?.length"
              class="flex flex-wrap gap-[8px]"
            >
              <Tag
                v-for="envType in row.envTypes"
                :key="envType"
                class="h-[20px] !justify-center"
                :class="envTypeTagClassMap[envType]"
              >
                {{ envTypeMap[envType]?.name || envType }}
              </Tag>
            </div>
            <span v-else>--</span>
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('操作')"
          :width="230"
        >
          <template #default="{ row }: { row: DevModeRuleOutputObj }">
            <div class="flex items-center gap-[12px]">
              <Button
                text
                theme="primary"
                @click="handleEdit(row)"
              >
                {{ $t('编辑') }}
              </Button>
              <PopConfirm
                :content="$t('确认删除该规则吗？')"
                trigger="click"
                width="288"
                @confirm="handleDelete(row)"
              >
                <Button
                  text
                  theme="primary"
                >
                  {{ $t('删除') }}
                </Button>
              </PopConfirm>
            </div>
          </template>
        </TableColumn>
      </Table>
    </div>
  </BkmsContent>

  <DevModeRuleSideslider
    v-model:is-show="sliderVisible"
    :occupied-env-types="occupiedEnvTypes"
    :rule="editingRule"
    :scene="sliderScene"
    @success="fetchRuleList"
  />
</template>

<script lang="ts" setup>
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Message, PopConfirm, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppSpecService } from '~/api/modules/v1';
  import BkmsContent from '~/components/bkms-content.vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useSpaceStore } from '~/stores/space';

  import { DEV_MODE_SUPPORTED_ENV_TYPES } from './app-spec-scope';
  import DevModeRuleSideslider from './dev-mode-rule-sideslider.vue';

  import type { AppSpecSliderScene } from './app-spec-scope';
  import type { DevModeRuleOutputObj } from '~/@types/v1/app-spec';

  const { t } = useI18n();
  const spaceStore = useSpaceStore();

  const loading = ref(false);
  const ruleList = ref<DevModeRuleOutputObj[]>([]);
  const sliderVisible = ref(false);
  const sliderScene = ref<AppSpecSliderScene>('addConfig');
  const editingRule = ref<DevModeRuleOutputObj | null>(null);

  /** 有规则数据时才展示配置项卡片 */
  const hasData = computed(() => ruleList.value.length > 0);

  const emit = defineEmits<{
    'has-data-change': [value: boolean];
    'loading-change': [value: boolean];
  }>();

  /** 已被其他规则占用的环境类型（编辑时排除当前规则已选择的环境） */
  const occupiedEnvTypes = computed(() => {
    const editingEnvTypes = new Set(editingRule.value?.envTypes || []);
    return ruleList.value.flatMap(item => (item.envTypes || []).filter(envType => !editingEnvTypes.has(envType)));
  });

  /** dev-mode 支持的环境类型都已配置时，不允许再新增（以列表全部规则为准，不受编辑态影响） */
  const isAddDisabled = computed(() => {
    const selectedEnvTypes = new Set(ruleList.value.flatMap(item => item.envTypes || []));
    return DEV_MODE_SUPPORTED_ENV_TYPES.every(envType => selectedEnvTypes.has(envType));
  });

  /**
   * 查询工作空间开发模式默认规则列表
   * GET /workspaces/{workspaceID}/app-spec/dev-mode
   */
  async function fetchRuleList() {
    if (!spaceStore.currentSpace) {
      ruleList.value = [];
      return;
    }
    loading.value = true;
    try {
      const result = await AppSpecService.listWorkspaceAppSpecDevModeRules({
        workspaceID: spaceStore.currentSpace,
      });
      ruleList.value = Array.isArray(result) ? result : [];
    } catch {
      // 列表拉取失败：保持空列表隐藏卡片；错误提示由全局拦截器统一弹出。
      ruleList.value = [];
    } finally {
      loading.value = false;
    }
  }

  /** 卡片内「新增规则」 */
  function handleCreate() {
    openSlider('createRule');
  }

  /** 确认删除后调用接口，成功则刷新列表 */
  async function handleDelete(row: DevModeRuleOutputObj) {
    if (!spaceStore.currentSpace || !row.id) return;
    await AppSpecService.deleteWorkspaceAppSpecDevModeRule({
      workspaceID: spaceStore.currentSpace,
      ruleID: row.id,
    });
    Message({ theme: 'success', message: t('删除成功') });
    await fetchRuleList();
  }

  function handleEdit(row: DevModeRuleOutputObj) {
    openSlider('editRule', row);
  }

  /** 打开开发模式侧滑，三种入口共用同一组件 */
  function openSlider(scene: AppSpecSliderScene, rule: DevModeRuleOutputObj | null = null) {
    if (scene !== 'editRule' && isAddDisabled.value) return;
    sliderScene.value = scene;
    editingRule.value = rule;
    sliderVisible.value = true;
  }

  // 空间 ID 就绪后再拉列表
  watch(
    () => spaceStore.currentSpace,
    spaceID => {
      if (spaceID) {
        fetchRuleList();
      } else {
        ruleList.value = [];
      }
    },
    { immediate: true },
  );

  watch(hasData, value => emit('has-data-change', value), { immediate: true });

  watch(loading, value => emit('loading-change', value), { immediate: true });

  defineExpose({
    /** 供父级菜单「添加」打开「新增配置项 | 开发模式」侧滑 */
    openCreate: () => openSlider('addConfig'),
  });
</script>

<style lang="postcss" scoped>
  .dev-mode-form-wrapper {
    box-shadow: 0 2px 4px 0 #1919290d;
  }

  :deep(.bkms-content-title) {
    background-color: #eaebf0;
    height: 42px;
  }

  /* 表头：背景白色 + 默认文字颜色 */
  :deep(.dev-mode-table .vxe-header--column) {
    background-color: #fff;
    color: #4d4f56;
  }

  /* 表格正文：默认文字颜色 */
  :deep(.dev-mode-table .vxe-body--column .vxe-cell) {
    color: #4d4f56;
  }
</style>
