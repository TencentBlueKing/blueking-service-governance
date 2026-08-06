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
  <div class="flex flex-col items-center justify-start">
    <div class="w-full max-w-[1112px] my-[16px]">
      <Input
        v-model.trim="searchTemplateValue"
        class="w-[360px]"
        clearable
        :placeholder="
          createPlaceholder({
            labels: ['模板名称'],
          })
        "
        type="search"
      />
    </div>
    <template v-if="templateList.length">
      <div class="flex items-center flex-wrap gap-[16px] max-w-[1112px]">
        <div
          v-for="item in templateList"
          :key="item.id"
          :class="[
            'flex flex-col items-center justify-center',
            'h-[180px] w-[360px] bg-[#fff] shadow cursor-pointer',
            'rounded-sm border-[1px]',
            activeCard?.id === item.id ? 'border-[#3A84FF]' : '',
            item.disabled ? 'opacity-50 cursor-not-allowed' : 'hover:border-[#3A84FF]',
          ]"
          @click="handleClick(item)"
        >
          <span
            v-if="item.icon"
            :class="[item.icon, 'size-[50px] leading-[50px]', item.disabled ? 'grayscale' : '']"
            :style="{ color: item.iconColor, fontSize: item.iconFontSize }"
          ></span>
          <span class="text-[#313238] leading-[24px] text-[16px] mt-[10px] font-bold">{{ item.title }}</span>
          <span class="mt-[12px] text-[#4D4F56] text-[12px]">{{ item.desc }}</span>
        </div>
      </div>
      <div class="mt-[32px]">
        <Button
          class="min-w-[88px]"
          theme="primary"
          @click="handleCreateApp"
          >{{ $t('下一步') }}</Button
        >
        <Button
          class="min-w-[88px] ml-[8px] bg-[#fff]"
          @click="cancel"
          >{{ $t('取消') }}</Button
        >
      </div>
    </template>
    <Exception
      v-else
      :description="$t('搜索为空')"
      type="search-empty"
    />
  </div>
</template>
<script lang="ts" setup>
  import { computed, onMounted, ref } from 'vue';

  import { Button, Exception, Input } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import { useAppStore } from '~/stores/apps';

  interface ICard {
    desc?: string;
    disabled?: boolean;
    icon?: string;
    iconColor?: string;
    iconFontSize?: string;
    id?: string;
    title?: string;
    type?: string;
  }

  const emits = defineEmits(['next', 'cancel', 'update-step', 'update-text']);

  const router = useRouter();
  const route = useRoute();
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const appStore = useAppStore();

  const searchTemplateValue = ref('');
  const activeCard = ref<ICard | null>({
    id: appStore.lastAppTemplateID || 'createTrpcTemplateApp',
  });
  const templateList = computed<ICard[]>(() => {
    const data = [
      {
        id: 'createTrpcTemplateApp',
        type: 'trpc',
        title: t('tRPC 应用'),
        desc: t('适用于使用 tRPC 协议的服务'),
        disabled: false,
        icon: 'bkms-icon bkms-icon-trpc',
        iconColor: '#1B44C8',
      },
      {
        id: 'createHelmTemplateApp',
        type: 'helm',
        title: t('Helm 应用'),
        desc: t('自定义应用编排，拥有灵活的可定制性'),
        disabled: false,
        icon: 'bkms-icon bkms-icon-HelmCharts',
        iconColor: '#0F1689',
        iconFontSize: '40px',
      },
      {
        id: 'createTAFTemplateApp',
        type: 'taf',
        title: 'TAF',
        desc: t('采用 TAF 框架实现的服务端应用'),
        disabled: false,
        icon: 'bkms-icon bkms-icon-taf',
        iconColor: '#1b44c8',
        iconFontSize: '40px',
      },
      {
        id: 'createAgonesTemplateApp',
        type: 'agones',
        title: t('Agones 应用'),
        desc: t('适用于多人在线游戏的专用服务器编排'),
        disabled: false,
        icon: 'bkms-icon bkms-icon-agones',
        iconColor: '#0F1689',
        iconFontSize: '40px',
      },
    ];
    return data.filter(
      item => item.title.toLocaleLowerCase().indexOf(searchTemplateValue.value?.toLocaleLowerCase()) > -1,
    );
  });

  function cancel() {
    emits('cancel');
    // 显式指定 fallback：resolveParent 推导到应用创建页(app/create)，但业务上应返回应用列表页(app)
    router.back({ name: 'app', params: { space: route.params.space } });
  }

  function handleClick(item: ICard) {
    if (item.disabled) return;
    activeCard.value = item;
    appStore.lastAppTemplateID = item.id || '';
    emits('update-text', item);
  }

  function handleCreateApp() {
    if (!activeCard.value?.id) return;
    router.push({
      name: activeCard.value?.id,
    });
    emits('next');
  }

  onMounted(() => {
    emits('update-step', 1);
  });
</script>
