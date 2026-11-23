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
  <LoginUserinfo
    :action-list="userActionList"
    :tenant="loginTenant"
    :userinfo="loginUserinfo"
  >
    <UserDisplay :value="userStore.userInfo.user_id" />
  </LoginUserinfo>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import LoginUserinfo from '@blueking/login-userinfo/vue3';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { useFrontendFeaturesStore } from '~/stores/frontend-features';
  import { useSpaceStore } from '~/stores/space';
  import { useUserStore } from '~/stores/user';

  import '@blueking/login-userinfo/vue3/vue3.css';

  // login-userinfo 面板操作项（对齐 @blueking/login-userinfo actionList 类型）
  interface IUserAction {
    href?: string;
    icon: string;
    target?: string;
    text: string;
    theme?: 'danger' | 'primary';
    handle?: () => void;
  }

  const { t } = useI18n();
  const router = useRouter();

  // 用户store
  const userStore = useUserStore();
  const featuresStore = useFrontendFeaturesStore();
  // 空间相关全局store
  const spaceStore = useSpaceStore();

  // 传给 login-userinfo 的用户信息（name 必填：头像首字母 + 兜底文案）
  const loginUserinfo = computed(() => ({
    name: userStore.currentUsername || '--',
  }));

  const loginTenant = computed(() => {
    const id = userStore.tenantId;
    // 多租户模式下展示当前租户。登录接口仅返回租户 ID，暂以 ID 作为展示名称。
    return featuresStore.enableMultiTenantMode && id ? { id, name: id } : undefined;
  });

  // 用户操作列表
  const userActionList = computed<IUserAction[]>(() => {
    const actions: IUserAction[] = [
      {
        text: t('退出登录'),
        icon: 'bkms-icon bkms-icon-daochu',
        theme: 'danger',
        handle: handleLogout,
      },
    ];
    if (userStore.hasPlatformRole) {
      actions.unshift({
        text: t('平台管理'),
        icon: 'bkms-icon bkms-icon-guanlikongjian-3',
        handle: handleGotoPlatform,
      });
    }
    return actions;
  });

  // 跳转平台管理
  function handleGotoPlatform() {
    spaceStore.updateCurrentSpace('');
    router.push({ name: 'platform' });
  }
  // 退出登录
  function handleLogout() {
    // 注销登录只注销当前登录态，清除bk_token，不做登录弹窗
    window.location.href = `${window.BK_LOGIN_URL}?is_from_logout=1&c_url=${encodeURIComponent(window.location.href)}`;
  }
</script>

<style lang="postcss" scoped>
  /* 深色导航下激活态高亮（组件默认 #3a84ff 偏暗） */
  .bk-login-userinfo.is-active {
    color: #c2cee5;
  }
</style>
