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
  <div class="flex bg-[#FFFFFF] gap-[24px]">
    <!-- 左侧：镜像详情 -->
    <div class="flex-1 min-w-0 py-[24px]">
      <FieldItem
        :container-height="32"
        :field-value="t('镜像仓库')"
        :field-width="100"
        :value="row.repository"
        value-color="#111214"
        value-max-width="100%"
      />
      <FieldItem
        :container-height="32"
        field-value="tag"
        :field-width="100"
        :value="row.tag"
        value-color="#111214"
      />
      <FieldItem
        :container-height="32"
        :field-value="t('源材料')"
        :field-width="100"
      >
        <template #value>
          <div
            v-if="row.sourceMaterial"
            class="flex min-w-0 max-w-full items-center text-[#111214]"
          >
            <!-- 与蓝盾一致：分支 / Tag / CommitID 都用分支图标标修订，短哈希另用 commit 图标 -->
            <span class="bkms-icon bkms-icon-branchs mr-[5px] shrink-0"></span>
            <span class="truncate">{{ row.sourceMaterial.revision }}</span>
            <template v-if="row.sourceMaterial.commitID">
              <span class="bkms-icon bkms-icon-commit mx-[5px] shrink-0 text-[18px]"></span>
              <Button
                class="shrink-0"
                :disabled="!row.sourceMaterial.commitURL"
                text
                theme="primary"
                @click="handleToCommit(row.sourceMaterial.commitURL || '')"
              >
                {{ row.sourceMaterial.commitID.slice(0, 8) }}
              </Button>
            </template>
          </div>
          <span
            v-else
            class="text-[#111214]"
            >--</span
          >
        </template>
      </FieldItem>
      <FieldItem
        :container-height="32"
        :field-value="t('大小')"
        :field-width="100"
        :value="formatSize(Number(row?.size))"
        value-color="#111214"
      />
      <FieldItem
        :container-height="32"
        :field-value="t('构建时间')"
        :field-width="100"
        :value="row.builtAt ? formatDateString(row.builtAt) : '--'"
        value-color="#111214"
      />
      <FieldItem
        :container-height="32"
        :field-value="t('摘要')"
        :field-width="100"
        :is-label-overflow="false"
        :value="row.digest || '--'"
        value-max-width="100%"
      >
      </FieldItem>
      <FieldItem
        class="!min-h-[32px] !h-auto !items-start"
        value-color="#111214"
      >
        <template #field>
          <div class="w-[100px] min-h-[32px] leading-[32px] text-align-end text-[#8F9FBD]">{{ t('已部署环境') }}：</div>
        </template>
        <template #value>
          <div
            v-if="groupedEnvEntries.length"
            class="flex flex-col gap-[8px] py-[4px]"
          >
            <div
              v-for="[envType, envs] in groupedEnvEntries"
              :key="envType"
              class="flex flex-wrap items-center gap-[4px]"
            >
              <Tag :class="envTypeTagClassMap[envType]">
                {{ getEnvTypeLabel(envType) }}
              </Tag>
              <Tag
                v-for="env in envs"
                :key="env.envName"
                class="bg-[#fff]"
                type="stroke"
              >
                {{ getEnvDisplayName(env?.envName ?? '') }}
              </Tag>
            </div>
          </div>
          <div
            v-else
            class="h-[32px] leading-[32px]"
          >
            --
          </div>
        </template>
      </FieldItem>
    </div>

    <!-- 右侧：部署记录时间线 -->
    <div class="w-[400px] min-h-[180px] flex-shrink-0 px-[16px] pt-[8px] bg-[#F7F9FD]">
      <div class="flex items-center h-[32px] leading-[32px]">
        <span class="text-[12px] text-[#111214]">{{ t('部署记录') }}</span>
        <span class="text-[12px] text-[#8F9FBD] ml-[4px]"> ({{ $t('仅展示前 5 条') }}) </span>
      </div>
      <div
        v-bkloading="{ loading: isDeployRecordLoading, color: '#F0F4FA]' }"
        class="flex-1 h-full"
      >
        <Timeline
          v-if="deployRecords.length"
          :list="timelineList"
        >
          <template #content="{ content }">
            <div class="text-[12px] text-[#8F9FBD]">{{ content }}</div>
          </template>
        </Timeline>
        <Exception
          v-else-if="!isDeployRecordLoading"
          :description="$t('暂无数据')"
          scene="part"
          type="empty"
        ></Exception>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { computed, h, onMounted, ref } from 'vue';

  import { Button, Exception, OverflowTitle, Tag, Timeline } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppImageOutputObj, DeployedEnvInfoOutputObj, ImageTagDeployRecordOutputObj } from '~/@types/v1/images';
  import { ImagesService } from '~/api/modules/v1';
  import { APP_DEPLOY_STATUS, HELM_DEPLOY_STATUS } from '~/common/enums/deploy';
  import { formatSize } from '~/common/util';
  import FieldItem from '~/components/field-item.vue';
  import SvgIcon from '~/components/svg-icon.vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';
  import useTime from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';

  type GroupedEnvs = Record<string, DeployedEnvInfoOutputObj[]>;

  interface IProps {
    envNameDisplayMap?: Record<string, string>;
    row: AppImageOutputObj;
  }

  const props = defineProps<IProps>();

  const { t } = useI18n();
  const { formatDateString } = useTime();
  const appDetailStore = useAppDetail();

  function getEnvDisplayName(envName: string) {
    return props.envNameDisplayMap?.[envName] || envName;
  }

  function getEnvTypeLabel(envType: string) {
    return envTypeMap[envType]?.name || envType;
  }

  /** 跳转代码仓库 commit 页 */
  function handleToCommit(url: string) {
    if (url) window.open(url, '_blank');
  }

  /** 按环境类型分组 */
  const groupedEnvs = computed<GroupedEnvs>(() => {
    const envs = props.row.deployedEnvs;
    if (!envs?.length) return {};
    return envs.reduce((acc, env) => {
      const type = env.envType || 'unknown';
      acc[type] ||= [];
      acc[type].push(env);
      return acc;
    }, {} as GroupedEnvs);
  });

  const groupedEnvEntries = computed(() => Object.entries(groupedEnvs.value));

  /** 部署记录 */
  const deployRecords = ref<ImageTagDeployRecordOutputObj[]>([]);
  const deployRecordCount = ref(0);
  const isDeployRecordLoading = ref(false);

  /** 部署状态配置：颜色、标签文案、时间线节点颜色 */
  interface DeployStatusConfig {
    color: string;
    label: string;
    timelineColor?: string;
  }

  const DEPLOY_STATUS_MAP: Record<string, DeployStatusConfig> = {
    [APP_DEPLOY_STATUS.DEPLOYED]: { color: '#2DCB56', label: '成功', timelineColor: 'green' },
    [HELM_DEPLOY_STATUS.PENDING_INSTALL]: { color: '#3B6EF5', label: '部署中' },
    [HELM_DEPLOY_STATUS.PENDING_UPGRADE]: { color: '#3B6EF5', label: '部署中' },
    [APP_DEPLOY_STATUS.DEPLOYING]: { color: '#3B6EF5', label: '部署中' },
    [APP_DEPLOY_STATUS.FAILED]: { color: '#EA3636', label: '失败', timelineColor: 'red' },
    [APP_DEPLOY_STATUS.POLLING_TIMEOUT]: { color: '#EA3636', label: '失败', timelineColor: 'red' },
    [APP_DEPLOY_STATUS.POLLING_BROKEN]: { color: '#EA3636', label: '失败', timelineColor: 'red' },
  };

  function getDeployStatusConfig(status: string): DeployStatusConfig {
    return DEPLOY_STATUS_MAP[normalizeDeployStatus(status)];
  }

  function normalizeDeployStatus(status?: string) {
    return status?.toLowerCase() || 'unknown';
  }

  const LOADING_TIMELINE_ICON = () =>
    h(SvgIcon, {
      height: '16px',
      icon: 'bkms-icon-loading',
      width: '16px',
      backgroundColor: 'transparent',
    });

  function buildTimelineTag(record: ImageTagDeployRecordOutputObj) {
    const tagText = t('{0} 部署 {1}', [record.operator, getEnvDisplayName(record?.envName ?? '')]);
    const { color, label } = getDeployStatusConfig(record?.status ?? '');
    const tagTextVNode = h(OverflowTitle, { type: 'tips', style: { color: '#4A5973', minWidth: 0 } }, () => tagText);
    const statusVNode = h('span', { style: { color, flexShrink: '0', fontSize: '12px' } }, `（${t(label)}）`);

    return h(
      'div',
      {
        style: {
          fontSize: '12px',
          display: 'flex',
          alignItems: 'center',
          gap: '4px',
          overflow: 'hidden',
          whiteSpace: 'nowrap',
        },
      },
      [tagTextVNode, statusVNode],
    );
  }

  const timelineList = computed(() =>
    deployRecords.value.map(record => {
      const config = getDeployStatusConfig(record?.status ?? '');
      const isLoading = config.color === '#3B6EF5';
      return {
        tag: buildTimelineTag(record),
        nodeType: 'vnode',
        color: config.timelineColor,
        content: record.createdAt ? formatDateString(record.createdAt) : '--',
        icon: isLoading ? LOADING_TIMELINE_ICON : undefined,
        type: 'default',
      };
    }),
  );

  async function fetchDeployRecords() {
    if (!appDetailStore.appID || !props.row.tag) return;
    isDeployRecordLoading.value = true;
    try {
      const res = await ImagesService.listImageTagDeployRecords({
        appID: appDetailStore.appID,
        tag: props.row.tag,
        page: 1,
        pageSize: 5,
      });
      deployRecords.value = res?.results || [];
      deployRecordCount.value = Number(res?.count) || 0;
    } catch {
      deployRecords.value = [];
      deployRecordCount.value = 0;
    } finally {
      isDeployRecordLoading.value = false;
    }
  }

  onMounted(() => {
    fetchDeployRecords();
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-timeline) {
    padding-left: 6px;
    li {
      padding-bottom: 12px;
    }
  }
  :deep(.bk-timeline-section) {
    margin-left: 4px;
    overflow: hidden;

    .bk-timeline-title {
      width: 100%;
      overflow: visible;
    }
  }
</style>
