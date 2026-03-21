<template>
  <div class="users-page">
    <div class="page-header">
      <h1 class="page-title">用户管理</h1>
      <button class="btn-primary" @click="showCreateModal = true">
        <PlusIcon :size="16" />
        新建用户
      </button>
    </div>

    <!-- 用户列表 -->
    <div class="user-table">
      <div class="table-header">
        <span class="col-name">用户名</span>
        <span class="col-role">角色</span>
        <span class="col-status">状态</span>
        <span class="col-actions">操作</span>
      </div>

      <div v-if="loading" class="empty-state">加载中...</div>
      <div v-else-if="users.length === 0" class="empty-state">暂无用户</div>

      <div v-for="user in users" :key="user.id" class="table-row">
        <span class="col-name">{{ user.username }}</span>
        <span class="col-role">
          <select
            class="role-select"
            :value="user.role"
            @change="changeRole(user, ($event.target as HTMLSelectElement).value)"
          >
            <option value="member">成员</option>
            <option value="admin">管理员</option>
          </select>
        </span>
        <span class="col-status">
          <span class="status-dot" :class="user.is_active ? 'dot-on' : 'dot-off'" />
          {{ user.is_active ? '启用' : '禁用' }}
        </span>
        <span class="col-actions">
          <button
            class="action-btn"
            :title="user.is_active ? '禁用' : '启用'"
            @click="toggleActive(user)"
          >
            <ToggleLeftIcon v-if="user.is_active" :size="16" />
            <ToggleRightIcon v-else :size="16" />
          </button>
          <button
            class="action-btn"
            title="重置密码"
            @click="openResetPwd(user)"
          >
            <KeyRoundIcon :size="16" />
          </button>
          <button
            class="action-btn action-danger"
            title="删除"
            @click="confirmDelete(user)"
          >
            <Trash2Icon :size="16" />
          </button>
        </span>
      </div>
    </div>

    <!-- 新建用户弹窗 -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal-card">
        <h2 class="modal-title">新建用户</h2>
        <form @submit.prevent="handleCreate">
          <div class="form-group">
            <label>用户名</label>
            <input v-model="createForm.username" type="text" placeholder="请输入用户名" required />
          </div>
          <div class="form-group">
            <label>密码</label>
            <input v-model="createForm.password" type="password" placeholder="请输入密码" required minlength="8" />
          </div>
          <div class="form-group">
            <label>角色</label>
            <select v-model="createForm.role">
              <option value="member">成员</option>
              <option value="admin">管理员</option>
            </select>
          </div>
          <div v-if="createError" class="error-text">{{ createError }}</div>
          <div class="modal-actions">
            <button type="button" class="btn-ghost" @click="showCreateModal = false">取消</button>
            <button type="submit" class="btn-primary" :disabled="creating">
              {{ creating ? '创建中...' : '创建' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- 重置密码弹窗 -->
    <div v-if="showResetModal" class="modal-overlay" @click.self="showResetModal = false">
      <div class="modal-card">
        <h2 class="modal-title">重置密码 — {{ resetTarget?.username }}</h2>
        <form @submit.prevent="handleResetPassword">
          <div class="form-group">
            <label>新密码</label>
            <input v-model="resetForm.password" type="password" placeholder="请输入新密码（至少8位）" required minlength="8" />
          </div>
          <div v-if="resetError" class="error-text">{{ resetError }}</div>
          <div class="modal-actions">
            <button type="button" class="btn-ghost" @click="showResetModal = false">取消</button>
            <button type="submit" class="btn-primary" :disabled="resetting">
              {{ resetting ? '重置中...' : '重置密码' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { PlusIcon, Trash2Icon, ToggleLeftIcon, ToggleRightIcon, KeyRoundIcon } from 'lucide-vue-next'

interface UserItem {
  id: string
  username: string
  role: string
  is_active: boolean
}

const users = ref<UserItem[]>([])
const loading = ref(false)
const showCreateModal = ref(false)
const creating = ref(false)
const createError = ref('')
const createForm = ref({ username: '', password: '', role: 'member' })

const showResetModal = ref(false)
const resetting = ref(false)
const resetError = ref('')
const resetTarget = ref<UserItem | null>(null)
const resetForm = ref({ password: '' })

async function fetchUsers() {
  loading.value = true
  try {
    const res = await axios.get('/api/users')
    users.value = res.data?.users ?? res.data ?? []
  } catch {
    users.value = []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.value.username || !createForm.value.password) return
  creating.value = true
  createError.value = ''
  try {
    await axios.post('/api/users', {
      username: createForm.value.username,
      password: createForm.value.password,
      role: createForm.value.role,
    })
    showCreateModal.value = false
    createForm.value = { username: '', password: '', role: 'member' }
    await fetchUsers()
  } catch (err: any) {
    createError.value = err.response?.data?.error || '创建失败'
  } finally {
    creating.value = false
  }
}

async function changeRole(user: UserItem, newRole: string) {
  try {
    await axios.put(`/api/users/${user.id}/role`, { role: newRole })
    user.role = newRole
  } catch {
    // revert on error
    await fetchUsers()
  }
}

function openResetPwd(user: UserItem) {
  resetTarget.value = user
  resetForm.value = { password: '' }
  resetError.value = ''
  showResetModal.value = true
}

async function handleResetPassword() {
  if (!resetTarget.value || !resetForm.value.password) return
  resetting.value = true
  resetError.value = ''
  try {
    await axios.put(`/api/users/${resetTarget.value.id}/password`, { password: resetForm.value.password })
    showResetModal.value = false
  } catch (err: any) {
    resetError.value = err.response?.data?.error || '重置失败'
  } finally {
    resetting.value = false
  }
}

async function toggleActive(user: UserItem) {
  try {
    await axios.put(`/api/users/${user.id}/active`, { active: !user.is_active })
    user.is_active = !user.is_active
  } catch {
    // ignore
  }
}

async function confirmDelete(user: UserItem) {
  if (!confirm(`确认删除用户「${user.username}」？此操作不可撤销。`)) return
  try {
    await axios.delete(`/api/users/${user.id}`)
    await fetchUsers()
  } catch {
    // ignore
  }
}

onMounted(fetchUsers)
</script>

<style scoped>
.users-page {
  padding: 24px 32px;
  height: 100%;
  overflow-y: auto;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-hover);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-ghost {
  padding: 8px 16px;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-ghost:hover {
  background: var(--bg-overlay);
}

.user-table {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.table-header,
.table-row {
  display: grid;
  grid-template-columns: 1fr 130px 100px 130px;
  align-items: center;
  padding: 12px 16px;
}

.table-header {
  background: var(--bg-overlay);
  font-size: 11px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border);
}

.table-row {
  border-bottom: 1px solid var(--border-subtle);
  font-size: 13px;
  color: var(--text-primary);
}

.table-row:last-child {
  border-bottom: none;
}

.empty-state {
  padding: 32px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.badge-admin {
  background: var(--accent-dim);
  color: var(--accent);
}

.badge-member {
  background: var(--bg-app);
  color: var(--text-secondary);
  border: 1px solid var(--border);
}

.role-select {
  height: 28px;
  padding: 0 8px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 5px;
  color: var(--text-primary);
  font-size: 12px;
  outline: none;
  cursor: pointer;
}

.role-select:focus {
  border-color: var(--accent);
}

.col-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-on {
  background: var(--green);
}

.dot-off {
  background: var(--text-tertiary);
}

.col-actions {
  display: flex;
  gap: 4px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s;
}

.action-btn:hover {
  background: var(--bg-overlay);
  color: var(--text-primary);
}

.action-danger:hover {
  background: rgba(239, 68, 68, 0.1);
  color: var(--red, #ef4444);
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9000;
}

.modal-card {
  width: 380px;
  padding: 28px 28px 24px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
}

.modal-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}

.form-group label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
}

.form-group input,
.form-group select {
  height: 38px;
  padding: 0 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
  width: 100%;
}

.form-group input:focus,
.form-group select:focus {
  border-color: var(--accent);
}

.error-text {
  font-size: 12px;
  color: var(--red, #ef4444);
  margin-bottom: 10px;
}

.modal-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 4px;
}
</style>
