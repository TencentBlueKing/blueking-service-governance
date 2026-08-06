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
  <Dialog
    v-model:is-show="isShow"
    :width="500"
  >
    <template #header>
      <div class="flex flex-col items-center">
        <SvgIcon
          :height="42"
          icon="bkms-icon-cuo"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ t('镜像 Tag 删除失败') }}
        </span>
      </div>
    </template>
    <div class="bg-[#F5F7FA] mb-[10px] py-[12px] px-[14px] text-[14px] leading-[22px] text-[#4D4F56]">
      <i18n-t keypath="删除需要具备写权限的凭证。跳转到：{0} 配置凭证后重试。">
        <Button
          text
          theme="primary"
          @click="handleGoConfig"
          >{{ $t('构建管理') }} - {{ $t('构建配置') }}</Button
        >
      </i18n-t>
    </div>
    <template #footer>
      <div class="flex justify-center">
        <Button
          class="mr-[8px]"
          theme="primary"
          @click="handleGoConfig"
        >
          {{ t('去配置') }}
        </Button>
        <Button @click="handleCancel">
          {{ t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { onBeforeUnmount, ref } from 'vue';

  import { Button, Dialog } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { useAppDetail } from '~/stores/app-detail';

  const isShow = defineModel<boolean>('isShow');

  const { t } = useI18n();
  const router = useRouter();
  const route = useRoute();
  const appDetailStore = useAppDetail();

  // 跳转定时器句柄：点击「取消」或组件卸载时需清理，避免误跳转
  const navTimer = ref<null | ReturnType<typeof setTimeout>>(null);

  /** 取消：关闭弹窗并撤销尚未执行的跳转 */
  function handleCancel() {
    isShow.value = false;
    if (navTimer.value) {
      clearTimeout(navTimer.value);
      navTimer.value = null;
    }
  }

  /** 跳转去配置构建凭证 */
  function handleGoConfig() {
    isShow.value = false;
    // 通过 store 瞬态标记携带「去配置」来源（镜像仓库），构建页据此自动打开对应来源的侧栏。
    // 不再走 route.params，避免污染路由且保证只生效一次（构建页会在打开侧栏时消费并清空标记）。
    appDetailStore.setPendingBuilderSource('imageRegistry');
    // 延迟跳转，避免在关闭对话框时，对话框的关闭动画和路由跳转动画同时进行
    navTimer.value = setTimeout(() => {
      navTimer.value = null;
      router.push({
        name: 'detail',
        params: {
          ...route.params,
          menuName: 'build',
        },
      });
    }, 300);
  }

  onBeforeUnmount(() => {
    if (navTimer.value) {
      clearTimeout(navTimer.value);
      navTimer.value = null;
    }
  });
</script>
