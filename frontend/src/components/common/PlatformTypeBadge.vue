<template>
  <div class="inline-flex items-center overflow-hidden rounded-md text-xs font-medium">
    <!-- Platform part -->
    <span :class="['inline-flex items-center gap-1 px-2 py-1', platformClass]">
      <PlatformIcon :platform="platform" size="xs" />
      <span>{{ platformLabel }}</span>
    </span>
    <!-- Type part -->
    <span :class="['inline-flex items-center gap-1 px-1.5 py-1', typeClass]">
      <!-- OAuth icon -->
      <svg
        v-if="type === 'oauth'"
        class="h-3 w-3"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
        />
      </svg>
      <!-- Setup Token icon -->
      <Icon v-else-if="type === 'setup-token'" name="shield" size="xs" />
      <!-- API Key icon -->
      <Icon v-else name="key" size="xs" />
      <span>{{ typeLabel }}</span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AccountPlatform, AccountType } from '@/types'
import PlatformIcon from './PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  platform: AccountPlatform
  type: AccountType
}

const props = defineProps<Props>()

const platformLabel = computed(() => {
  if (props.platform === 'anthropic') return 'Anthropic'
  if (props.platform === 'openai') return 'OpenAI'
  if (props.platform === 'antigravity') return 'Antigravity'
  if (props.platform === 'sora') return 'Sora'
  if (props.platform === 'deepseek') return 'DeepSeek'
  if (props.platform === 'qwen') return 'Qwen'
  if (props.platform === 'glm') return 'GLM'
  if (props.platform === 'kimi') return 'Kimi'
  if (props.platform === 'iflow') return 'iFlow'
  if (props.platform === 'kiro') return 'Kiro'
  return 'Gemini'
})

const typeLabel = computed(() => {
  switch (props.type) {
    case 'oauth':
      return 'OAuth'
    case 'setup-token':
      return 'Token'
    case 'apikey':
      return 'Key'
    default:
      return props.type
  }
})

const platformClass = computed(() => {
  if (props.platform === 'anthropic') {
    return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
  }
  if (props.platform === 'openai') {
    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
  }
  if (props.platform === 'antigravity') {
    return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
  }
  if (props.platform === 'sora') {
    return 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-400'
  }
  if (props.platform === 'deepseek') {
    return 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400'
  }
  if (props.platform === 'qwen') {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  }
  if (props.platform === 'glm') {
    return 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-400'
  }
  if (props.platform === 'kimi') {
    return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400'
  }
  if (props.platform === 'iflow') {
    return 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400'
  }
  if (props.platform === 'kiro') {
    return 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-400'
  }
  return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
})

const typeClass = computed(() => {
  if (props.platform === 'anthropic') {
    return 'bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400'
  }
  if (props.platform === 'openai') {
    return 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400'
  }
  if (props.platform === 'antigravity') {
    return 'bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400'
  }
  if (props.platform === 'sora') {
    return 'bg-rose-100 text-rose-600 dark:bg-rose-900/30 dark:text-rose-400'
  }
  if (props.platform === 'deepseek') {
    return 'bg-sky-100 text-sky-600 dark:bg-sky-900/30 dark:text-sky-400'
  }
  if (props.platform === 'qwen') {
    return 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400'
  }
  if (props.platform === 'glm') {
    return 'bg-teal-100 text-teal-600 dark:bg-teal-900/30 dark:text-teal-400'
  }
  if (props.platform === 'kimi') {
    return 'bg-indigo-100 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400'
  }
  if (props.platform === 'iflow') {
    return 'bg-cyan-100 text-cyan-600 dark:bg-cyan-900/30 dark:text-cyan-400'
  }
  if (props.platform === 'kiro') {
    return 'bg-rose-100 text-rose-600 dark:bg-rose-900/30 dark:text-rose-400'
  }
  return 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
})
</script>
