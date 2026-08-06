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
    <Select
      ref="selectRef"
      v-test="'space-selector'"
      filterable
      :model-value="spaceStore.currentSpace"
      :popover-min-width="300"
      :search-placeholder="$t('搜索空间ID、名称')"
      :search-with-pinyin="false"
      @change="handleChangeSpace"
      @toggle="isSpacePopoverShow = !isSpacePopoverShow"
    >
      <template #trigger>
        <span class="flex items-center text-[#eaebf0] text-[14px] cursor-pointer">
          <Loading
            class="mr-[5px]"
            color="#182132"
            :loading="spaceStore.isLoading"
            :opacity="1"
            size="small"
            theme="default"
          >
            <span class="text-[#DCDEE5]">{{ spaceName }}</span>
          </Loading>
          <AngleDownFill :class="['ml-[5px] text-[12px]', isSpacePopoverShow ? '' : 'rotate-180']" />
        </span>
      </template>
      <Select.Option
        v-for="item in enabledSpaces"
        :id="item.id"
        :key="item.id"
        :name="`${item.displayName}(${item.id})`"
      />
      <template #extension>
        <div class="flex w-full items-center">
          <div
            class="flex-1 text-center flex cursor-pointer justify-center items-center"
            @click="handleCreateTeamSpace"
          >
            <Plus
              height="20px"
              width="20px"
            />
            <Button
              class="!text-[14px]"
              text
            >
              {{ $t('新建') }}
            </Button>
          </div>
          <Divider direction="vertical" />
          <div
            class="flex-1 text-center flex cursor-pointer justify-center align-items-center"
            @click="handleGotoHome"
          >
            <i class="bkms-icon bkms-icon-guanlikongjian-3 mr-[5px] mt-[1px]"></i>
            <Button
              class="!text-[14px]"
              text
              >{{ $t('空间管理') }}</Button
            >
          </div>
        </div>
      </template>
    </Select>
    <TeamSpace
      v-model:is-show="isShow"
      type="create"
    />
  </div>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Button, Divider, Loading, Select } from 'bkui-vue';
  import { AngleDownFill, Plus } from 'bkui-vue/lib/icon';
  import { useRoute, useRouter } from 'vue-router';
  import { useSpaceStore } from '~/stores/space';

  import TeamSpace from './team-space.vue';

  const spaceStore = useSpaceStore();
  const router = useRouter();
  const route = useRoute();

  const isSpacePopoverShow = ref(false);
  const isShow = ref(false);

  // ref
  const selectRef = ref();

  // 空间名称
  const spaceName = computed(() => {
    if (!spaceStore.currentSpace || !spaceStore.list?.length) return '';
    return spaceStore.list.find(item => item.id === spaceStore.currentSpace)?.displayName || '';
  });

  // 过滤启用中的空间
  const enabledSpaces = computed(() => spaceStore.list.filter(item => item.state === spaceStore.spaceState.Ready));

  // 切换空间
  function handleChangeSpace(name: string) {
    // 更新当前空间缓存
    spaceStore.updateCurrentSpace(name);
    router.replace({
      name: route.name,
      params: {
        space: name,
      },
    });
  }

  // 创建空间
  function handleCreateTeamSpace() {
    // 打开侧栏前，先隐藏下拉框
    selectRef.value?.hidePopover?.();
    isShow.value = true;
  }

  // 跳转首页
  function handleGotoHome() {
    spaceStore.updateCurrentSpace('');
    router.push({
      name: 'home',
    });
  }
</script>

<style lang="postcss" scoped></style>
