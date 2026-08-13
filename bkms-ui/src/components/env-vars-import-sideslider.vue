<template>
  <Sideslider
    v-model:is-show="visible"
    :before-close="handleBeforeClose"
    quick-close
    render-directive="if"
    :width="960"
    @hidden="resetState"
  >
    <template #header>
      <div class="flex items-center gap-[12px]">
        <span>{{ title || $t('导入变量') }}</span>
        <template v-if="showTargetInfo">
          <Divider
            class="h-[14px] mx-[0] text-[12px]"
            color="#DCDEE5"
            direction="vertical"
            type="solid"
          />
          <span class="text-[12px] font-normal text-[#979BA5]">
            {{ targetLabel || $t('环境') }}：{{ targetName || '--' }}
          </span>
          <Tag
            v-if="targetTag"
            size="small"
          >
            {{ targetTag }}
          </Tag>
        </template>
      </div>
    </template>
    <div class="h-full overflow-y-auto p-[24px]">
      <Upload
        :key="uploadKey"
        accept=".env"
        :custom-request="handlePreviewRequest"
        :disabled="isBusy"
        :handle-res-code="() => true"
        :is-show-preview="false"
        :limit="1"
        :multiple="false"
        name="file"
        :size="1"
        theme="draggable"
        @delete="handleFileDelete"
        @error="handleUploadError"
        @success="handlePreviewSuccess"
      >
        <div class="flex h-[92px] flex-col items-center justify-center text-[14px] text-[#63656E]">
          <UploadIcon
            class="mb-[8px] text-[32px] text-[#979BA5]"
            height="32"
            width="32"
          />
          <span class="text-[12px]">
            {{ $t('将文件拖到此处或') }} <span class="text-[#3A84FF]">{{ $t('点击上传') }}</span>
          </span>
        </div>
      </Upload>
      <div class="mt-[8px] text-[12px]">
        {{ $t('仅支持 .env 类型文件，文件大小不超过 1 MiB') }}，
        <Button
          :disabled="isBusy"
          :loading="isTemplateDownloading"
          text
          theme="primary"
          @click="handleDownloadTemplate"
          >{{ $t('点击下载模板') }}
        </Button>
      </div>
      <div
        v-if="previewLoading"
        class="flex min-h-[360px] flex-col items-center justify-center"
      >
        <Loading class="animate-spin text-[52px] text-[#3A84FF]" />
        <div class="mt-[20px] text-[20px] text-[#313238]">{{ $t('文件正在解析中...') }}</div>
      </div>
      <section
        v-else-if="previewData"
        class="mt-[24px]"
      >
        <div class="mb-[16px] flex items-center justify-between">
          <div class="text-[14px] font-bold">
            {{ $t('请确认以下变量信息（共 {count} 个）', { count: totalCount }) }}
          </div>
          <Radio.Group
            v-model="activeFilter"
            type="capsule"
          >
            <Radio.Button label="all">
              <span class="inline-flex items-center">
                <i
                  class="bkms-icon bkms-icon-quanbu-xuanzhong mr-[4px]"
                  :class="activeFilter === 'all' ? 'text-[#3A84FF]' : 'text-[#979BA5]'"
                ></i>
                {{ $t('全部') }} {{ totalCount }}
              </span>
            </Radio.Button>
            <Radio.Button label="new">
              <span class="inline-flex items-center">
                <i class="env-var-action-icon env-var-action-icon-new"></i>
                {{ $t('新增') }} {{ newCount }}
              </span>
            </Radio.Button>
            <Radio.Button label="overwrite">
              <span class="inline-flex items-center">
                <i class="env-var-action-icon env-var-action-icon-overwrite"></i>
                {{ $t('覆盖') }} {{ overwriteCount }}
              </span>
            </Radio.Button>
          </Radio.Group>
        </div>
        <Table
          :data="filteredItems"
          :row-height="48"
        >
          <TableColumn
            field="key"
            label="Key"
            :min-width="160"
          />
          <TableColumn
            label="Value"
            :min-width="200"
          >
            <template #default="{ row }">
              <template v-if="row.action === 'overwrite'">
                <div class="leading-[20px]">{{ row.value || '--' }}</div>
                <div class="text-[#979BA5] leading-[20px]">{{ $t('原值') }}：{{ row.originalValue || '--' }}</div>
              </template>
              <template v-else>{{ row.value || '--' }}</template>
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('描述')"
            :min-width="180"
          >
            <template #default="{ row }">
              {{ row.description || '--' }}
            </template>
          </TableColumn>
          <TableColumn
            v-if="showEffectiveScope"
            :label="$t('生效环境类型')"
            width="140"
          >
            <template #default="{ row }">
              <template v-if="!row.effectiveScope?.type">
                <Tag theme="danger">{{ $t('缺失') }}</Tag>
                <div class="text-[#979BA5] text-[12px] leading-[20px]">{{ $t('默认设置为「所有」') }}</div>
              </template>
              <Tag
                v-else
                :class="getScopeDisplay(row.effectiveScope.type, row.effectiveScope.value || '').tagClass"
              >
                {{ getScopeDisplay(row.effectiveScope.type, row.effectiveScope.value || '').label }}
              </Tag>
            </template>
          </TableColumn>
          <TableColumn
            :label="$t('操作')"
            width="140"
          >
            <template #default="{ row }"
              ><Tag :theme="row.action === 'overwrite' ? 'warning' : 'success'">{{
                row.action === 'overwrite' ? $t('覆盖') : $t('新增')
              }}</Tag></template
            >
          </TableColumn>
        </Table>
      </section>
    </div>
    <template #footer>
      <Button
        :disabled="!previewData || previewLoading"
        :loading="importLoading"
        theme="primary"
        @click="handleImport"
        >{{ previewData ? $t('导入') : $t('确定') }}</Button
      >
      <Button
        class="ml-[8px]"
        :disabled="isBusy"
        @click="handleCancel"
        >{{ $t('取消') }}</Button
      >
    </template>
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Divider, Message, Radio, Sideslider, Tag, Upload } from 'bkui-vue';
  import { Loading, Upload as UploadIcon } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { appendTraceId } from '~/api/trace-id';
  import { useFileExport } from '~/composables/use-file-export';
  import { getScopeDisplay } from '~/composables/use-scope-display';

  import type { UploadRequestOptions } from 'bkui-vue/lib/upload/upload.type';
  import type { EnvVarImportPreviewOutputObj, EnvVarImportPreviewSummaryOutputObj } from '~/@types/v1/envvars';

  type FilterType = 'all' | 'new' | 'overwrite';
  const props = withDefaults(
    defineProps<{
      downloadTemplateRequest: () => Promise<Response>;
      importRequest: (file: File) => Promise<EnvVarImportPreviewSummaryOutputObj>;
      previewRequest: (file: File) => Promise<EnvVarImportPreviewOutputObj>;
      showEffectiveScope?: boolean;
      showTargetInfo?: boolean;
      targetLabel?: string;
      targetName?: string;
      targetTag?: string;
      templateFilename: string;
      title?: string;
      visible: boolean;
    }>(),
    {
      showEffectiveScope: false,
      showTargetInfo: true,
      targetLabel: '',
      targetName: '',
      targetTag: '',
      title: '',
    },
  );
  const emit = defineEmits<{
    success: [summary: EnvVarImportPreviewSummaryOutputObj];
    'update:visible': [value: boolean];
  }>();

  const { t } = useI18n();

  const activeFilter = ref<FilterType>('all');
  const importFile = ref<File>(); // 当前待导入的文件
  const importLoading = ref(false); // 导入按钮 loading
  const previewData = ref<EnvVarImportPreviewOutputObj>(); // 服务端返回的预览数据
  const previewLoading = ref(false); // 文件解析 loading
  const uploadKey = ref(0); // 用于重置 Upload 组件
  const { exportFile, isExporting: isTemplateDownloading } = useFileExport();

  // 双向绑定 visible，Sideslider 关闭时通过 @hidden 重置状态
  const visible = computed({ get: () => props.visible, set: value => emit('update:visible', value) });

  // 是否处于忙碌状态（解析中或导入中），用于禁用操作
  const isBusy = computed(() => previewLoading.value || importLoading.value || isTemplateDownloading.value);

  // ── 预览数据派生 ──
  const previewItems = computed(() => previewData.value?.items || []);
  const totalCount = computed(() => previewData.value?.summary?.total ?? previewItems.value.length);
  const newCount = computed(
    () => previewData.value?.summary?.new ?? previewItems.value.filter(item => item.action === 'new').length,
  );
  const overwriteCount = computed(
    () =>
      previewData.value?.summary?.overwrite ?? previewItems.value.filter(item => item.action === 'overwrite').length,
  );
  const filteredItems = computed(() =>
    activeFilter.value === 'all'
      ? previewItems.value
      : previewItems.value.filter(item => item.action === activeFilter.value),
  );

  // 错误提示
  function getErrorMessage(error: unknown, fallback: string, showTraceId = true) {
    const apiError = error as { error?: { message?: string; traceId?: string }; message?: string; traceId?: string };
    const message = apiError.error?.message || apiError.message || fallback;
    return showTraceId ? appendTraceId(message, apiError.error?.traceId || apiError.traceId) : message;
  }
  // 忙时禁止关闭面板
  function handleBeforeClose() {
    return !isBusy.value;
  }
  function handleCancel() {
    if (!isBusy.value) visible.value = false;
  }
  // 下载模板
  function handleDownloadTemplate() {
    return exportFile({
      request: props.downloadTemplateRequest,
      fallbackFilename: props.templateFilename,
    });
  }
  function handleFileDelete() {
    importFile.value = undefined;
    previewData.value = undefined;
    activeFilter.value = 'all';
  }
  // 确认导入：调用 importRequest，成功后关闭面板
  async function handleImport() {
    if (!importFile.value || !previewData.value || isBusy.value) return;
    importLoading.value = true;
    try {
      const summary = await props.importRequest(importFile.value);
      Message({ theme: 'success', message: t('导入成功') });
      emit('success', summary);
      visible.value = false;
    } catch (error) {
      showError(error, t('导入失败'));
    } finally {
      importLoading.value = false;
    }
  }
  // 自定义上传：校验文件 → 调用 previewRequest 获取预览数据
  function handlePreviewRequest(options: UploadRequestOptions) {
    const file = options.file;
    previewData.value = undefined;
    activeFilter.value = 'all';
    importFile.value = file;
    const error = validateFile(file);
    if (error) return Promise.reject(new Error(error));
    previewLoading.value = true;
    return props
      .previewRequest(file)
      .catch(error => {
        const apiError = error as { error?: { traceId?: string }; traceId?: string };
        const uploadError = new Error(`${t('上传失败')}: ${getErrorMessage(error, t('文件解析失败'), false)}`);
        Object.assign(uploadError, { traceId: apiError.error?.traceId || apiError.traceId });
        throw uploadError;
      })
      .finally(() => {
        previewLoading.value = false;
      });
  }
  function handlePreviewSuccess(response: EnvVarImportPreviewOutputObj) {
    previewData.value = response;
  }
  function handleUploadError(file: File, _files: unknown[], error: unknown) {
    previewData.value = undefined;
    const validationError = validateFile(file);
    showError(validationError ? new Error(validationError) : error, t('文件解析失败'));
  }
  // Sideslider 关闭后重置所有状态
  function resetState() {
    activeFilter.value = 'all';
    importFile.value = undefined;
    previewData.value = undefined;
    previewLoading.value = false;
    importLoading.value = false;
    uploadKey.value += 1;
  }
  function showError(error: unknown, fallback: string) {
    Message({ theme: 'error', message: getErrorMessage(error, fallback) });
  }
  // 校验 .env 文件格式与大小
  function validateFile(file: File) {
    if (!file.name.toLowerCase().endsWith('.env')) return t('仅支持 .env 类型文件');
    if (!file.size) return t('文件不能为空');
    if (file.size > 1024 * 1024) return t('文件大小不能超过 1 MiB');
    return '';
  }
</script>

<style scoped>
  /* stylelint-disable-next-line selector-class-pattern */
  :deep(.bk-upload-list__item--fail .bk-upload-list__item-message) {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .env-var-action-icon {
    box-sizing: border-box;
    display: inline-block;
    width: 10px;
    height: 10px;
    margin-right: 4px;
    background-clip: padding-box;
    border: 2px solid;
    border-radius: 50%;
  }

  .env-var-action-icon-new {
    background-color: #2caf5e;
    border-color: rgb(44 175 94 / 20%);
  }

  .env-var-action-icon-overwrite {
    background-color: #f59500;
    border-color: rgb(245 149 0 / 20%);
  }
</style>
