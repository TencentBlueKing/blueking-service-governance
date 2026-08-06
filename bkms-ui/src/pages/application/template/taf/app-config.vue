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
  <Form
    ref="formRef"
    class="flex flex-col p-[20px] !max-w-[1400px] w-full"
    :label-width="135"
    :model="formData"
  >
    <ToggleCard
      class="bg-[#fff] shadow-sm px-[16px]"
      :name="$t('应用配置')"
    >
      <Form.FormItem :label="$t('启动命令')">
        <RepeatableInput
          ref="commandRef"
          v-model="formData.command"
          :add-text="$t('添加启动命令')"
          class="max-w-[600px]"
          :required="formData.command.length > 0"
          trim-on-input
        />
      </Form.FormItem>
      <Form.FormItem :label="$t('命令参数')">
        <RepeatableInput
          ref="argsRef"
          v-model="formData.args"
          :add-text="$t('添加命令参数')"
          class="max-w-[600px]"
          :required="formData.args.length > 0"
          trim-on-input
        />
      </Form.FormItem>
      <Form.FormItem :label="$t('环境变量')">
        <KeyValue
          v-model="formData.envVars"
          class="max-w-[600px]"
          :key-placeholder="$t('请输入变量名')"
          textarea
          :value-placeholder="$t('请输入变量值')"
        />
      </Form.FormItem>
    </ToggleCard>
    <ToggleCard
      :class="['bg-[#fff] shadow-sm p-[16px] mt-[16px] mb-[20px] min-h-0', { 'flex-1 h-full': isYamlCardExpanded }]"
      content-class="h-full px-[24px] flex flex-col"
      :name="$t('框架配置')"
      @change="handleYamlCardChange"
    >
      <Form.FormItem
        class="mb-0"
        :label="$t('配置文件路径')"
        :label-width="110"
        property="tafSpec.filePath"
        required
      >
        <Input
          v-model.trim="formData.tafSpec.filePath"
          class="max-w-[600px]"
        />
      </Form.FormItem>
      <FlexRow>
        <template #right>
          <IconTextButton
            v-show="isYamlCardExpanded"
            :active="showEnvVar"
            class="text-[12px] p-[10px] transform-translate-y-[-50%]"
            icon="bkms-icon bkms-icon-variable"
            :text="$t('环境变量')"
            @click="toggleEnvVarShow()"
          />
        </template>
      </FlexRow>
      <ResizeLayout
        ref="resizeLayoutRef"
        :border="false"
        class="h-[60vh] min-h-0 yaml-sideslider-layout"
        initial-divide="50%"
        placement="right"
      >
        <template #aside>
          <!-- 默认环境变量 -->
          <ViewDefaultEnvVars
            app-type="taf"
            class="h-full ml-[16px]"
            :custom-request-fn="handleGetVarEnv"
            :env-list="envList"
          />
        </template>
        <template #main>
          <!-- 代码编辑器 -->
          <ResizeLayout
            ref="errorRef"
            :auto-minimize="true"
            :border="false"
            class="h-full"
            :disabled="!editorErr.message?.length"
            :max="300"
            :min="100"
            placement="bottom"
          >
            <template #aside>
              <EditorStatus
                v-show="!!editorErr.message?.length"
                :message="editorErr.message"
              />
            </template>
            <template #main>
              <MsEditor
                ref="msEditorRef"
                v-model="formData.tafSpec.fileContent"
                class="h-full"
                @error="handleEditorErr"
              >
                <template #title>
                  <div class="flex items-center">
                    <span class="text-[14px] text-[#C4C6CC]">{{ formData.tafSpec.fileName }}</span>
                    <Popover
                      v-model:is-show="isPopoverShow"
                      :padding="16"
                      placement="bottom"
                      theme="light"
                      trigger="manual"
                      :width="320"
                      @after-hidden="handlePopoverHidden"
                    >
                      <EditLine
                        class="ml-[8px] text-[#979BA5] cursor-pointer hover:text-[#3A84FF]"
                        @click="handleEdit"
                      />
                      <template #content>
                        <Form
                          ref="popoverFormRef"
                          class="mt-[16px]"
                          form-type="vertical"
                          :model="popoverFormData"
                          :rules="popoverFormRules"
                        >
                          <Form.FormItem
                            :label="$t('配置文件名称')"
                            :label-width="110"
                            property="fileName"
                            required
                          >
                            <Input
                              v-model.trim="popoverFormData.fileName"
                              :maxlength="100"
                              :placeholder="$t('请输入')"
                              @enter="handleConfirm"
                            />
                          </Form.FormItem>
                        </Form>
                        <div class="mt-[8px] flex justify-end gap-[8px]">
                          <Button
                            size="small"
                            @click="handleCancel"
                          >
                            {{ $t('取消') }}
                          </Button>
                          <Button
                            size="small"
                            theme="primary"
                            @click="handleConfirm"
                          >
                            {{ $t('确定') }}
                          </Button>
                        </div>
                      </template>
                    </Popover>
                  </div>
                </template>
              </MsEditor>
            </template>
          </ResizeLayout>
        </template>
      </ResizeLayout>
      <p
        v-show="isShowContentTip"
        class="text-[#EA3636]"
      >
        {{ $t('必填项') }}
      </p>
    </ToggleCard>
  </Form>
