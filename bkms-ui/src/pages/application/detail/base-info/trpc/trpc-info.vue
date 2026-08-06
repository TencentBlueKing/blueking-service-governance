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
    <Skeleton :loading="isLoading || appDetailStore.loading">
      <template #loading>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="grid grid-cols-2 gap-4 gap-y-2 my-[16px] pl-[60px]">
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
        </div>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="grid grid-cols-2 gap-4 gap-y-2 my-[16px] pl-[60px]">
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
        </div>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="grid grid-cols-2 gap-4 gap-y-2 my-[16px] pl-[60px]">
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
        </div>
        <div>
          <Layout.shape
            :height="28"
            width="100%"
          />
          <Layout.shape
            class="mt-[12px] ml-[16px]"
            :height="32"
            :width="240"
          />
          <Layout.shape
            class="mt-[12px] mx-[16px]"
            :height="32"
            width="calc(100% - 32px)"
          />
          <Layout.shape
            class="my-[12px] ml-[16px]"
            :height="32"
            :width="110"
          />
        </div>
      </template>
      <div class="flex flex-col gap-[16px]">
        <BkmsContent
          class="info-title shadow-[0_2px_4px_0_#1919290d]"
          :title="$t('基础信息')"
        >
          <div class="grid grid-cols-2 gap-[12px] gap-y-2 p-[16px] pl-[60px] bg-[#fff]">
            <FieldItem
              :container-height="20"
              :field-value="$t('类型')"
            >
              <template #value>
                <div class="shrink-0">
                  <TypeIcon :type="appType">
                    <template #label>
                      <span class="text-[#313238] leading-[20px] text-[14px] font-bold ml-[6px]">
                        {{ appType === 'trpc' ? $t('tRPC 应用') : $t('TAF 应用') }}
                      </span>
                    </template>
                  </TypeIcon>
                </div>
              </template>
            </FieldItem>
            <FieldItem
              :container-height="20"
              :field-value="$t('空间名称')"
              :value="appData?.workspaceID"
              value-color="#313238"
            />
            <FieldItem
              :container-height="20"
              :field-value="$t('应用名称')"
              :value="appData?.name"
              value-color="#313238"
            />
            <FieldItem
              :container-height="20"
              :field-value="`${$t('应用')} ID`"
              :value="appData?.id"
              value-color="#313238"
            />
          </div>
        </BkmsContent>
        <BkmsContent
          class="info-title shadow-[0_2px_4px_0_#1919290d]"
          show-edit-icon
          :title="$t('构建配置')"
          @edit="handleBeforeEditBuilder"
        >
          <div class="grid grid-cols-2 gap-[12px] gap-y-2 p-[16px] pl-[60px] bg-[#fff]">
            <FieldItem
              :class="builderType === 'pipeline' ? 'col-span-2' : ''"
              :container-height="20"
              :field-value="$t('来源')"
              :value="curBuilderType"
              value-color="#313238"
            />
            <template v-if="builderType === 'codeRepository'">
              <FieldItem
                :container-height="20"
                :field-value="$t('代码库')"
                :value="getBuildConfigFieldValue('repoBuildConfig.repoURL')"
                value-color="#313238"
                value-max-width="100%"
              />
              <FieldItem
                :container-height="20"
                :field-value="$t('默认分支')"
                :value="getBuildConfigFieldValue('repoBuildConfig.defaultBranch')"
                value-color="#313238"
              />
              <FieldItem
                :container-height="20"
                :field-value="$t('构建目录')"
                :value="getBuildConfigFieldValue('repoBuildConfig.sourceDir')"
                value-color="#313238"
              />
              <!-- 构建方式 -->
              <FieldItem
                :container-height="20"
                :field-value="$t('构建方式')"
                value-color="#313238"
              >
                <template #value>
                  <span>{{
                    appData?.buildConfig?.repoBuildConfig?.imageBuildMode === 'platform'
                      ? $t('平台通用构建')
                      : $t('使用仓库 Dockerfile')
                  }}</span>
                </template>
              </FieldItem>
              <FieldItem
                :container-height="20"
                :field-value="$t('推荐版本号')"
                :value="tagConfigDisplayText"
                value-color="#313238"
              />
              <!-- 使用仓库 Dockerfile 模式 -->
              <template v-if="appData?.buildConfig?.repoBuildConfig?.imageBuildMode !== 'platform'">
                <FieldItem
                  :container-height="20"
                  :field-value="$t('Dockerfile 路径')"
                  :value="getBuildConfigFieldValue('repoBuildConfig.dockerfile')"
                  value-color="#313238"
                  value-max-width="100%"
                />
              </template>
              <!-- 平台通用构建模式 -->
              <template v-else>
                <FieldItem
                  :container-height="20"
                  :field-value="$t('构建镜像')"
                  :value="getBuildConfigFieldValue('repoBuildConfig.platformBuildConfig.builderImage')"
                  value-color="#313238"
                  value-max-width="100%"
                />
                <FieldItem
                  :container-height="20"
                  :field-value="$t('运行镜像')"
                  :value="getBuildConfigFieldValue('repoBuildConfig.platformBuildConfig.runnerImage')"
                  value-color="#313238"
                  value-max-width="100%"
                />
                <!-- 高级命令 -->
                <FieldItem
                  :class="['col-span-2 !items-start', { 'my-[4px]': platformCommandsPreBuild }]"
                  container-height="unset"
                  :field-value="$t('高级命令')"
                >
                  <template #value>
                    <span v-if="isPlatformCommandsEmpty">--</span>
                    <div
                      v-else
                      class="flex flex-col gap-0 flex-1 border rounded"
                    >
                      <ToggleCard
                        v-if="platformCommandsPreBuild"
                        class="rounded-[2px] overflow-hidden"
                        content-class="px-[38px] py-[8px] mt-0 border-t"
                        header-class="!hover:bg-[#f0f1f5]"
                        normal-bg-color="#F5F7FA"
                        type="normal"
                      >
                        <template #icon>
                          <i class="bkms-icon bkms-icon-angle-right font-bold"></i>
                        </template>
                        <template #title>
                          <div class="flex items-center gap-[8px] ml-[10px]">
                            <span>{{ $t('编译前置命令') }}</span>
                            <Tag theme="info">{{ $t('builder 阶段') }}</Tag>
                          </div>
                        </template>
                        <pre
                          class="m-0 text-[12px] font-mono leading-[22px] text-[#63656e] whitespace-pre-wrap"
                          v-text="platformCommandsPreBuild"
                        ></pre>
                      </ToggleCard>
                      <ToggleCard
                        v-if="platformCommandsBuild"
                        class="rounded-[2px] overflow-hidden"
                        content-class="px-[38px] py-[8px] mt-0 border-t"
                        header-class="!hover:bg-[#f0f1f5]"
                        normal-bg-color="#F5F7FA"
                        type="normal"
                      >
                        <template #icon>
                          <i class="bkms-icon bkms-icon-angle-right font-bold"></i>
                        </template>
                        <template #title>
                          <div class="flex items-center gap-[8px] ml-[10px]">
                            <span>{{ $t('编译命令') }}</span>
                            <Tag theme="info">{{ $t('builder 阶段') }}</Tag>
                          </div>
                        </template>
                        <pre
                          class="m-0 text-[12px] font-mono leading-[22px] text-[#63656e] whitespace-pre-wrap"
                          v-text="platformCommandsBuild"
                        ></pre>
                      </ToggleCard>
                      <ToggleCard
                        v-if="platformCommandsRuntimeEnv"
                        class="rounded-[2px] overflow-hidden"
                        content-class="px-[38px] py-[8px] mt-0 border-t"
                        header-class="!hover:bg-[#f0f1f5]"
                        normal-bg-color="#F5F7FA"
                        type="normal"
                      >
                        <template #icon>
                          <i class="bkms-icon bkms-icon-angle-right font-bold"></i>
                        </template>
                        <template #title>
                          <div class="flex items-center gap-[8px] ml-[10px]">
                            <span>{{ $t('运行环境命令') }}</span>
                            <Tag theme="success">{{ $t('runner 阶段') }}</Tag>
                          </div>
                        </template>
                        <pre
                          class="m-0 text-[12px] font-mono leading-[22px] text-[#63656e] whitespace-pre-wrap"
                          v-text="platformCommandsRuntimeEnv"
                        ></pre>
                      </ToggleCard>
                      <ToggleCard
                        v-if="platformCommandsStart"
                        class="rounded-[2px] overflow-hidden"
                        content-class="px-[38px] py-[8px] mt-0 border-t"
                        header-class="!hover:bg-[#f0f1f5]"
                        normal-bg-color="#F5F7FA"
                        type="normal"
                      >
                        <template #icon>
                          <i class="bkms-icon bkms-icon-angle-right font-bold"></i>
                        </template>
                        <template #title>
                          <div class="flex items-center gap-[8px] ml-[10px]">
                            <span>{{ $t('镜像入口命令') }}</span>
                            <Tag theme="success">{{ $t('runner 阶段') }}</Tag>
                          </div>
                        </template>
                        <pre
                          class="m-0 text-[12px] font-mono leading-[22px] text-[#63656e] whitespace-pre-wrap"
                          v-text="platformCommandsStart"
                        ></pre>
                      </ToggleCard>
                    </div>
                  </template>
                </FieldItem>
              </template>
              <!-- 转换 -->
              <FieldItem
                class="!items-start"
                :container-height="isEmpty(appData?.buildConfig?.repoBuildConfig?.dockerBuildArgs) ? 20 : 'auto'"
                :field-value="$t('构建参数')"
                value-color="#313238"
              >
                <template #value>
                  <div v-if="isEmpty(appData?.buildConfig?.repoBuildConfig?.dockerBuildArgs)">--</div>
                  <div
                    v-else
                    class="flex flex-wrap gap-[4px]"
                  >
                    <Tag
                      v-for="(v, k) in appData?.buildConfig?.repoBuildConfig?.dockerBuildArgs || {}"
                      :key="`${k}+${v}`"
                      class="max-w-[200px]"
                    >
                      <OverflowTitle type="tips">{{ `${k} = ${v}` }}</OverflowTitle>
                    </Tag>
                  </div>
                </template>
              </FieldItem>
            </template>
            <template v-else-if="builderType === 'pipeline'">
              <FieldItem
                :container-height="20"
                :field-value="$t('流水线库')"
                :value="getBuildConfigFieldValue('pipelineBuildConfig.pipelineID')"
                value-color="#313238"
              />
              <FieldItem
                :container-height="20"
                :field-value="$t('流水线参数配置')"
              >
                <template #value>
                  <Button
                    text
                    theme="primary"
                    @click="handleBeforeEditBuilder"
                    >{{ $t('查看配置') }}</Button
                  >
                </template>
              </FieldItem>
            </template>
          </div>
        </BkmsContent>

        <div class="flex items-center gap-[12px] mt-[8px]">
          <Button
            theme="danger"
            @click="showDeleteDialog = true"
          >
            {{ $t('删除应用') }}
          </Button>
          <div class="flex items-center gap-[4px] text-[12px]">
            <ExclamationCircleShape class="text-[14px] text-[#F59500]" />
            <span class="text-[#4D4F56]">
              {{ $t('删除应用后，所有相关的配置、部署信息和操作记录将永久丢失，此操作不可恢复。') }}
            </span>
          </div>
        </div>
      </div>
      <!-- 删除应用弹窗 -->
      <DeleteAppDialog v-model:is-show="showDeleteDialog" />
      <!-- 构建编辑 -->
      <EditBuilderConfig
        v-model:builder-data="builderData"
        v-model:is-show="showBuilderDialog"
        :language="appData?.appModelSpec?.trpcSpec?.language"
        :type="builderType"
        @confirm="handleUpdateApp('buildConfig', builderData)"
      />
    </Skeleton>
  </div>
