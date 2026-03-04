import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { IFlowTokenInfo } from '@/api/admin/iflow'

export function useIFlowOAuth() {
  const appStore = useAppStore()
  const { t } = useI18n()

  const loading = ref(false)
  const error = ref('')

  const resetState = () => {
    loading.value = false
    error.value = ''
  }

  const authenticateWithCookie = async (
    cookie: string,
    proxyId?: number | null
  ): Promise<IFlowTokenInfo | null> => {
    if (!cookie.trim()) {
      error.value = t('admin.accounts.oauth.iflow.pleaseEnterCookie')
      return null
    }

    loading.value = true
    error.value = ''

    try {
      const tokenInfo = await adminAPI.iflow.authenticateWithCookie(cookie.trim(), proxyId)
      return tokenInfo
    } catch (err: any) {
      error.value =
        err.response?.data?.detail || t('admin.accounts.oauth.iflow.failedToAuth')
      appStore.showError(error.value)
      return null
    } finally {
      loading.value = false
    }
  }

  const buildCredentials = (tokenInfo: IFlowTokenInfo): Record<string, unknown> => {
    return {
      api_key: tokenInfo.api_key,
      cookie: tokenInfo.cookie,
      expires_at: tokenInfo.expires_at,
      email: tokenInfo.email
    }
  }

  return {
    loading,
    error,
    resetState,
    authenticateWithCookie,
    buildCredentials
  }
}
