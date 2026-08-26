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
  <div
    v-bkloading="{ isLoading }"
    class="flex h-full min-h-[480px] w-full items-center justify-center bg-white"
  >
    <div
      v-if="isForbidden"
      class="-mt-12 text-center"
    >
      <img
        alt="403"
        class="h-[152px] w-[308px]"
        src="@/assets/permissions.png"
      />
      <h2 class="mb-[24px] mt-[24px] text-[20px] font-normal leading-[28px] text-[#979ba5]">
        {{ t('您没有访问当前空间的权限') }}
      </h2>
      <div class="flex justify-center gap-[12px]">
        <div
          v-for="role in roles"
          :key="role.code"
          v-bk-tooltips="{
            content: t('暂无权限申请地址'),
            disabled: Boolean(applyURLs[role.code]),
          }"
        >
          <Button
            :disabled="!applyURLs[role.code]"
            theme="primary"
            @click="handleApply(role.code)"
          >
            {{ t(role.label) }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';

  import { Button, Message } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { WorkspaceService } from '~/api/modules/v1/workspace';
  import { useSpaceStore } from '~/stores/space';

  interface PermissionDetail {
    code?: string;
    extras?: Partial<Record<RoleCode, string>>;
  }

  type RoleCode = 'admin' | 'developer';

  interface WorkspaceLookupError {
    status?: number;
    error?: {
      details?: PermissionDetail[];
    };
  }

  const roles: Array<{ code: RoleCode; label: string }> = [
    { code: 'admin', label: '申请成为管理员' },
    { code: 'developer', label: '申请成为开发者' },
  ];

  const route = useRoute();
  const router = useRouter();
  const spaceStore = useSpaceStore();
  const { t } = useI18n();

  const isForbidden = ref(false);
  const isLoading = ref(true);
  const roleGroupIDs = ref<Partial<Record<RoleCode, string>>>({});
  // 统一去掉末尾斜杠，避免拼接申请路径时出现双斜杠。
  const iamURL = import.meta.env.BK_IAM_URL.trim().replace(/\/$/, '');

  const applyURLs = computed<Record<RoleCode, string>>(() => ({
    admin: buildApplyURL(roleGroupIDs.value.admin),
    developer: buildApplyURL(roleGroupIDs.value.developer),
  }));

  function buildApplyURL(groupID?: string) {
    if (!iamURL || !groupID) return '';
    return `${iamURL}/apply-join-user-group?id=${encodeURIComponent(groupID)}`;
  }

  function getQueryValue(value: unknown) {
    return typeof value === 'string' ? value : '';
  }

  function getRoleGroupIDs(details: PermissionDetail[] = []) {
    // 服务端将各角色对应的用户组 ID 放在 IAM_NO_PERMISSION 明细的 extras 中。
    const iamDetail = details.find(detail => detail.code === 'IAM_NO_PERMISSION');
    return iamDetail?.extras ?? {};
  }

  function handleApply(roleCode: RoleCode) {
    const url = applyURLs.value[roleCode];

    if (!url) return;
    window.open(url, '_blank', 'noopener,noreferrer');
  }

  function normalizeError(error: unknown): WorkspaceLookupError {
    return error && typeof error === 'object' ? (error as WorkspaceLookupError) : {};
  }

  async function verifyWorkspace() {
    const workspaceID = getQueryValue(route.query.workspaceID);
    if (!workspaceID) {
      await router.replace({ name: '404' });
      return;
    }

    try {
      // 空间列表不包含目标空间并不一定代表资源不存在，详情接口才是最终判断依据。
      const workspace = await WorkspaceService.getWorkspace(
        { workspaceID },
        {
          interceptorErr: false,
          needStatus: true,
        },
      );

      if (workspace.state !== spaceStore.spaceState.Ready) {
        await router.replace({ name: '404' });
        return;
      }

      // 详情可访问说明列表缓存已过期，补回 Store 后恢复用户原本访问的页面。
      spaceStore.upsertWorkspace(workspace);
      const redirect = getQueryValue(route.query.redirect);
      // 只接受站内地址，并阻止再次进入 403 检查页造成循环跳转。
      await router.replace(redirect.startsWith('/') && !redirect.startsWith('/403') ? redirect : { name: 'home' });
    } catch (error: unknown) {
      const lookupError = normalizeError(error);
      if (lookupError.status === 403) {
        // 无权限时展示申请入口，按钮是否可用取决于 IAM 地址和角色用户组 ID。
        roleGroupIDs.value = getRoleGroupIDs(lookupError.error?.details);
        isForbidden.value = true;
        return;
      }
      // 空间不存在时继续沿用现有 404 页面。
      if (lookupError.status === 404) {
        await router.replace({ name: '404' });
        return;
      }
      Message({
        theme: 'error',
        message: t('请求异常'),
      });
      await router.replace({ name: 'home' });
    } finally {
      isLoading.value = false;
    }
  }

  onMounted(verifyWorkspace);
</script>
