/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

import { computed, ref, watch } from 'vue';

import { ImagesService } from '~/api/modules/v1/images';

import type { RuntimeImageOutputObj, RuntimeImageTagOutputObj } from '~/@types/v1/images';

export interface UsePlatformBuildImagesOptions {
  /** 当前语言，用于 API 过滤镜像列表 */
  getLanguage: () => string | undefined;
  /** 读取已存储的构建/运行镜像值（如 "golang:1.22"），用于新数据加载后的回填匹配 */
  getStoredImage: (type: 'builder' | 'runner') => string | undefined;
  /** 将选中结果写回 form data（如 "golang:1.22"） */
  setImageValue: (type: 'builder' | 'runner', value: string) => void;
}

export function usePlatformBuildImages(options: UsePlatformBuildImagesOptions) {
  const { getStoredImage, setImageValue, getLanguage } = options;

  // ========== State refs ==========
  const builderImages = ref<RuntimeImageOutputObj[]>([]);
  const runnerImages = ref<RuntimeImageOutputObj[]>([]);
  const selectedBuilderImageId = ref('');
  const selectedRunnerImageId = ref('');
  const builderImageTags = ref<RuntimeImageTagOutputObj[]>([]);
  const runnerImageTags = ref<RuntimeImageTagOutputObj[]>([]);
  const selectedBuilderTag = ref('');
  const selectedRunnerTag = ref('');
  const builderTagLoading = ref(false);
  const runnerTagLoading = ref(false);
  const staleBuilderWarning = ref(false);
  const staleRunnerWarning = ref(false);

  // ========== Utility functions ==========
  function extractVersion(tag: string): string {
    if (!tag) return '';
    const alpineMatch = tag.match(/alpine(\d+(?:\.\d+)*)/i);
    if (alpineMatch) return alpineMatch[1];
    const matches = tag.match(/(\d+(?:\.\d+)*)/g);
    if (matches && matches.length > 0) return matches[matches.length - 1];
    return tag;
  }

  function parseImageValue(fullValue: string, images: RuntimeImageOutputObj[]): { imageId: string; tag: string } {
    if (!fullValue || !images.length) return { imageId: '', tag: '' };
    const lastColon = fullValue.lastIndexOf(':');
    if (lastColon === -1) return { imageId: '', tag: fullValue };
    const name = fullValue.slice(0, lastColon);
    const tag = fullValue.slice(lastColon + 1);
    const image = images.find(i => i.name === name);
    return { imageId: image?.id ?? '', tag };
  }

  // ========== Computed ==========
  const imageVersionMismatchWarning = computed(() => {
    if (!selectedBuilderTag.value || !selectedRunnerTag.value) return null;
    // 仅当构建镜像 tag 包含 alpine 且运行镜像 name 包含 alpine 时，才检查版本一致性
    const builderHasAlpine = /alpine/i.test(selectedBuilderTag.value);
    const runnerImage = runnerImages.value.find(i => i.id === selectedRunnerImageId.value);
    const runnerHasAlpine = /alpine/i.test(runnerImage?.name ?? '');

    if (!builderHasAlpine || !runnerHasAlpine) return null;
    const bv = extractVersion(selectedBuilderTag.value);
    const rv = extractVersion(selectedRunnerTag.value);
    if (bv === rv) return null;
    return { builderVersion: bv, runnerVersion: rv };
  });

  // ========== API functions ==========
  async function fetchImageTags(type: 'builder' | 'runner', imageId: string, keyword?: string) {
    const loadingRef = type === 'builder' ? builderTagLoading : runnerTagLoading;
    const tagsRef = type === 'builder' ? builderImageTags : runnerImageTags;
    loadingRef.value = true;
    try {
      const res = await ImagesService.listPlatformBuildImageTags({
        imageID: imageId,
        ...(keyword ? { keyword } : {}),
        page: 1,
        pageSize: 100,
      });
      const data = res as { results?: RuntimeImageTagOutputObj[] };
      tagsRef.value = data.results ?? [];
      if (tagsRef.value.length > 0) {
        const selectedTagRef = type === 'builder' ? selectedBuilderTag : selectedRunnerTag;
        if (!selectedTagRef.value && tagsRef.value[0].tag) {
          selectedTagRef.value = tagsRef.value[0].tag;
        }
      }
    } catch (err) {
      console.error(`Failed to fetch ${type} image tags:`, err);
    } finally {
      loadingRef.value = false;
    }
  }

  async function fetchPlatformBuildImages(type: 'builder' | 'runner') {
    try {
      const language = getLanguage();
      const res = await ImagesService.listPlatformBuildImages({ type, ...(language ? { language } : {}) });
      const images = (res as { results?: RuntimeImageOutputObj[] }).results ?? [];

      const imagesRef = type === 'builder' ? builderImages : runnerImages;
      const selectedImageIdRef = type === 'builder' ? selectedBuilderImageId : selectedRunnerImageId;
      const selectedTagRef = type === 'builder' ? selectedBuilderTag : selectedRunnerTag;
      const staleWarningRef = type === 'builder' ? staleBuilderWarning : staleRunnerWarning;
      const imageTagsRef = type === 'builder' ? builderImageTags : runnerImageTags;

      imagesRef.value = images;

      // 回填已存值
      const existingValue = getStoredImage(type);
      if (existingValue) {
        const parsed = parseImageValue(existingValue, images);
        if (parsed.imageId) {
          staleWarningRef.value = false;
          selectedImageIdRef.value = parsed.imageId;
          await fetchImageTags(type, parsed.imageId);
          if (parsed.tag) {
            const tagExists = imageTagsRef.value.some(t => t.tag === parsed.tag);
            if (tagExists) {
              selectedTagRef.value = parsed.tag;
            } else {
              selectedTagRef.value = '';
              staleWarningRef.value = true;
            }
          }
        } else {
          staleWarningRef.value = true;
          selectedImageIdRef.value = '';
          selectedTagRef.value = '';
        }
      }

      // 无既有值时（或回填失败时）自动选中第一个镜像
      if (!selectedImageIdRef.value && images.length > 0 && images[0].id) {
        selectedImageIdRef.value = images[0].id;
        staleWarningRef.value = false;
        await fetchImageTags(type, images[0].id);
      }
    } catch (err) {
      console.error(`Failed to fetch ${type} images:`, err);
    }
  }

  // ========== Event handlers ==========
  function handleBuilderImageChange(imageId: string) {
    selectedBuilderImageId.value = imageId;
    selectedBuilderTag.value = '';
    builderImageTags.value = [];
    if (imageId) {
      fetchImageTags('builder', imageId);
    }
  }

  function handleRunnerImageChange(imageId: string) {
    selectedRunnerImageId.value = imageId;
    selectedRunnerTag.value = '';
    runnerImageTags.value = [];
    if (imageId) {
      fetchImageTags('runner', imageId);
    }
  }

  // ========== Reset ==========
  function resetState() {
    selectedBuilderImageId.value = '';
    selectedBuilderTag.value = '';
    selectedRunnerImageId.value = '';
    selectedRunnerTag.value = '';
    builderImageTags.value = [];
    runnerImageTags.value = [];
    staleBuilderWarning.value = false;
    staleRunnerWarning.value = false;
  }

  // ========== Sync watchers ==========
  watch([selectedBuilderImageId, selectedBuilderTag], ([imageId, tag]) => {
    if (imageId && tag) {
      const image = builderImages.value.find(i => i.id === imageId);
      if (image) {
        setImageValue('builder', `${image.name}:${tag}`);
      }
    } else {
      setImageValue('builder', '');
    }
  });

  watch([selectedRunnerImageId, selectedRunnerTag], ([imageId, tag]) => {
    if (imageId && tag) {
      const image = runnerImages.value.find(i => i.id === imageId);
      if (image) {
        setImageValue('runner', `${image.name}:${tag}`);
      }
    } else {
      setImageValue('runner', '');
    }
  });

  return {
    // State refs
    builderImages,
    runnerImages,
    selectedBuilderImageId,
    selectedRunnerImageId,
    builderImageTags,
    runnerImageTags,
    selectedBuilderTag,
    selectedRunnerTag,
    builderTagLoading,
    runnerTagLoading,
    staleBuilderWarning,
    staleRunnerWarning,
    // Computed
    imageVersionMismatchWarning,
    // API methods
    fetchImageTags,
    fetchPlatformBuildImages,
    // Event handlers
    handleBuilderImageChange,
    handleRunnerImageChange,
    // Reset
    resetState,
  };
}
