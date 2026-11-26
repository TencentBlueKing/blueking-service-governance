<template>
  <Alert theme="warning">
    <template #title>
      <div class="leading-[20px]">
        <template v-if="props.requiresBuilderImage">
          <i18n-t
            keypath="构建产物需放到产物目录（命令中可通过 {0} 引用），runner 阶段会将该目录下的全部内容拷贝到运行镜像的 {1} 目录，例如："
            tag="span"
          >
            <span class="font-mono font-bold">${BKMS_ARTIFACT_DIR}</span>
            <span class="font-mono font-bold">/app</span>
          </i18n-t>
          <div class="flex items-center justify-between bg-[#fff] mt-[8px] h-[52px] px-[12px]">
            <code class="font-mono">
              go build -o <span class="text-[#3A84FF]">{{ buildOutputPath }}</span> .
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
        </template>
        <i18n-t
          v-else
          keypath="源码会整体拷贝到产物目录并作为工作目录（命令中可通过 {0} 引用），构建结束时该目录不能为空。"
          tag="span"
        >
          <span class="font-mono font-bold">${BKMS_ARTIFACT_DIR}</span>
        </i18n-t>
      </div>
    </template>
  </Alert>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Alert, Button } from 'bkui-vue';
  import { Copy } from 'bkui-vue/lib/icon';
  import { useCopy } from '~/composables/use-copy';

  const props = withDefaults(
    defineProps<{
      appName?: string;
      requiresBuilderImage?: boolean;
    }>(),
    {
      appName: '',
      requiresBuilderImage: true,
    },
  );
  const buildOutputPath = computed(() => `\${BKMS_ARTIFACT_DIR}/${props.appName || '{{ appName }}'}`);
  const buildCommandExample = computed(() => `go build -o ${buildOutputPath.value} .`);
  const { copyText } = useCopy();
</script>
