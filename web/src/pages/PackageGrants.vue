<template>
  <div class="package-grants-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <button class="back-btn" @click="goBack">
          <ChevronLeftIcon :size="20" />
        </button>
        <div>
          <h1 class="page-title">资源包授权 - {{ packageName }}</h1>
          <p class="page-desc">管理可访问此资源包的用户</p>
        </div>
      </div>
      <div class="header-right">
        <button class="btn-primary" @click="showGrantDialog = true">
          <UserPlusIcon :size="16" />
          <span>授权用户</span>
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <LoaderIcon :size="24" class="spin" />
      <span>加载中...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="grants.length === 0" class="empty-state">
      <UsersIcon :size="48" />
      <p>暂无授权用户</p>
      <p class="hint">为此资源包添加第一个用户</p>
      <button class="btn-primary" @click="showGrantDialog = true">
        <UserPlusIcon :size="16" />
        <span>授权用户</span>
      </button>
    </div>

    <!-- 授权用户列表 -->
    <div v-else class="grants-table">
      <div class="table-header">
        <span class="col-user">用户</span>
        <span class="col-role">角色</span>
        <span class="col-granted">授权时间</span>
        <span class="col-actions">操作</span>
      </div>

      <div v-for="grant in grants" :key="grant.user_id" class="table-row">
        <span class="col-user">
          <div class="user-info">
            <div class="user-avatar">{{ getUserInitials(grant.username) }}</div>
            <div class="user-details">
              <div class="user-name">{{ grant.username }}</div>
              <div class="user-email">{{ grant.email || '-' }}</div>
            </div>
          </div>
        </span>
        <span class="col-role">
          <span class="role-badge" :class="grant.role">
            {{ grant.role === 'admin' ? '管理员' : '普通用户' }}
          </span>
        </span>
        <span class="col-granted">{{ formatDateTime(grant.granted_at) }}</span>
        <span class="col-actions">
          <button class="btn-danger-sm" @click="revokeGrant(grant.user_id)">
            <TrashIcon :size="14" />
            <span>撤销</span>
          </button>
        </span>
      </div>
    </div>

    <!-- 授权用户对话框 -->
    <div v-if="showGrantDialog" class="modal-overlay" @click.self="showGrantDialog = false">
      <div class="modal-card">
        <h2 class="modal-title">授权用户</h2>

        <div class="form-group">
          <label>选择用户 *</label>
          <select v-model="selectedUserId">
            <option value="">请选择用户</option>
            <option v-for="user in availableUsers" :key="user.id" :value="user.id">
              {{ user.username }} ({{ user.email }})
            </option>
          </select>
        </div>

        <div class="modal-actions">
          <button class="btn-ghost" @click="showGrantDialog = false">取消</button>
          <button class="btn-primary" @click="submitGrant" :disabled="!selectedUserId || submitting">
            <span v-if="submitting" class="loading-dot" />
            <span v-else>确认授权</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  ChevronLeftIcon,
  UserPlusIcon,
  UsersIcon,
  LoaderIcon,
  TrashIcon,
} from 'lucide-vue-next'
import { useToast } from 'vue-sonner'
import { resourcePackageApi } from '@/api/resource'
import axios from 'axios'

const router = useRouter()
const route = useRoute()
const toast = useToast()

const loading = ref(false)
const submitting = ref(false)
const packageName = ref('')
const grants = ref<any[]>([])
const allUsers = ref<any[]>([])
const showGrantDialog = ref(false)
const selectedUserId = ref('')

const packageId = route.params.id as string

onMounted(async () => {
  await Promise.all([loadPackageInfo(), loadGrants(), loadUsers()])
})

async function loadPackageInfo() {
  try {
    const data = await resourcePackageApi.getPackage(packageId)
    packageName.value = data.package.name
  } catch {
    packageName.value = packageId
  }
}

async function loadGrants() {
  loading.value = true
  try {
    const grantsData = await resourcePackageApi.getPackageGrants(packageId)
    // Enrich grants with user info
    const userIds = grantsData.map((g: any) => g.user_id)
    if (userIds.length > 0) {
      const usersRes = await axios.get('/api/users')
      const users = usersRes.data.users || []
      grants.value = grantsData.map((g: any) => {
        const user = users.find((u: any) => u.id === g.user_id)
        return {
          ...g,
          username: user?.username || g.user_id,
          email: user?.email || '',
          role: user?.role || 'member',
        }
      })
    }
  } catch (error: any) {
    toast.error(error.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadUsers() {
  try {
    const res = await axios.get('/api/users')
    const users = res.data.users || []
    // Filter out users who already have grants
    const grantedUserIds = grants.value.map((g) => g.user_id)
    allUsers.value = users.filter((u: any) => !grantedUserIds.includes(u.id))
  } catch (error: any) {
    toast.error(error.response?.data?.error || '加载用户失败')
  }
}

const availableUsers = ref(allUsers.value)

function getUserInitials(username: string): string {
  return username.substring(0, 2).toUpperCase()
}

function formatDateTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN')
}

function goBack() {
  router.push('/resource-packages')
}

async function submitGrant() {
  if (!selectedUserId.value) return

  submitting.value = true
  try {
    await resourcePackageApi.grantToUser(packageId, selectedUserId.value)
    toast.success('授权成功')
    showGrantDialog.value = false
    selectedUserId.value = ''
    await Promise.all([loadGrants(), loadUsers()])
  } catch (error: any) {
    toast.error(error.response?.data?.error || '授权失败')
  } finally {
    submitting.value = false
  }
}

async function revokeGrant(userId: string) {
  if (!confirm('确定要撤销该用户的授权吗？')) return

  try {
    await resourcePackageApi.revokeGrant(packageId, userId)
    toast.success('撤销成功')
    await Promise.all([loadGrants(), loadUsers()])
  } catch (error: any) {
    toast.error(error.response?.data?.error || '撤销失败')
  }
}
</script>

<style scoped>
.package-grants-page {
  padding: 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-primary);
}

.back-btn:hover {
  background: var(--bg-overlay);
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 4px 0;
}

.page-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
}

.loading-state,
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-secondary);
  gap: 12px;
}

.empty-hint {
  font-size: 12px;
  color: var(--text-tertiary);
}

.grants-table {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
}

.table-header,
.table-row {
  display: grid;
  grid-template-columns: 1fr 120px 180px 120px;
  padding: 12px 16px;
  align-items: center;
  gap: 16px;
}

.table-header {
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
}

.table-row {
  border-bottom: 1px solid var(--border-subtle);
}

.table-row:last-child {
  border-bottom: none;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 36px;
  height: 36px;
  background: var(--accent-dim);
  color: var(--accent);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
}

.user-details {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.user-email {
  font-size: 12px;
  color: var(--text-secondary);
}

.role-badge {
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.role-badge.admin {
  background: rgba(139, 92, 246, 0.15);
  color: #8b5cf6;
}

.role-badge.member {
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
}

.btn-danger-sm {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 4px;
  color: #ef4444;
  font-size: 12px;
  cursor: pointer;
}

.btn-danger-sm:hover {
  background: rgba(239, 68, 68, 0.2);
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
  width: 100%;
  max-width: 400px;
}

.modal-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 20px 0;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.form-group select {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  box-sizing: border-box;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 24px;
}
</style>
