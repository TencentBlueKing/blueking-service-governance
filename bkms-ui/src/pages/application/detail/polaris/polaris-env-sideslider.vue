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
  <Sideslider
    v-model:is-show="isShow"
    render-directive="if"
    :width="960"
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader
        :title="$t('关联环境')"
        :title-size="16"
      >
        <span class="truncate text-[#979BA5]">
          {{ currentConfig?.polarisName || currentConfig?.name || '--' }}
        </span>
      </DividerHeader>
    </template>

    <div class="px-[24px] py-[20px]">
      <section>
        <div class="mb-[12px] flex items-center">
          <strong class="text-[14px] text-[#313238]">{{ $t('已部署环境') }}（{{ deployedRows.length }}）</strong>
          <span class="text-[#979BA5]">
            {{ $t('权重调整后立即同步到该环境全部实例；健康实例数与总权重来自北极星，约有 15 秒延迟') }}
          </span>
          <Button
            class="ml-[8px]"
            :disabled="loading"
            text
            theme="primary"
            @click="loadEnvInstanceStats"
            @mousedown.stop
          >
            <i class="bkms-icon bkms-icon-refresh mr-[2px]"></i>
            {{ $t('刷新') }}
          </Button>
        </div>
        <Table
          v-bkloading="{ loading }"
          :data="deployedRows"
        >
          <template #empty>
            <TableException />
          </template>
          <TableColumn
            field="displayName"
            :label="$t('目标环境')"
            min-width="220"
          >
            <template #default="{ row }">
              <div class="flex items-center">
                <span class="truncate text-[#3A84FF]">{{ row.displayName }}</span>
                <Tag
                  v-if="row.envTypeName"
                  class="ml-[8px] shrink-0"
                  :class="row.envTypeClass"
                  size="small"
                >
                  {{ row.envTypeName }}
                </Tag>
              </div>
            </template>
          </TableColumn>
          <TableColumn
            field="weight"
            :label="$t('单实例权重')"
            min-width="180"
          >
            <template #default="{ row }">
              <span>{{ row.weight }}</span>
              <Tag
                v-if="row.weightOverriddenInstanceCount > 0"
                v-bk-tooltips="
                  $t('该环境下有 {count} 个实例在实例列表中单独调整过权重；总权重按各实例实际权重求和', {
                    count: row.weightOverriddenInstanceCount,
                  })
                "
                class="ml-[8px]"
                size="small"
                theme="warning"
              >
                {{ $t('部分实例已覆盖') }}
              </Tag>
            </template>
          </TableColumn>
          <TableColumn
            field="healthyCount"
            :label="$t('健康实例数')"
            min-width="120"
          >
            <template #header>
              <span class="inline-flex items-center">
                {{ $t('健康实例数') }}
                <i
                  v-bk-tooltips="$t('来自北极星实时数据，约 15 秒延迟')"
                  class="bkms-icon bkms-icon-circle-info ml-[4px] cursor-pointer text-[14px] text-[#C4C6CC]"
                ></i>
              </span>
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('总权重（占比）')"
            min-width="220"
          >
            <template #header>
              <span class="inline-flex items-center">
                {{ $t('总权重（占比）') }}
                <i
                  v-bk-tooltips="
                    $t('按各健康实例的实际权重逐个求和，不是「单实例权重 × 健康实例数」；数据来自北极星，约 15 秒延迟')
                  "
                  class="bkms-icon bkms-icon-circle-info ml-[4px] cursor-pointer text-[14px] text-[#C4C6CC]"
                ></i>
              </span>
            </template>
            <template #default="{ row }">
              <span class="text-[#313238]">{{ getRowTotalWeight(row) }}</span>
              <span class="ml-[6px] text-[#979BA5]">（{{ getRowPercent(row) }}）</span>
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('操作')"
            width="140"
          >
            <template #default="{ row }">
              <Button
                text
                theme="primary"
                @click="handleShowWeightDialog(row)"
              >
                {{ $t('调整权重') }}
              </Button>
            </template>
          </TableColumn>
        </Table>
      </section>

      <section class="mt-[24px]">
        <div class="mb-[12px] flex items-center">
          <strong class="text-[14px] text-[#313238]">{{ $t('未部署环境') }}（{{ undeployedRows.length }}）</strong>
          <span class="text-[#979BA5]">{{ $t('权重在该环境首次部署时生效') }}</span>
        </div>
        <Table :data="undeployedRows">
          <template #empty>
            <TableException />
          </template>
          <TableColumn
            field="displayName"
            :label="$t('目标环境')"
            min-width="360"
          >
            <template #default="{ row }">
              <div class="flex items-center">
                <span class="truncate text-[#3A84FF]">{{ row.displayName }}</span>
                <Tag
                  v-if="row.envTypeName"
                  class="ml-[8px] shrink-0"
                  :class="row.envTypeClass"
                  size="small"
                >
                  {{ row.envTypeName }}
                </Tag>
              </div>
            </template>
          </TableColumn>
          <TableColumn
            field="weight"
            :label="$t('单实例权重')"
            min-width="260"
          />
          <TableColumn
            :label="$t('操作')"
            width="140"
          >
            <template #default="{ row }">
              <Button
                text
                theme="primary"
                @click="handleShowWeightDialog(row)"
              >
                {{ $t('调整权重') }}
              </Button>
            </template>
          </TableColumn>
        </Table>
      </section>
    </div>
  </Sideslider>

  <!-- 权重调整弹窗 -->
  <PolarisEnvWeightDialog
    v-model:is-show="weightDialogVisible"
    :deployed="editingEnv?.deployed ?? false"
    :display-name="editingEnv?.displayName || ''"
    :env-type="editingEnv?.envType"
    :healthy-count="editingEnv?.healthyCount ?? 0"
    :loading="submitting"
    :old-env-total-weight="editingEnv?.healthyInstanceWeight ?? 0"
    :total-healthy-instance-weight="totalHealthyInstanceWeight"
    :weight="editingEnv?.weight ?? DEFAULT_ENV_WEIGHT"
    :weight-overridden-instance-count="editingEnv?.weightOverriddenInstanceCount ?? 0"
    @confirm="handleConfirmWeight"
  />
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Message, Sideslider, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { PolarisConfigService } from '~/api/modules/v1';
  import DividerHeader from '~/components/divider-header.vue';
  import TableException from '~/components/table-exception.vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import { useAppDetail } from '~/stores/app-detail';

  import PolarisEnvWeightDialog from './polaris-env-weight-dialog.vue';

  import type { EnvOutput } from '~/@types/v1/env';
  import type { EnvInstanceStatsOutput, PolarisConfigOutputObj } from '~/@types/v1/polaris-config';

  interface EnvWeightRow {
    deployed: boolean;
    displayName: string;
    envType?: string;
    envTypeClass?: string;
    envTypeName?: string;
    healthyCount: number;
    healthyInstanceWeight: number;
    name: string;
    weight: number;
    weightOverriddenInstanceCount: number;
  }

  const DEFAULT_ENV_WEIGHT = 100;

  const props = defineProps<{
    config?: PolarisConfigOutputObj;
    envList: EnvOutput[];
  }>();

  const emit = defineEmits<{
    (e: 'updated', config: PolarisConfigOutputObj): void;
  }>();

  const isShow = defineModel<boolean>('isShow', { default: false });
  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const currentConfig = ref<PolarisConfigOutputObj>();
  const envInstanceStats = ref<Record<string, EnvInstanceStatsOutput>>({});
  const totalHealthyInstanceWeight = ref(0);
  const loading = ref(false);
  const loadFailed = ref(false);
  const weightDialogVisible = ref(false);
  const editingEnv = ref<EnvWeightRow>();
  const submitting = ref(false);

  /** 每次加载或关闭侧栏时递增，用于识别并丢弃已过期的异步请求结果。 */
  let loadVersion = 0;

  /** 将环境元数据转换为索引，避免构造每一行时重复遍历环境列表。 */
  const envMap = computed(() => new Map(props.envList.map(env => [env.name, env])));

  /** 保持 scope 的配置顺序，并追加已移出 scope 但仍保留部署快照的环境。 */
  const associatedEnvNames = computed(() => {
    const scopeEnvNames = currentConfig.value?.scopeEnvNames || [];
    const deployedEnvNames = Object.entries(currentConfig.value?.envStates || {})
      .filter(([, state]) => Boolean(state?.appliedFields))
      .map(([envName]) => envName);
    return Array.from(new Set([...scopeEnvNames, ...deployedEnvNames]));
  });

  /** 根据关联环境名称生成表格行；统计失败时统一交由 Table 展示空状态。 */
  const envRows = computed<EnvWeightRow[]>(() => {
    if (loadFailed.value) return [];
    return associatedEnvNames.value.map(createEnvWeightRow);
  });

  const deployedRows = computed(() => envRows.value.filter(row => row.deployed));
  const undeployedRows = computed(() => envRows.value.filter(row => !row.deployed));

  /** 根据配置和环境元数据构造单条权重表格数据。 */
  function createEnvWeightRow(envName: string): EnvWeightRow {
    const env = envMap.value.get(envName);
    const instanceStats = envInstanceStats.value[envName];
    const envType = env?.type || '';
    const deployed = isDeployedEnv(envName, currentConfig.value);

    return {
      name: envName,
      displayName: env?.displayName || envName,
      envType,
      envTypeName: envTypeMap[envType]?.name,
      envTypeClass: envTypeTagClassMap[envType],
      deployed,
      weight: currentConfig.value?.envWeights?.[envName] ?? DEFAULT_ENV_WEIGHT,
      healthyCount: deployed ? (instanceStats?.healthyInstanceCount ?? 0) : 0,
      healthyInstanceWeight: deployed ? (instanceStats?.healthyInstanceWeight ?? 0) : 0,
      weightOverriddenInstanceCount: deployed ? (instanceStats?.weightOverriddenInstanceCount ?? 0) : 0,
    };
  }

  /** 计算当前环境总权重在北极星服务全部健康实例权重中的占比。 */
  function getRowPercent(row: EnvWeightRow) {
    if (totalHealthyInstanceWeight.value === 0) return '0.00%';
    return `${((getRowTotalWeight(row) / totalHealthyInstanceWeight.value) * 100).toFixed(2)}%`;
  }

  /** 获取北极星返回的环境健康实例实际权重总和。 */
  function getRowTotalWeight(row: EnvWeightRow) {
    return row.healthyInstanceWeight;
  }

  /** 关闭侧栏并清理本次配置关联的临时状态。 */
  function handleClosed() {
    loadVersion += 1;
    loading.value = false;
    loadFailed.value = false;
    envInstanceStats.value = {};
    totalHealthyInstanceWeight.value = 0;
    weightDialogVisible.value = false;
  }

  /** 提交环境权重，同步配置后重新拉取实例统计与全量健康权重。 */
  async function handleConfirmWeight(weight: number) {
    if (submitting.value || !currentConfig.value?.name || !editingEnv.value) return;

    submitting.value = true;
    try {
      const updatedConfig = await PolarisConfigService.putEnvWeight({
        appID: appDetailStore.appID,
        configName: currentConfig.value.name,
        envName: editingEnv.value.name,
        weight,
      });
      currentConfig.value = updatedConfig;
      emit('updated', updatedConfig);
      weightDialogVisible.value = false;
      Message({
        message: t('操作成功'),
        theme: 'success',
      });
      await loadEnvInstanceStats();
    } catch (error) {
      console.error(error);
    } finally {
      submitting.value = false;
    }
  }

  /** 记录当前环境，并打开对应的权重调整弹窗。 */
  function handleShowWeightDialog(row: EnvWeightRow) {
    editingEnv.value = row;
    weightDialogVisible.value = true;
  }

  /** 判断异步统计结果是否仍属于当前打开的侧栏。 */
  function isActiveLoad(version: number) {
    return version === loadVersion && isShow.value;
  }

  /** 判断环境是否存在已生效的部署快照。 */
  function isDeployedEnv(envName: string, config?: PolarisConfigOutputObj) {
    return Boolean(config?.envStates?.[envName]?.appliedFields);
  }

  /** 从后端统一统计接口读取各环境的北极星实例统计。 */
  async function loadEnvInstanceStats() {
    const config = currentConfig.value;
    const version = ++loadVersion;
    envInstanceStats.value = {};
    totalHealthyInstanceWeight.value = 0;
    loadFailed.value = false;

    if (!config?.name) {
      loading.value = false;
      return;
    }

    loading.value = true;
    try {
      const result = await PolarisConfigService.getEnvInstanceStats({
        appID: appDetailStore.appID,
        configName: config.name,
      });

      if (!isActiveLoad(version)) return;
      envInstanceStats.value = result?.envStats || {};
      totalHealthyInstanceWeight.value = result?.totalHealthyInstanceWeight ?? 0;
    } catch (error) {
      if (isActiveLoad(version)) {
        // 任一环境统计失败时整体置空，避免展示不完整的权重占比。
        loadFailed.value = true;
      }
      console.error(error);
    } finally {
      if (version === loadVersion) {
        loading.value = false;
      }
    }
  }

  /** 跟随父组件配置变化，确保权重更新后侧栏使用最新配置。 */
  watch(
    () => props.config,
    config => {
      currentConfig.value = config;
    },
    { immediate: true },
  );

  /** 打开或切换北极星配置时重新统计健康实例。 */
  watch(
    () => [isShow.value, props.config?.name] as const,
    ([show]) => {
      if (!show || !props.config) return;
      currentConfig.value = props.config;
      loadEnvInstanceStats();
    },
  );
</script>
