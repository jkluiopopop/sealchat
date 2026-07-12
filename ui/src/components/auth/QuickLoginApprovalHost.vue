<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'

import { useQuickLoginStore } from '@/stores/quickLogin'

const quickLogin = useQuickLoginStore()
const message = useMessage()
const approving = ref(false)

const activeApproval = computed(() => quickLogin.activeApproval)
const visible = computed(() => !!activeApproval.value)

const formatTime = (value: number) => {
  if (!value) {
    return '-'
  }
  return new Date(value).toLocaleString()
}

const closeModal = () => {
  if (!activeApproval.value) {
    return
  }
  quickLogin.dismiss(activeApproval.value.requestId)
}

const handleApprove = async () => {
  if (!activeApproval.value || approving.value) {
    return
  }
  approving.value = true
  const requestId = activeApproval.value.requestId
  try {
    await quickLogin.approve(requestId)
    message.success('已确认快捷登录')
  } catch (err) {
    message.error((err as any)?.message || (err as any)?.response?.data?.message || '确认失败')
  } finally {
    approving.value = false
  }
}

const handleUpdateShow = (next: boolean) => {
  if (!next) {
    closeModal()
  }
}

onMounted(() => {
  quickLogin.bindGatewayEvents()
})

onBeforeUnmount(() => {
  quickLogin.unbindGatewayEvents()
})
</script>

<template>
  <n-modal
    :show="visible"
    preset="card"
    title="快捷登录确认"
    style="width: min(520px, 92vw)"
    @update:show="handleUpdateShow"
  >
    <n-space vertical size="large">
      <n-alert type="info" :show-icon="false">
        另一个未登录端正在请求使用当前账号完成快捷登录。
      </n-alert>

      <n-descriptions label-placement="left" :column="1" bordered size="small">
        <n-descriptions-item label="账号输入">
          {{ activeApproval?.accountInput || '-' }}
        </n-descriptions-item>
        <n-descriptions-item label="发起时间">
          {{ formatTime(activeApproval?.requestedAt || 0) }}
        </n-descriptions-item>
        <n-descriptions-item label="浏览器">
          {{ activeApproval?.requesterBrowser || '-' }}
        </n-descriptions-item>
        <n-descriptions-item label="终端">
          {{ activeApproval?.requesterDevice || '-' }}
        </n-descriptions-item>
        <n-descriptions-item label="IP">
          {{ activeApproval?.requesterIP || '-' }}
        </n-descriptions-item>
      </n-descriptions>

      <div class="quick-login-approval-host__actions">
        <n-button type="primary" block :loading="approving" @click="handleApprove">
          确认登录
        </n-button>
      </div>
    </n-space>
  </n-modal>
</template>

<style scoped>
.quick-login-approval-host__actions {
  display: flex;
  justify-content: flex-end;
}
</style>
