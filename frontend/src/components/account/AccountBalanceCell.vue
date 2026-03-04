<template>
  <div v-if="isEligible" class="flex items-center gap-1.5">
    <!-- Cached balance display -->
    <template v-if="hasCachedBalance && !loading">
      <!-- Panel type badge -->
      <span
        v-if="cachedPanelType"
        class="inline-block rounded px-1 py-0.5 text-[9px] font-medium bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"
      >
        {{ cachedPanelType }}
      </span>

      <!-- Balance with tooltip showing details -->
      <span
        :class="balanceColorClass"
        class="group relative cursor-default text-xs font-mono font-medium"
      >
        ${{ formattedBalance }}
        <!-- Tooltip with details -->
        <span
          class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-1.5 -translate-x-1/2 whitespace-nowrap rounded bg-gray-900 px-2.5 py-1.5 text-[10px] font-normal text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 dark:bg-gray-700"
        >
          <div v-if="cachedTotalQuota" class="leading-relaxed">
            <div>{{ t('admin.accounts.balance.total') }}: ${{ cachedTotalQuota.toFixed(2) }}</div>
            <div>{{ t('admin.accounts.balance.used') }}: ${{ cachedUsedQuota.toFixed(2) }}</div>
            <div v-if="cachedUserRole && cachedUserRole !== '-'">{{ cachedUserRole }}</div>
          </div>
        </span>
      </span>

      <span v-if="cachedUpdatedAt" class="text-[10px] text-gray-400 dark:text-gray-500">
        {{ relativeTime }}
      </span>
    </template>

    <!-- Error state -->
    <span v-else-if="cachedError && !loading" class="text-[10px] text-red-500" :title="cachedError">
      {{ t('admin.accounts.balance.error') }}
    </span>

    <!-- No data yet -->
    <span v-else-if="!loading" class="text-xs text-gray-400 dark:text-gray-500">
      {{ t('admin.accounts.balance.noData') }}
    </span>

    <!-- Loading spinner -->
    <span v-if="loading" class="flex items-center gap-1 text-[10px] text-gray-400">
      <svg class="h-3 w-3 animate-spin" viewBox="0 0 24 24" fill="none">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </span>

    <!-- Refresh button -->
    <button
      v-if="!loading"
      @click.stop="fetchBalance"
      class="inline-flex items-center rounded p-0.5 text-gray-400 hover:text-primary-500 dark:text-gray-500 dark:hover:text-primary-400"
      :title="t('admin.accounts.balance.fetch')"
    >
      <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
    </button>
  </div>
  <div v-else class="text-xs text-gray-400 dark:text-gray-500">-</div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'

const props = defineProps<{
  account: Account
}>()

const emit = defineEmits<{
  (e: 'balance-updated'): void
}>()

const { t } = useI18n()
const loading = ref(false)

const isEligible = computed(() => {
  const acct = props.account
  if (acct.type !== 'apikey' && acct.type !== 'upstream') return false
  const creds = acct.credentials as Record<string, unknown> | undefined
  if (!creds) return false
  return !!creds.api_key && !!creds.base_url
})

const extra = computed(() => props.account.extra as Record<string, unknown> | undefined)

const hasCachedBalance = computed(() => {
  return extra.value?.upstream_balance_updated_at != null &&
    !extra.value?.upstream_balance_error
})

const cachedBalance = computed(() => {
  if (!extra.value) return null
  const val = extra.value.upstream_balance
  return typeof val === 'number' ? val : null
})

const cachedUsedQuota = computed(() => {
  if (!extra.value) return 0
  const val = extra.value.upstream_used_quota
  return typeof val === 'number' ? val : 0
})

const cachedTotalQuota = computed(() => {
  if (!extra.value) return 0
  const val = extra.value.upstream_total_quota
  return typeof val === 'number' ? val : 0
})

const cachedPanelType = computed(() => {
  if (!extra.value) return null
  const val = extra.value.upstream_balance_panel_type
  return typeof val === 'string' && val !== '' ? val : null
})

const cachedUserRole = computed(() => {
  if (!extra.value) return null
  const val = extra.value.upstream_balance_user_role
  return typeof val === 'string' && val !== '' ? val : null
})

const cachedError = computed(() => {
  if (!extra.value) return null
  const err = extra.value.upstream_balance_error
  return typeof err === 'string' && err !== '' ? err : null
})

const cachedUpdatedAt = computed(() => {
  if (!extra.value) return null
  const val = extra.value.upstream_balance_updated_at
  return typeof val === 'string' ? val : null
})

const formattedBalance = computed(() => {
  if (cachedBalance.value === null) return '0.00'
  return cachedBalance.value.toFixed(2)
})

const balanceColorClass = computed(() => {
  const bal = cachedBalance.value
  if (bal === null) return 'text-gray-500'
  if (bal > 10) return 'text-green-600 dark:text-green-400'
  if (bal >= 1) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-red-600 dark:text-red-400'
})

const relativeTime = computed(() => {
  if (!cachedUpdatedAt.value) return ''
  const updated = new Date(cachedUpdatedAt.value)
  const now = Date.now()
  const diffMs = now - updated.getTime()
  if (diffMs < 0) return ''

  const diffSec = Math.floor(diffMs / 1000)
  if (diffSec < 60) return t('admin.accounts.balance.lastUpdated', { time: `${diffSec}s` })
  const diffMin = Math.floor(diffSec / 60)
  if (diffMin < 60) return t('admin.accounts.balance.lastUpdated', { time: `${diffMin}m` })
  const diffHour = Math.floor(diffMin / 60)
  if (diffHour < 24) return t('admin.accounts.balance.lastUpdated', { time: `${diffHour}h` })
  const diffDay = Math.floor(diffHour / 24)
  return t('admin.accounts.balance.lastUpdated', { time: `${diffDay}d` })
})

const fetchBalance = async () => {
  loading.value = true
  try {
    const result = await adminAPI.accounts.getBalance(props.account.id)
    // Initialize extra if missing
    if (!props.account.extra) {
      (props.account as any).extra = {}
    }
    const ex = props.account.extra as Record<string, unknown>
    ex.upstream_balance = result.balance
    ex.upstream_used_quota = result.used_quota
    ex.upstream_total_quota = result.total_quota
    ex.upstream_balance_panel_type = result.panel_type
    ex.upstream_balance_user_role = result.user_role || ''
    ex.upstream_balance_user_name = result.user_name || ''
    ex.upstream_balance_user_email = result.user_email || ''
    ex.upstream_balance_updated_at = result.updated_at
    ex.upstream_balance_error = result.error || ''
    emit('balance-updated')
  } catch (e: any) {
    console.error('Failed to fetch balance:', e)
    if (!props.account.extra) {
      (props.account as any).extra = {}
    }
    const ex = props.account.extra as Record<string, unknown>
    ex.upstream_balance_error = e?.message || 'request failed'
    ex.upstream_balance_updated_at = new Date().toISOString()
  } finally {
    loading.value = false
  }
}
</script>
