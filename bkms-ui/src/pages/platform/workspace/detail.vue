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
  <section class="min-h-full bg-[#F5F7FA]">
    <MsHeader
      :title="t('空间详情')"
      :trigger-back="handleBack"
    >
      <span class="mx-[8px] h-[20px] w-[1px] bg-[#dcdee5]"></span>
      <span class="text-[16px] leading-[24px] text-[#979ba5]">{{ workspaceTitle }}</span>
    </MsHeader>

    <div class="p-[24px]">
      <!-- 加载失败 -->
      <Exception
        v-if="isError"
        class="mt-[120px]"
        :description="t('空间详情加载失败，请稍后重试')"
        scene="part"
        :title="t('加载失败')"
        type="500"
      >
        <Button
          theme="primary"
          @click="fetchWorkspaceDetail"
        >
          {{ t('重试') }}
        </Button>
      </Exception>

      <Skeleton
        v-else
        :loading="isLoading"
        :once="false"
      >
        <template #loading>
          <!-- 基本信息骨架屏 -->
          <div class="min-h-[160px] p-[24px] pt-[28px] mb-[24px]">
            <div class="flex items-center">
              <Layout.shape
                :height="72"
                :width="72"
              />
              <div class="flex-1 min-w-0 ml-[26px]">
                <Layout.shape
                  :height="20"
                  :width="180"
                />
                <div class="grid grid-cols-4 gap-x-[80px] mt-[8px]">
                  <Layout.formItem
                    :item-height="22"
                    :item-width="80"
                    :label-height="22"
                    :label-width="48"
                  />
                </div>
                <div class="flex mt-[8px]">
                  <Layout.shape
                    class="mr-[20px]"
                    :height="22"
                    :width="48"
                  />
                  <Layout.shape
                    :height="22"
                    width="60%"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- 快捷操作骨架屏 -->
          <div>
            <Layout.shape
              :height="42"
              width="100%"
            />
            <div class="flex items-center h-[100px] px-[120px]">
              <Layout.shape
                :height="20"
                :width="120"
              />
              <Layout.shape
                class="ml-[12px]"
                :height="20"
                :width="180"
              />
              <Layout.shape
                class="ml-[12px]"
                :height="32"
                :width="140"
              />
            </div>
          </div>
        </template>

        <!-- 空间基本信息 -->
        <div class="flex items-center min-h-[160px] p-[24px] pt-[28px] bg-white shadow-[0_2px_4px_0_#1919290d]">
          <!-- 首字母头像 -->
          <div
            class="relative flex shrink-0 items-center justify-center w-[72px] h-[72px] text-[32px] font-bold text-white rounded-[8px] bg-[#3a84ff]"
          >
            {{ workspaceInitial }}
          </div>

          <div class="min-w-0 ml-[26px] flex flex-col">
            <div class="text-[16px] font-bold text-[#313238] truncate">
              {{ workspaceTitle }}
            </div>
            <div class="flex flex-col">
              <div class="flex flex-wrap gap-x-[80px]">
                <ProbeDetailItem
                  :label="t('状态')"
                  :line-height="22"
                >
                  <StatusIcon
                    :status="workspaceDetail?.state || ''"
                    :status-color-map="workspaceStatusColorMap"
                    :status-text-map="workspaceStatusTextMap"
                  />
                </ProbeDetailItem>
                <ProbeDetailItem
                  :label="t('创建者')"
                  :line-height="22"
                  :value="workspaceDetail?.creator || '--'"
                />
                <ProbeDetailItem
                  :label="t('更新者')"
                  :line-height="22"
                  :value="workspaceDetail?.updater || '--'"
                />
              </div>
              <ProbeDetailItem
                :label="t('描述')"
                :line-height="22"
                :value="workspaceDetail?.description || '--'"
              />
            </div>
          </div>
        </div>

        <!-- 快捷操作 -->
        <BkmsContent
          class="mt-[24px] shadow-[0_2px_4px_0_#1919290d]"
          :collapsible="true"
          :title="t('快捷操作')"
        >
          <div class="flex items-center h-[100px] px-[120px] text-[14px] text-[#63656e] bg-[#fff]">
            <span>{{ t('空间权限') }}：</span>
            <span class="mr-[12px] text-[#313238]">{{ quickPermissionText }}</span>
            <!-- 管理员：退出空间 | 非管理员：成为管理员 -->
            <Button
              :class="{ 'mr-[1px]': !isCurrentUserWorkspaceAdmin }"
              :loading="isPermissionSubmitting"
              :style="{ borderRadius: isCurrentUserWorkspaceAdmin ? '2px' : '2px 0 0 2px' }"
              :theme="quickPermissionButtonTheme"
              @click="handleQuickPermissionAction"
            >
              {{ quickPermissionButtonText }}
            </Button>
            <!-- 非管理员可申请临时管理员 -->
            <Dropdown
              v-if="!isCurrentUserWorkspaceAdmin"
              placement="bottom-end"
              :popover-options="{ boundary: 'body', clickContentAutoHide: true }"
              trigger="click"
            >
              <Button
                class="!min-w-[32px] !px-0"
                style="border-radius: 0 2px 2px 0"
                theme="primary"
              >
                <AngleDownLine class="text-[12px]" />
              </Button>
              <template #content>
                <Dropdown.DropdownMenu>
                  <Dropdown.DropdownItem @click="handleGrantWorkspaceAdmin(true)">
                    {{ t('成为临时管理员，2 小时后自动退出') }}
                  </Dropdown.DropdownItem>
                </Dropdown.DropdownMenu>
              </template>
            </Dropdown>
            <!-- 管理员可访问空间 -->
            <Button
              v-if="isCurrentUserWorkspaceAdmin"
              v-bk-tooltips="{
                content: $t('空间已停用'),
                disabled: !isWorkspaceDisabled,
              }"
              class="ml-[8px]"
              :disabled="isWorkspaceDisabled"
              text
              theme="primary"
              @click="handleVisitWorkspace"
            >
              {{ $t('访问空间') }}
              <i class="bkms-icon bkms-icon-jump-link ml-[4px]"></i>
            </Button>
          </div>
        </BkmsContent>
      </Skeleton>
    </div>
  </section>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue';

  import { Button, Dropdown, Exception, Message } from 'bkui-vue';
  import { AngleDownLine } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { PlatmgtService } from '~/api/modules/v1';
  import BkmsContent from '~/components/bkms-content.vue';
  import MsHeader from '~/components/ms-header.vue';
  import Layout from '~/components/skeleton/skeleton-layout';
  import Skeleton from '~/components/skeleton/skeleton.vue';
  import StatusIcon from '~/components/status-icon.vue';
  import { useSpaceStore } from '~/stores/space';
  import { useUserStore } from '~/stores/user';

  import ProbeDetailItem from '../../application/detail/app-config/components/probe-detail-item.vue';

  import type { WorkspaceInfoOutput } from '~/@types/v1/platmgt';

  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const spaceStore = useSpaceStore();
  const userStore = useUserStore();

  // --- state ---
  const workspaceDetail = ref<null | WorkspaceInfoOutput>(null);
  /** 当前用户是否为空间管理员 */
  const isCurrentUserWorkspaceAdmin = ref(false);
  const isLoading = ref(false);
  const isError = ref(false);
  const isPermissionSubmitting = ref(false);
  /** 竞态控制：workspaceID 快速切换时，仅最后一次请求的结果生效 */
  const requestSequence = ref(0);

  const workspaceID = computed(() => String(route.params.workspaceID || ''));
  /** 当前登录用户名 */
  const currentUsername = computed(() => userStore.currentUsername);
  const quickPermissionText = computed(() =>
    isCurrentUserWorkspaceAdmin.value ? t('您已经是空间管理员') : t('您不具备管理权限'),
  );
  const quickPermissionButtonText = computed(() =>
    isCurrentUserWorkspaceAdmin.value ? t('退出空间') : t('成为管理员'),
  );
  const quickPermissionButtonTheme = computed(() => (isCurrentUserWorkspaceAdmin.value ? 'danger' : 'primary'));
  const isWorkspaceDisabled = computed(() => workspaceDetail.value?.state === 'Disabled');
  const workspaceTitle = computed(() => workspaceDetail.value?.displayName || workspaceID.value || '--');
  // 空间状态颜色映射
  const workspaceStatusColorMap = {
    Ready: 'green',
    Processing: 'orange',
    Disabled: 'gray',
  };
  const workspaceStatusTextMap = computed(() => ({
    Ready: t('启用中'),
    Processing: t('处理中'),
    Disabled: t('已停用'),
  }));
  /** 空间名称首字母，用于头像展示 */
  const workspaceInitial = computed(() => {
    const name = (workspaceDetail.value?.displayName || workspaceDetail.value?.id || workspaceID.value).trim();
    return Array.from(name)[0] || '--';
  });

  /** 查询当前用户空间管理员状态，驱动快捷操作按钮的展示状态 */
  async function fetchWorkspaceAdminStatus(id: string) {
    if (!currentUsername.value) {
      isCurrentUserWorkspaceAdmin.value = false;
      return;
    }

    const roleStatus = await PlatmgtService.getWorkspaceRoleStatus(
      {
        workspaceID: id,
        roleCode: 'admin',
        username: currentUsername.value,
      },
      { interceptorErr: false, validateCode: false },
    ).catch(() => ({ hasRole: false }));
    isCurrentUserWorkspaceAdmin.value = !!roleStatus.hasRole;
  }

  async function fetchWorkspaceDetail() {
    if (!workspaceID.value) return;

    const currentSequence = ++requestSequence.value;
    isLoading.value = true;
    isError.value = false;

    try {
      const [detail] = await Promise.all([
        PlatmgtService.getPlatWorkspace({ workspaceID: workspaceID.value }),
        fetchWorkspaceAdminStatus(workspaceID.value),
      ]);
      if (currentSequence !== requestSequence.value) return; // 丢弃过期请求
      workspaceDetail.value = detail;
    } catch {
      if (currentSequence !== requestSequence.value) return;
      workspaceDetail.value = null;
      isCurrentUserWorkspaceAdmin.value = false;
      isError.value = true;
    } finally {
      if (currentSequence === requestSequence.value) {
        isLoading.value = false;
      }
    }
  }

  function handleBack() {
    router.push({
      name: 'platformItem',
      params: {
        menuName: 'workspace',
      },
    });
  }

  /** 授予当前用户空间管理员身份，isTemporary=true 表示临时管理员 */
  async function handleGrantWorkspaceAdmin(isTemporary: boolean) {
    if (!validateQuickPermissionContext()) return;

    await submitPermissionRequest(
      PlatmgtService.grantWorkspaceAdmin({ workspaceID: workspaceID.value, isTemporary }, { validateCode: false }),
    );
  }

  /** 主按钮操作：管理员退出空间，非管理员成为永久管理员 */
  async function handleQuickPermissionAction() {
    if (!validateQuickPermissionContext()) return;

    await submitPermissionRequest(
      isCurrentUserWorkspaceAdmin.value
        ? PlatmgtService.revokeWorkspaceAdmin({ workspaceID: workspaceID.value }, { validateCode: false })
        : PlatmgtService.grantWorkspaceAdmin(
            {
              workspaceID: workspaceID.value,
              isTemporary: false,
            },
            { validateCode: false },
          ),
    );
  }

  /** 跳转至空间内应用列表 */
  function handleVisitWorkspace() {
    if (!workspaceID.value || isWorkspaceDisabled.value) return;

    router.push({
      name: 'app',
      params: {
        space: workspaceID.value,
      },
    });
  }

  /** 统一处理权限变更请求，成功后刷新当前用户管理员状态 */
  async function submitPermissionRequest(permissionRequest: Promise<unknown>) {
    if (isPermissionSubmitting.value) return;

    isPermissionSubmitting.value = true;
    try {
      const isSuccess = await permissionRequest.then(() => true).catch(() => false);
      if (!isSuccess) return;

      await spaceStore.handleGetWorkspaceList();
      await fetchWorkspaceAdminStatus(workspaceID.value);
      Message({
        message: t('操作成功'),
        theme: 'success',
      });
    } finally {
      isPermissionSubmitting.value = false;
    }
  }

  /** 校验快捷权限操作所需的空间 ID 和当前用户名 */
  function validateQuickPermissionContext() {
    if (!workspaceID.value || !currentUsername.value) {
      Message({
        message: t('当前用户信息获取失败，请刷新后重试'),
        theme: 'warning',
      });
      return false;
    }

    return true;
  }

  watch(workspaceID, fetchWorkspaceDetail);

  onMounted(async () => {
    if (!userStore.currentUsername) {
      await userStore.getRoleInfo();
    }
    fetchWorkspaceDetail();
  });
</script>

<style lang="postcss" scoped>
  /* 折叠面板标题背景 */
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
