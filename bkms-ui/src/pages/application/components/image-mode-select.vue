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
    <Button.ButtonGroup class="flex items-center mb-[8px]">
      <Button
        class="flex-1"
        :selected="imageMode === 'platform'"
        @click="handleImageModeChange('platform')"
        >{{ $t('平台推荐镜像') }}</Button
      >
      <Button
        class="flex-1"
        :selected="imageMode === 'custom'"
        @click="handleImageModeChange('custom')"
        >{{ $t('自定义镜像') }}</Button
      >
    </Button.ButtonGroup>
    <!-- 平台推荐镜像 -->
    <template v-if="imageMode === 'platform'">
      <div class="bg-[#F0F4FA] mt-[16px] p-[16px]">
        <div class="flex gap-[6px] items-start">
          <Form.FormItem
            class="flex-grow-2 !mb-0"
            error-display-type="tooltips"
            :property="imageProperty"
            required
            :rules="platformRules.image"
          >
            <Select
              v-model="platformSelection.imageId"
              class="w-full"
              display-key="name"
              filterable
              id-key="id"
              :input-search="false"
              :list="images"
              :placeholder="$t('请选择镜像')"
              @change="handleImageChange"
            />
          </Form.FormItem>
          <Form.FormItem
            class="!mb-0 w-[180px]"
            error-display-type="tooltips"
            :property="tagProperty"
            required
            :rules="platformRules.tag"
          >
            <Select
              v-model="platformSelection.tag"
              class="w-full"
              :disabled="!platformSelection.imageId"
              display-key="tag"
              filterable
              id-key="tag"
              :list="tags"
              :loading="tagLoading"
              :placeholder="$t('请选择镜像 Tag')"
              :remote-method="searchTag"
            />
          </Form.FormItem>
        </div>
        <div
          v-if="staleWarning"
          class="text-[12px] text-[#fe9c00] mt-[8px]"
        >
          {{ $t('该镜像已不在当前平台镜像列表中，请重新选择') }}
        </div>
        <div
          v-if="staleTagWarning"
          class="text-[12px] text-[#fe9c00] mt-[8px]"
        >
          {{ $t('所选镜像 Tag 已不在当前镜像的 Tag 列表中，请重新选择') }}
        </div>
        <div class="text-[12px] text-[#8F9FBD] mt-[4px] leading-[20px]">
          {{ descText }}
        </div>
      </div>
    </template>
    <!-- 自定义镜像 -->
    <template v-else>
      <CustomImageSelect
        v-model:name="customSelection.name"
        v-model:tag="customSelection.tag"
        :repo-info="customRepoInfo"
        :type="props.type"
        :validate-prefix="imageProperty"
      />
    </template>
    <!-- 版本一致性告警（仅运行镜像） -->
    <Alert
      v-if="versionMismatchWarning"
      class="mt-[4px]"
      theme="warning"
    >
      <template #title>
        <div class="leading-[22px]">
          {{
            $t(
              '构建镜像的版本（{0}）与运行镜像（{1}）不一致，可能导致 glibc/musl 等基础库差异，引发运行时兼容问题。建议保持两者一致，如已确认可忽略。',
              [versionMismatchWarning.builderVersion, versionMismatchWarning.runnerVersion],
            )
          }}
        </div>
      </template>
    </Alert>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';

  import { Alert, Button, Form, Select } from 'bkui-vue';
  import { debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { ImagesService } from '~/api/modules/v1/images';
  import CustomImageSelect from '~/pages/application/components/custom-image-select.vue';

  import type { RuntimeImageOutputObj, RuntimeImageTagOutputObj } from '~/@types/v1/images';

  type CustomImageMode = 'custom' | 'platform';

  interface CustomImageRepoInfo {
    /** 仓库凭证/密码 */
    password: string;
    /** 仓库地址（imageRegistry.registry），作为自定义镜像名前缀，如 mirrors.example.com/bkms/proj */
    repositoryAddress: string;
    /** 用户名 */
    username: string;
  }

  interface IProps {
    /** 当前语言，用于 API 过滤镜像列表 */
    language?: string;
    /** 镜像字段值：platform 格式 "name:tag"，custom 格式 "完整仓库名:tag"（完整仓库名含前缀） */
    modelValue?: string;
    /** 仅 runner 传入：构建镜像字段值，用于版本一致性告警 */
    peerImageValue?: string;
    /** 空间镜像仓库信息（父级异步拉取就绪后传入；未绑定仓库时为 null） */
    repoInfo?: CustomImageRepoInfo | null;
    /** 父级仓库信息是否仍在加载中；结束后组件才执行初始化（仓库未绑定也会结束为 false） */
    repoInfoLoading?: boolean;
    /** 镜像类型：构建镜像（builder）/ 运行镜像（runner） */
    type: 'builder' | 'runner';
    /** 表单 property 前缀（拼完整路径），如 "repoBuildConfig" 或 "buildConfig.repoBuildConfig" */
    validatePrefix?: string;
  }

  const props = withDefaults(defineProps<IProps>(), {
    validatePrefix: '',
    language: '',
    peerImageValue: '',
    repoInfoLoading: false,
  });
  const emit = defineEmits<{
    /** 初始化完成（含回填 emit flush 后触发），供父级清理 useLeaveConfirm 的 dirty 误判 */
    'init-complete': [];
    'update:modelValue': [value: string];
  }>();
  const { t } = useI18n();

  // ========== 实例状态 ==========
  const images = ref<RuntimeImageOutputObj[]>([]);
  const tags = ref<RuntimeImageTagOutputObj[]>([]);
  const tagLoading = ref(false);
  /** 镜像已不在平台列表中（整体失效） */
  const staleWarning = ref(false);
  /** 镜像仍在但存值 Tag 已不在最新列表中（被自动替换） */
  const staleTagWarning = ref(false);
  const imageMode = ref<CustomImageMode>('platform');
  // ========== 选中值状态（按模式分组，避免多个 selected* 顶层 ref 散落） ==========
  const platformSelection = ref({ imageId: '', tag: '' });
  const customSelection = ref({ name: '', tag: '' });
  // ========== 自定义镜像状态（数据请求已下沉子组件，仅保留表单值） ==========
  const customRepoInfo = ref<CustomImageRepoInfo | null>(null);

  // ========== Computed ==========
  const descText = computed(() =>
    props.type === 'builder'
      ? t('用于编译源码的镜像（含 Go 工具链），对应 builder 阶段基础镜像。')
      : t('运行服务的基础镜像（精简、无编译器），对应 runner 阶段基础镜像。'),
  );
  const prefix = computed(() => (props.validatePrefix ? `${props.validatePrefix}.` : ''));
  const imageProperty = computed(
    () => `${prefix.value}platformBuildConfig.${props.type === 'builder' ? 'builderImage' : 'runnerImage'}`,
  );
  /** Tag FormItem property（虚拟路径，仅用于被父级 form.validate() 收集；值由自定义 validator 决定） */
  const tagProperty = computed(() => `${imageProperty.value}Tag`);

  /** 平台模式校验规则：镜像 / Tag 各自 FormItem 收集 */
  const platformRules = computed(() => ({
    image: [
      {
        required: true,
        message: t(props.type === 'builder' ? '构建镜像不能为空' : '运行镜像不能为空'),
        validator: () => !!platformSelection.value.imageId,
        trigger: 'change',
      },
    ],
    tag: [
      {
        required: true,
        message: t('请选择镜像 Tag'),
        validator: () => !!platformSelection.value.tag,
        trigger: 'change',
      },
    ],
  }));
  // 版本一致性告警（仅 runner）：peerImageValue 为构建镜像值，解析两侧 tag 比较
  const versionMismatchWarning = computed(() => {
    if (props.type !== 'runner') return null;
    const peerTag = parseTagFromValue(props.peerImageValue);
    if (!peerTag || !platformSelection.value.tag) return null;
    const builderHasAlpine = /alpine/i.test(peerTag);
    const runnerImage = images.value.find(i => i.id === platformSelection.value.imageId);
    const runnerHasAlpine = /alpine/i.test(runnerImage?.name ?? '');
    if (!builderHasAlpine || !runnerHasAlpine) return null;
    const builderVersion = extractVersion(peerTag);
    const runnerVersion = extractVersion(platformSelection.value.tag);
    if (builderVersion === runnerVersion) return null;
    return { builderVersion, runnerVersion };
  });

  // ========== Utility ==========
  /**
   * 根据存值 + 仓库信息应用镜像模式并回填自定义镜像（initState 与 watch(repoInfo) 共用）
   * @returns true 表示最终为平台模式（是否拉取列表由调用方结合场景决定）
   */
  function applyStoredImageValue(storedValue: string, repoInfo: CustomImageRepoInfo | null): boolean {
    const repositoryAddress = storedValue ? (repoInfo?.repositoryAddress ?? '') : '';
    if (repositoryAddress && isCustomImageValue(storedValue, repositoryAddress)) {
      // 已存值为自定义镜像地址 → 自定义模式回填（候选/tag 由子组件 watch 自动拉取）
      imageMode.value = 'custom';
      const { name, tag } = parseCustomImageValue(storedValue, repositoryAddress);
      customSelection.value.name = name;
      customSelection.value.tag = tag;
      return false;
    }
    imageMode.value = 'platform';
    return true;
  }

  function extractVersion(tag: string): string {
    if (!tag) return '';
    const alpineMatch = tag.match(/alpine(\d+(?:\.\d+)*)/i);
    if (alpineMatch) return alpineMatch[1];
    const matches = tag.match(/(\d+(?:\.\d+)*)/g);
    if (matches && matches.length > 0) return matches[matches.length - 1];
    return tag;
  }

  // ========== API（平台推荐镜像）==========
  /** 拉取平台推荐镜像 Tag 列表；autoSelect 为 false 时（回填场景）不自动选中，避免中间空态触发 emit('') */
  async function fetchImageTags(imageId: string, keyword?: string, autoSelect = true) {
    tagLoading.value = true;
    try {
      const res = await ImagesService.listPlatformBuildImageTags({
        imageID: imageId,
        ...(keyword ? { keyword } : {}),
        page: 1,
        pageSize: 100,
      });
      const data = res as { results?: RuntimeImageTagOutputObj[] };
      tags.value = data.results ?? [];
      if (autoSelect && tags.value.length > 0) {
        if (!platformSelection.value.tag && tags.value[0].tag) {
          platformSelection.value.tag = tags.value[0].tag;
        }
      }
    } catch (err) {
      console.error(`Failed to fetch ${props.type} image tags:`, err);
    } finally {
      tagLoading.value = false;
    }
  }

  /**
   * 拉取平台推荐镜像列表。restoreSelection 为 true 时按已存值回填选中（编辑回显）；
   * 为 false 时仅刷新列表，保留当前选中（模式切换触发，避免自定义存值被误判清空 / staleWarning 误报）
   */
  async function fetchPlatformBuildImages(restoreSelection = true) {
    try {
      const language = props.language;
      const res = await ImagesService.listPlatformBuildImages({ type: props.type, ...(language ? { language } : {}) });
      const list = (res as { results?: RuntimeImageOutputObj[] }).results ?? [];
      images.value = list;

      // 回填已存值：先拉 tags，再同 tick 一次性设置 imageId+tag，避免中间空态触发 watch emit('') 污染 modelValue
      const existingValue = props.modelValue ?? '';
      if (restoreSelection && existingValue) {
        const parsed = parseImageValue(existingValue, list);
        if (parsed.imageId) {
          staleWarning.value = false;
          staleTagWarning.value = false;
          await fetchImageTags(parsed.imageId, undefined, false);
          platformSelection.value.imageId = parsed.imageId;
          if (parsed.tag && tags.value.some(t => t.tag === parsed.tag)) {
            platformSelection.value.tag = parsed.tag;
          } else if (parsed.tag) {
            // 存值 Tag 已不在最新列表：自动替换为首个可用 Tag，并明确提示避免镜像版本被无声改变
            platformSelection.value.tag = tags.value[0]?.tag ?? '';
            staleTagWarning.value = true;
          } else {
            platformSelection.value.tag = tags.value[0]?.tag ?? '';
            staleWarning.value = true;
          }
        } else {
          staleWarning.value = true;
          staleTagWarning.value = false;
          platformSelection.value.imageId = '';
          platformSelection.value.tag = '';
        }
      }
      // 不自动选中任何镜像：创建场景由用户自行选择，编辑场景避免误覆盖存值（含自定义镜像）
    } catch (err) {
      console.error(`Failed to fetch ${props.type} images:`, err);
    }
  }

  // ========== Event handlers ==========
  function handleImageChange(imageId: string) {
    platformSelection.value.imageId = imageId;
    platformSelection.value.tag = '';
    tags.value = [];
    // 重新选择镜像后清除失效提示
    staleWarning.value = false;
    staleTagWarning.value = false;
    if (imageId) {
      fetchImageTags(imageId);
    }
  }

  /** 切换镜像模式：平台推荐镜像与自定义镜像是两套独立数据，各自保留已选值，互不预填。
   * 两模式 FormItem 互斥渲染（v-if/v-else），切换时另一模式 FormItem 卸载，错误态随之清除
   * 切回平台推荐时刷新镜像列表（对齐自定义镜像每次进入拉取的行为），仅刷新不重放回填 */
  function handleImageModeChange(mode: CustomImageMode) {
    if (mode === imageMode.value) return;
    imageMode.value = mode;
    if (mode === 'platform') {
      fetchPlatformBuildImages(false);
    }
  }

  // ========== 初始化 ==========
  // 由 watch(repoInfoLoading) 在父级仓库信息加载结束后触发，挂载时数据必已就绪；
  // initialized 防重：repoInfoLoading 意外翻转多次时仅执行一次
  let initialized = false;
  async function initState() {
    if (initialized) return;
    initialized = true;
    const storedValue = props.modelValue ?? '';
    // 仓库信息由父级就绪后传入，未绑定仓库时为 null
    const repoInfo = props.repoInfo ?? null;
    customRepoInfo.value = repoInfo;
    // 平台列表不依赖仓库信息，应照常拉取：
    // - 创建场景 → 平台模式，拉列表
    // - 编辑 + 仓库未绑定（repoInfo 恒 null）→ 无法判断自定义，先按平台拉列表保证可用
    // - 编辑 + 存值自定义镜像 → 识别自定义模式，不拉平台列表
    if (applyStoredImageValue(storedValue, repoInfo)) {
      await fetchPlatformBuildImages();
    }
    // 回填 emit 经 writeBackModelValue watch 在下一轮 flush 完成，等待后再通知父级清理 dirty，
    // 避免初始化回显被 useLeaveConfirm 误判为用户改动
    await nextTick();
    emit('init-complete');
  }

  /** 判断字段值是否为自定义镜像完整地址（以仓库地址开头） */
  function isCustomImageValue(fullValue: string, repositoryAddress: string): boolean {
    return !!fullValue && !!repositoryAddress && fullValue.startsWith(`${repositoryAddress}/`);
  }

  /** 从自定义镜像完整地址解析出 完整仓库名（含前缀）/tag */
  function parseCustomImageValue(fullValue: string, repositoryAddress: string): { name: string; tag: string } {
    if (!isCustomImageValue(fullValue, repositoryAddress)) return { name: '', tag: '' };
    const lastColon = fullValue.lastIndexOf(':');
    if (lastColon === -1) return { name: fullValue, tag: '' };
    return { name: fullValue.slice(0, lastColon), tag: fullValue.slice(lastColon + 1) };
  }

  function parseImageValue(fullValue: string, imageList: RuntimeImageOutputObj[]): { imageId: string; tag: string } {
    if (!fullValue || !imageList.length) return { imageId: '', tag: '' };
    const lastColon = fullValue.lastIndexOf(':');
    if (lastColon === -1) return { imageId: '', tag: fullValue };
    const name = fullValue.slice(0, lastColon);
    const tag = fullValue.slice(lastColon + 1);
    const image = imageList.find(i => i.name === name);
    return { imageId: image?.id ?? '', tag };
  }

  /** 从镜像字段值解析 tag（"name:tag" 或 "repo/name:tag" 取最后冒号后） */
  function parseTagFromValue(fullValue: string): string {
    if (!fullValue) return '';
    const lastColon = fullValue.lastIndexOf(':');
    return lastColon === -1 ? '' : fullValue.slice(lastColon + 1);
  }

  // ========== Watchers ==========
  // 写回 modelValue 统一入口：platform / custom 两种模式由 imageMode 分支，
  // 避免双 watch 各自维护写回逻辑。回填、用户选择、清空均经此方法。
  // （不能改用 Select @change：初始化回填/代码赋值不触发 change 事件）
  function writeBackModelValue() {
    if (imageMode.value === 'platform') {
      if (platformSelection.value.imageId && platformSelection.value.tag) {
        const image = images.value.find(i => i.id === platformSelection.value.imageId);
        if (image?.name) {
          emit('update:modelValue', `${image.name}:${platformSelection.value.tag}`);
        }
      } else {
        emit('update:modelValue', '');
      }
      return;
    }
    if (imageMode.value === 'custom') {
      // name 已含仓库前缀，落库值 = `${name}:${tag}`
      // 镜像名校验由 CustomImageSelect 内部 name FormItem 的 with-validate 自动触发，此处仅负责写回
      if (customSelection.value.name && customSelection.value.tag) {
        emit('update:modelValue', `${customSelection.value.name}:${customSelection.value.tag}`);
      } else {
        emit('update:modelValue', '');
      }
    }
  }

  watch([platformSelection, customSelection], writeBackModelValue, { deep: true });

  // 语言变化时重新拉取平台推荐镜像列表
  watch(
    () => props.language,
    () => {
      if (imageMode.value === 'platform') {
        fetchPlatformBuildImages();
      }
    },
  );

  // 等待父级仓库信息加载结束（repoInfoLoading 翻转）后再初始化，避免挂载时 repoInfo 未就绪
  // 误走平台模式拉取多余列表。监听加载状态而非 repoInfo 值：仓库未绑定时 repoInfo 恒 null 不变化，
  // 若监听 repoInfo 将永不触发导致组件卡在空态；挂载时已就绪（loading 已结束）由 immediate 兜底。
  watch(
    () => props.repoInfoLoading,
    loading => {
      if (loading) return;
      initState();
    },
    { immediate: true },
  );

  // ========== 远程搜索防抖 ==========
  const searchTag = debounce((keyword: string) => {
    if (platformSelection.value.imageId) {
      fetchImageTags(platformSelection.value.imageId, keyword);
    }
  }, 300);

  onBeforeUnmount(() => {
    searchTag.cancel();
  });
</script>
