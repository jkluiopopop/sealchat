import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { chatEvent, useChatStore } from './chat'
import { api } from './_config'

const DEFAULT_QUICK_LOGIN_HINT = '若该账号已在其他端在线，可向其发起一次快捷登录确认'

export interface QuickLoginApprovalItem {
  requestId: string
  accountInput: string
  requestedAt: number
  expiresAt: number
  requesterIP: string
  requesterBrowser: string
  requesterDevice: string
}

export interface QuickLoginCheckResponse {
  showQuickLoginButton: boolean
  hint?: string
}

export interface QuickLoginRequestPayload {
  account: string
  requesterBrowser?: string
  requesterDevice?: string
}

export interface QuickLoginRequestResponse {
  requestId: string
  requesterToken: string
  expiresAt: number
}

export interface QuickLoginPollPayload {
  requestId: string
  requesterToken: string
}

export interface QuickLoginPollResponse {
  status: 'pending' | 'approved' | 'denied' | 'expired' | 'consumed'
  token?: string
}

type UserAgentDataLike = {
  brands?: Array<{ brand?: string; version?: string }>
  platform?: string
}

const normalizeLabel = (value: unknown) => String(value || '').trim()

const detectBrowserName = (ua: string, uaData?: UserAgentDataLike | null) => {
  const brands = Array.isArray(uaData?.brands) ? uaData?.brands : []
  const brand = brands
    .map((item) => normalizeLabel(item?.brand))
    .find((item) => item && item !== 'Not A;Brand' && item !== 'Not)A;Brand' && item !== 'Chromium')
  if (brand) {
    return brand
  }
  if (/Edg\//i.test(ua)) return 'Microsoft Edge'
  if (/OPR\//i.test(ua) || /Opera/i.test(ua)) return 'Opera'
  if (/Firefox\//i.test(ua)) return 'Firefox'
  if (/Chrome\//i.test(ua) || /CriOS\//i.test(ua)) return 'Chrome'
  if (/Safari\//i.test(ua)) return 'Safari'
  return '未知浏览器'
}

const detectDeviceType = (ua: string, uaData?: UserAgentDataLike | null) => {
  const platform = normalizeLabel(uaData?.platform || '')
  if (/Android|iPhone|iPad|iPod|Mobile|webOS|BlackBerry|IEMobile|Opera Mini/i.test(`${ua} ${platform}`)) {
    return '移动端'
  }
  return '桌面端'
}

export const detectQuickLoginRequesterEnvironment = () => {
  if (typeof navigator === 'undefined') {
    return {
      browser: '未知浏览器',
      device: '桌面端',
    }
  }
  const ua = normalizeLabel(navigator.userAgent)
  const uaData = (navigator as Navigator & { userAgentData?: UserAgentDataLike }).userAgentData
  return {
    browser: detectBrowserName(ua, uaData),
    device: detectDeviceType(ua, uaData),
  }
}

export const useQuickLoginStore = defineStore('quick-login', () => {
  const approverQueue = ref<QuickLoginApprovalItem[]>([])
  const gatewayBound = ref(false)
  let gatewayHandler: ((event: any) => void) | null = null

  const activeApproval = computed(() => approverQueue.value[0] || null)
  const hint = DEFAULT_QUICK_LOGIN_HINT

  const enqueueApproval = (item: QuickLoginApprovalItem) => {
    if (!item.requestId) {
      return
    }
    const existingIndex = approverQueue.value.findIndex((entry) => entry.requestId === item.requestId)
    if (existingIndex >= 0) {
      approverQueue.value.splice(existingIndex, 1, item)
      return
    }
    approverQueue.value.push(item)
    approverQueue.value.sort((a, b) => a.requestedAt - b.requestedAt)
  }

  const dismiss = (requestId: string) => {
    const normalized = normalizeLabel(requestId)
    if (!normalized) {
      return
    }
    approverQueue.value = approverQueue.value.filter((item) => item.requestId !== normalized)
  }

  const bindGatewayEvents = () => {
    if (gatewayBound.value) {
      return
    }
    gatewayHandler = (event: any) => {
      const payload = event?.quickLoginRequested
      const requestId = normalizeLabel(payload?.requestId)
      if (!requestId) {
        return
      }
      enqueueApproval({
        requestId,
        accountInput: normalizeLabel(payload?.accountInput),
        requestedAt: Number(payload?.requestedAt || Date.now()),
        expiresAt: Number(payload?.expiresAt || 0),
        requesterIP: normalizeLabel(payload?.requesterIP),
        requesterBrowser: normalizeLabel(payload?.requesterBrowser),
        requesterDevice: normalizeLabel(payload?.requesterDevice),
      })
    }
    chatEvent.on('quick-login-requested' as any, gatewayHandler as any)
    gatewayBound.value = true
  }

  const unbindGatewayEvents = () => {
    if (!gatewayBound.value || !gatewayHandler) {
      return
    }
    chatEvent.off('quick-login-requested' as any, gatewayHandler as any)
    gatewayHandler = null
    gatewayBound.value = false
  }

  const approve = async (requestId: string) => {
    const normalized = normalizeLabel(requestId)
    if (!normalized) {
      throw new Error('缺少请求ID')
    }
    const chat = useChatStore()
    const resp = await chat.sendAPI<{ ok: boolean; requestId: string; status: string }>('auth.quick_login.approve', {
      request_id: normalized,
    } as any)
    dismiss(normalized)
    return resp
  }

  const deny = async (requestId: string) => {
    const normalized = normalizeLabel(requestId)
    if (!normalized) {
      throw new Error('缺少请求ID')
    }
    const chat = useChatStore()
    const resp = await chat.sendAPI<{ ok: boolean; requestId: string; status: string }>('auth.quick_login.deny', {
      request_id: normalized,
    } as any)
    dismiss(normalized)
    return resp
  }

  const check = async (account: string) => {
    const resp = await api.post<QuickLoginCheckResponse>('api/v1/auth/quick-login/check', {
      account: normalizeLabel(account),
    })
    return resp.data
  }

  const request = async (payload: QuickLoginRequestPayload) => {
    const resp = await api.post<QuickLoginRequestResponse>('api/v1/auth/quick-login/request', {
      account: normalizeLabel(payload.account),
      requesterBrowser: normalizeLabel(payload.requesterBrowser),
      requesterDevice: normalizeLabel(payload.requesterDevice),
    })
    return resp.data
  }

  const poll = async (payload: QuickLoginPollPayload) => {
    const resp = await api.post<QuickLoginPollResponse>('api/v1/auth/quick-login/poll', {
      requestId: normalizeLabel(payload.requestId),
      requesterToken: normalizeLabel(payload.requesterToken),
    })
    return resp.data
  }

  return {
    activeApproval,
    approverQueue,
    approve,
    bindGatewayEvents,
    check,
    deny,
    dismiss,
    hint,
    poll,
    request,
    unbindGatewayEvents,
  }
})