</template>
<script setup lang="ts">
  import { computed, onBeforeMount, ref, watch } from 'vue';

  import { Button, Message, OverflowTitle, Tag } from 'bkui-vue';
  import { ExclamationCircleShape } from 'bkui-vue/lib/icon';
  import { cloneDeep, get, isEmpty, set } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { AppDetailOutputObj, BuildConfigOutputObj } from '~/@types/v1/app';
  import { UpdateBuildConfigRequest } from '~/@types/v1/builds';
  import { BuildsService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import ToggleCard from '~/components/toggle-card.vue';
  import { getTagConfigDisplayText, normalizeTagConfig } from '~/composables/use-tag-config';
  import DeleteAppDialog from '~/pages/application/components/delete-app-dialog.vue';
  import TypeIcon from '~/pages/application/components/type-icon.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import EditBuilderConfig from './edit-builder-config.vue';

  type BuilderData = {
    imageBuildConfig?: BuildConfigOutputObj['imageBuildConfig'];
    pipelineBuildConfig: BuildConfigOutputObj['pipelineBuildConfig'] | null;
    repoBuildConfig: BuildConfigOutputObj['repoBuildConfig'] | null;
    sourceType?: string;
    tagConfig?: BuildConfigOutputObj['tagConfig'] | null;
  };

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  // 应用类型（根据实际应用类型动态确定）
  const appType = computed(() => appDetailStore.appType);

  // 应用数据
  const appData = ref<AppDetailOutputObj>();

  // 删除应用
  const showDeleteDialog = ref(false);

  // 推荐版本号展示文本
  const tagConfigDisplayText = computed(() => getTagConfigDisplayText(appData.value?.buildConfig?.tagConfig, t));

  // 平台通用构建命令展示文本
  const platformCommands = computed(() => {
    return appData.value?.buildConfig?.repoBuildConfig?.platformBuildConfig?.commands || {};
  });
  const platformCommandsPreBuild = computed(() => {
    const cmds = platformCommands.value.preBuild;
    if (isEmpty(cmds)) return '';
    return (cmds as string[]).join('\n');
  });
  const platformCommandsBuild = computed(() => {
    const cmds = platformCommands.value.build;
    if (isEmpty(cmds)) return '';
    return (cmds as string[]).join('\n');
  });
  const platformCommandsRuntimeEnv = computed(() => {
    const cmds = platformCommands.value.runtimeEnv;
    if (isEmpty(cmds)) return '';
    return (cmds as string[]).join('\n');
  });
  const platformCommandsStart = computed(() => {
    const start = platformCommands.value.start;
    if (!start) return '';
    return start;
  });
  const isPlatformCommandsEmpty = computed(() => {
    const cmds = platformCommands.value;
    return isEmpty(cmds?.preBuild) && isEmpty(cmds?.build) && isEmpty(cmds?.runtimeEnv) && !cmds?.start;
  });

  /**
   * 保存组件
   */
  const updating = ref(false);
  // 构建 trpc UpdateBuildConfig 请求数据
  function formatUpdateBuildConfig(data: AppDetailOutputObj): UpdateBuildConfigRequest {
    const { buildConfig: buildConfigData } = data;
    return {
      appID: appDetailStore.appID,
      sourceType: buildConfigData?.sourceType,
      tagConfig: normalizeTagConfig(buildConfigData?.tagConfig),
      // 源码仓库类型
      codeRepo: buildConfigData?.repoBuildConfig,
      // 流水线类型
      pipeline: buildConfigData?.pipelineBuildConfig,
    } as UpdateBuildConfigRequest;
  }

  // 获取构建配置对应字段展示值
  function getBuildConfigFieldValue(key: string): string {
    const value = get(appData.value?.buildConfig, key);
    return value || '--';
  }

  // 更新构建配置信息
  async function handleUpdateApp(key: string, value: BuilderData) {
    if (!key || !appData.value) return false;

    updating.value = true;
    const data = cloneDeep<AppDetailOutputObj>(appData.value);
    set(data, key, value); // 更新数据
    const parmas = formatUpdateBuildConfig(data);
    const result = await BuildsService.updateBuildConfig(parmas, { validateCode: false })
      .then(() => true)
      .catch(() => false);
    updating.value = false;
    if (result) {
      Message({
        message: t('操作成功'),
        theme: 'success',
        delay: 1500,
      });
      showBuilderDialog.value = false;
      getData();
    }
    return result;
  }

  // 编辑应用构建配置
  const showBuilderDialog = ref(false);
  const builderData = ref<BuilderData>({
    pipelineBuildConfig: null,
    repoBuildConfig: null,
  } as BuilderData);
  const builderType = ref<'codeRepository' | 'pipeline'>('codeRepository'); // 构建类型，默认源码仓库

  const curBuilderType = computed(() => {
    const builderTypeTextMap = {
      codeRepository: t('源码仓库'),
      pipeline: t('流水线'),
    };
    return builderTypeTextMap[builderType.value];
  });

  // function handleChangeBuilderType(type: typeof builderType.value) {
  //   if (type === 'pipeline' && builderData.value) {
  //     builderData.value.pipeline = {
  //       name: '',
  //       pID: '',
  //       params: {},
  //     };
  //     (builderData.value as Partial<Builder>).source = undefined; // 清除源码仓库配置
  //   } else if (type === 'source' && builderData.value) {
  //     builderData.value.source = {
  //       name: '',
  //       git: '',
  //       branch: '',
  //       credential: '',
  //       system: 'linux',
  //       dockerfile: {
  //         content: '',
  //         path: '',
  //       },
  //     };
  //     (builderData.value as Partial<Builder>).pipeline = undefined; // 清除流水线配置
  //   }
  //   builderType.value = type;
  // }
  function handleBeforeEditBuilder() {
    builderData.value = cloneDeep(appData.value?.buildConfig || {}) as unknown as BuilderData;
    showBuilderDialog.value = true;
  }

  /**
   * 获取应用数据
   */
  const isLoading = ref(false);
  async function getData() {
    isLoading.value = true;
    try {
      const detail = await appDetailStore.fetchAppDetail();
      appData.value = detail || ({} as AppDetailOutputObj);
      builderType.value = (appData.value?.buildConfig?.sourceType as typeof builderType.value) || 'codeRepository';
    } finally {
      isLoading.value = false;
    }
  }

  // 监听应用详情变化，更新页面数据
  watch(
    () => appDetailStore.appDetail,
    newDetail => {
      appData.value = newDetail || ({} as AppDetailOutputObj);
      builderType.value = (newDetail?.buildConfig?.sourceType as typeof builderType.value) || 'codeRepository';
    },
    { immediate: true },
  );

  onBeforeMount(() => {
    getData();
  });
</script>

<style lang="postcss" scoped>
  .info-title :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
