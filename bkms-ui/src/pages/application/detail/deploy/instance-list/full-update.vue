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
    :before-close="handleBeforeClose"
    :class="{ 'full-update-sideslider': showBuildLog }"
    render-directive="if"
    :width="showBuildLog ? 1200 : 640"
  >
    <template #header>
      <DividerHeader>
        <template #title>
          <span class="text-[16px]">{{ $t('部署') }}</span>
        </template>
        <span v-if="appDetailStore.app">
          {{ appDetailStore.app }}
        </span>
        <span v-if="trpcDeployStore?.curEnvItem?.displayName">
          {{ `${$t('环境')}: ${trpcDeployStore.curEnvItem.displayName}` }}
        </span>
      </DividerHeader>
    </template>
    <template v-if="!showBuildLog">
      <Form
        ref="formRef"
        class="pt-[18px] px-[24px]"
        form-type="vertical"
        :model="formModel"
      >
        <Form.FormItem
          class="!mb-[14px]"
          :label="$t('更新内容')"
          required
        >
          <Radio.Group
            v-model="formModel.updateContent"
            @change="handleUpdateContentChange"
          >
            <Radio
              v-bk-tooltips="{ content: deployDisabledReason, disabled: !deployDisabledReason }"
              :disabled="!!deployDisabledReason"
              label="both"
              >{{ $t('镜像+配置') }}</Radio
            >
            <Radio
              v-bk-tooltips="{ content: deployDisabledReason, disabled: !deployDisabledReason }"
              :disabled="!!deployDisabledReason"
              label="config"
              >{{ $t('仅配置') }}</Radio
            >
            <Radio
              :disabled="isFederationEnv"
              label="image"
            >
              <span
                v-bk-tooltips="{
                  content: $t('联邦集群不支持原地重启'),
                  disabled: !isFederationEnv,
                }"
              >
                {{ $t('仅镜像（原地重启）') }}
              </span>
            </Radio>
          </Radio.Group>
        </Form.FormItem>

        <Alert
          v-if="formModel.updateContent !== 'both'"
          class="mb-[24px]"
          closable
          theme="info"
        >
          {{ alertContent }}
        </Alert>

        <Form.FormItem
          v-if="formModel.updateContent !== 'image'"
          :label="$t('实例数')"
          property="replicas"
          required
          :rules="replicasRules"
        >
          <Input
            v-model.number="formModel.replicas"
            :min="1"
            :precision="0"
            type="number"
          />
          <div
            v-if="hasInvalidEffectiveReplicas"
            class="mt-[4px] text-[12px] leading-[20px] text-[#979BA5]"
          >
            {{ $t('实例删除后副本数已变为 0，请填写期望实例数后重新部署') }}
          </div>
        </Form.FormItem>

        <Form.FormItem
          v-if="formModel.updateContent === 'image'"
          :label="$t('镜像 Tag')"
          property="imageTag"
          required
        >
          <ImageSelect
            ref="imageSelectRef"
            v-model:value="formModel.imageTag"
          />
        </Form.FormItem>

        <template v-if="formModel.updateContent === 'both'">
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
          <CodeRevisionField
            v-if="imageSource === 'code'"
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
              v-else
              ref="imageSelectRef"
              v-model:value="formModel.imageTag"
            />
          </Form.FormItem>
        </template>

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
        class="mx-[24px] mb-[24px] rounded-[2px] bg-[#F5F7FA] p-[16px]"
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
                <span class="font-bold">{{
                  $t('{0}/{1}', [row.updatedCount, releasePreview.summary.totalReplicas])
                }}</span>
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
    </template>
    <template v-else>
      <BuildLogPanel
        :active="showBuildLog"
        :build-info="buildInfo"
      />
    </template>

    <div
      v-if="!showBuildLog"
      class="mt-[32px] px-[24px]"
    >
      <Button
        :loading="loading"
        theme="primary"
        @click="handleDeploy"
      >
        {{ $t('部署') }}
      </Button>
      <Button
        class="ml-[8px]"
        :loading="loading"
        @click="handleClose"
      >
        {{ $t('取消') }}
      </Button>
    </div>

    <!-- 新建策略：叠加在部署抽屉之上的独立抽屉，保存后自动选中并刷新发布预览 -->
    <StrategySideslider
      v-model:is-show="createStrategyVisible"
      :existing-names="existingStrategyNames"
      mode="create"
      @success="handleStrategyCreated"
    />
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, onMounted, reactive, ref, watch } from 'vue';

  import { Alert, Button, Form, Input, Message, Radio, Select, Sideslider, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { InstanceService } from '~/api/modules/v1';
  import { ReleaseStrategiesService } from '~/api/modules/v1/release-strategies';
  import { type RevisionType, revisionRequestFields } from '~/components/code-revision/revision';
  import { isAppModelAppType } from '~/composables/app-type';
  import { useAppRepoRefSelect } from '~/composables/use-app-repo-ref-select';
  import { useAppVisibleEnvs } from '~/composables/use-app-visible-envs';
  import useIsFederationEnv from '~/composables/use-is-federation-env';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useRecommendTag } from '~/composables/use-recommend-tag';
  import ImageSelect from '~/pages/application/components/image-select.vue';
  import BuildLogPanel from '~/pages/application/detail/components/view-build-log/build-log-panel.vue';
  import StrategySideslider from '~/pages/basic/release-strategies/components/strategy-sideslider.vue';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';
  import { useTrpcDeployStore } from '~/stores/trpc-deploy';

  import {
    type ReleasePreviewResult,
    type ReleasePreviewRow,
    buildBatchSquares,
    buildReleasePreview,
    buildStrategyHint,
    formatOptionTarget,
  } from '../release-preview';
  import { type DeployableAppType, type DeployParams, useDeployAPIs } from '../use-deploy';

  import type { AppModelDeployRecordOutputObj } from '~/@types/v1/deploy';
  import type { UpdateAppInstancesRequest } from '~/@types/v1/instance';
  import type { BatchOutputObj, ReleaseStrategyOutputObj } from '~/@types/v1/release-strategies';
  import type { BuildInfo, BuildStatus } from '~/pages/application/detail/components/view-build-log/type';

  type ImageSourceType = 'code' | 'image';

  const isShow = defineModel<boolean>('isShow');
  const emit = defineEmits<{
    update: [keepOpen: boolean];
  }>();
  const props = defineProps<{
    effectiveReplicas?: number;
    isProdEnv?: boolean;
    latestBuildId?: string;
    latestBuildStatus?: BuildStatus;
  }>();

  const { t } = useI18n();
  const router = useRouter();
  const spaceStore = useSpaceStore();
  const trpcDeployStore = useTrpcDeployStore();
  const appDetailStore = useAppDetail();
  const { checkDeployEnv, getEnvDeployDisabledReason } = useAppVisibleEnvs();
  const deployDisabledReason = computed(() => getEnvDeployDisabledReason(trpcDeployStore.curEnvItem));
  const isFederationEnv = useIsFederationEnv(() => trpcDeployStore.curEnvItem);

  const { workspaceId, repoAlias, branchSelectRef, prepareBranchAfterMount } = useAppRepoRefSelect(
    () => appDetailStore.appDetail?.buildConfig?.repoBuildConfig?.repoAlias || '',
  );

  const imageSource = ref<ImageSourceType>('image');
  const formRef = ref();
  const imageSelectRef = ref();
  const showCodeSource = computed(() => appDetailStore.appDetail?.buildConfig?.sourceType !== 'pipeline');
  const defaultRevision = ref('');
  const formModel = reactive<{
    imageTag: string;
    releaseMode: 'all' | 'canary';
    releaseStrategyID: string;
    replicas?: number;
    revision: string;
    revisionType: RevisionType;
    updateContent: 'both' | 'config' | 'image';
  }>({
    revision: '',
    revisionType: 'branch',
    imageTag: '',
    updateContent: 'both',
    releaseMode: 'all',
    releaseStrategyID: '',
  });
  // 切换为“仅配置”时会保留之前的镜像来源，因此只有“镜像+配置”选择源码时才触发构建。
  const shouldBuildFromSource = computed(() => formModel.updateContent === 'both' && imageSource.value === 'code');
  const { getDefaultBranch, fetchRecommendTag } = useRecommendTag(
    () => ({ revisionType: formModel.revisionType, revision: formModel.revision }),
    {
      manualFetchOnly: computed(() => !repoAlias.value),
      onRecommend: tag => {
        if (shouldBuildFromSource.value) {
          formModel.imageTag = tag;
        }
      },
    },
  );

  /** 来源或修订确认后重算推荐版本号，包括用户改过的版本号 */
  function handleRevisionConfirm() {
    fetchRecommendTag();
  }
  const { confirmBox, forceCleanDirtyTag, withPausedWatch } = useLeaveConfirm(formModel);

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
  // 排序：空间自定义策略在前、平台内置策略在后（各自保持创建时间序）
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
   * 支持范围：仅应用模型应用（tRPC / TAF）、主工作负载为 GameDeployment、非联邦集群、镜像直连部署；
   * 「仅镜像（原地重启）」走实例更新接口，同样不支持分批发布。
   */
  const canaryDisabledReason = computed(() => {
    if (!isAppModelAppType(appDetailStore.appType)) return t('仅应用模型应用支持分批发布');
    if (isFederationEnv.value) return t('联邦集群环境暂不支持分批发布');
    // 主工作负载资格检查完成前不允许选择分批发布，避免「先选后禁」留下已选策略透传给后端。
    if (workloadKindLoading.value) return t('正在检查分批发布资格，请稍候');
    // 主工作负载明确为 Deployment 时禁用；无部署记录（未知）时不因该条件禁用，由后端兜底校验。
    if (mainWorkloadKind.value === 'Deployment') return t('仅主工作负载为 GameDeployment 的应用支持分批发布');
    if (formModel.updateContent === 'image') return t('原地重启暂不支持分批发布');
    // 切换为「仅配置」时会保留之前的镜像来源，因此只有「镜像+配置」选择源码时才算构建部署。
    if (shouldBuildFromSource.value) return t('构建部署暂不支持分批发布');
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
    if (!isAppModelAppType(appDetailStore.appType)) return;
    const envName = trpcDeployStore.curEnvItem?.name ?? '';
    if (!envName) return;
    workloadKindLoading.value = true;
    const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
    const res = await deployAPIs
      .listDeployRecords({
        appID: appDetailStore.appID,
        envName,
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

  /** 拉取工作空间发布策略；侧滑内容随打开重建，挂载时加载一次即可。 */
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

  function isValidReplicas(value: number | string | undefined) {
    return Number.isInteger(Number(value)) && Number(value) >= 1;
  }

  const replicasRules = [
    {
      validator: isValidReplicas,
      message: t('请输入大于或等于 1 的整数'),
      trigger: 'blur',
    },
  ];

  // 仅在明确取到 0（实例被删光）时提示；undefined 表示规格未知，不能断言为 0。
  const hasInvalidEffectiveReplicas = computed(() => props.effectiveReplicas === 0);

  const alertContent = computed(() => {
    if (formModel.updateContent === 'config') {
      return t('本次更新仅变更应用的配置信息（包括环境变量等），镜像 Tag 保持不变');
    }
    return t(
      '本次更新仅变更镜像 Tag，应用的配置信息（包括环境变量等）保持不变。仅更新容器镜像，不重建 Pod，更新速度更快',
    );
  });
  const loading = ref(false);
  const curImageTag = ref('');

  // 构建日志相关
  const showBuildLog = ref(false);
  /** buildAndCreateDeploy 返回的构建信息，统一供提示、日志请求和流水线跳转使用。 */
  const buildInfo = reactive<BuildInfo>({
    buildID: '',
    imageTag: '',
    operator: '',
    pipelineID: '',
    revision: '',
    status: 'running',
  });
  // 通过接口获取当前镜像 Tag
  async function fetchCurrentImageTag() {
    const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
    const res = await deployAPIs
      .listDeployRecords({
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name ?? '',
        page: 1,
        pageSize: 1,
      })
      .catch(() => ({ count: 0, results: [] as AppModelDeployRecordOutputObj[] }));
    curImageTag.value = res!.results[0]?.imageTag?.split(':')?.pop() || '';
  }

  // 侧边栏关闭前确认
  function handleBeforeClose(): Promise<boolean> {
    return confirmBox();
  }

  /** 切换镜像来源；源码模式 prepare 默认分支并预拉列表 */
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

  // 关闭弹窗
  async function handleClose() {
    if (await handleBeforeClose()) {
      // 关闭 ImageSelect 的下拉框
      imageSelectRef.value?.closeDropdown?.();
      isShow.value = false;
    }
  }

  /**
   * 配置+镜像、仅配置
   */
  async function handleConfigAndImage() {
    const env = trpcDeployStore.curEnvItem;
    if (!env?.name || !checkDeployEnv(env)) return false;
    try {
      // 根据应用类型获取对应的部署 API
      const deployAPIs = useDeployAPIs(appDetailStore.appType as DeployableAppType);
      // 根据镜像来源选择部署方式
      const params: DeployParams = {
        appID: appDetailStore.appID,
        envName: env.name,
        imageTag: formModel.imageTag,
        replicas: Number(formModel.replicas),
      };
      if (shouldBuildFromSource.value) {
        Object.assign(params, revisionRequestFields(showCodeSource.value, formModel.revisionType, formModel.revision));
        const buildRes = await deployAPIs.buildAndCreateDeploy!(params);
        // 后端状态类型为 string，这里收窄为 BuildTips 支持的状态。
        const validStatuses: BuildStatus[] = ['failed', 'pollingBroken', 'running', 'success', 'warning'];
        const status = validStatuses.includes(buildRes.status as BuildStatus)
          ? (buildRes.status as BuildStatus)
          : 'warning';
        // 将后续展示、日志请求和流水线跳转所需字段统一保存到 buildInfo。
        Object.assign(buildInfo, {
          buildID: buildRes.buildID || '',
          imageTag: buildRes.params?.BKMS_IMAGE_TAG || formModel.imageTag,
          operator: buildRes.operator || '',
          pipelineID: buildRes.pipelineID || '',
          revision: buildRes.revision || formModel.revision,
          status,
        });
      } else {
        // 已构建镜像 和 “仅配置”均直接部署；“仅配置”的镜像 Tag 已回填为当前部署版本。
        // 已构建镜像 + 分批发布 + 已选策略时透传发布策略；构建部署接口无该字段。
        if (formModel.releaseMode === 'canary' && formModel.releaseStrategyID) {
          params.releaseStrategyID = formModel.releaseStrategyID;
        }
        await deployAPIs.createDeployDirectly!(params);
      }
      return true;
    } catch (error) {
      console.warn(error);
      return false;
    }
  }

  // 部署
  async function handleDeploy() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    // 提交前按最新资格兜底校验：分批发布已失效（如主工作负载异步判定为 Deployment）时拦截，
    // 不把后端必拒的 releaseStrategyID 透传出去。
    if (formModel.releaseMode === 'canary' && canaryDisabledReason.value) {
      Message({ theme: 'error', message: canaryDisabledReason.value });
      return;
    }

    loading.value = true;
    try {
      let result = false;
      if (formModel.updateContent !== 'image') {
        result = await handleConfigAndImage();
      } else {
        result = await handleImage();
      }

      if (!result) return;

      if (shouldBuildFromSource.value) {
        showBuildLog.value = true;
      }

      forceCleanDirtyTag(() => {
        // 源码构建后保留当前侧滑，以便持续展示流式日志；其他部署方式沿用成功后关闭侧滑的交互。
        emit('update', shouldBuildFromSource.value);
        Message({
          theme: 'success',
          message: t('操作成功'),
        });
      });
    } catch (error) {
      console.warn(error);
    } finally {
      loading.value = false;
    }
  }

  /**
   * 仅镜像
   */
  async function handleImage() {
    if (isFederationEnv.value) return false;
    try {
      const params: UpdateAppInstancesRequest = {
        appID: appDetailStore.appID,
        envName: trpcDeployStore.curEnvItem?.name ?? '',
        imageTag: formModel.imageTag,
        updateStrategy: 'InplaceUpdate',
        // 全量更新传空数组
        instanceIDs: [],
      };
      await InstanceService.updateAppInstances(params);
      return true;
    } catch (error) {
      console.warn(error);
      return false;
    }
  }

  /**
   * 切换更新内容
   * @param val 'both' | 'config' | 'image'
   */
  function handleUpdateContentChange(val: 'both' | 'config' | 'image') {
    // 仅配置时,使用当前镜像 Tag;其他情况清空让用户选择
    formModel.imageTag = val === 'config' ? curImageTag.value : '';
    // 仅镜像（原地重启）不支持分批发布，切回时静默重置，避免触发禁用重置提示
    if (val === 'image' && formModel.releaseMode === 'canary') {
      formModel.releaseMode = 'all';
      formModel.releaseStrategyID = '';
    }
  }

  watch(isShow, async val => {
    if (!val) {
      withPausedWatch(() => {
        formModel.revision = '';
        formModel.revisionType = 'branch';
        formModel.updateContent = 'both';
        formModel.imageTag = '';
        formModel.replicas = undefined;
        formModel.releaseMode = 'all';
        formModel.releaseStrategyID = '';
      });
      imageSource.value = 'image';
      showBuildLog.value = false;
      formRef.value?.clearValidate();
    } else {
      // 打开弹窗时回填当前生效实例数；withPausedWatch 避免初始化赋值被标记为未保存修改。
      withPausedWatch(() => {
        formModel.updateContent = deployDisabledReason.value ? 'image' : 'both';
        formModel.replicas = isValidReplicas(props.effectiveReplicas) ? props.effectiveReplicas : undefined;
      });
      // 打开弹窗时获取当前镜像 Tag
      await fetchCurrentImageTag();
    }
  });

  // 关闭构建日志内容区时重置本次构建信息。
  watch(
    showBuildLog,
    val => {
      if (!val) {
        Object.assign(buildInfo, {
          buildID: '',
          imageTag: '',
          operator: '',
          pipelineID: '',
          revision: '',
          status: 'running',
        });
      }
    },
    { immediate: false },
  );

  // 部署页轮询到同一构建的新状态后，同步更新当前日志提示。
  watch([() => props.latestBuildId, () => props.latestBuildStatus], ([latestBuildId, latestBuildStatus]) => {
    if (showBuildLog.value && latestBuildId === buildInfo.buildID && latestBuildStatus) {
      buildInfo.status = latestBuildStatus;
    }
  });
</script>

<style lang="postcss" scoped>
  .full-update-sideslider :deep(.bk-modal-content) {
    height: calc(100% - 52px) !important;
    overflow: hidden !important;
    scrollbar-gutter: auto !important;
    > div {
      height: 100%;
      .bk-sideslider-content {
        height: 100%;
      }
    }
  }
</style>
