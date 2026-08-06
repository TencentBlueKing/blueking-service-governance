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
  <div>
    <Skeleton
      class="bg-[#fff] pb-[18px]"
      :full-height="false"
      :loading="isLoading || appDetailStore.loading"
    >
      <template #loading>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="grid grid-cols-2 gap-4 gap-y-2 text-[12px] my-[20px] pl-[60px]">
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
        </div>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="grid grid-cols-2 gap-4 gap-y-2 text-[12px] my-[20px] pl-[60px]">
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
        </div>
        <Layout.shape
          :height="28"
          width="100%"
        />
        <div class="grid grid-cols-2 gap-4 gap-y-2 text-[12px] my-[20px] pl-[60px]">
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
          <Layout.formItem />
        </div>
      </template>
      <div class="flex flex-col gap-[16px]">
        <BkmsContent
          class="info-title shadow-[0_2px_4px_0_#1919290d]"
          :title="$t('基础信息')"
        >
          <div class="grid grid-cols-[minmax(0,40%)_minmax(0,60%)] gap-[12px] gap-y-2 text-[12px] py-[16px] bg-[#fff]">
            <DetailItem
              class="!h-[20px]"
              :label="$t('类型')"
              :label-width="160"
            >
              <TypeIcon :type="isAgones ? 'agones' : 'helm'">
                <template #label>
                  <span class="text-[#313238] leading-[20px] ml-[6px]">
                    {{ $t(isAgones ? 'Agones 应用' : 'Helm 应用') }}
                  </span>
                </template>
              </TypeIcon>
            </DetailItem>
            <DetailItem
              class="!h-[20px]"
              :label="$t('工作空间')"
              :label-width="100"
              :value="appData?.workspaceID"
            >
            </DetailItem>
            <DetailItem
              class="!h-[20px]"
              :label="$t('应用名称')"
              :label-width="160"
              :value="appData?.name"
            >
            </DetailItem>
            <DetailItem
              class="!h-[20px]"
              :label="$t('应用 ID')"
              :label-width="100"
              :value="appData?.id"
            >
            </DetailItem>
          </div>
        </BkmsContent>
        <BkmsContent
          class="info-title shadow-[0_2px_4px_0_#1919290d]"
          show-edit-icon
          :title="$t('Helm chart 配置')"
          @edit="showSourceConfig = true"
        >
          <div class="grid grid-cols-[minmax(0,40%)_minmax(0,60%)] gap-[12px] gap-y-2 text-[12px] py-[16px] bg-[#fff]">
            <DetailItem
              class="!h-[20px]"
              :label="$t('Helm Chart 来源')"
              :label-width="160"
              :value="appData?.helmSpec?.helmSource?.repoType"
            >
            </DetailItem>
            <template v-if="appData?.helmSpec?.helmSource?.repoType === 'GitRepo'">
              <DetailItem
                class="!h-[20px]"
                :label="$t('代码库')"
                :label-width="100"
              >
                <div class="flex items-center">
                  <span>{{ appData?.helmSpec?.helmSource?.gitRepoConfig?.repoURL }}</span>
                  <Button
                    class="ml-[6px]"
                    text
                    @click="copyText(appData?.helmSpec?.helmSource?.gitRepoConfig?.repoURL || '')"
                  >
                    <Copy
                      class="hover:text-[#3A84FF]"
                      :title="$t('复制')"
                    />
                  </Button>
                </div>
              </DetailItem>
              <DetailItem
                class="!h-[20px]"
                :label="$t('分支')"
                :label-width="160"
                :value="appData?.helmSpec?.helmSource?.gitRepoConfig?.revision"
              />
              <DetailItem
                class="!h-[20px]"
                :label="$t('路径')"
                :label-width="100"
                :value="appData?.helmSpec?.helmSource?.gitRepoConfig?.sourceDir"
              />
            </template>
            <template v-else-if="appData?.helmSpec?.helmSource?.repoType === 'HelmRepo'">
              <DetailItem
                class="!h-[20px]"
                :label="$t('仓库地址')"
                :label-width="100"
              >
                <div class="flex items-center">
                  <span>{{ appData?.helmSpec?.helmSource?.helmRepoConfig?.repoURL || '--' }}</span>
                  <Button
                    v-if="appData?.helmSpec?.helmSource?.helmRepoConfig?.repoURL"
                    class="ml-[6px]"
                    text
                    @click="copyText(appData?.helmSpec?.helmSource?.helmRepoConfig?.repoURL || '')"
                  >
                    <Copy
                      class="hover:text-[#3A84FF]"
                      :title="$t('复制')"
                    />
                  </Button>
                </div>
              </DetailItem>
              <DetailItem
                class="!h-[20px]"
                label="Chart"
                :label-width="160"
                :value="appData?.helmSpec?.helmSource?.helmRepoConfig?.chartName || '--'"
              >
              </DetailItem>
            </template>
          </div>
        </BkmsContent>
        <BkmsContent
          class="info-title shadow-[0_2px_4px_0_#1919290d]"
          show-edit-icon
          :title="$t('构建配置')"
          @edit="showBuildConfig = true"
        >
          <div class="grid grid-cols-[minmax(0,40%)_minmax(0,60%)] gap-[12px] gap-y-2 text-[12px] py-[16px] bg-[#fff]">
            <DetailItem
              class="!h-[20px]"
              :label="$t('应用镜像来源')"
              :label-width="160"
            >
              {{ appData?.buildConfig?.sourceType === 'imageRegistry' ? $t('镜像仓库') : $t('代码仓库') }}
            </DetailItem>
            <DetailItem
              class="!h-[20px]"
              :label="appData?.buildConfig?.sourceType === 'imageRegistry' ? $t('镜像仓库') : $t('代码仓库')"
              :label-width="100"
            >
              <div
                v-if="repoURL"
                class="flex items-center"
              >
                <span>{{ repoURL }}</span>
                <Button
                  class="ml-[6px]"
                  text
                  @click="copyText(repoURL)"
                >
                  <Copy
                    class="hover:text-[#3A84FF]"
                    :title="$t('复制')"
                  />
                </Button>
              </div>
              <span v-else>--</span>
            </DetailItem>
            <template v-if="appData?.buildConfig?.sourceType === 'codeRepository'">
              <DetailItem
                class="!h-[20px]"
                :label="$t('默认分支')"
                :label-width="160"
              >
                {{ appData?.buildConfig?.repoBuildConfig?.defaultBranch || '--' }}
              </DetailItem>
              <DetailItem
                class="!h-[20px]"
                :label="$t('构建目录')"
                :label-width="100"
              >
                {{ appData?.buildConfig?.repoBuildConfig?.sourceDir || '--' }}
              </DetailItem>
              <DetailItem
                class="!h-[20px]"
                :label="$t('Dockerfile 路径')"
                :label-width="160"
              >
                {{ appData?.buildConfig?.repoBuildConfig?.dockerfile || '--' }}
              </DetailItem>
              <DetailItem
                :class="isEmpty(appData?.buildConfig?.repoBuildConfig?.dockerBuildArgs) ? '!h-[20px]' : ''"
                :label="$t('构建参数')"
                :label-width="100"
              >
                {{ isEmpty(appData?.buildConfig?.repoBuildConfig?.dockerBuildArgs) ? '--' : '' }}
                <div class="flex flex-wrap gap-[4px]">
                  <Tag
                    v-for="(v, k) in appData?.buildConfig?.repoBuildConfig?.dockerBuildArgs || {}"
                    :key="`${k}+${v}`"
                    class="max-w-[200px]"
                  >
                    <OverflowTitle type="tips">{{ `${k} = ${v}` }}</OverflowTitle>
                  </Tag>
                </div>
              </DetailItem>
            </template>
            <DetailItem
              v-if="appData?.buildConfig?.sourceType !== 'imageRegistry'"
              class="!h-[20px]"
              :label="$t('推荐版本号')"
              :label-width="160"
              :value="tagConfigDisplayText"
            />
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
      <!-- Helm chart 配置 -->
      <HelmSourceConfigSideslider
        :data="appData.helmSpec"
        :is-show="showSourceConfig"
        @close="showSourceConfig = false"
        @update="handleUpdate"
      />
      <!-- 构建配置 -->
      <HelmBuildConfigSideslider
        :data="appData.buildConfig"
        :is-show="showBuildConfig"
        @close="showBuildConfig = false"
        @update="handleUpdate"
      />
    </Skeleton>
  </div>
