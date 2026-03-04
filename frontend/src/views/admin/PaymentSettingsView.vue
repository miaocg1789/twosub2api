<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <!-- Payment Settings Form -->
      <form v-else @submit.prevent="saveSettings" class="space-y-6">
        <!-- Enable Payment -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.payment.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.payment.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.payment.enabled') }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.enabledHint') }}</p>
              </div>
              <Toggle v-model="form.payment_enabled" />
            </div>
          </div>
        </div>

        <!-- Payment Config - Only show when enabled -->
        <template v-if="form.payment_enabled">
          <!-- General Settings -->
          <div class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('admin.settings.payment.generalTitle') }}
              </h2>
            </div>
            <div class="space-y-5 p-6">
              <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.currency') }}</label>
                  <select v-model="form.payment_currency" class="input">
                    <option value="CNY">CNY</option>
                    <option value="USD">USD</option>
                  </select>
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.currencyHint') }}</p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.exchangeRate') }}</label>
                  <input v-model.number="form.payment_exchange_rate" type="number" step="0.0001" min="0" class="input" placeholder="1.0" />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.exchangeRateHint') }}</p>
                </div>
              </div>

              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.presetAmounts') }}</label>
                <input v-model="form.payment_preset_amounts" type="text" class="input" placeholder="10,20,50,100" />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.presetAmountsHint') }}</p>
              </div>

              <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.minAmount') }}</label>
                  <input v-model.number="form.payment_min_amount" type="number" step="0.01" min="0" class="input" placeholder="1" />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.minAmountHint') }}</p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.maxAmount') }}</label>
                  <input v-model.number="form.payment_max_amount" type="number" step="0.01" min="0" class="input" placeholder="10000" />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.maxAmountHint') }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- Alipay Section -->
          <div class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.payment.alipayTitle') }}</h2>
            </div>
            <div class="space-y-4 p-6">
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.alipayMode') }}</label>
                <select v-model="alipayMode" class="input">
                  <option value="disabled">{{ t('admin.settings.payment.alipayModeDisabled') }}</option>
                  <option value="regular">{{ t('admin.settings.payment.alipayModeRegular') }}</option>
                  <option value="f2f">{{ t('admin.settings.payment.alipayModeF2f') }}</option>
                </select>
              </div>
              <div v-if="alipayMode !== 'disabled'" class="space-y-4">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.alipayAppId') }}</label>
                  <input v-model="form.payment_alipay_app_id" type="text" class="input font-mono text-sm" />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.alipayAppIdHint') }}</p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.alipayPrivateKey') }}</label>
                  <input v-model="form.payment_alipay_private_key" type="password" class="input font-mono text-sm" :placeholder="form.payment_alipay_private_key_configured ? t('admin.settings.payment.alipayPrivateKeyHint') : t('admin.settings.payment.alipayPrivateKeyPlaceholder')" />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.alipayPublicKey') }}</label>
                  <input v-model="form.payment_alipay_public_key" type="password" class="input font-mono text-sm" :placeholder="form.payment_alipay_public_key_configured ? t('admin.settings.payment.alipayPublicKeyHint') : t('admin.settings.payment.alipayPublicKeyPlaceholder')" />
                </div>
              </div>
            </div>
          </div>

          <!-- WeChat Pay Section -->
          <div class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.payment.wechatTitle') }}</h2>
            </div>
            <div class="space-y-4 p-6">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.payment.wechatEnabled') }}</label>
                </div>
                <Toggle v-model="form.payment_wechat_enabled" />
              </div>
              <div v-if="form.payment_wechat_enabled" class="space-y-4">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.wechatAppId') }}</label>
                  <input v-model="form.payment_wechat_app_id" type="text" class="input font-mono text-sm" />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.wechatAppIdHint') }}</p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.wechatMchId') }}</label>
                  <input v-model="form.payment_wechat_mch_id" type="text" class="input font-mono text-sm" />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.wechatMchIdHint') }}</p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.wechatApiKey') }}</label>
                  <input v-model="form.payment_wechat_api_key" type="password" class="input font-mono text-sm" :placeholder="form.payment_wechat_api_key_configured ? t('admin.settings.payment.wechatApiKeyHint') : t('admin.settings.payment.wechatApiKeyPlaceholder')" />
                </div>
              </div>
            </div>
          </div>

          <!-- EasyPay Section -->
          <div class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.payment.epayTitle') }}</h2>
            </div>
            <div class="space-y-4 p-6">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.payment.epayEnabled') }}</label>
                </div>
                <Toggle v-model="form.payment_epay_enabled" />
              </div>
              <div v-if="form.payment_epay_enabled" class="space-y-4">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.epayType') }}</label>
                  <select v-model="form.payment_epay_type" class="input">
                    <option value="alipay">{{ t('admin.settings.payment.epayTypeAlipay') }}</option>
                    <option value="wxpay">{{ t('admin.settings.payment.epayTypeWechat') }}</option>
                    <option value="both">{{ t('admin.settings.payment.epayTypeBoth') }}</option>
                  </select>
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.epayTypeHint') }}</p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.epayApiUrl') }}</label>
                  <input v-model="form.payment_epay_api_url" type="url" class="input font-mono text-sm" :placeholder="t('admin.settings.payment.epayApiUrlPlaceholder')" />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.epayApiUrlHint') }}</p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.epayPid') }}</label>
                  <input v-model="form.payment_epay_pid" type="text" class="input font-mono text-sm" />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.payment.epayPidHint') }}</p>
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.payment.epayKey') }}</label>
                  <input v-model="form.payment_epay_key" type="password" class="input font-mono text-sm" :placeholder="form.payment_epay_key_configured ? t('admin.settings.payment.epayKeyHint') : t('admin.settings.payment.epayKeyPlaceholder')" />
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- Referral Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('referral.title') }}
            </h2>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="font-medium text-gray-900 dark:text-white">{{ t('referral.enabled') }}</label>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('referral.enabledHint') }}</p>
              </div>
              <Toggle v-model="form.referral_enabled" />
            </div>
            <div v-if="form.referral_enabled">
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('referral.commissionRateLabel') }} (%)</label>
              <input
                v-model.number="form.referral_commission_rate"
                type="number"
                step="0.1"
                min="0"
                max="100"
                class="input w-48"
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('referral.commissionRateHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Save Button -->
        <div class="flex justify-end">
          <button type="submit" :disabled="saving" class="btn btn-primary px-8 py-2.5">
            <svg
              v-if="saving"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import type { UpdateSettingsRequest } from '@/api/admin/settings'
import AppLayout from '@/components/layout/AppLayout.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const saving = ref(false)

// 支付宝模式：将两个布尔值映射为单个下拉选项
const alipayMode = computed({
  get() {
    if (form.payment_alipay_f2f_enabled) return 'f2f'
    if (form.payment_alipay_enabled) return 'regular'
    return 'disabled'
  },
  set(val: string) {
    form.payment_alipay_enabled = val === 'regular'
    form.payment_alipay_f2f_enabled = val === 'f2f'
  }
})

const form = reactive({
  payment_enabled: false,
  payment_currency: 'CNY',
  payment_exchange_rate: 1,
  payment_preset_amounts: '10,20,50,100',
  payment_min_amount: 1,
  payment_max_amount: 10000,
  payment_alipay_enabled: false,
  payment_alipay_app_id: '',
  payment_alipay_private_key: '',
  payment_alipay_private_key_configured: false,
  payment_alipay_public_key: '',
  payment_alipay_public_key_configured: false,
  payment_alipay_f2f_enabled: false,
  payment_wechat_enabled: false,
  payment_wechat_app_id: '',
  payment_wechat_mch_id: '',
  payment_wechat_api_key: '',
  payment_wechat_api_key_configured: false,
  payment_epay_enabled: false,
  payment_epay_type: 'alipay',
  payment_epay_api_url: '',
  payment_epay_pid: '',
  payment_epay_key: '',
  payment_epay_key_configured: false,
  referral_enabled: false,
  referral_commission_rate: 0
})

async function loadSettings() {
  loading.value = true
  try {
    const settings = await adminAPI.settings.getSettings()
    // Only assign payment-related fields
    form.payment_enabled = settings.payment_enabled ?? false
    form.payment_currency = settings.payment_currency ?? 'CNY'
    form.payment_exchange_rate = settings.payment_exchange_rate ?? 1
    form.payment_preset_amounts = settings.payment_preset_amounts ?? '10,20,50,100'
    form.payment_min_amount = settings.payment_min_amount ?? 1
    form.payment_max_amount = settings.payment_max_amount ?? 10000
    form.payment_alipay_enabled = settings.payment_alipay_enabled ?? false
    form.payment_alipay_app_id = settings.payment_alipay_app_id ?? ''
    form.payment_alipay_private_key_configured = settings.payment_alipay_private_key_configured ?? false
    form.payment_alipay_public_key_configured = settings.payment_alipay_public_key_configured ?? false
    form.payment_alipay_f2f_enabled = settings.payment_alipay_f2f_enabled ?? false
    form.payment_wechat_enabled = settings.payment_wechat_enabled ?? false
    form.payment_wechat_app_id = settings.payment_wechat_app_id ?? ''
    form.payment_wechat_mch_id = settings.payment_wechat_mch_id ?? ''
    form.payment_wechat_api_key_configured = settings.payment_wechat_api_key_configured ?? false
    form.payment_epay_enabled = settings.payment_epay_enabled ?? false
    form.payment_epay_type = settings.payment_epay_type ?? 'alipay'
    form.payment_epay_api_url = settings.payment_epay_api_url ?? ''
    form.payment_epay_pid = settings.payment_epay_pid ?? ''
    form.payment_epay_key_configured = settings.payment_epay_key_configured ?? false
    form.referral_enabled = (settings as any).referral_enabled ?? false
    form.referral_commission_rate = ((settings as any).referral_commission_rate ?? 0) * 100
    // Clear sensitive fields
    form.payment_alipay_private_key = ''
    form.payment_alipay_public_key = ''
    form.payment_wechat_api_key = ''
    form.payment_epay_key = ''
  } catch (error: any) {
    appStore.showError(
      t('admin.settings.failedToLoad') + ': ' + (error.message || t('common.unknownError'))
    )
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  try {
    const payload: UpdateSettingsRequest = {
      payment_enabled: form.payment_enabled,
      payment_currency: form.payment_currency,
      payment_exchange_rate: form.payment_exchange_rate,
      payment_preset_amounts: form.payment_preset_amounts,
      payment_min_amount: form.payment_min_amount,
      payment_max_amount: form.payment_max_amount,
      payment_alipay_enabled: form.payment_alipay_enabled,
      payment_alipay_app_id: form.payment_alipay_app_id,
      payment_alipay_private_key: form.payment_alipay_private_key || undefined,
      payment_alipay_public_key: form.payment_alipay_public_key || undefined,
      payment_alipay_f2f_enabled: form.payment_alipay_f2f_enabled,
      payment_wechat_enabled: form.payment_wechat_enabled,
      payment_wechat_app_id: form.payment_wechat_app_id,
      payment_wechat_mch_id: form.payment_wechat_mch_id,
      payment_wechat_api_key: form.payment_wechat_api_key || undefined,
      payment_epay_enabled: form.payment_epay_enabled,
      payment_epay_type: form.payment_epay_type,
      payment_epay_api_url: form.payment_epay_api_url,
      payment_epay_pid: form.payment_epay_pid,
      payment_epay_key: form.payment_epay_key || undefined,
      referral_enabled: form.referral_enabled,
      referral_commission_rate: form.referral_commission_rate / 100
    }
    const updated = await adminAPI.settings.updateSettings(payload)
    // Refresh configured flags
    form.payment_alipay_private_key_configured = updated.payment_alipay_private_key_configured ?? form.payment_alipay_private_key_configured
    form.payment_alipay_public_key_configured = updated.payment_alipay_public_key_configured ?? form.payment_alipay_public_key_configured
    form.payment_wechat_api_key_configured = updated.payment_wechat_api_key_configured ?? form.payment_wechat_api_key_configured
    form.payment_epay_key_configured = updated.payment_epay_key_configured ?? form.payment_epay_key_configured
    // Clear sensitive fields
    form.payment_alipay_private_key = ''
    form.payment_alipay_public_key = ''
    form.payment_wechat_api_key = ''
    form.payment_epay_key = ''
    // Refresh cached public settings
    await appStore.fetchPublicSettings(true)
    appStore.showSuccess(t('admin.settings.settingsSaved'))
  } catch (error: any) {
    appStore.showError(
      t('admin.settings.failedToSave') + ': ' + (error.message || t('common.unknownError'))
    )
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>
