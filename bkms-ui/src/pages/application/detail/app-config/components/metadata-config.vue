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
  <BkmsContent
    :collapsible="true"
    :show-edit-icon="false"
  >
    <template #title>
      <div class="flex items-center">
        <UnderLineTips
          class="line-height-[18px]"
          :description="
            $t(
              '为应用的运行实例添加自定义标签 / 注解，常用于监控采集、调度筛选、资源关联等场景。填写的内容会追加在平台默认配置之上，不会覆盖平台已有内容。',
            )
          "
          :placement="PlacementEnum.RIGHT"
        >
          {{ $t('元数据配置') }}
        </UnderLineTips>
        <EnvScopeTag :current-env="currentEnv" />
      </div>
    </template>
    <div class="bg-[#fff] p-[16px]">
      <div class="grid grid-cols-3 gap-[16px]">
        <div
          v-for="section in sectionItems"
          :key="section.key"
          class="metadata-card border border-[#EAEBF0] rounded-[8px] min-w-0 overflow-hidden"
        >
          <div class="flex items-center gap-[10px] h-[32px] px-[12px] bg-[#F5F7FA] border-b border-[#EAEBF0]">
            <span
              :class="[
                'text-[12px] font-bold text-[#4D4F56]',
                { 'section-modified-label': controllers[section.key].hasOverride.value },
              ]"
            >
              {{ section.label }}
            </span>
            <Button
              v-if="!editingSection"
              class="!hover:text-[#3A84FF]"
              text
              @click="handleEdit(section.key)"
            >
              <EditLine />
            </Button>
          </div>

          <div class="px-[12px] py-[16px]">
            <template v-if="editingSection === section.key">
              <div class="flex items-center justify-between mb-[16px]">
                <Radio.Group
                  v-model="inputModes[section.key]"
                  type="capsule"
                  @change="(mode: string) => handleModeChange(section.key, mode)"
                >
                  <Radio.Button label="table">
                    {{ $t('表格模式') }}
                  </Radio.Button>
                  <Radio.Button label="text">
                    {{ $t('文本模式') }}
                  </Radio.Button>
                </Radio.Group>

                <Copy
                  v-if="inputModes[section.key] === 'text'"
                  class="text-[#979BA5] text-[16px] hover:text-[#3A84FF] cursor-pointer"
                  :title="$t('复制')"
                  @click="handleCopy"
                />
              </div>

              <KeyValue
                v-if="inputModes[section.key] === 'table'"
                ref="keyValueRefs"
                v-model="controllers[section.key].draft.value"
                :key-rules="getMetadataKeyRules(section.key)"
                :key-unique-rule="uniqueKeyRule"
                :value-rules="getMetadataValueRules(section.key)"
              />

              <Form
                v-else
                ref="textFormRefs"
                form-type="vertical"
                :model="textContent"
              >
                <Form.FormItem
                  class="!mb-0"
                  :property="section.key"
                  :rules="getTextRules(section.key)"
                >
                  <Input
                    v-model="textContent[section.key]"
                    :placeholder="$t('请输入参数名和参数值，如 {0}，多个参数换行分隔', ['key=value'])"
                    :rows="8"
                    type="textarea"
                  />
                </Form.FormItem>
              </Form>

              <div class="flex items-center gap-[8px] mt-[24px]">
                <Button
                  class="!w-[64px] !min-w-[64px]"
                  :loading="controllers[section.key].saving.value"
                  theme="primary"
                  @click="handleSave(section.key)"
                >
                  {{ $t('保存') }}
                </Button>
                <Button
                  class="!w-[64px] !min-w-[64px]"
                  @click="handleCancel(section.key)"
                >
                  {{ $t('取消') }}
                </Button>
                <Button
                  v-if="currentEnv && !currentEnv.isDefault"
                  :loading="controllers[section.key].resetting.value"
                  @click="handleResetToDefault(section.key)"
                >
                  {{ $t('恢复默认设置') }}
                </Button>
              </div>
            </template>

            <template v-else>
              <div
                v-if="displayEntries(section.key).length"
                class="flex flex-col"
              >
                <ProbeDetailItem
                  v-for="[key, value] in displayEntries(section.key)"
                  :key="key"
                  :label="key"
                  :value="value"
                >
                  {{ value ?? '--' }}
                </ProbeDetailItem>
              </div>
              <Exception
                v-else
                class="normal-exception"
                scene="part"
                type="empty"
              >
                <template #type>
                  <img
                    class="h-[100px]"
                    src="/empty.svg"
                  />
                </template>
                <template #description>
                  <span>{{ $t('尚未配置') }}，</span>
                  <Button
                    text
                    theme="primary"
                    @click="handleEdit(section.key)"
                  >
                    {{ $t('立即配置') }}
                  </Button>
                </template>
              </Exception>
            </template>
          </div>
        </div>
      </div>
    </div>
  </BkmsContent>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue';

  import { Button, Exception, Form, Input, Message, Radio } from 'bkui-vue';
  import { Copy, EditLine } from 'bkui-vue/lib/icon';
  import { PlacementEnum } from 'bkui-vue/lib/shared';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { AppSpecService } from '~/api/modules/v1/app-spec';
  import { BKMS_REGEX } from '~/common/const';
  import { copyText } from '~/common/util';
  import BkmsContent from '~/components/bkms-content.vue';
  import KeyValue, { type FormRule } from '~/components/key-value.vue';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';

  import EnvScopeTag from './env-scope-tag.vue';
  import ProbeDetailItem from './probe-detail-item.vue';

  import type { ExtendedEnv } from './types';
  import type {
    AppSpecAnnotationsInput,
    AppSpecAnnotationsOutput,
    AppSpecLabelsInput,
    AppSpecLabelsOutput,
  } from '~/@types/v1/app-spec';

  /** 文本模式中重复 key 的位置 */
  type DuplicateTextKey = { key: string; line: number };
  /** 编辑模式：表格或文本 */
  type InputMode = 'table' | 'text';
  /** 文本模式中不合法的 key/value 位置 */
  type InvalidTextMetadata = { key: string; line: number };
  /** 开启 needStatus 后，HTTP 异常响应中携带的状态码 */
  type MetadataApiError = { status?: number };
  /** API 响应：可能已解包（直接返回 data）或未解包（嵌套在 data 字段中） */
  type MetadataApiResponse = MetadataSectionOutput | null | { data?: MetadataSectionOutput };
  /** 元数据键值对 */
  type MetadataRecord = Record<string, string>;
  /** 保存/设置元数据时的请求体 */
  type MetadataSectionInput = AppSpecAnnotationsInput | AppSpecLabelsInput;
  /** 元数据 section 标识：标签或注解 */
  type MetadataSectionKey = 'annotations' | 'labels';
  /** section 的 API 配置，注入到工厂函数中以区分 labels/annotations */
  interface MetadataSectionOptions {
    /** section 所属字段名 */
    field: MetadataSectionKey;
    /** 删除环境级覆盖 */
    deleteEnv: (appID: string, envName: string) => Promise<unknown>;
    /** 获取应用默认配置 */
    fetchDefault: (appID: string) => Promise<MetadataApiResponse>;
    /** 获取环境覆盖配置（404 时 reject，需调用方自行 catch） */
    fetchEnvOverride: (appID: string, envName: string) => Promise<MetadataApiResponse>;
    /** 保存应用默认配置 */
    saveDefault: (appID: string, payload: MetadataSectionInput) => Promise<unknown>;
    /** 保存环境覆盖配置 */
    saveEnv: (appID: string, envName: string, payload: MetadataSectionInput) => Promise<unknown>;
  }

  /** API 返回的元数据 section 数据 */
  type MetadataSectionOutput = AppSpecAnnotationsOutput | AppSpecLabelsOutput;

  interface Props {
    /** 当前选中的环境 */
    currentEnv: ExtendedEnv | null;
  }

  const props = defineProps<Props>();

  const emit = defineEmits<{
    'env-modified-change': [];
    'loading-change': [value: boolean];
  }>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  /** 供 useLeaveConfirm 监听的脏数据模型，编辑态下任何变更都会同步到此处 */
  const dirtyModel = reactive({
    labels: {},
    annotations: {},
  });
  const { confirmBox, forceCleanDirtyTag } = useLeaveConfirm(dirtyModel);

  const sectionItems = computed(() => [
    { key: 'labels' as MetadataSectionKey, label: '标签（Labels）' },
    { key: 'annotations' as MetadataSectionKey, label: '注解（Annotations）' },
  ]);

  const inputModes = reactive<Record<MetadataSectionKey, InputMode>>({
    labels: 'table',
    annotations: 'table',
  });
  const textContent = reactive<Record<MetadataSectionKey, string>>({
    labels: '',
    annotations: '',
  });

  const editingSection = ref<MetadataSectionKey | null>(null);
  const keyValueRefs = ref<{ validate: () => Promise<boolean> }[]>([]);
  const loading = ref(true);
  const textFormRefs = ref<InstanceType<typeof Form>[]>([]);
  const uniqueKeyRule = { message: t('Key 不能重复'), trigger: 'blur' };

  /** 各元数据类型由系统维护、禁止用户写入的 key。 */
  const reservedMetadataKeys: Record<MetadataSectionKey, Set<string>> = {
    labels: new Set(['app.kubernetes.io/name', 'io.tencent.bcs.dev/deletion-allow']),
    annotations: new Set([
      'controller.kubernetes.io/pod-deletion-cost',
      'io.tencent.bcs.dev/update-strategy-type-allow',
    ]),
  };

  /** Labels 和 Annotations 共用同一套状态管理逻辑，差异仅由字段名和 API 方法决定 */
  function createMetadataSectionController(options: MetadataSectionOptions) {
    const model = ref<MetadataRecord>({});
    const draft = ref<MetadataRecord>({});
    const snapshot = ref<MetadataRecord>({});
    const defaultCache = ref<MetadataRecord>({});
    const hasOverride = ref(false);
    const saving = ref(false);
    const resetting = ref(false);

    /** 深拷贝并标准化数据，确保返回值始终为普通对象 */
    const normalize = (data?: MetadataRecord | null): MetadataRecord => cloneDeep(data || {});
    /**
     * 解包 API 响应中的 data 字段
     * v1Fetch 在运行时可能已经自动解包；这里同时兼容原始响应类型和已解包的运行时返回值。
     */
    const unwrapSectionData = (data: MetadataApiResponse): MetadataSectionOutput | null => {
      if (!data) return null;
      if (typeof data === 'object' && 'data' in data) return (data as { data?: MetadataSectionOutput }).data ?? null;
      return data as MetadataSectionOutput;
    };
    /** 从 API 响应中提取当前 section 的字段数据并标准化 */
    const getSectionRecord = (data: MetadataApiResponse) =>
      normalize((unwrapSectionData(data) as Record<string, MetadataRecord | undefined>)?.[options.field]);
    /** 将内存中的 record 构建为符合 API 请求体的 payload 格式 */
    const buildPayload = (record: MetadataRecord): MetadataSectionInput => ({ [options.field]: normalize(record) });

    /** 获取应用的默认配置并缓存，用于对比环境覆盖及回退 */
    async function fetchDefault() {
      if (!appDetailStore.appID) return {};

      const result = await options.fetchDefault(appDetailStore.appID);
      defaultCache.value = getSectionRecord(result);
      return defaultCache.value;
    }

    /**
     * 根据当前环境加载对应的元数据
     * - 默认环境：直接读取 default 配置，不展示覆盖标识
     * - 普通环境：优先读取环境级 override；未单独配置（404）时回显 default 配置
     */
    async function loadEnvData(env: ExtendedEnv) {
      const defaultValue = await fetchDefault();

      if (env.isDefault) {
        model.value = normalize(defaultValue);
        hasOverride.value = false;
        return;
      }

      try {
        const override = await options.fetchEnvOverride(appDetailStore.appID, env.name);
        model.value = getSectionRecord(override);
        hasOverride.value = true;
      } catch (error: unknown) {
        if ((error as MetadataApiError)?.status === 404) {
          model.value = normalize(defaultValue);
          hasOverride.value = false;
          return;
        }
        throw error;
      }
    }

    /** 进入编辑模式：保存当前数据快照，初始化草稿数据 */
    function enterEdit() {
      snapshot.value = normalize(model.value);
      draft.value = normalize(model.value);
    }

    /** 取消编辑：将草稿恢复到进入编辑时的快照状态 */
    function cancelEdit() {
      draft.value = normalize(snapshot.value);
    }

    /**
     * 保存元数据配置
     * - 默认环境：保存到 default endpoint，更新本地缓存
     * - 普通环境：保存到 env override endpoint，标记存在覆盖并通知父页面刷新
     */
    async function save(currentEnv: ExtendedEnv | null) {
      if (!currentEnv || !appDetailStore.appID) return false;

      saving.value = true;
      try {
        const payload = buildPayload(draft.value);
        if (currentEnv.isDefault) {
          await options.saveDefault(appDetailStore.appID, payload);
          defaultCache.value = normalize(draft.value);
          hasOverride.value = false;
        } else {
          await options.saveEnv(appDetailStore.appID, currentEnv.name, payload);
          hasOverride.value = true;
          emit('env-modified-change');
        }

        model.value = normalize(draft.value);
        snapshot.value = normalize(draft.value);
        Message({ theme: 'success', message: t('操作成功') });
        return true;
      } finally {
        saving.value = false;
      }
    }

    /**
     * 恢复为默认设置：删除环境级覆盖配置，使当前环境继承默认值
     * 仅普通环境可用，默认环境直接忽略此操作。
     */
    async function resetToDefault(currentEnv: ExtendedEnv | null) {
      if (!currentEnv || currentEnv.isDefault || !appDetailStore.appID) return false;

      resetting.value = true;
      try {
        await options.deleteEnv(appDetailStore.appID, currentEnv.name);
        model.value = normalize(defaultCache.value);
        draft.value = normalize(defaultCache.value);
        snapshot.value = normalize(defaultCache.value);
        hasOverride.value = false;
        Message({ theme: 'success', message: t('操作成功') });
        emit('env-modified-change');
        return true;
      } finally {
        resetting.value = false;
      }
    }

    return {
      draft,
      defaultCache,
      hasOverride,
      model,
      resetting,
      saving,
      cancelEdit,
      enterEdit,
      loadEnvData,
      resetToDefault,
      save,
    };
  }

  function getLabelValueRuleMessage() {
    return t('以字母或数字开头和结尾，中间可含 字母、数字、- _ .，最长 63 字符。');
  }

  /** 生成表格模式的 key 校验规则：必填、格式合法且不能使用系统保留 key。 */
  function getMetadataKeyRules(sectionKey: MetadataSectionKey): FormRule[] {
    return [
      ...getRequiredKeyRules(),
      {
        message: getMetadataRuleMessage(),
        trigger: 'blur',
        validator: (value: unknown) => !value || isValidMetadataKey(String(value).trim()),
      },
      {
        message: t('系统保留字段，禁止写入'),
        trigger: 'blur',
        validator: (value: unknown) => !reservedMetadataKeys[sectionKey].has(String(value ?? '').trim()),
      },
    ];
  }

  function getMetadataRuleMessage() {
    return t('以字母或数字开头和结尾，中间可含 字母、数字、- _ .，最长 63 字符。');
  }

  /** Labels 在原有必填规则基础上追加格式校验；Annotations 仅保留必填规则。 */
  function getMetadataValueRules(sectionKey: MetadataSectionKey): FormRule[] {
    if (sectionKey === 'annotations') return getRequiredValueRules(sectionKey);

    return [
      ...getRequiredValueRules(sectionKey),
      {
        message: getLabelValueRuleMessage(),
        pattern: BKMS_REGEX.kubernetesLabelValueRegex,
        trigger: 'blur',
      },
    ];
  }

  function getRequiredKeyRules(): FormRule[] {
    return [{ required: true, message: getMetadataRuleMessage(), trigger: 'blur' }];
  }

  function getRequiredValueRules(sectionKey: MetadataSectionKey): FormRule[] {
    if (sectionKey === 'annotations') {
      return [{ required: true, message: t('value 必填项'), trigger: 'blur' }];
    }
    return [{ required: true, message: getLabelValueRuleMessage(), trigger: 'blur' }];
  }

  /** 元数据 key 支持 name 或 prefix/name；prefix 必须为小写 DNS 名称。 */
  function isValidMetadataKey(key: string): boolean {
    const segments = key.split('/');
    if (segments.length === 1) return BKMS_REGEX.kubernetesMetadataNameRegex.test(segments[0]);
    if (segments.length !== 2) return false;

    return (
      BKMS_REGEX.kubernetesMetadataPrefixRegex.test(segments[0]) &&
      BKMS_REGEX.kubernetesMetadataNameRegex.test(segments[1])
    );
  }

  /** Labels 和 Annotations 各自独立一个 controller 实例，通过工厂函数注入对应 API */
  const controllers = {
    labels: createMetadataSectionController({
      field: 'labels',
      fetchDefault: appID => AppSpecService.getAppDefaultAppSpecLabels({ appID }),
      fetchEnvOverride: (appID, envName) =>
        AppSpecService.getEnvAppSpecLabels({ appID, envName }, { interceptorErr: false, needStatus: true }),
      saveDefault: (appID, payload) =>
        AppSpecService.setAppDefaultAppSpecLabels({ appID, ...(payload as AppSpecLabelsInput) }),
      saveEnv: (appID, envName, payload) =>
        AppSpecService.setEnvAppSpecLabels({ appID, envName, ...(payload as AppSpecLabelsInput) }),
      deleteEnv: (appID, envName) => AppSpecService.deleteEnvAppSpecLabels({ appID, envName }),
    }),
    annotations: createMetadataSectionController({
      field: 'annotations',
      fetchDefault: appID => AppSpecService.getAppDefaultAppSpecAnnotations({ appID }),
      fetchEnvOverride: (appID, envName) =>
        AppSpecService.getEnvAppSpecAnnotations({ appID, envName }, { interceptorErr: false, needStatus: true }),
      saveDefault: (appID, payload) =>
        AppSpecService.setAppDefaultAppSpecAnnotations({ appID, ...(payload as AppSpecAnnotationsInput) }),
      saveEnv: (appID, envName, payload) =>
        AppSpecService.setEnvAppSpecAnnotations({ appID, envName, ...(payload as AppSpecAnnotationsInput) }),
      deleteEnv: (appID, envName) => AppSpecService.deleteEnvAppSpecAnnotations({ appID, envName }),
    }),
  };

  /** 获取指定 section 可用于展示的键值对列表（过滤空 key） */
  function displayEntries(sectionKey: MetadataSectionKey) {
    return Object.entries(controllers[sectionKey].model.value).filter(([key]) => key);
  }

  /** 查找文本模式中首个重复的 key；key 去除首尾空格后按大小写敏感方式比较 */
  function findDuplicateTextKey(text: string): DuplicateTextKey | null {
    const keys = new Set<string>();
    const lines = text.split('\n');
    for (let index = 0; index < lines.length; index += 1) {
      const line = lines[index].trim();
      if (!line) continue;

      const key = line.substring(0, line.indexOf('=')).trim();
      if (keys.has(key)) return { key, line: index + 1 };
      keys.add(key);
    }

    return null;
  }

  /** 定位文本模式中首个 Label value 格式错误的行号。 */
  function findInvalidLabelValueLine(text: string): null | number {
    const lines = text.split('\n');
    for (let index = 0; index < lines.length; index += 1) {
      const line = lines[index].trim();
      if (!line) continue;

      const value = line.substring(line.indexOf('=') + 1).trim();
      if (!BKMS_REGEX.kubernetesLabelValueRegex.test(value)) return index + 1;
    }
    return null;
  }

  /** 定位文本模式中首个 key 格式错误的位置。 */
  function findInvalidTextKey(text: string): InvalidTextMetadata | null {
    const lines = text.split('\n');
    for (let index = 0; index < lines.length; index += 1) {
      const line = lines[index].trim();
      if (!line) continue;

      const key = line.substring(0, line.indexOf('=')).trim();
      if (!isValidMetadataKey(key)) return { key, line: index + 1 };
    }
    return null;
  }

  /** 校验文本模式中的非空行，每行必须且只能包含一个等号，且 key、value 均不能为空。 */
  function findInvalidTextLine(text: string): null | number {
    const lines = text.split('\n');
    for (let index = 0; index < lines.length; index += 1) {
      const line = lines[index].trim();
      if (!line) continue;

      const delimiterIndex = line.indexOf('=');
      if (
        delimiterIndex <= 0 ||
        delimiterIndex !== line.lastIndexOf('=') ||
        !line.substring(0, delimiterIndex).trim() ||
        !line.substring(delimiterIndex + 1).trim()
      ) {
        return index + 1;
      }
    }
    return null;
  }

  /** 定位文本模式中首个系统保留 key 的位置。 */
  function findReservedTextKey(text: string, sectionKey: MetadataSectionKey): InvalidTextMetadata | null {
    const lines = text.split('\n');
    for (let index = 0; index < lines.length; index += 1) {
      const line = lines[index].trim();
      if (!line) continue;

      const key = line.substring(0, line.indexOf('=')).trim();
      if (reservedMetadataKeys[sectionKey].has(key)) return { key, line: index + 1 };
    }
    return null;
  }

  /** 文本模式表单规则：格式校验优先，格式正确后再检查重复 key */
  function getTextRules(sectionKey: MetadataSectionKey): FormRule[] {
    const text = textContent[sectionKey];
    const duplicateKey = findDuplicateTextKey(text);
    const invalidLine = findInvalidTextLine(text);
    const invalidKey = findInvalidTextKey(text);
    const reservedKey = findReservedTextKey(text, sectionKey);
    const invalidValueLine = sectionKey === 'labels' ? findInvalidLabelValueLine(text) : null;

    const rules: FormRule[] = [
      {
        message: t('第 {0} 行格式错误，{1}', [
          invalidLine ?? 1,
          sectionKey === 'labels' ? getLabelValueRuleMessage() : t('value 必填项'),
        ]),
        trigger: 'blur',
        validator: (value: unknown) => findInvalidTextLine(String(value ?? '')) === null,
      },
      {
        message: t('第 {0} 行的 Key {1} 格式不正确，{2}', [
          invalidKey?.line ?? 1,
          invalidKey?.key ?? '',
          getMetadataRuleMessage(),
        ]),
        trigger: 'blur',
        validator: (value: unknown) => {
          const currentText = String(value ?? '');
          return findInvalidTextLine(currentText) !== null || findInvalidTextKey(currentText) === null;
        },
      },
      {
        message: t('第 {0} 行的 Key {1} 为系统保留字段，禁止写入', [reservedKey?.line ?? 1, reservedKey?.key ?? '']),
        trigger: 'blur',
        validator: (value: unknown) => {
          const currentText = String(value ?? '');
          return findInvalidTextLine(currentText) !== null || findReservedTextKey(currentText, sectionKey) === null;
        },
      },
      {
        message: t('第 {0} 行的 key {1} 重复', [duplicateKey?.line ?? 1, duplicateKey?.key ?? '']),
        trigger: 'blur',
        validator: (value: unknown) => {
          const currentText = String(value ?? '');
          return findInvalidTextLine(currentText) !== null || findDuplicateTextKey(currentText) === null;
        },
      },
    ];

    if (sectionKey === 'labels') {
      rules.splice(3, 0, {
        message: t('第 {0} 行的 Value 格式不正确，{1}', [invalidValueLine ?? 1, getLabelValueRuleMessage()]),
        trigger: 'blur',
        validator: (value: unknown) => {
          const currentText = String(value ?? '');
          return findInvalidTextLine(currentText) !== null || findInvalidLabelValueLine(currentText) === null;
        },
      });
    }

    return rules;
  }

  /** 取消编辑：恢复草稿数据并退出编辑模式 */
  function handleCancel(sectionKey: MetadataSectionKey) {
    controllers[sectionKey].cancelEdit();
    textContent[sectionKey] = recordToText(controllers[sectionKey].draft.value);
    editingSection.value = null;
    forceCleanDirtyTag();
  }

  /** 复制当前编辑 section 的文字模式内容到剪贴板 */
  function handleCopy() {
    if (editingSection.value) {
      copyText(textContent[editingSection.value]);
    }
  }

  /**
   * 点击编辑按钮进入编辑模式
   * 当已有 section 处于编辑中时不响应其他 section 的编辑请求，确保同一时间只编辑一个 section。
   */
  function handleEdit(sectionKey: MetadataSectionKey) {
    if (editingSection.value && editingSection.value !== sectionKey) return;

    controllers[sectionKey].enterEdit();
    inputModes[sectionKey] = 'table';
    textContent[sectionKey] = recordToText(controllers[sectionKey].draft.value);
    editingSection.value = sectionKey;
    forceCleanDirtyTag();
  }

  /**
   * 响应环境切换：确认是否有未保存的脏数据，通过后重新加载两个 section 的数据
   * 由父组件通过 defineExpose 调用。
   * @returns 是否成功切换环境
   */
  async function handleEnvChange(env: ExtendedEnv) {
    if (!(await confirmBox())) return false;

    editingSection.value = null;
    loading.value = true;
    try {
      await Promise.all([controllers.labels.loadEnvData(env), controllers.annotations.loadEnvData(env)]);
      forceCleanDirtyTag();
    } finally {
      loading.value = false;
    }

    return true;
  }

  /**
   * 切换输入模式（表格/文字）
   * - 切到文字模式：将草稿数据同步到文本输入框
   * - 切到表格模式：将文本内容解析回草稿数据
   */
  function handleModeChange(sectionKey: MetadataSectionKey, mode: string) {
    if (mode === 'text') {
      syncTextFromDraft(sectionKey);
    } else if (mode === 'table') {
      syncDraftFromText(sectionKey);
    }
  }

  /** 恢复为默认设置，成功后退出编辑模式 */
  async function handleResetToDefault(sectionKey: MetadataSectionKey) {
    const reset = await controllers[sectionKey].resetToDefault(props.currentEnv);
    if (reset) {
      editingSection.value = null;
      forceCleanDirtyTag();
    }
  }

  /**
   * 保存当前编辑内容
   * 若在文字模式下，先将文本内容同步到草稿数据，再调用保存接口。
   */
  async function handleSave(sectionKey: MetadataSectionKey) {
    if (inputModes[sectionKey] === 'table') {
      const [keyValue] = keyValueRefs.value;
      if (!keyValue || !(await keyValue.validate())) return;
    } else {
      const [textForm] = textFormRefs.value;
      if (!textForm) return;
      try {
        await textForm.validate(sectionKey);
      } catch {
        return;
      }
      syncDraftFromText(sectionKey);
    }

    const saved = await controllers[sectionKey].save(props.currentEnv);
    if (saved) {
      editingSection.value = null;
      forceCleanDirtyTag();
    }
  }

  function recordToText(data: MetadataRecord): string {
    return Object.entries(data)
      .filter(([key]) => key)
      .map(([key, value]) => `${key}=${value}`)
      .join('\n');
  }

  /** 将文字输入框内容解析后同步到草稿数据 */
  function syncDraftFromText(sectionKey: MetadataSectionKey) {
    controllers[sectionKey].draft.value = textToRecord(textContent[sectionKey]);
  }

  /** 将草稿数据序列化为文本，同步到文字输入框 */
  function syncTextFromDraft(sectionKey: MetadataSectionKey) {
    textContent[sectionKey] = recordToText(controllers[sectionKey].draft.value);
  }

  /**
   * 将 key=value 换行分隔的文本解析为键值对 record
   * - 忽略空行
   * - 解析失败的行（不含 = 或以 = 开头）直接跳过
   */
  function textToRecord(text: string): MetadataRecord {
    if (!text.trim()) return {};

    const result: MetadataRecord = {};
    for (const line of text.split('\n')) {
      const trimmedLine = line.trim();
      if (!trimmedLine) continue;

      const delimiterIndex = trimmedLine.indexOf('=');
      if (delimiterIndex <= 0) continue;

      const key = trimmedLine.substring(0, delimiterIndex).trim();
      const value = trimmedLine.substring(delimiterIndex + 1).trim();
      if (key) {
        result[key] = value;
      }
    }

    return result;
  }

  /** 同步编辑态变更到 dirtyModel，驱动 useLeaveConfirm 的离开确认逻辑 */
  watch(
    () => ({
      editingSection: editingSection.value,
      labels: controllers.labels.draft.value,
      annotations: controllers.annotations.draft.value,
      textContent: { ...textContent },
    }),
    value => {
      Object.assign(dirtyModel, cloneDeep(value));
    },
    { deep: true },
  );

  watch(loading, val => emit('loading-change', val), { immediate: true });

  defineExpose({
    handleEnvChange,
    loading,
  });
</script>

<style lang="postcss" scoped>
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }

  :deep(.bk-form-label) {
    color: #4d4f56;
  }

  :deep(.metadata-card .bk-radio-group) {
    background-color: #eaebf0;
  }

  .section-modified-label {
    border-left: 3px solid #ff9c01;
    padding-left: 6px;
  }

  :deep(.bk-form-error) {
    position: static;
  }
</style>
