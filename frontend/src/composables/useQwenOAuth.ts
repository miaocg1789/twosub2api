import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { QwenTokenInfo, QwenDeviceFlowResponse } from '@/api/admin/qwen'

export type QwenDeviceFlowStatus = 'idle' | 'initiated' | 'polling' | 'success' | 'error' | 'expired'

export function useQwenOAuth() {
  const appStore = useAppStore()
  const { t } = useI18n()

  const loading = ref(false)
  const error = ref('')
  const deviceFlowInfo = ref<QwenDeviceFlowResponse | null>(null)
  const deviceFlowStatus = ref<QwenDeviceFlowStatus>('idle')
  const sessionId = ref('')

  const resetState = () => {
    loading.value = false
    error.value = ''
    deviceFlowInfo.value = null
    deviceFlowStatus.value = 'idle'
    sessionId.value = ''
  }

  const initiateDeviceFlow = async (proxyId?: number | null): Promise<boolean> => {
    loading.value = true
    error.value = ''
    deviceFlowInfo.value = null
    deviceFlowStatus.value = 'idle'

    try {
      const payload: Record<string, unknown> = {}
      if (proxyId) payload.proxy_id = proxyId

      const response = await adminAPI.qwen.initiateDeviceFlow(payload as any)
      deviceFlowInfo.value = response
      sessionId.value = response.session_id
      deviceFlowStatus.value = 'initiated'
      return true
    } catch (err: any) {
      error.value =
        err.response?.data?.detail || t('admin.accounts.oauth.qwen.failedToInitiateFlow')
      appStore.showError(error.value)
      deviceFlowStatus.value = 'error'
      return false
    } finally {
      loading.value = false
    }
  }

  const pollForToken = async (): Promise<QwenTokenInfo | null> => {
    if (!sessionId.value) {
      error.value = t('admin.accounts.oauth.qwen.noActiveSession')
      return null
    }

    loading.value = true
    error.value = ''
    deviceFlowStatus.value = 'polling'

    try {
      const tokenInfo = await adminAPI.qwen.pollToken(sessionId.value)
      deviceFlowStatus.value = 'success'
      return tokenInfo
    } catch (err: any) {
      const detail = err.response?.data?.detail || ''
      if (detail.includes('过期') || detail.includes('expired')) {
        deviceFlowStatus.value = 'expired'
        error.value = t('admin.accounts.oauth.qwen.deviceCodeExpired')
      } else {
        deviceFlowStatus.value = 'error'
        error.value = detail || t('admin.accounts.oauth.qwen.pollFailed')
      }
      return null
    } finally {
      loading.value = false
    }
  }

  const validateRefreshToken = async (
    refreshToken: string,
    proxyId?: number | null
  ): Promise<QwenTokenInfo | null> => {
    if (!refreshToken.trim()) {
      error.value = t('admin.accounts.oauth.qwen.pleaseEnterRefreshToken')
      return null
    }

    loading.value = true
    error.value = ''

    try {
      const tokenInfo = await adminAPI.qwen.refreshQwenToken(refreshToken.trim(), proxyId)
      return tokenInfo
    } catch (err: any) {
      error.value =
        err.response?.data?.detail || t('admin.accounts.oauth.qwen.failedToValidateRT')
      return null
    } finally {
      loading.value = false
    }
  }

  const buildCredentials = (tokenInfo: QwenTokenInfo): Record<string, unknown> => {
    let expiresAt: string | undefined
    if (typeof tokenInfo.expires_at === 'number' && Number.isFinite(tokenInfo.expires_at)) {
      expiresAt = Math.floor(tokenInfo.expires_at).toString()
    } else if (typeof tokenInfo.expires_at === 'string' && tokenInfo.expires_at.trim()) {
      expiresAt = tokenInfo.expires_at.trim()
    }

    return {
      access_token: tokenInfo.access_token,
      refresh_token: tokenInfo.refresh_token,
      token_type: tokenInfo.token_type,
      expires_at: expiresAt,
      resource_url: tokenInfo.resource_url
    }
  }

  return {
    loading,
    error,
    deviceFlowInfo,
    deviceFlowStatus,
    sessionId,
    resetState,
    initiateDeviceFlow,
    pollForToken,
    validateRefreshToken,
    buildCredentials
  }
}
