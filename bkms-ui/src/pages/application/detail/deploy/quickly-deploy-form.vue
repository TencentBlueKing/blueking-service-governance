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
  <Form
    ref="formRef"
    form-type="vertical"
    :model="formModel"
  >
    <Form.FormItem
      :label="$t('实例数')"
      property="replicas"
      required
    >
      <Input
        v-model.number="formModel.replicas"
        :min="1"
        :precision="0"
        type="number"
      />
    </Form.FormItem>

    <Form.FormItem
      :label="$t('镜像来源')"
      required
    >
      <Button.ButtonGroup class="flex items-center">
        <Button
          class="flex-1"
          :selected="imageSource === 'image'"
          @click="handleChangeImageSource('image')"
        >
          {{ $t('已构建镜像') }}
        </Button>
        <Button
          v-bk-tooltips="{
            content: $t('生产类型环境只能部署已经晋级的镜像 Tag'),
            disabled: !isProdEnv,
          }"
          class="flex-1"
          :disabled="isProdEnv"
          :selected="imageSource === 'code'"
          @click="handleChangeImageSource('code')"
        >
          {{ $t('从源码构建') }}
        </Button>
      </Button.ButtonGroup>
    </Form.FormItem>
    <template v-if="imageSource === 'code'">
      <CodeRevisionField
        ref="branchSelectRef"
        v-model="formModel.revision"
        v-model:revision-type="formModel.revisionType"
        :default-revision="defaultRevision"
        :repository-id="repoAlias"
        required
        :show-source="showCodeSource"
        :workspace-id="workspaceId"
        @confirm="handleRevisionConfirm"
      />
    </template>
    <Form.FormItem
      :label="$t('镜像 Tag')"
      property="imageTag"
      required
    >
      <Input
        v-if="imageSource === 'code'"
        v-model.trim="formModel.imageTag"
      />
      <ImageSelect
        v-else-if="envName"
        ref="imageSelectRef"
        v-model:value="formModel.imageTag"
        :env-name="envName"
        :env-type="envType"
      />
      <!-- 占位 -->
      <Select
        v-else
        v-bk-tooltips="$t('请选择环境')"
        disabled
      />
    </Form.FormItem>

    <!-- 发布方式：一次更新全部 / 按策略分批更新 -->
    <Form.FormItem
      :label="$t('发布方式')"
      required
    >
      <Button.ButtonGroup class="flex items-center">
        <Button
          class="flex-1"
          :selected="formModel.releaseMode === 'all'"
          @click="handleChangeReleaseMode('all')"
        >
          {{ $t('一次更新全部实例') }}
        </Button>
        <Button
          v-bk-tooltips="{
            content: canaryDisabledReason,
            disabled: !canaryDisabledReason,
          }"
          class="flex-1"
          :disabled="!!canaryDisabledReason"
          :selected="formModel.releaseMode === 'canary'"
          @click="handleChangeReleaseMode('canary')"
        >
          {{ $t('按策略分批更新') }}
        </Button>
      </Button.ButtonGroup>
    </Form.FormItem>

    <!-- 发布策略：分批发布时必选 -->
    <Form.FormItem
      v-if="formModel.releaseMode === 'canary'"
      :label="$t('发布策略')"
      property="releaseStrategyID"
      required
      :rules="releaseStrategyRules"
    >
      <div class="w-full">
        <div class="flex items-center gap-[8px]">
          <Select
            ref="strategySelectRef"
            v-model="formModel.releaseStrategyID"
            class="flex-1"
            filterable
            :list="strategyOptions"
            :loading="strategyLoading"
            :no-data-text="$t('暂无发布策略，请先在工作空间创建')"
            :placeholder="$t('请选择发布策略')"
          >
            <template #optionRender="{ item }">
              <span class="flex min-w-0 flex-1 items-center justify-between gap-[8px]">
                <span class="truncate">{{ item.label }}</span>
                <span
                  v-if="item.isBuiltin"
                  class="shrink-0 rounded-[2px] bg-[#F0F1F5] px-[6px] text-[10px] leading-[16px] text-[#4D4F56]"
                >
                  {{ $t('平台内置') }}
                </span>
              </span>
            </template>
            <template #extension>
              <div
                class="flex w-full items-center gap-[8px] justify-center cursor-pointer"
                @click="handleCreateStrategy"
              >
                <Plus
                  class="border-[1px] border-solid rounded-[50%] border-[#979BA5]"
                  fill="#979BA5"
                  height="16px"
                  width="16px"
                />
                <span class="text-[#4D4F56] !text-[12px]">
                  {{ $t('新建策略') }}
                </span>
              </div>
            </template>
          </Select>
          <!-- 跳转空间设置「发布策略」页管理策略 -->
          <span
            class="shrink-0 cursor-pointer text-[12px] text-[#3A84FF]"
            @click="handleViewStrategies"
          >
            {{ $t('查看') }}
          </span>
        </div>
        <div
          v-if="strategyHintText"
          class="mt-[8px] text-[12px] leading-[20px] text-[#979BA5]"
        >
          {{ strategyHintText }}
        </div>
      </div>
    </Form.FormItem>
  </Form>

  <!-- 发布预览：摘要栏 + 批次时间线 -->
  <div
    v-if="releasePreview"
    class="rounded-[2px] bg-[#F5F7FA] p-[16px] mb-[24px]"
  >
    <div class="mb-[12px] flex items-baseline gap-[8px]">
      <span class="text-[14px] font-bold leading-[22px] text-[#313238]">{{ $t('发布预览') }}</span>
      <span class="text-[12px] leading-[20px] text-[#979BA5]">
        {{ $t('按所选环境的 {0} 个实例换算', [releasePreview.summary.totalReplicas]) }}
      </span>
    </div>
    <!-- 摘要栏：label 在上，数值 + 单位在下 -->
    <div
      class="mb-[16px] flex items-stretch rounded-[2px] border border-[#DCDEE5] bg-white px-[24px] py-[14px]"
      data-testid="release-summary"
    >
      <template
        v-for="(item, index) in releaseSummaryItems"
        :key="item.label"
      >
        <div
          v-if="index > 0"
          class="mx-[24px] h-[36px] w-[1px] shrink-0 self-center bg-[#DCDEE5]"
        ></div>
        <div class="min-w-0 flex-1">
          <div class="text-[12px] leading-[16px] text-[#979BA5]">{{ item.label }}</div>
          <div class="mt-[6px] flex items-baseline">
            <span class="text-[20px] font-semibold leading-[24px] text-[#313238]">{{ item.value }}</span>
            <span class="ml-[4px] text-[12px] leading-[16px] text-[#63656E]">{{ item.unit }}</span>
          </div>
        </div>
      </template>
    </div>
    <!-- 批次时间线：编号圆节点 + 批次描述 + 进度方块 + 暂停标签 -->
    <div class="relative pl-[30px]">
      <div class="absolute bottom-[22px] left-[12px] top-[12px] w-[2px] bg-[#DCDEE5]"></div>
      <div
        v-for="row in releasePreview.rows"
        :key="row.batchIndex"
        class="relative pb-[16px] last:pb-0"
      >
        <span
          class="absolute left-[-30px] top-0 flex h-[24px] w-[24px] items-center justify-center rounded-full bg-[#3A84FF] text-[12px] font-medium text-white"
        >
          {{ row.batchIndex }}
        </span>
        <div class="flex items-center justify-between gap-[8px]">
          <div class="flex min-w-0 items-center gap-[6px]">
            <span class="text-[13px] font-bold leading-[20px] text-[#313238]">{{ batchTitle(row) }}</span>
            <Tag
              v-if="row.isFull"
              theme="info"
            >
              {{ $t('全量') }}
            </Tag>
          </div>
          <div class="shrink-0 text-[13px] leading-[20px] text-[#313238]">
            <span class="font-bold">{{ $t('{0}/{1}', [row.updatedCount, releasePreview.summary.totalReplicas]) }}</span>
            <span class="ml-[2px] text-[#979BA5]">{{ $t('实例') }}</span>
          </div>
        </div>
        <div
          class="mt-[8px] flex gap-[5px]"
          data-testid="batch-squares"
        >
          <!-- 方块按实例数比例均分宽度（如 1 实例 = 100% 宽）；浅蓝=本批要发布，深蓝=此前批次已发布，灰=未发布 -->
          <span
            v-for="(square, squareIndex) in buildBatchSquares(row, releasePreview.summary.totalReplicas)"
            :key="squareIndex"
            class="h-[10px] flex-1 rounded-[2px]"
            :class="{
              'bg-[#3A84FF]': square === 'prev',
              'bg-[#A3C5FD]': square === 'new',
              'bg-[#DCDEE5]': square === 'pending',
            }"
          ></span>
        </div>
        <Tag
          v-if="row.pause"
          class="mt-[10px]"
          :theme="row.pause.mode === 'permanent' ? 'warning' : 'info'"
        >
          {{
            row.pause.mode === 'permanent'
              ? $t('永久暂停 · 手动继续')
              : $t('暂停 {0} 秒后自动继续', [row.pause.seconds])
          }}
        </Tag>
      </div>
    </div>
  </div>

  <!-- 新建策略：叠加在部署抽屉之上的独立抽屉，保存后自动选中并刷新发布预览 -->
  <StrategySideslider
    v-model:is-show="createStrategyVisible"
    :existing-names="existingStrategyNames"
    mode="create"
    @success="handleStrategyCreated"
  />

  <!-- “资源规格” 和 “环境变量” 统一预检查弹窗 -->
  <EnvVarPrecheckDialog
    v-if="precheckBeforeSubmit"
    v-model:is-show="isShowPrecheckDialog"
    :env-id="precheckEnvId"
    :env-name="precheckEnvName"
    :mismatches="federationMismatches"
    :missing-required-cluster-addons="missingRequiredClusterAddons"
    :undefined-vars="undefinedVars"
    @cancel="cancelDeploy"
    @go-modify="cancelDeploy"
    @still-deploy="continueDeploy"
  />
