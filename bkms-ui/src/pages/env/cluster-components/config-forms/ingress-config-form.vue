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
    form-type="vertical"
    :model="innerFormData"
  >
    <div class="flex items-start gap-[16px]">
      <Form.FormItem
        class="flex-1"
        :label="$t('命名空间')"
        property="namespace"
      >
        <Input
          v-model="innerFormData.namespace"
          disabled
          placeholder="bcs-system"
        />
      </Form.FormItem>

      <Form.FormItem
        class="flex-1"
        :label="$t('Chart 版本')"
        property="chartVersion"
        required
      >
        <Select
          v-model="innerFormData.chartVersion"
          filterable
          :list="chartVersionOptions"
          :placeholder="$t('请选择Chart版本')"
        />
      </Form.FormItem>
    </div>

    <div class="flex items-start gap-[16px]">
      <Form.FormItem
        class="flex-1"
        :label="$t('CLB 区域')"
        property="tencentcloudRegion"
        required
      >
        <Select
          v-model="innerFormData.tencentcloudRegion"
          filterable
          :list="regionOptions"
          :placeholder="$t('请选择CLB区域')"
        />
        <p class="text-[#979BA5] text-[12px] mt-[4px]">tencentcloudRegion，{{ $t('默认 CLB 区域') }}</p>
      </Form.FormItem>

      <Form.FormItem
        class="flex-1"
        :label="`${$t('腾讯云')} SecretID`"
        property="secretId"
        required
      >
        <Input
          v-model.trim="innerFormData.secretId"
          :placeholder="$t('请输入腾讯云 SecretID')"
        />
        <p class="text-[#979BA5] text-[12px] mt-[4px]">tencentcloudSecretID</p>
      </Form.FormItem>
    </div>

    <Form.FormItem
      :label="`${t('腾讯云')} SecretKey`"
      property="secretKey"
      required
    >
      <Input
        v-model.trim="innerFormData.secretKey"
        :placeholder="$t('请输入腾讯云 SecretKey')"
        type="password"
      />
      <p class="text-[#979BA5] text-[12px] mt-[4px]">tencentcloudSecretKey</p>
    </Form.FormItem>
  </Form>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue';

  import { Form, Input, Select } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ClusterAddonInfoOutput } from '~/@types/v1/cluster-addon';
  import { parseYamlValues } from '~/common/util';

  interface Emits {
    (e: 'update:formData', value: FormDataOutput): void;
  }

  /** 对外输出的接口数据结构 */
  interface FormDataOutput {
    chartVersion: string;
    namespace: string;
    values: {
      tencentcloudRegion: string;
      tencentcloudSecretID: string;
      tencentcloudSecretKey: string;
    };
  }

  interface InnerFormDataModel {
    chartVersion: string;
    namespace: string;
    secretId: string;
    secretKey: string;
    tencentcloudRegion: string;
  }

  interface Props {
    addonInfo?: ClusterAddonInfoOutput | null;
    formData?: FormDataOutput | null;
    isUpdate?: boolean;
  }

  const INITIAL_FORM_DATA: InnerFormDataModel = {
    chartVersion: '',
    namespace: 'bcs-system',
    secretId: '',
    secretKey: '',
    tencentcloudRegion: '',
  };

  const REGION_OPTIONS = [
    { value: 'ap-guangzhou', labelKey: '华南地区(广州)' },
    { value: 'ap-shenzhen-fsi', labelKey: '华南地区(深圳金融)' },
    { value: 'ap-guangzhou-open', labelKey: '华南地区(广州OPEN)' },
    { value: 'ap-shenzhen', labelKey: '华南地区(深圳)' },
    { value: 'ap-guangzhou-wxzf', labelKey: '华南地区(广州微信支付)' },
    { value: 'ap-shanghai', labelKey: '华东地区(上海)' },
    { value: 'ap-shanghai-fsi', labelKey: '华东地区(上海金融)' },
    { value: 'ap-jinan-ec', labelKey: '华东地区(济南)' },
    { value: 'ap-hangzhou-ec', labelKey: '华东地区(杭州)' },
    { value: 'ap-nanjing', labelKey: '华东地区(南京)' },
    { value: 'ap-fuzhou-ec', labelKey: '华东地区(福州)' },
    { value: 'ap-hefei-ec', labelKey: '华东地区(合肥)' },
    { value: 'ap-shanghai-wxzf', labelKey: '华东地区(上海微信支付)' },
    { value: 'ap-beijing', labelKey: '华北地区(北京)' },
    { value: 'ap-tianjin', labelKey: '华北地区(天津)' },
    { value: 'ap-beijing-fsi', labelKey: '华北地区(北京金融)' },
    { value: 'ap-shijiazhuang-ec', labelKey: '华北地区(石家庄)' },
    { value: 'ap-wuhan-ec', labelKey: '华中地区(武汉)' },
    { value: 'ap-changsha-ec', labelKey: '华中地区(长沙)' },
    { value: 'ap-zhengzhou-ec', labelKey: '华中地区(郑州)' },
    { value: 'ap-chengdu', labelKey: '西南地区(成都)' },
    { value: 'ap-chongqing', labelKey: '西南地区(重庆)' },
    { value: 'ap-xian-ec', labelKey: '西北地区(西安)' },
    { value: 'ap-shenyang-ec', labelKey: '东北地区(沈阳)' },
    { value: 'ap-hongkong', labelKey: '港澳台地区(中国香港)' },
    { value: 'ap-seoul', labelKey: '亚太东北(首尔)' },
    { value: 'ap-tokyo', labelKey: '亚太东北(东京)' },
    { value: 'ap-singapore', labelKey: '亚太东南(新加坡)' },
    { value: 'ap-bangkok', labelKey: '亚太东南(曼谷)' },
    { value: 'ap-jakarta', labelKey: '亚太东南(雅加达)' },
    { value: 'na-siliconvalley', labelKey: '美国西部(硅谷)' },
    { value: 'eu-frankfurt', labelKey: '欧洲地区(法兰克福)' },
    { value: 'na-ashburn', labelKey: '美国东部(弗吉尼亚)' },
    { value: 'sa-saopaulo', labelKey: '南美地区(圣保罗)' },
    { value: 'me-saudi-arabia', labelKey: '中东地区(利雅得)' },
  ] as const;

  const props = withDefaults(defineProps<Props>(), {
    addonInfo: null,
    formData: null,
    isUpdate: false,
  });

  const emit = defineEmits<Emits>();

  defineExpose({
    validate,
  });

  const { t } = useI18n();

  const formRef = ref();

  const innerFormData = reactive<InnerFormDataModel>({ ...INITIAL_FORM_DATA });

  /** Chart 版本选项 */
  const chartVersionOptions = computed(() => {
    const versions = props.addonInfo?.chartInfo?.availableVersions || [];
    return versions.map(v => ({ label: v, value: v }));
  });

  /** CLB 区域选项（label 走 i18n） */
  const regionOptions = computed(() => REGION_OPTIONS.map(r => ({ value: r.value, label: t(r.labelKey) })));

  /** 监听 addonInfo / isUpdate 变化，统一回填表单 */
  watch(
    [() => props.addonInfo, () => props.isUpdate],
    ([addonInfo, isUpdate]) => {
      if (!addonInfo) {
        applyFormData(INITIAL_FORM_DATA);
        return;
      }
      applyFormData(isUpdate ? buildUpdateFormData(addonInfo) : buildInstallFormData(addonInfo));
    },
    { immediate: true },
  );

  // 监听内部表单变化，向外同步
  watch(innerFormData, () => {
    emitFormData();
  });

  /** 统一写入表单数据 */
  function applyFormData(values: InnerFormDataModel) {
    Object.assign(innerFormData, values);
    emitFormData();
  }

  /** 安装模式：构建表单回填数据 */
  function buildInstallFormData(addonInfo: ClusterAddonInfoOutput): InnerFormDataModel {
    return {
      ...INITIAL_FORM_DATA,
      chartVersion: addonInfo.chartInfo?.defaultChartVersion || '',
    };
  }

  /** 更新模式：构建表单回填数据 */
  function buildUpdateFormData(addonInfo: ClusterAddonInfoOutput): InnerFormDataModel {
    const currentValues = parseYamlValues(addonInfo.installInfo?.currentValues);
    return {
      ...INITIAL_FORM_DATA,
      chartVersion: addonInfo.installInfo?.currentChartVersion || '',
      namespace: 'bcs-system',
      secretId: decodeBase64(String(currentValues.tencentcloudSecretID ?? '')),
      secretKey: decodeBase64(String(currentValues.tencentcloudSecretKey ?? '')),
      tencentcloudRegion: String(currentValues.tencentcloudRegion ?? ''),
    };
  }

  /** Base64 解码（回填时使用） */
  function decodeBase64(str: string): string {
    if (!str) return '';
    try {
      return atob(str);
    } catch {
      return str;
    }
  }

  /** 向外同步表单数据（转换为接口数据结构） */
  function emitFormData() {
    emit('update:formData', {
      chartVersion: innerFormData.chartVersion,
      namespace: innerFormData.namespace,
      values: {
        tencentcloudRegion: innerFormData.tencentcloudRegion,
        tencentcloudSecretID: encodeBase64(innerFormData.secretId),
        tencentcloudSecretKey: encodeBase64(innerFormData.secretKey),
      },
    });
  }

  /** Base64 编码（提交时使用） */
  function encodeBase64(str: string): string {
    if (!str) return '';
    try {
      return btoa(str);
    } catch {
      return str;
    }
  }

  /** 校验表单 */
  async function validate(): Promise<void> {
    await formRef.value?.validate?.();
  }
</script>
