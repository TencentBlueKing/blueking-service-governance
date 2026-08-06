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
  <div class="h-full overflow-auto min-w-[1366px]">
    <!-- 无空间：引导页 -->
    <template v-if="!skeletonLoading && (overviewList?.length ?? 0) === 0">
      <EmptyHome
        @create="isShowCreateSpace = true"
        @guide="handleGuide"
      />
    </template>

    <!-- 有空间 -->
    <template v-else>
      <div class="flex justify-center mb-[50px]">
        <div class="max-w-[1600px] w-full mt-[24px] px-[24px] pb-[24px]">
          <Skeleton
            :full-height="false"
            :loading="skeletonLoading"
          >
            <template #loading>
              <div class="flex items-center justify-between p-[16px]">
                <div class="flex items-center">
                  <Layout.shape
                    :height="60"
                    :width="60"
                  />
                  <div class="flex flex-col ml-[16px]">
                    <Layout.shape
                      :height="32"
                      :width="160"
                    />
                    <Layout.shape
                      class="mt-[6px]"
                      :height="22"
                      :width="120"
                    />
                  </div>
                </div>
                <Layout.shape
                  :height="32"
                  :width="110"
                />
              </div>
              <div class="flex items-center justify-between px-[16px] pt-[16px] my-[16px]">
                <Layout.shape
                  :height="30"
                  :width="200"
                />
                <Layout.shape
                  :height="30"
                  :width="200"
                />
              </div>
              <div class="grid grid-cols-2 gap-[16px] pb-[24px] px-[16px]">
                <Layout.shape
                  v-for="i in 2"
                  :key="i"
                  :height="300"
                  width="100%"
                />
              </div>
            </template>

            <!-- 顶部问候栏 -->
            <div
              class="flex items-center justify-between p-[16px] mb-[24px] rounded-[8px] bg-[#FFF] shadow-[0_2px_4px_0_#1919290d]"
            >
              <div class="flex items-center">
                <div class="w-[60px] h-[60px] rounded-[12px] bg-[#F0F5FF] flex items-center justify-center mr-[16px]">
                  <Coffee class="w-[28px] h-[28px] animate-bounce-coffee" />
                </div>
                <div>
                  <div class="text-[20px] font-bold text-[#313238]">
                    {{ greetingText }}，
                    <span class="bg-clip-text text-transparent bg-gradient-to-r from-[#614BFA] to-[#3A84FF]"
                      >{{ userStore.userInfo.user_id || '--' }} ~
                    </span>
                  </div>
                  <div class="text-[14px] text-[#979BA5] mt-[4px]">{{ $t('欢迎回到蓝鲸服务治理平台') }} ~</div>
                </div>
              </div>
              <Button
                theme="primary"
                @click="isShowCreateSpace = true"
              >
                <Plus
                  :height="24"
                  :width="24"
                />
                {{ $t('新建空间') }}
              </Button>
            </div>

            <!-- 最近操作区域 -->
            <div class="flex items-center justify-between mb-[16px]">
              <span class="text-[16px] font-bold text-[#313238]">{{ $t('最近操作') }}</span>
              <Button
                text
                theme="primary"
                @click="handleGoAllSpaces"
              >
                {{ $t('启用空间') }}（{{ enableSpaceCount }}）
                <i class="bkms-icon bkms-icon-expand-small rotate-270 text-[16px] text-[#3a84ff]"></i>
              </Button>
            </div>

            <!-- 空间卡片列表 -->
            <div :class="['grid gap-[16px]', (overviewList?.length ?? 0) === 1 ? 'grid-cols-1' : 'grid-cols-2']">
              <div
                v-for="(space, spaceIndex) in overviewList"
                :key="space.id"
                class="bg-[#fff] rounded-[8px] shadow-[0_2px_4px_0_#1919290d] overflow-hidden"
              >
                <!-- 卡片头部 -->
                <div class="flex items-center justify-between p-[16px]">
                  <div
                    class="flex items-center cursor-pointer"
                    @click="handleChangeSpace(space)"
                  >
                    <div
                      class="w-[40px] h-[40px] rounded-[8px] flex items-center justify-center text-[#fff] text-[18px] font-bold mr-[12px] shrink-0"
                      :style="{ backgroundColor: getSpaceColor(spaceIndex) }"
                    >
                      {{ getFirstChar(space?.displayName ?? '') }}
                    </div>
                    <div class="flex flex-col">
                      <span class="text-[14px] font-bold text-[#313238]">{{ space.displayName }}</span>
                      <span class="text-[12px] text-[#979BA5]">
                        {{ $t('共 {0} 个应用', [space.apps?.length ?? 0]) }}
                      </span>
                    </div>
                  </div>
                  <Button
                    text
                    theme="primary"
                    @click="handleChangeSpace(space)"
                  >
                    {{ $t('进入空间') }}
                    <i class="bkms-icon bkms-icon-expand-small rotate-270 text-[16px] text-[#3a84ff]"></i>
                  </Button>
                </div>

                <div class="p-[16px] pt-[0]">
                  <Table
                    class="w-full"
                    :data="(space.apps ?? []).slice(0, 5)"
                    :row-height="42"
                    show-overflow="tooltip"
                  >
                    <template #empty>
                      <TableException />
                    </template>
                    <TableColumn
                      field="name"
                      :label="$t('应用名称')"
                      :min-width="120"
                      show-overflow="tooltip"
                    >
                      <template #default="{ row }: { row: AppInfoOutputObj }">
                        <div class="flex items-center leading-[22px]">
                          <TypeIcon
                            classes="min-w-[40px] inline-block"
                            :show-label="false"
                            :type="row.type"
                          />
                          <Button
                            class="app-name-button"
                            text
                            theme="primary"
                            @click="handleShowAppDetail(space?.id ?? '', row)"
                            >{{ row.name || '--' }}</Button
                          >
                        </div>
                      </template>
                    </TableColumn>
                    <TableColumn
                      field="deployedEnvs"
                      :label="$t('已部署环境')"
                      :min-width="220"
                    >
                      <template #default="{ row }: { row: AppInfoOutputObj }">
                        <div class="flex gap-[4px]">
                          <MoreTag
                            v-if="groupEnvs(row?.deployedEnvs ?? []).length"
                            :data="groupEnvs(row?.deployedEnvs ?? [])"
                          >
                            <template #default="{ item: env }">
                              <Popover
                                :disabled="!getGroupedEnvNames(env).length"
                                theme="popover-dark-translucent"
                                trigger="hover"
                              >
                                <Tag
                                  class="mr-[4px]"
                                  :theme="envTypeMap[getGroupedEnvType(env)]?.theme || 'info'"
                                >
                                  {{ getGroupedEnvLabel(env) }}（{{ getGroupedEnvCount(env) }}）
                                </Tag>
                                <template #content>
                                  <div class="flex flex-col gap-[6px]">
                                    <div
                                      v-for="item in getGroupedEnvEnvs(env)"
                                      :key="item.id"
                                      class="flex items-center gap-[6px]"
                                    >
                                      <ColorIcon
                                        :icon="
                                          row.type && item?.deployStatus
                                            ? getDeployStatusInfo(row.type as AppType, item.deployStatus)?.icon
                                            : ''
                                        "
                                        :size="12"
                                      />
                                      <span>{{ item.displayName }}</span>
                                      <span class="text-[#979BA5]">
                                        （{{
                                          row.type && item?.deployStatus
                                            ? getDeployStatusInfo(row.type as AppType, item.deployStatus)?.text
                                            : '--'
                                        }}）
                                      </span>
                                    </div>
                                  </div>
                                </template>
                              </Popover>
                            </template>
                          </MoreTag>
                          <span
                            v-else
                            class="text-[#979BA5]"
                            >--</span
                          >
                        </div>
                      </template>
                    </TableColumn>
                    <TableColumn
                      field="lastOperatedAt"
                      :label="$t('最后操作时间')"
                      :min-width="100"
                    >
                      <template #default="{ row }: { row: AppInfoOutputObj }">
                        <span
                          v-bk-tooltips="{
                            content: formatRelativeTimeWithTooltip(row.lastOperatedAt).tooltip,
                            disabled: !formatRelativeTimeWithTooltip(row.lastOperatedAt).tooltip,
                          }"
                          class="text-[#979BA5]"
                          >{{ formatRelativeTimeWithTooltip(row.lastOperatedAt).text }}</span
                        >
                      </template>
                    </TableColumn>
                  </Table>
                </div>
              </div>
            </div>
          </Skeleton>
        </div>
      </div>
    </template>

    <!-- 新增空间 -->
    <TeamSpace
      v-model:is-show="isShowCreateSpace"
      @confirm="handleRefresh"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Popover, Tag } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useRouter } from 'vue-router';
  import { WorkspaceService } from '~/api/modules/v1';
  import { DOC_LINKS } from '~/common/const';
  import ColorIcon from '~/components/color-icon.vue';
  import MoreTag from '~/components/more-tag.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { AppType } from '~/composables/app-type';
  import { useDeployStatusMap } from '~/composables/use-deploy-status';
  import { envTypeMap } from '~/composables/use-env-manager';
  import { formatRelativeTimeWithTooltip, useGreeting } from '~/composables/use-time';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';
  import { useUserStore } from '~/stores/user';

  import TypeIcon from '../application/components/type-icon.vue';
  import EmptyHome from './empty-home.vue';
  import TeamSpace from './team-space.vue';
  import Coffee from '@/assets/coffee.svg';

  import type { AppDeployedEnvOutputObj, AppInfoOutputObj } from '~/@types/v1/app';
  import type { WorkspaceWithAppsOutputObj } from '~/@types/v1/workspace';

  /** groupEnvs 返回的分组环境对象 */
  interface GroupedEnv {
    count: number;
    envs: AppDeployedEnvOutputObj[];
    label: string;
    names: string[];
    type: string;
  }

  const router = useRouter();
  const spaceStore = useSpaceStore();
  const userStore = useUserStore();
  const appDetailStore = useAppDetail();
  const { getDeployStatusInfo } = useDeployStatusMap();

  const skeletonLoading = ref(false);
  const isShowCreateSpace = ref(false);

  // 可用空间数量
  const enableSpaceCount = computed(
    () => spaceStore.list?.filter(item => item.state === spaceStore.spaceState.Ready).length ?? 0,
  );

  // 空间 icon 颜色循环
  const spaceColors = ['#3A84FF', '#2CAF5E', '#F59500', '#6228FF', '#EF6317', '#EA3636'];

  function getGroupedEnvCount(item: unknown): number {
    return isGroupedEnv(item) ? item.count : 0;
  }

  function getGroupedEnvEnvs(item: unknown): AppDeployedEnvOutputObj[] {
    return isGroupedEnv(item) ? item.envs : [];
  }

  function getGroupedEnvLabel(item: unknown): string {
    return isGroupedEnv(item) ? item.label : '';
  }

  /** 安全获取 GroupedEnv 的属性 */
  function getGroupedEnvNames(item: unknown): string[] {
    return isGroupedEnv(item) ? item.names : [];
  }

  function getGroupedEnvType(item: unknown): string {
    return isGroupedEnv(item) ? item.type : '';
  }

  // 按 type 分组统计环境数量
  function groupEnvs(envs: AppDeployedEnvOutputObj[]): GroupedEnv[] {
    if (!envs?.length) return [];
    const countMap: Record<string, number> = {};
    const envMap: Record<string, AppDeployedEnvOutputObj[]> = {};
    envs.forEach(env => {
      if (!env.type) return;
      countMap[env.type] = (countMap[env.type] ?? 0) + 1;
      (envMap[env.type] ??= []).push(env);
    });
    return Object.entries(countMap).map(([type, count]) => ({
      count,
      label: envTypeMap[type]?.name ?? type,
      names: (envMap[type] ?? []).map(e => e.displayName).filter((name): name is string => !!name),
      envs: envMap[type] ?? [],
      type,
    }));
  }

  /** 类型守卫：检查 unknown 对象是否为 GroupedEnv */
  function isGroupedEnv(item: unknown): item is GroupedEnv {
    return (
      typeof item === 'object' &&
      item !== null &&
      'count' in item &&
      'envs' in item &&
      'label' in item &&
      'names' in item &&
      'type' in item
    );
  }

  // 概览列表（含应用）
  const overviewList = ref<WorkspaceWithAppsOutputObj[]>([]);

  // 问候语
  const { greetingText } = useGreeting();

  // 获取空间首字符
  function getFirstChar(name: string) {
    return name?.charAt(0) || '?';
  }

  function getSpaceColor(index: number) {
    return spaceColors[index % spaceColors.length];
  }

  // 切换空间
  function handleChangeSpace(space: WorkspaceWithAppsOutputObj) {
    spaceStore.updateCurrentSpace(space.id ?? '');
    router.push({ name: 'app', params: { space: space.id ?? '' } });
  }

  // 跳转全部空间
  function handleGoAllSpaces() {
    router.push({ name: 'spaceList' });
  }

  // 查看接入指引
  function handleGuide() {
    window.open(`${import.meta.env.BK_DOC_URL}${DOC_LINKS.ACCESS_GUIDE}`, '_blank');
  }

  // 刷新：新建空间后重新拉取
  async function handleRefresh() {
    await Promise.all([spaceStore.handleGetWorkspaceList(), loadOverview()]);
  }

  // 进入应用详情
  function handleShowAppDetail(spaceId: string, app: AppInfoOutputObj) {
    spaceStore.updateCurrentSpace(spaceId);
    appDetailStore.updateAppID(app.id || '');
    router.push({
      name: 'detail',
      params: {
        space: spaceId,
        name: app.name,
        menuName: 'info',
        type: app.type || '',
      },
    });
  }

  // 获取概览数据（空间 + 应用）
  async function loadOverview() {
    const res = await WorkspaceService.listWorkspacesOverview({ limit: 6 }).catch(() => []);
    overviewList.value = Array.isArray(res) ? res : [];
  }

  onMounted(async () => {
    skeletonLoading.value = true;
    await Promise.all([spaceStore.handleGetWorkspaceList(), loadOverview()]);
    skeletonLoading.value = false;
  });
</script>

<style lang="postcss" scoped>
  @keyframes bounce-coffee {
    0% {
      transform: translateY(0);
    }

    40% {
      transform: translateY(-6px);
    }

    100% {
      transform: translateY(0);
    }
  }

  .animate-bounce-coffee {
    animation: bounce-coffee 1.5s cubic-bezier(0.25, 0.46, 0.45, 0.94) 0.5s infinite;
  }

  :deep(.app-name-button .bk-button-text) {
    line-height: 22px;
  }
</style>
