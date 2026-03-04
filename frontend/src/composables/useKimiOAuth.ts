import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { KimiTokenInfo, KimiDeviceFlowResponse } from '@/api/admin/kimi'

export type KimiDeviceFlowStatus = 'idle' | 'initiated' | 'polling' | 'success' | 'error' | 'expired'

export function useKimiOAuth() {
  const appStore = useAppStore()
  const { t } = useI18n()

  const loading = ref(false)
  const error = ref('')
  const deviceFlowInfo = ref<KimiDeviceFlowResponse | null>(null)
  const deviceFlowStatus = ref<KimiDeviceFlowStatus>('idle')
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

      const response = await adminAPI.kimi.initiateDeviceFlow(payload as any)
      deviceFlowInfo.value = response
      sessionId.value = response.session_id
      deviceFlowStatus.value = 'initiated'
      return true
    } catch (err: any) {
      error.value =
        err.response?.data?.detail || t('admin.accounts.oauth.kimi.failedToInitiateFlow')
      appStore.showError(error.value)
      deviceFlowStatus.value = 'error'
      return false
    } finally {
      loading.value = false
    }
  }

  const pollForToken = async (): Promise<KimiTokenInfo | null> => {
    if (!sessionId.value) {
      error.value = t('admin.accounts.oauth.kimi.noActiveSession')
      return null
    }

    loading.value = true
    error.value = ''
    deviceFlowStatus.value = 'polling'

    try {
      const tokenInfo = await adminAPI.kimi.pollToken(sessionId.value)
      deviceFlowStatus.value = 'success'
      return tokenInfo
    } catch (err: any) {
      const detail = err.response?.data?.detail || ''
      if (detail.includes('过期') || detail.includes('expired')) {
        deviceFlowStatus.value = 'expired'
        error.value = t('admin.accounts.oauth.kimi.deviceCodeExpired')
      } else {
        deviceFlowStatus.value = 'error'
        error.value = detail || t('admin.accounts.oauth.kimi.pollFailed')
      }
      return null
    } finally {
      loading.value = false
    }
  }

  const validateRefreshToken = async (
    refreshToken: string,
    proxyId?: number | null
  ): Promise<KimiTokenInfo | null> => {
    if (!refreshToken.trim()) {
      error.value = t('admin.accounts.oauth.kimi.pleaseEnterRefreshToken')
      return null
    }

    loading.value = true
    error.value = ''

    try {
      const tokenInfo = await adminAPI.kimi.refreshKimiToken(refreshToken.trim(), proxyId)
      return tokenInfo
    } catch (err: any) {
      error.value =
        err.response?.data?.detail || t('admin.accounts.oauth.kimi.failedToValidateRT')
      return null
    } finally {
      loading.value = false
    }
  }

  const buildCredentials = (tokenInfo: KimiTokenInfo): Record<string, unknown> => {
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
      scope: tokenInfo.scope,
      expires_at: expiresAt,
      device_id: tokenInfo.device_id
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