</template>
<script setup lang="ts">
  import { computed, onBeforeMount, ref, watch } from 'vue';

  import { Button, OverflowTitle, Tag } from 'bkui-vue';
  import { Copy, ExclamationCircleShape } from 'bkui-vue/lib/icon';
  import { isEmpty } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { AppDetailOutputObj } from '~/@types/v1/app';
  import { copyText } from '~/common/util';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useAgonesFromAppDetail } from '~/composables/use-agones';
  import { getTagConfigDisplayText } from '~/composables/use-tag-config';
  import DeleteAppDialog from '~/pages/application/components/delete-app-dialog.vue';
  import TypeIcon from '~/pages/application/components/type-icon.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import HelmBuildConfigSideslider from './helm-build-config.vue';
  import HelmSourceConfigSideslider from './helm-source-config.vue';

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const appData = ref<AppDetailOutputObj>({} as AppDetailOutputObj);

  // 使用 Agones Hook 获取应用类型相关信息
  const { isAgones } = useAgonesFromAppDetail(appData);
  const isLoading = ref(false);

  // 来源信息
  const repoURL = computed(() => {
    if (appData.value?.buildConfig?.sourceType === 'imageRegistry') {
      return appData.value?.buildConfig?.imageBuildConfig?.name;
    }
    if (appData.value?.buildConfig?.sourceType === 'codeRepository') {
      return appData.value?.buildConfig?.repoBuildConfig?.repoURL;
    }
    return '';
  });

  // 推荐版本号展示文本
  const tagConfigDisplayText = computed(() => getTagConfigDisplayText(appData.value?.buildConfig?.tagConfig, t));

  const showSourceConfig = ref(false);
  // 构建配置
  const showBuildConfig = ref<boolean>(false);
  // 删除应用
  const showDeleteDialog = ref(false);

  // 获取应用信息
  async function handleGetApp() {
    isLoading.value = true;
    try {
      const detail = await appDetailStore.fetchAppDetail();
      appData.value = detail || ({} as AppDetailOutputObj);
    } finally {
      isLoading.value = false;
    }
  }

  function handleUpdate() {
    showBuildConfig.value = false;
    showSourceConfig.value = false;
    handleGetApp();
  }

  // 监听应用详情变化
  watch(
    () => appDetailStore.appDetail,
    newDetail => {
      appData.value = newDetail || ({} as AppDetailOutputObj);
    },
    { immediate: true },
  );

  onBeforeMount(() => {
    handleGetApp();
  });
</script>

<style lang="postcss" scoped>
  .info-title :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
