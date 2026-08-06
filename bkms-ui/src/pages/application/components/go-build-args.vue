<template>
  <KeyValue
    v-model="argsModel"
    :value-placeholder="goProxyValuePlaceholders"
  />
  <div
    v-if="language === 'go' && golangProxyUrl"
    class="flex items-center justify-between gap-[24px] bg-[#f5f7fa] mt-[16px] p-[16px]"
  >
    <div class="flex flex-wrap gap-x-[4px] text-[13px] leading-[22px]">
      <i18n-t keypath="若 go.mod 有使用工蜂私有仓库依赖包，构建参数中需要添加 {0}，取值见：">
        <span class="font-mono font-bold">GOPROXY、GOPRIVATE、GOSUMDB</span>
      </i18n-t>
      <a
        v-if="golangProxyUrl"
        class="text-[#3a84ff]"
        :href="golangProxyUrl"
        rel="noopener noreferrer"
        target="_blank"
      >
        Goproxy for Tencent
        <Share class="ml-[4px]" />
      </a>
    </div>
    <Button
      class="shrink-0"
      text
      theme="primary"
      @click="handleAddGoProxyArgs"
    >
      <span class="bkms-icon bkms-icon-plus-circle-shape text-[14px]"></span>
      <span class="ml-[6px]">{{ $t('一键添加') }}</span>
    </Button>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Button } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import KeyValue from '~/components/key-value.vue';

  defineProps<{
    language?: string;
  }>();

  const modelValue = defineModel<Record<string, string>>();
  const { t } = useI18n();
  const golangProxyUrl = import.meta.env.BK_GOLANG_PROXY_URL;
  const goProxyArgKeys = ['GOPROXY', 'GOPRIVATE', 'GOSUMDB'];
  const goProxyValuePlaceholders = computed(() => ({
    GOPROXY: '如 https://<用户名>:<凭证>@<代理地址>,direct',
    GOPRIVATE: t('一般可留空'),
    GOSUMDB: '如 <校验库地址>+<公钥>',
  }));
  const argsModel = computed({
    get: () => modelValue.value ?? {},
    set: value => {
      modelValue.value = value;
    },
  });

  function handleAddGoProxyArgs() {
    const args = { ...argsModel.value };
    goProxyArgKeys.forEach(key => {
      if (!(key in args)) {
        args[key] = '';
      }
    });
    argsModel.value = args;
  }
</script>
