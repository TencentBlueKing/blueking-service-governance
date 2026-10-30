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
  <div class="bg-[#F0F4FA] mt-[16px] p-[16px]">
    <Alert
      class="mb-[12px]"
      theme="warning"
    >
      <template #title>
        <div class="flex items-center text-[#4A5973]">
          <i18n-t keypath="请先将自定义的镜像推送至空间，{0}">
            <CustomImagePushGuide
              :image-name="name"
              :image-tag="tag"
              :password="repoInfo?.password ?? ''"
              :repository-address="repositoryAddress"
              :username="repoInfo?.username ?? ''"
            >
              <Button
                text
                theme="primary"
                >{{ $t('查看推送指引') }}</Button
              >
            </CustomImagePushGuide>
          </i18n-t>
        </div>
      </template>
    </Alert>
    <div class="flex gap-[6px] items-start">
      <Input
        class="flex-grow-2"
        disabled
        :model-value="repositoryAddress"
        :placeholder="$t('请先将自定义的镜像推送至空间')"
        :with-validate="false"
      />
      <Form.FormItem
        ref="nameFormItemRef"
        class="flex-grow-1 min-w-[180px] !mb-0"
        error-display-type="tooltips"
        :property="nameProperty"
        required
        :rules="nameRules"
      >
        <Input
          class="w-full"
          clearable
          :model-value="displayName"
          :placeholder="$t('请输入镜像名称')"
          @change="handleNameChange"
        />
      </Form.FormItem>
      <div class="flex-grow-1 min-w-[180px] flex items-center gap-[6px]">
        <Form.FormItem
          ref="tagFormItemRef"
          class="flex-1 min-w-[120px] !mb-0"
          error-display-type="tooltips"
          :property="tagProperty"
          required
          :rules="tagRules"
        >
          <Select
            v-model="tag"
            class="w-full"
            display-key="tag"
            filterable
            id-key="tag"
            :list="tags"
            :loading="tagsLoading"
            :placeholder="$t('请选择镜像 Tag')"
            :remote-method="searchTag"
            :with-validate="false"
          />
        </Form.FormItem>
        <Button
          v-bk-tooltips="$t('刷新 Tag 列表')"
          class="!min-w-[32px] !px-0"
          :disabled="!name || refreshing"
          text
          theme="primary"
          @click="handleRefresh"
        >
          <i
            :class="[
              'bkms-icon bkms-icon-refresh text-[16px]',
              refreshing ? 'animate-spin [animation-duration:1.5s]' : '',
            ]"
          ></i>
        </Button>
      </div>
    </div>
    <!-- 快捷填入镜像名称：为空（未推送过任何镜像）时整个区域不展示，避免空数据加载期间「先闪出再消失」 -->
    <div
      v-if="displayNames.length"
      class="mt-[8px] flex items-start text-[12px] leading-[20px]"
    >
      <span class="text-[#8F9FBD] shrink-0 mr-[4px]">{{ $t('快捷填入镜像名称') }}：</span>
      <div class="flex flex-wrap items-center gap-x-[12px] gap-y-[4px] min-w-[24px] min-h-[16px]">
        <Button
          v-for="item in displayNames"
          :key="item.name"
          class="!text-[12px] leading-[20px] inline-block"
          text
          theme="primary"
          @click="handleQuickFill(item.name)"
        >
          {{ item.name }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

  import { Alert, Button, Form, Input, Message, Select } from 'bkui-vue';
  import { debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { ImagesService } from '~/api/modules/v1/images';
  import CustomImagePushGuide from '~/pages/application/components/custom-image-push-guide.vue';
  import { useSpaceStore } from '~/stores/space';

  import type { CustomRuntimeImageOutputObj, CustomRuntimeImageTagOutputObj } from '~/@types/v1/images';

  interface IProps {
    /** 仓库信息（推送指引展示用） */
    repoInfo?: null | RepoInfo;
    /** 镜像类型：构建镜像 / 运行镜像 */
    type: 'builder' | 'runner';
    /** 表单 property 前缀（拼完整路径），如 "platformBuildConfig.builderImage"；缺省时内部 FormItem 不参与父级 form.validate() */
    validatePrefix?: string;
  }

  interface RepoInfo {
    password?: string;
    repositoryAddress?: string;
    username?: string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    repoInfo: null,
    validatePrefix: '',
  });

  const { t } = useI18n();
  const spaceStore = useSpaceStore();

  const name = defineModel<string>('name', { required: true });
  const tag = defineModel<string>('tag', { required: true });

  // ========== 数据状态（组件内部自治） ==========
  const names = ref<CustomRuntimeImageOutputObj[]>([]);
  const tags = ref<CustomRuntimeImageTagOutputObj[]>([]);
  const tagsLoading = ref(false);
  /** 镜像名表单项实例：切换镜像名时清除旧校验错误 */
  const nameFormItemRef = ref<{ clearValidate: () => void }>();
  /** Tag 表单项实例：切换镜像名清空 tag 时清除旧校验错误 */
  const tagFormItemRef = ref<{ clearValidate: () => void }>();
  const refreshing = ref(false);

  // ========== Computed ==========
  /** 仓库地址：空间绑定的镜像仓库地址，只读（disabled Input 展示），用于裁剪展示与校验镜像名前缀 */
  const repositoryAddress = computed(() => props.repoInfo?.repositoryAddress ?? '');

  /** 裁剪仓库地址前缀，得到纯镜像名（列表展示 / 手输回显均不带前缀）。
   *  目前镜像仓库不能修改，空间绑定的仓库地址固定不变，因此对任意镜像名裁剪该前缀都是安全的 */
  const trimRepoPrefix = (fullName: string) => {
    const prefix = repositoryAddress.value ? `${repositoryAddress.value}/` : '';
    return prefix && fullName.startsWith(prefix) ? fullName.slice(prefix.length) : fullName;
  };

  /** 快捷填入候选：提前将 name 裁剪为纯镜像名，点击后填入 Input 的也是不带前缀的镜像名 */
  const displayNames = computed(() =>
    names.value.map(item => ({
      ...item,
      name: trimRepoPrefix(item.name ?? ''),
    })),
  );

  /** 编辑回显：当前完整名裁剪前缀后作为 Input 的展示值 */
  const displayName = computed(() => trimRepoPrefix(name.value));

  /** 传入镜像名是否已落库（命中候选）。候选 name 可能为完整名或纯名，统一裁剪前缀后比较，保证命中判断鲁棒 */
  function isKnownImageName(imageName: string): boolean {
    if (!imageName) return false;
    const trimmed = trimRepoPrefix(imageName);
    return names.value.some(item => trimRepoPrefix(item.name ?? '') === trimmed);
  }

  // ========== 表单校验（单点治理：name / Tag 各自 FormItem） ==========
  /** 镜像名/Tag 的 FormItem property（虚拟路径，仅用于被父级 form.validate() 收集；值由自定义 validator 决定） */
  const nameProperty = computed(() => (props.validatePrefix ? `${props.validatePrefix}.customImageName` : ''));
  const tagProperty = computed(() => (props.validatePrefix ? `${props.validatePrefix}.customImageTag` : ''));

  /** 镜像名规则：必填 + 格式（完整名以仓库地址为前缀、不含 tag）。用户输入/展示为裁剪前缀后的镜像名，内部 name 保存完整名 */
  const nameRules = computed(() => [
    {
      required: true,
      message: t(props.type === 'builder' ? '构建镜像不能为空' : '运行镜像不能为空'),
      validator: () => !!name.value,
      trigger: 'change',
    },
    {
      required: true,
      message: t('请输入镜像名称，且不能包含 tag'),
      validator: () => {
        // 空值交给必填规则；仓库地址未就绪时跳过格式校验，避免对用户输入误报
        if (!name.value || !repositoryAddress.value) return true;
        // name.value 为补前缀后的完整名，去掉仓库前缀后即用户输入部分；
        // 前缀由 normalizeName 无条件补齐，此处只需校验用户输入部分不含 tag
        return !name.value.slice(repositoryAddress.value.length + 1).includes(':');
      },
      trigger: 'change',
    },
  ]);

  /** Tag 规则：必填 */
  const tagRules = computed(() => [
    {
      required: true,
      message: t('请选择镜像 Tag'),
      validator: () => !!tag.value,
      trigger: 'change',
    },
  ]);

  // ========== 联动 ==========
  /** 已加载 tag 的镜像名，避免重复请求同一镜像 */
  const lastLoadedName = ref('');

  /** 手输值补全仓库前缀为完整镜像名。
   *  目前镜像仓库不能修改，且候选 list 已提前裁剪前缀，因此选中/手输拿到的都是不带前缀的镜像名，统一无条件补前缀；
   *  空值保持为空，允许清空 */
  const normalizeName = (raw: string) => {
    if (!raw) return '';
    const prefix = repositoryAddress.value ? `${repositoryAddress.value}/` : '';
    return `${prefix}${raw}`;
  };

  /** 切换镜像名后更新本地值并同步清除 name/tag 旧校验错误：
   *  清空 tag 后若旧错误不立即清除，会渲染出错误帧直到 loadTags 网络返回才清；
   *  同步 clearValidate 在 Vue 批处理内与清空同一 tick 完成，错误态不会被渲染 */
  function applyNameChange(fullName: string) {
    name.value = fullName;
    tag.value = '';
    tags.value = [];
    nameFormItemRef.value?.clearValidate();
    tagFormItemRef.value?.clearValidate();
  }

  /** Input @change（用户手输）。根据是否命中已落库候选判断：
   *  命中（手输恰好等于候选名）→ 先刷新后端 Tag 快照再拉取；
   *  未命中（手动输入新名字）→ 直接实时拉取，不做 refresh。
   *  同步预置 lastLoadedName，使异步触发的 watch(name) 被防重拦截，避免重复请求 */
  function handleNameChange(value: string) {
    const fullName = normalizeName(value);
    applyNameChange(fullName);
    if (isKnownImageName(fullName)) {
      lastLoadedName.value = fullName;
      refreshTagsForName(fullName);
    } else {
      loadTagsForName(fullName);
    }
  }

  /** 快捷填入：候选项必已落库，直接刷新后端 Tag 快照 + 拉取，无需命中判断 */
  function handleQuickFill(name: string) {
    const fullName = normalizeName(name);
    applyNameChange(fullName);
    lastLoadedName.value = fullName;
    refreshTagsForName(fullName);
  }
  // ========== API 请求（组件内部自治） ==========
  /** 刷新按钮统一入口：
   *  已落库镜像（候选列表中存在）→ 先刷新后端 Tag 快照，成功后再拉取最新列表；
   *  手动输入（未落库）→ 无快照可刷新，直接重拉 Tags（后端实时拉取） */
  async function handleRefresh() {
    const workspaceID = spaceStore.currentSpace;
    const targetName = name.value;
    if (!workspaceID || !targetName) return;
    refreshing.value = true;
    try {
      const succeeded = await refreshTagsForName(targetName);
      if (succeeded) {
        Message.success(t('镜像 Tag 刷新成功'));
      }
    } catch (err) {
      console.error('Failed to refresh custom image tags:', err);
      Message.error(t('刷新镜像 Tag 失败'));
    } finally {
      refreshing.value = false;
    }
  }

  /** 完整镜像名合法性（组件内联）：非空、以仓库地址为前缀，且不含 tag/digest。
   *  仓库地址可含端口冒号（如 host:8080/proj），故只看「前缀之后」的镜像路径是否含冒号。
   *  用于请求前守卫：接口要求含前缀的完整名 */
  function isCustomImageNameValid(imageName: string): boolean {
    if (!imageName || !repositoryAddress.value || !imageName.startsWith(`${repositoryAddress.value}/`)) return false;
    return !imageName.slice(repositoryAddress.value.length + 1).includes(':');
  }

  /** 获取自定义镜像名称候选列表（一次拉全，作为快捷填入项） */
  async function loadNames() {
    const workspaceID = spaceStore.currentSpace;
    if (!workspaceID) return;
    try {
      const res = await ImagesService.listCustomBuildImages({
        workspaceID,
        type: props.type,
      });
      names.value = res.results ?? [];
    } catch (err) {
      console.error(`Failed to fetch ${props.type} custom images:`, err);
    }
  }

  /** 获取自定义镜像 Tag 列表 */
  async function loadTags(imageName?: string, keyword?: string) {
    const workspaceID = spaceStore.currentSpace;
    const targetName = imageName ?? name.value;
    if (!workspaceID || !targetName) return;
    // 切换镜像名后 tag 已被清空，清除旧校验错误，避免新列表未就绪时残留报错
    tagFormItemRef.value?.clearValidate();
    tagsLoading.value = true;
    try {
      const res = await ImagesService.listCustomBuildImageTags({
        workspaceID,
        name: targetName,
        ...(keyword ? { keyword } : {}),
        page: 1,
        pageSize: 100,
      });
      tags.value = res.results ?? [];
      // 仅当未选中 tag 且列表非空时自动选中第一个
      if (!tag.value && tags.value.length > 0 && tags.value[0].tag) {
        tag.value = tags.value[0].tag;
      }
    } catch (err) {
      console.error('Failed to fetch custom image tags:', err);
      tags.value = [];
    } finally {
      tagsLoading.value = false;
    }
  }

  /** 校验通过且非重复时才拉取 tag */
  function loadTagsForName(imageName: string) {
    if (!imageName || imageName === lastLoadedName.value || !isCustomImageNameValid(imageName)) {
      return;
    }
    lastLoadedName.value = imageName;
    loadTags(imageName);
  }

  /** 拉取 tag 统一入口（快捷填入 / 刷新按钮复用）：
   *  已落库镜像（候选列表中存在）→ 先刷新后端 Tag 快照，成功后再拉取最新列表；
   *  手动输入（未落库）→ 无快照可刷新，直接重拉 Tags（后端实时拉取）。
   *  返回是否成功完成拉取（刷新失败 / 已有刷新任务时为 false） */
  async function refreshTagsForName(imageName: string): Promise<boolean> {
    const workspaceID = spaceStore.currentSpace;
    if (!workspaceID || !imageName) return false;
    if (isKnownImageName(imageName)) {
      const res = await ImagesService.refreshCustomBuildImageTags({
        workspaceID,
        name: imageName,
      });
      if (res?.status === 'failed') {
        Message.error(res.message || t('刷新镜像 Tag 失败'));
        return false;
      }
      if (res?.status === 'refreshing') {
        Message.warning(t('已有刷新任务进行中，请稍后重试'));
        return false;
      }
    }
    await loadTags(imageName);
    lastLoadedName.value = imageName;
    return true;
  }

  // ========== 远程搜索防抖 ==========
  const searchTag = debounce((keyword: string) => {
    if (name.value) {
      loadTags(name.value, keyword);
    }
  }, 300);

  // ========== 初始化 ==========
  // 编辑回显 name 与仓库地址就绪时拉取 tag（immediate 覆盖挂载前已回填的 name）；
  // 交互场景走 @change / 快捷填入 → loadTagsForName 的 lastLoadedName 防重，不会重复请求
  watch(
    [name, () => props.repoInfo?.repositoryAddress],
    ([nameValue]) => {
      if (nameValue) {
        loadTagsForName(nameValue);
      }
    },
    { immediate: true },
  );

  onBeforeUnmount(() => {
    searchTag.cancel();
  });

  // 挂载时拉取候选列表（独立于仓库地址，只要空间 ID 存在即请求）
  onMounted(() => {
    loadNames();
  });

  defineExpose({
    loadNames,
    loadTags,
    handleRefresh,
  });
</script>
