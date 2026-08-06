<template>
  <Alert theme="warning">
    <template #title>
      <div class="leading-[20px]">
        <i18n-t
          keypath="构建产物需放到 {0} 下，runner 阶段仅从此路径拷贝进运行镜像。不限定命令形式（{1} 等均可），只要最终产物落到该路径即可，例如："
          tag="span"
        >
          <span class="font-mono font-bold">
            {{ buildOutputPath }}
          </span>
          <span class="font-mono font-bold">go build -o、cp</span>
        </i18n-t>
        <div class="flex items-center justify-between bg-[#fff] mt-[8px] h-[52px] px-[12px]">
          <code class="font-mono">
            go build -o <span class="text-[#3A84FF]">{{ buildOutputPath }}</span>
          </code>
          <Button
            text
            theme="primary"
            @click="copyText(buildCommandExample)"
          >
            <Copy class="mr-[4px]" />
            {{ $t('复制') }}
          </Button>
        </div>
      </div>
    </template>
  </Alert>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Alert, Button } from 'bkui-vue';
  import { Copy } from 'bkui-vue/lib/icon';
  import { useCopy } from '~/composables/use-copy';

  const props = defineProps<{
    appName?: string;
  }>();
  const buildOutputPath = computed(() => `/out/${props.appName || '{{ appName }}'}`);
  const buildCommandExample = computed(() => `go build -o ${buildOutputPath.value}`);
  const { copyText } = useCopy();
</script>
