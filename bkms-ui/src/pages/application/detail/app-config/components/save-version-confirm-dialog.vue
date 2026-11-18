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
    :before-close="canClose"
    class="save-version-confirm-dialog"
    header-align="left"
    width="80vw"
    @hidden="handleHidden"
  >
    <template #header>
      <div class="flex flex-wrap items-center gap-[8px] pr-[24px]">
        <span>{{ $t('保存并创建新版本') }}</span>
        <span
          v-if="preview?.fileName"
          class="min-w-0 truncate text-[12px] text-[#979BA5]"
          :title="preview.fileName"
        >
          {{ preview.fileName }}
        </span>
        <Tag v-if="isUnchanged">{{ $t('内容无差异') }}</Tag>
        <Tag
          v-if="diffStats.added"
          theme="success"
        >
          {{ $t('新增') }} +{{ diffStats.added }}
        </Tag>
        <Tag
          v-if="diffStats.deleted"
          theme="danger"
        >
          {{ $t('删除') }} -{{ diffStats.deleted }}
        </Tag>
      </div>
    </template>

    <template v-if="preview">
      <Exception
        v-if="isEmpty"
        class="!h-auto shrink-0 py-[16px]"
        scene="part"
        type="empty"
      >
        <template #title>{{ $t('还没有配置内容') }}</template>
        <template #description>
          <i18n-t keypath="确认后将生成空的 {version}，可供之后对比和回滚。">
            <template #version
              ><strong>{{ nextVersion }}</strong></template
            >
          </i18n-t>
        </template>
      </Exception>
      <template v-else>
        <i18n-t
          v-if="isUnchanged"
          class="mb-[12px] shrink-0 leading-[22px]"
          keypath="与当前已保存内容一致。确认后仍会生成 {version}。"
          tag="p"
        >
          <template #version
            ><strong>{{ nextVersion }}</strong></template
          >
        </i18n-t>
        <i18n-t
          v-else
          class="mb-[12px] shrink-0 leading-[22px]"
          keypath="请先核对与当前版本的差异。确认后将生成 {version}，并记入版本历史。"
          tag="p"
        >
          <template #version
            ><strong>{{ nextVersion }}</strong></template
          >
        </i18n-t>
        <div
          class="min-h-[80px] overflow-hidden"
          :style="{ height: `${diffHeight}px` }"
        >
          <MsEditor
            v-if="isShow"
            is-diff
            :lang="preview.lang"
            :model-value="preview.content"
            :options="diffOptions"
            :original="preview.originalContent"
            readonly
            :show-copy="false"
            :target-title="$t('即将保存 {0}', [nextVersion])"
            :title="
              preview.currentVersion ? $t('当前版本 {0}', [`V${preview.currentVersion}`]) : $t('当前内容（尚无版本）')
            "
            @diff-stats="handleDiffStats"
          />
        </div>
      </template>
      <Form
        ref="formRef"
        class="mt-[12px] shrink-0"
        form-type="vertical"
        :model="formData"
        :rules="rules"
      >
        <Form.FormItem
          :label="$t('{0} 版本描述', [nextVersion])"
          property="description"
          required
        >
          <Input
            v-model="formData.description"
            :placeholder="$t('请输入版本描述')"
            :rows="3"
            type="textarea"
          />
        </Form.FormItem>
      </Form>
    </template>

    <template #footer>
      <div class="flex justify-end">
        <Button
          class="mr-[8px]"
          :loading="loading"
          theme="primary"
          @click="handleConfirm"
        >
          {{ $t('确认保存') }}
        </Button>
        <Button
          :disabled="loading"
          @click="isShow = false"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';

  import { Button, Dialog, Exception, Form, Input, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';

  import type { SaveVersionPreview } from './types';
  import type { editor } from 'monaco-editor';

  const props = defineProps<{ preview: null | SaveVersionPreview }>();

  const emit = defineEmits<{ (e: 'confirm', description: string): void }>();

  const isShow = defineModel<boolean>('isShow', { default: false });
  const { t } = useI18n();
  const nextVersion = computed(() => `V${(props.preview?.currentVersion ?? 0) + 1}`);
  const isEmpty = computed(() => props.preview?.originalContent === '' && props.preview.content === '');
  const isUnchanged = computed(
    () => props.preview && !isEmpty.value && props.preview.originalContent === props.preview.content,
  );
  const diffStats = ref({ added: 0, deleted: 0 });
  const diffOptions = {
    diffWordWrap: 'off',
    ignoreTrimWhitespace: false,
    lineHeight: 20,
    renderSideBySide: true,
    useInlineViewWhenSpaceIsLimited: false,
  } satisfies editor.IDiffEditorConstructionOptions;
  const diffHeight = computed(() => {
    const countLines = (content: string) => content.split(/\r\n|\r|\n/).length;
    const rows = Math.max(
      4,
      countLines(props.preview?.originalContent ?? ''),
      countLines(props.preview?.content ?? ''),
    );
    return 40 + rows * diffOptions.lineHeight + 12;
  });

  /** 表单数据 */
  const formData = reactive({
    description: '',
  });

  /** 表单校验规则 */
  const rules = computed(() => ({
    description: [
      {
        message: t('请输入版本描述'),
        required: true,
        trigger: 'blur',
        validator: (val: string) => val.trim().length > 0,
      },
    ],
  }));

  /** 表单 ref */
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  const loading = ref(false);
  const canClose = () => !loading.value;

  /** 确认保存 */
  async function handleConfirm() {
    if (loading.value || !formRef.value) return;
    loading.value = true;
    try {
      await formRef.value.validate();
    } catch {
      stopLoading();
      return;
    }
    emit('confirm', formData.description.trim());
    // 由父组件负责关闭弹窗（在 handleSave 成功后）
  }

  function handleDiffStats({ added, deleted }: { added: number; deleted: number }) {
    // Monaco 会把空文本的占位行也计入差异。
    diffStats.value = {
      added: props.preview?.content ? added : 0,
      deleted: props.preview?.originalContent ? deleted : 0,
    };
  }

  /** 停止 loading */
  function stopLoading() {
    loading.value = false;
  }

  defineExpose({ stopLoading });

  /** 关闭弹窗 */
  function handleHidden() {
    stopLoading();
    formData.description = '';
    diffStats.value = { added: 0, deleted: 0 };
    formRef.value?.clearValidate();
  }
</script>

<style lang="postcss" scoped>
  .save-version-confirm-dialog :deep(.bk-modal-wrapper) {
    max-width: 90vw;
  }

  .save-version-confirm-dialog :deep(.bk-modal-body) {
    display: flex;
    flex-direction: column;
    max-height: 90vh;
  }

  .save-version-confirm-dialog :deep(.bk-modal-header),
  .save-version-confirm-dialog :deep(.bk-modal-footer) {
    flex-shrink: 0;
  }

  .save-version-confirm-dialog :deep(.bk-modal-content),
  .save-version-confirm-dialog :deep(.bk-dialog-content) {
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .save-version-confirm-dialog :deep(.bk-modal-content > div) {
    display: flex !important;
    flex-direction: column;
    min-height: 0;
  }

  .save-version-confirm-dialog :deep(.bk-modal-content) {
    overflow-y: auto;
  }

  .save-version-confirm-dialog :deep(.bk-dialog-content) {
    margin-bottom: 0;
  }

  .save-version-confirm-dialog :deep(.bk-dialog-footer) {
    padding-bottom: 16px;
    background-color: unset;
    border: none;
  }
</style>