</template>
<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue';

  import { Button, Form, Input, Popover, ResizeLayout } from 'bkui-vue';
  import { EditLine } from 'bkui-vue/lib/icon';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import FlexRow from '~/components/flex-row.vue';
  import KeyValue from '~/components/key-value.vue';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import RepeatableInput from '~/components/repeatable-input.vue';
  import useEnvManager from '~/composables/use-env-manager';

  import type { AppModelSpecOutputObj } from '~/@types/v1/app';
  import type { IMonacoEditorErrorMarkerItem } from '~/common/util';

  interface IProps {
    value: TafAppModelSpecForm;
  }

  /** 表单场景下，tafSpec/command/args/envVars 是确定存在的，需要声明为必需 */
  type TafAppModelSpecForm = Omit<AppModelSpecOutputObj, 'args' | 'command' | 'envVars' | 'tafSpec' | 'trpcSpec'> & {
    args: string[];
    command: string[];
    envVars: Record<string, string>[];
    tafSpec: Omit<NonNullable<AppModelSpecOutputObj['tafSpec']>, 'fileName' | 'filePath'> & {
      fileName: string;
      filePath: string;
    };
  };

  const props = defineProps<IProps>();

  const { t } = useI18n();

  const { envList, handleGetEnvList } = useEnvManager();

  const formData = ref<TafAppModelSpecForm>(cloneDeep(props.value));

  // Popover 相关状态
  const isPopoverShow = ref(false);
  const popoverFormRef = ref<InstanceType<typeof Form> | null>(null);
  const popoverFormData = ref({
    fileName: '',
  });
  const popoverFormRules = {
    fileName: [
      {
        required: true,
        message: t('请输入配置文件名称'),
        trigger: 'blur',
      },
    ],
  };

  // 监听 Popover 显示状态，打开时初始化数据
  watch(isPopoverShow, newVal => {
    if (newVal) {
      popoverFormData.value.fileName = formData.value.tafSpec.fileName;
    }
  });

  function handleCancel() {
    isPopoverShow.value = false;
    popoverFormData.value.fileName = '';
    popoverFormRef.value?.clearValidate();
  }

  async function handleConfirm() {
    if (await popoverFormRef.value?.validate()) {
      formData.value.tafSpec.fileName = popoverFormData.value.fileName.trim();
      isPopoverShow.value = false;
      popoverFormData.value.fileName = '';
      popoverFormRef.value?.clearValidate();
    }
  }

  function handleEdit() {
    isPopoverShow.value = true;
  }

  function handlePopoverHidden() {
    popoverFormData.value.fileName = '';
    popoverFormRef.value?.clearValidate();
  }

  // yaml异常
  const editorErr = ref<{
    message: string[];
    type: string;
  }>({
    type: '',
    message: [],
  });
  const errorRef = ref<InstanceType<typeof ResizeLayout> | null>(null);
  function getValue() {
    formData.value.envVars = formData.value.envVars.map(item => ({
      key: item.key,
      value: item.value,
      description: '',
    }));
    return formData.value;
  }
  function handleEditorErr(err: IMonacoEditorErrorMarkerItem[]) {
    // 捕获编辑器错误提示
    editorErr.value.type = 'content'; // 编辑内容错误
    editorErr.value.message = err.map(item => item.message);
    hideOrShowError();
  }

  function hideOrShowError() {
    if (!editorErr.value?.message?.length && errorRef.value) {
      errorRef.value.asideRef.hidden = true;
    } else if (editorErr.value?.message?.length && errorRef.value) {
      errorRef.value.asideRef.hidden = false;
    }
  }

  function resetStatus() {
    editorErr.value = {
      type: '',
      message: [],
    };
    hideOrShowError();
  }
  const firstTrigger = ref(false);
  const isShowContentTip = computed(() => !formData.value.tafSpec.fileContent && firstTrigger.value);
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  const commandRef = ref<InstanceType<typeof RepeatableInput> | null>(null);
  const argsRef = ref<InstanceType<typeof RepeatableInput> | null>(null);
  async function validate() {
    firstTrigger.value = true;
    const [valid, commandValid, argsValid] = await Promise.all([
      formRef.value?.validate(),
      commandRef.value?.validate(),
      argsRef.value?.validate(),
    ]);

    if (
      !valid ||
      !commandValid ||
      !argsValid ||
      editorErr.value?.message?.length ||
      !formData.value.tafSpec.fileContent
    )
      return false;
    return true;
  }

  const showEnvVar = ref(false);
  const isYamlCardExpanded = ref(true);
  const resizeLayoutRef = ref<InstanceType<typeof ResizeLayout> | null>(null);

  // 获取应用环境变量
  function handleGetVarEnv(env: string) {
    const envID = envList.value.find(item => item.name === env)?.id;
    if (!envID) return Promise.resolve([]);
    return ApiServerService.ListEnvAvailableEnvVars({ envID });
  }

  // 处理yaml卡片展开/收起状态变化
  function handleYamlCardChange(expanded: boolean) {
    isYamlCardExpanded.value = expanded;
  }

  function setEnvVarAsideVisible(visible: boolean) {
    if (resizeLayoutRef.value?.asideRef) {
      resizeLayoutRef.value.asideRef.hidden = !visible;
    }
  }

  function toggleEnvVarShow() {
    showEnvVar.value = !showEnvVar.value;
    setEnvVarAsideVisible(showEnvVar.value);
  }

  // 初始化隐藏错误侧栏
  watch(
    [editorErr.value, errorRef],
    () => {
      hideOrShowError();
    },
    { immediate: true },
  );

  onMounted(async () => {
    await handleGetEnvList();
    setEnvVarAsideVisible(showEnvVar.value);
  });

  defineExpose({
    getValue,
    validate,
    resetStatus,
  });
</script>