</template>

<script lang="ts" setup>
  import { computed, onMounted, reactive, ref, watch } from 'vue';

  import { Button, Form, Input, Message, Select, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { ReleaseStrategiesService } from '~/api/modules/v1/release-strategies';
  import { type RevisionType, revisionRequestFields } from '~/components/code-revision/revision';
  import { isAppModelAppType } from '~/composables/app-type';
  import { useAppRepoRefSelect } from '~/composables/use-app-repo-ref-select';
  import { useAppVisibleEnvs } from '~/composables/use-app-visible-envs';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useRecommendTag } from '~/composables/use-recommend-tag';
  import StrategySideslider from '~/pages/basic/release-strategies/components/strategy-sideslider.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import ImageSelect from '../../components/image-select.vue';
  import EnvVarPrecheckDialog from './env-var-precheck-dialog.vue';
  import {
    type ReleasePreviewResult,
    type ReleasePreviewRow,
    buildBatchSquares,
    buildReleasePreview,
    buildStrategyHint,
    formatOptionTarget,
  } from './release-preview';
  import { type DeployableAppType, type DeployParams, useDeployAPIs } from './use-deploy';
  import { useDeployPrecheck } from './use-deploy-precheck';

  import type { AppModelDeployRecordOutputObj } from '~/@types/v1/deploy';
  import type { EnvOutput } from '~/@types/v1/env';
  import type { BatchOutputObj, ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';

  // 镜像来源类型：从源码构建 / 已构建镜像
  type ImageSourceType = 'code' | 'image';

  const props = withDefaults(
    defineProps<{
      effectiveReplicas?: number; // 当前生效的实例数，用于重置表单时回填
      envName?: string;
      envType?: string;
      /** 目标环境是否为联邦集群环境；联邦集群暂不支持按策略分批发布。 */
      isFederationEnv?: boolean;
      isProdEnv?: boolean;
      /** 是否在提交表单时执行预检查；普通部署已在打开侧栏前完成检查。 */
      precheckBeforeSubmit?: boolean;
      /** 总览入口强制使用显式传入的目标环境，防止未选中时误回退到 store 中的旧环境。 */
      useProvidedEnv?: boolean;
    }>(),
    {
      precheckBeforeSubmit: true,
    },
  );

  const { t } = useI18n();
  const router = useRouter();
  const spaceStore = useSpaceStore();
  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  const { checkDeployEnv } = useAppVisibleEnvs();
  const {
    cancelDeploy,
    continueDeploy,
    federationMismatches,
    isShowPrecheckDialog,
    missingRequiredClusterAddons,
    precheck,
    precheckEnvName,
    precheckEnvId,
    undefinedVars,
  } = useDeployPrecheck();

  const { workspaceId, repoAlias, branchSelectRef, prepareBranchAfterMount } = useAppRepoRefSelect(
    () => appDetailStore.appDetail?.buildConfig?.repoBuildConfig?.repoAlias || '',
  );

  // 当前选择的镜像来源
  const imageSource = ref<ImageSourceType>('image');
  const formRef = ref();
  const imageSelectRef = ref();
  // 表单数据模型
  const showCodeSource = computed(() => appDetailStore.appDetail?.buildConfig?.sourceType !== 'pipeline');
  const defaultRevision = ref('');
  const formModel = reactive<{
    imageTag: string;
    releaseMode: 'all' | 'canary';
    releaseStrategyID: string;
    replicas: number;
    revision: string;
    revisionType: RevisionType;
  }>({
    replicas: 1,
    imageTag: '',
    revision: '',
    revisionType: 'branch',
    releaseMode: 'all',
    releaseStrategyID: '',
  });

  // 实例列表入口沿用 store 当前环境；总览入口只认侧栏选中的目标环境，即使暂时为空也不回退。
  const envName = computed(() =>
    props.useProvidedEnv ? (props.envName ?? '') : props.envName || trpcDeployStore.curEnvItem?.name || '',
  );
  const envType = computed(() =>
    props.useProvidedEnv ? (props.envType ?? '') : props.envType || trpcDeployStore.curEnvItem?.type || '',
  );
  // 是否为生产环境：props 优先，否则按 envType 判断
  const isProdEnv = computed(() => props.isProdEnv ?? envType.value === 'production');

  // 推荐镜像 Tag：源码构建模式下自动推荐
  const { getDefaultBranch, fetchRecommendTag } = useRecommendTag(
    () => ({ revisionType: formModel.revisionType, revision: formModel.revision }),
    {
      manualFetchOnly: computed(() => !repoAlias.value),
      onRecommend: tag => {
        if (imageSource.value === 'code') {
          formModel.imageTag = tag;
        }
      },
    },
  );

  /** 来源或修订确认后重算推荐版本号，包括用户改过的版本号 */
  function handleRevisionConfirm() {
    fetchRecommendTag();
  }

  // ---------- 发布策略：一次更新全部实例 / 按策略分批发布 ----------

  // 工作空间发布策略列表（含平台内置策略，isBuiltin 标识）
  const releaseStrategies = ref<ReleaseStrategyOutputObj[]>([]);
  const strategyLoading = ref(false);
  const strategySelectRef = ref();
  /** 「新建策略」抽屉：叠加在部署抽屉之上，独立维护显隐 */
  const createStrategyVisible = ref(false);
  /** 空间内已有策略名称：新建抽屉用于名称唯一性校验 */
  const existingStrategyNames = computed(() => releaseStrategies.value.map(item => item.name ?? ''));
  // 选项展示：策略名 + 批次序列摘要，如「生产谨慎发布(1个,25%,50%,全量)」；
  // 排序：空间自定义策略在前、平台内置策略在后（各自保持创建时间序，见设计稿 2141:28311 注释）
  const strategyOptions = computed(() => {
    const options = releaseStrategies.value.map(item => ({
      isBuiltin: item.isBuiltin === true,
      value: item.id ?? '',
      label: buildStrategyOptionLabel(item.name ?? '', item.batches),
    }));
    return options.sort((a, b) => Number(a.isBuiltin) - Number(b.isBuiltin));
  });

  /** 选项摘要拼装：全量/个词条走 i18n */
  function buildStrategyOptionLabel(name: string, batches: BatchOutputObj[] | undefined): string {
    const summary = (batches ?? [])
      .map(batch => formatOptionTarget(batch.target, { full: t('全量'), unit: t('个') }))
      .join(',');
    return summary ? `${name}(${summary})` : name;
  }

  /**
   * 分批发布不可用的原因；为空表示可用。
   * 支持范围：仅应用模型应用（tRPC / TAF）、主工作负载为 GameDeployment、非联邦集群、镜像直连部署。
   */
  const canaryDisabledReason = computed(() => {
    if (!isAppModelAppType(appDetailStore.appType)) return t('仅应用模型应用支持分批发布');
    if (props.isFederationEnv) return t('联邦集群环境暂不支持分批发布');
    // 主工作负载资格检查完成前不允许选择分批发布，避免「先选后禁」留下已选策略透传给后端。
    if (workloadKindLoading.value) return t('正在检查分批发布资格，请稍候');
    // 主工作负载明确为 Deployment 时禁用；无部署记录（未知）时不因该条件禁用，由后端兜底校验。
    if (mainWorkloadKind.value === 'Deployment') return t('仅主工作负载为 GameDeployment 的应用支持分批发布');
    if (imageSource.value === 'code') return t('构建部署暂不支持分批发布');
    return '';
  });

  // 分批发布时策略必选；一次更新全部实例时不校验
  const releaseStrategyRules = computed(() =>
    formModel.releaseMode === 'canary' ? [{ required: true, message: t('请选择发布策略'), trigger: 'change' }] : [],
  );

  /**
   * 主工作负载类型（GameDeployment / Deployment），来自最近一次部署记录；
   * 无部署记录时为空字符串，表示未知（不因此禁用分批发布，由后端兜底校验）。
   */
  const mainWorkloadKind = ref('');
  /** 主工作负载资格检查是否进行中：检查完成前禁用分批发布，避免选择后再被回退。 */
  const workloadKindLoading = ref(false);

  /** 拉取最近一条部署记录以获取主工作负载类型；仅应用模型应用且已选环境时执行。 */
  async function fetchMainWorkloadKind() {
    if (!isAppModelAppType(appDetailStore.appType) || !envName.value) return;
    workloadKindLoading.value = true;
    const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
    const res = await deployAPIs
      .listDeployRecords({
        appID: appDetailStore.appID,
        envName: envName.value,
        page: 1,
        pageSize: 1,
      })
      .catch(() => ({ count: 0, results: [] as AppModelDeployRecordOutputObj[] }));
    mainWorkloadKind.value = res?.results?.[0]?.workloadKind ?? '';
    workloadKindLoading.value = false;
  }

  // 资格异步返回导致已选分批模式失效时，重置为一次更新全部并提示，避免带着后端必拒的参数提交。
  watch(canaryDisabledReason, reason => {
    if (!reason || formModel.releaseMode !== 'canary') return;
    formModel.releaseMode = 'all';
    formModel.releaseStrategyID = '';
    Message({ theme: 'warning', message: t('当前环境不支持分批发布，已重置为一次更新全部实例') });
  });

  /** 拉取工作空间发布策略；侧栏按环境名重建，随挂载加载一次即可。 */
  async function fetchReleaseStrategies() {
    if (!spaceStore.currentSpace) return;
    strategyLoading.value = true;
    try {
      releaseStrategies.value =
        (await ReleaseStrategiesService.listReleaseStrategies({ workspaceID: spaceStore.currentSpace })) ?? [];
    } catch (err) {
      console.error(err);
    } finally {
      strategyLoading.value = false;
    }
  }

  /**
   * 打开「新建策略」抽屉：Select 的 extension 区点击被组件库阻止冒泡，下拉不会自动收起，
   * 需显式关闭，避免下拉浮层压在抽屉之上。
   */
  function handleCreateStrategy() {
    strategySelectRef.value?.hidePopover?.();
    createStrategyVisible.value = true;
  }

  /** 新建成功：关闭抽屉、刷新策略列表并选中新策略，发布预览随选中项自动重算。 */
  async function handleStrategyCreated(strategy?: ReleaseStrategyOutputObj) {
    createStrategyVisible.value = false;
    await fetchReleaseStrategies();
    if (!strategy?.id) return;
    formModel.releaseStrategyID = strategy.id;
  }

  /** 「查看」跳转空间设置下的发布策略页，集中管理空间内策略。 */
  function handleViewStrategies() {
    router.push({
      name: 'basicItem',
      params: { space: spaceStore.currentSpace, menuName: 'releaseStrategies' },
    });
  }

  onMounted(() => {
    fetchReleaseStrategies();
    fetchMainWorkloadKind();
  });

  const selectedStrategy = computed(() =>
    releaseStrategies.value.find(item => item.id === formModel.releaseStrategyID),
  );

  /** 发布预览：选中策略后按实例数实时换算批次时间线与摘要 */
  const releasePreview = computed<null | ReleasePreviewResult>(() => {
    if (formModel.releaseMode !== 'canary' || !selectedStrategy.value) return null;
    return buildReleasePreview(selectedStrategy.value.batches, Number(formModel.replicas));
  });

  const releaseSummaryItems = computed(() => {
    const summary = releasePreview.value?.summary;
    if (!summary) return [];
    return [
      { label: t('批次数'), value: summary.batchCount, unit: t('批') },
      { label: t('暂停次数'), value: summary.pauseCount, unit: t('次') },
      { label: t('实例总数'), value: summary.totalReplicas, unit: t('个') },
      { label: t('流量批次'), value: summary.trafficBatchCount, unit: t('个') },
    ];
  });

  /**
   * 策略说明文案：先 1 个实例，确认后累计到 25%、50%，最后自动全量。
   * 末批非全量时后端仍会补齐剩余实例（UpdatingPrimary 阶段），因此收尾固定为「最后自动全量」，
   * 与策略预览、发布预览中的「系统补齐全量批」保持一致。
   */
  const strategyHintText = computed(() => {
    const hint = buildStrategyHint(selectedStrategy.value?.batches);
    if (!hint) return '';
    const parts: string[] = [];
    // 仅「真的只有一个批次且为全量」才只提示「最后自动全量」，避免「先更新到 100%」的重复表述；
    // 两批策略（如 25% → 100%）不能用中间批次数判断，否则首批灰度阶段会被漏掉。
    const isSingleFull = hint.batchCount === 1 && hint.lastIsFull;
    if (!isSingleFull) {
      parts.push(hint.firstIsPercent ? t('先更新到 {0}', [hint.firstTarget]) : t('先 {0} 个实例', [hint.firstTarget]));
    }
    if (hint.middleTargets.length > 0) {
      parts.push(t('确认后累计到 {0}', [hint.middleTargets.join('、')]));
    }
    parts.push(t('最后自动全量'));
    return parts.join('，');
  });

  /** 批次标题：首批数值 target 为「N 个实例」，其余为「累计 N 个（P%）」 */
  function batchTitle(row: ReleasePreviewRow): string {
    const head = t('第 {0} 批', [row.batchIndex]);
    if (row.batchIndex === 1 && !row.isPercent) {
      return `${head} · ${t('{0} 个实例', [row.updatedCount])}`;
    }
    const percentSuffix = row.isPercent ? `（${row.targetText}）` : '';
    return `${head} · ${t('累计 {0} 个', [row.updatedCount])}${percentSuffix}`;
  }

  /** 切换发布方式；切回「一次更新全部」时清空已选策略。 */
  function handleChangeReleaseMode(mode: 'all' | 'canary') {
    formModel.releaseMode = mode;
    if (mode === 'all') {
      formModel.releaseStrategyID = '';
    }
  }
  // 离开确认：表单有变更时提示用户
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

  // 清除表单校验结果
  function clearValidate() {
    formRef.value?.clearValidate?.();
  }

  // 关闭镜像选择下拉框
  function closeDropdown() {
    imageSelectRef.value?.closeDropdown?.();
  }

  function confirmLeave() {
    return confirmBox();
  }

  // 执行部署：根据镜像来源调用直接部署或构建+部署接口
  async function deploy(targetEnv: EnvOutput) {
    if (!targetEnv.name || !checkDeployEnv(targetEnv)) return false;
    // 提交前按最新资格兜底校验：分批发布已失效（如主工作负载异步判定为 Deployment）时拦截，
    // 不把后端必拒的 releaseStrategyID 透传出去。
    if (formModel.releaseMode === 'canary' && canaryDisabledReason.value) {
      Message({ theme: 'error', message: canaryDisabledReason.value });
      return false;
    }
    const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
    const params: DeployParams = {
      appID: appDetailStore.appID,
      envName: targetEnv.name,
      imageTag: formModel.imageTag,
      replicas: Number(formModel.replicas),
    };
    if (imageSource.value === 'image') {
      // 仅镜像直连部署 + 分批发布 + 已选策略时透传发布策略；构建部署接口无该字段。
      if (formModel.releaseMode === 'canary' && formModel.releaseStrategyID) {
        params.releaseStrategyID = formModel.releaseStrategyID;
      }
      await deployAPIs.createDeployDirectly!(params);
    } else {
      Object.assign(params, revisionRequestFields(showCodeSource.value, formModel.revisionType, formModel.revision));
      await deployAPIs.buildAndCreateDeploy!(params);
    }
    return true;
  }

  /** 切换镜像来源；源码模式 prepare 默认分支并预拉列表，镜像模式清空源码相关字段。 */
  async function handleChangeImageSource(source: ImageSourceType) {
    imageSource.value = source;
    if (source === 'code') {
      // 构建部署不支持分批发布，切到源码构建时回落为一次更新全部
      if (formModel.releaseMode === 'canary') {
        formModel.releaseMode = 'all';
        formModel.releaseStrategyID = '';
      }
      defaultRevision.value = getDefaultBranch();
      formModel.revisionType = 'branch';
      await prepareBranchAfterMount(defaultRevision.value);
      // 无仓库别名时 prepare 不会发出确认，需要主动拉一次推荐版本号
      if (!repoAlias.value) {
        await fetchRecommendTag();
      }
    } else {
      formModel.revision = '';
      formModel.revisionType = 'branch';
      formModel.imageTag = '';
    }
  }

  // 重置表单：回填实例数，其余字段清空，并统一清理内部 UI 状态
  function reset(replicas = props.effectiveReplicas || 1) {
    withPausedWatch(() => {
      formModel.replicas = replicas;
      formModel.imageTag = '';
      formModel.revision = '';
      formModel.revisionType = 'branch';
      formModel.releaseMode = 'all';
      formModel.releaseStrategyID = '';
      imageSource.value = 'image';
    });
    forceCleanDirtyTag(() => {
      clearValidate();
      closeDropdown();
    });
  }

  // 提交部署：内部完成校验、部署和脏标记清理
  async function submit(targetEnv: EnvOutput) {
    const valid = await validate();
    if (!valid) return false;

    if (props.precheckBeforeSubmit) {
      const precheckPassed = await precheck(targetEnv);
      if (!precheckPassed) return false;
    }

    if (!(await deploy(targetEnv))) return false;
    forceCleanDirtyTag();
    return true;
  }

  // 校验表单
  async function validate() {
    return await formRef.value?.validate?.().catch(() => false);
  }

  // 暴露业务级方法供父组件调用
  defineExpose({
    confirmLeave,
    reset,
    submit,
    validate,
  });
</script>
