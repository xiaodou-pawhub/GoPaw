<template>
  <div class="teams-page">
    <!-- Header -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">团队管理</h1>
        <span class="page-subtitle">管理您的团队和成员</span>
      </div>
      <button class="create-btn" @click="showCreateModal = true">
        <PlusIcon :size="16" />
        <span>创建团队</span>
      </button>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <LoadingSpinner />
      <span>加载中...</span>
    </div>

    <!-- Empty State -->
    <div v-else-if="teams.length === 0" class="empty-state">
      <UsersIcon :size="48" class="empty-icon" />
      <h3>暂无团队</h3>
      <p>创建您的第一个团队，开始协作</p>
      <button class="create-btn" @click="showCreateModal = true">
        <PlusIcon :size="16" />
        <span>创建团队</span>
      </button>
    </div>

    <!-- Teams Grid -->
    <div v-else class="teams-grid">
      <div
        v-for="team in teams"
        :key="team.id"
        class="team-card"
        :class="{ active: currentTeam?.id === team.id }"
        @click="selectTeam(team)"
      >
        <div class="team-header">
          <div class="team-avatar">
            {{ team.avatar || team.name.charAt(0).toUpperCase() }}
          </div>
          <div class="team-badges">
            <span v-if="team.owner_id === user?.id" class="badge owner">所有者</span>
            <span v-if="currentTeam?.id === team.id" class="badge current">当前</span>
          </div>
        </div>

        <div class="team-info">
          <h3 class="team-name">{{ team.name }}</h3>
          <p class="team-slug">{{ team.slug }}</p>
          <p class="team-desc">{{ team.description || '暂无描述' }}</p>
        </div>

        <div class="team-meta">
          <span class="meta-item">
            <UsersIcon :size="14" />
            {{ getMemberCount(team.id) }} 成员
          </span>
          <span class="meta-item">
            <CalendarIcon :size="14" />
            {{ formatDate(team.created_at) }}
          </span>
        </div>

        <div class="team-actions">
          <button class="action-btn" @click.stop="viewTeam(team)">
            <EyeIcon :size="14" />
            <span>查看</span>
          </button>
          <button v-if="team.owner_id === user?.id" class="action-btn" @click.stop="editTeam(team)">
            <PencilIcon :size="14" />
            <span>编辑</span>
          </button>
          <button v-if="team.owner_id === user?.id" class="action-btn danger" @click.stop="confirmDelete(team)">
            <TrashIcon :size="14" />
            <span>删除</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Team Detail Panel -->
    <div v-if="selectedTeam" class="team-detail-panel">
      <div class="panel-header">
        <h2>{{ selectedTeam.name }}</h2>
        <button class="close-btn" @click="selectedTeam = null">
          <XIcon :size="20" />
        </button>
      </div>

      <!-- Tabs -->
      <div class="panel-tabs">
        <button
          :class="['tab', { active: activeTab === 'members' }]"
          @click="activeTab = 'members'"
        >
          成员
        </button>
        <button
          :class="['tab', { active: activeTab === 'settings' }]"
          @click="activeTab = 'settings'"
        >
          设置
        </button>
      </div>

      <!-- Members Tab -->
      <div v-if="activeTab === 'members'" class="tab-content">
        <div class="members-header">
          <h3>团队成员</h3>
          <button class="invite-btn" @click="showInviteModal = true">
            <UserPlusIcon :size="16" />
            邀请成员
          </button>
        </div>

        <div v-if="membersLoading" class="members-loading">
          <LoadingSpinner size="small" />
        </div>

        <div v-else class="members-list">
          <div v-for="member in members" :key="member.id" class="member-item">
            <div class="member-avatar">
              {{ member.user?.display_name?.charAt(0) || member.user?.username?.charAt(0) || '?' }}
            </div>
            <div class="member-info">
              <span class="member-name">{{ member.user?.display_name || member.user?.username }}</span>
              <span class="member-email">{{ member.user?.email }}</span>
            </div>
            <div class="member-role">
              <span :class="['role-badge', member.role]">{{ getRoleLabel(member.role) }}</span>
            </div>
            <button
              v-if="selectedTeam.owner_id === user?.id && member.user_id !== user?.id"
              class="remove-btn"
              @click="removeMember(member)"
            >
              <TrashIcon :size="14" />
            </button>
          </div>
        </div>
      </div>

      <!-- Settings Tab -->
      <div v-if="activeTab === 'settings'" class="tab-content">
        <div class="settings-form">
          <div class="form-group">
            <label>团队名称</label>
            <input v-model="editForm.name" type="text" />
          </div>
          <div class="form-group">
            <label>团队描述</label>
            <textarea v-model="editForm.description" rows="3" />
          </div>
          <button class="save-btn" @click="saveTeamSettings" :disabled="saving">
            {{ saving ? '保存中...' : '保存更改' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Create Team Modal -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal">
        <div class="modal-header">
          <h3>创建团队</h3>
          <button class="close-btn" @click="showCreateModal = false">
            <XIcon :size="20" />
          </button>
        </div>
        <form class="modal-body" @submit.prevent="handleCreateTeam">
          <div class="form-group">
            <label>团队名称 *</label>
            <input v-model="createForm.name" type="text" required placeholder="输入团队名称" />
          </div>
          <div class="form-group">
            <label>团队描述</label>
            <textarea v-model="createForm.description" rows="3" placeholder="输入团队描述（可选）" />
          </div>
          <div class="modal-actions">
            <button type="button" class="cancel-btn" @click="showCreateModal = false">取消</button>
            <button type="submit" class="submit-btn" :disabled="creating">
              {{ creating ? '创建中...' : '创建团队' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Invite Modal -->
    <div v-if="showInviteModal" class="modal-overlay" @click.self="showInviteModal = false">
      <div class="modal">
        <div class="modal-header">
          <h3>邀请成员</h3>
          <button class="close-btn" @click="showInviteModal = false">
            <XIcon :size="20" />
          </button>
        </div>
        <form class="modal-body" @submit.prevent="handleInvite">
          <div class="form-group">
            <label>邮箱地址 *</label>
            <input v-model="inviteForm.email" type="email" required placeholder="输入成员邮箱" />
          </div>
          <div class="form-group">
            <label>角色</label>
            <select v-model="inviteForm.role">
              <option value="admin">管理员</option>
              <option value="member">成员</option>
              <option value="guest">访客</option>
            </select>
          </div>
          <div class="modal-actions">
            <button type="button" class="cancel-btn" @click="showInviteModal = false">取消</button>
            <button type="submit" class="submit-btn" :disabled="inviting">
              {{ inviting ? '发送中...' : '发送邀请' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-if="deletingTeam"
      title="删除团队"
      :message="`确定要删除团队「${deletingTeam.name}」吗？此操作不可撤销。`"
      @confirm="handleDelete"
      @cancel="deletingTeam = null"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import {
  PlusIcon,
  UsersIcon,
  CalendarIcon,
  EyeIcon,
  PencilIcon,
  TrashIcon,
  XIcon,
  UserPlusIcon,
} from 'lucide-vue-next'
import { useUserStore } from '@/stores/user'
import { teamApi, type Team, type TeamMember } from '@/api/auth'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

const userStore = useUserStore()

const loading = ref(true)
const teams = ref<Team[]>([])
const user = ref(userStore.user)
const currentTeam = ref(userStore.currentTeam)
const selectedTeam = ref<Team | null>(null)
const activeTab = ref('members')
const members = ref<TeamMember[]>([])
const membersLoading = ref(false)

// Modals
const showCreateModal = ref(false)
const showInviteModal = ref(false)
const creating = ref(false)
const inviting = ref(false)
const saving = ref(false)
const deletingTeam = ref<Team | null>(null)

// Forms
const createForm = ref({
  name: '',
  description: '',
})

const editForm = ref({
  name: '',
  description: '',
})

const inviteForm = ref({
  email: '',
  role: 'member',
})

// Member counts (mock data for now)
const memberCounts = ref<Record<string, number>>({})

onMounted(async () => {
  await loadTeams()
})

watch(selectedTeam, async (team) => {
  if (team) {
    editForm.value = {
      name: team.name,
      description: team.description || '',
    }
    await loadMembers(team.id)
  }
})

async function loadTeams() {
  loading.value = true
  try {
    const response = await teamApi.list()
    if (response.code === 200) {
      teams.value = response.data
    }
  } catch (error) {
    console.error('Failed to load teams:', error)
  } finally {
    loading.value = false
  }
}

async function loadMembers(teamId: string) {
  membersLoading.value = true
  try {
    const response = await teamApi.getMembers(teamId)
    if (response.code === 200) {
      members.value = response.data
      memberCounts.value[teamId] = response.data.length
    }
  } catch (error) {
    console.error('Failed to load members:', error)
  } finally {
    membersLoading.value = false
  }
}

function selectTeam(team: Team) {
  userStore.setCurrentTeam(team)
  currentTeam.value = team
}

function viewTeam(team: Team) {
  selectedTeam.value = team
}

function editTeam(team: Team) {
  selectedTeam.value = team
  activeTab.value = 'settings'
}

function getMemberCount(teamId: string): number {
  return memberCounts.value[teamId] || 0
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

function getRoleLabel(role: string): string {
  const labels: Record<string, string> = {
    owner: '所有者',
    admin: '管理员',
    member: '成员',
    guest: '访客',
  }
  return labels[role] || role
}

async function handleCreateTeam() {
  if (!createForm.value.name) return

  creating.value = true
  try {
    await userStore.createTeam(createForm.value)
    showCreateModal.value = false
    createForm.value = { name: '', description: '' }
    await loadTeams()
  } catch (error: any) {
    alert(error.message || '创建失败')
  } finally {
    creating.value = false
  }
}

async function handleInvite() {
  if (!selectedTeam.value || !inviteForm.value.email) return

  inviting.value = true
  try {
    await teamApi.inviteMember(selectedTeam.value.id, inviteForm.value)
    showInviteModal.value = false
    inviteForm.value = { email: '', role: 'member' }
    alert('邀请已发送')
  } catch (error: any) {
    alert(error.response?.data?.message || '邀请失败')
  } finally {
    inviting.value = false
  }
}

async function saveTeamSettings() {
  if (!selectedTeam.value) return

  saving.value = true
  try {
    await teamApi.update(selectedTeam.value.id, editForm.value)
    selectedTeam.value.name = editForm.value.name
    selectedTeam.value.description = editForm.value.description
    await loadTeams()
    alert('保存成功')
  } catch (error: any) {
    alert(error.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function confirmDelete(team: Team) {
  deletingTeam.value = team
}

async function handleDelete() {
  if (!deletingTeam.value) return

  try {
    await teamApi.delete(deletingTeam.value.id)
    teams.value = teams.value.filter(t => t.id !== deletingTeam.value!.id)
    if (currentTeam.value?.id === deletingTeam.value.id) {
      userStore.setCurrentTeam(teams.value[0] || null)
    }
    deletingTeam.value = null
    selectedTeam.value = null
  } catch (error: any) {
    alert(error.response?.data?.message || '删除失败')
  }
}

async function removeMember(member: TeamMember) {
  if (!selectedTeam.value || !confirm('确定要移除该成员吗？')) return

  try {
    await teamApi.removeMember(selectedTeam.value.id, member.user_id)
    members.value = members.value.filter(m => m.user_id !== member.user_id)
  } catch (error: any) {
    alert(error.response?.data?.message || '移除失败')
  }
}
</script>

<style scoped>
.teams-page {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: #666;
  margin: 0;
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: #667eea;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.create-btn:hover {
  background: #5a6fd6;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.empty-icon {
  color: #ccc;
  margin-bottom: 16px;
}

.empty-state h3 {
  margin: 0 0 8px;
  font-size: 18px;
}

.empty-state p {
  margin: 0 0 20px;
  color: #666;
}

.teams-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.team-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  cursor: pointer;
  transition: all 0.2s;
  border: 2px solid transparent;
}

.team-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}

.team-card.active {
  border-color: #667eea;
}

.team-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.team-avatar {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 20px;
  font-weight: 600;
}

.team-badges {
  display: flex;
  gap: 6px;
}

.badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}

.badge.owner {
  background: #e3f2fd;
  color: #1976d2;
}

.badge.current {
  background: #e8f5e9;
  color: #388e3c;
}

.team-info {
  margin-bottom: 12px;
}

.team-name {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 4px;
}

.team-slug {
  font-size: 12px;
  color: #999;
  margin: 0 0 8px;
}

.team-desc {
  font-size: 13px;
  color: #666;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.team-meta {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #666;
}

.team-actions {
  display: flex;
  gap: 8px;
  border-top: 1px solid #f0f0f0;
  padding-top: 16px;
}

.action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 8px;
  background: #f5f5f5;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  color: #666;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn:hover {
  background: #e0e0e0;
}

.action-btn.danger:hover {
  background: #ffebee;
  color: #e53935;
}

/* Team Detail Panel */
.team-detail-panel {
  position: fixed;
  top: 0;
  right: 0;
  width: 400px;
  height: 100vh;
  background: white;
  box-shadow: -4px 0 16px rgba(0, 0, 0, 0.1);
  z-index: 100;
  display: flex;
  flex-direction: column;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #f0f0f0;
}

.panel-header h2 {
  margin: 0;
  font-size: 18px;
}

.close-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: #666;
  padding: 4px;
}

.close-btn:hover {
  color: #333;
}

.panel-tabs {
  display: flex;
  border-bottom: 1px solid #f0f0f0;
}

.tab {
  flex: 1;
  padding: 12px;
  background: none;
  border: none;
  font-size: 14px;
  color: #666;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab.active {
  color: #667eea;
  border-bottom-color: #667eea;
}

.tab-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.members-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.members-header h3 {
  margin: 0;
  font-size: 14px;
}

.invite-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #667eea;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
}

.members-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.member-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f9f9f9;
  border-radius: 8px;
}

.member-avatar {
  width: 36px;
  height: 36px;
  background: #e0e0e0;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  color: #666;
}

.member-info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.member-name {
  font-size: 14px;
  font-weight: 500;
}

.member-email {
  font-size: 12px;
  color: #999;
}

.role-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}

.role-badge.owner {
  background: #e3f2fd;
  color: #1976d2;
}

.role-badge.admin {
  background: #fff3e0;
  color: #f57c00;
}

.role-badge.member {
  background: #f5f5f5;
  color: #666;
}

.role-badge.guest {
  background: #f5f5f5;
  color: #999;
}

.remove-btn {
  background: none;
  border: none;
  color: #999;
  cursor: pointer;
  padding: 4px;
}

.remove-btn:hover {
  color: #e53935;
}

/* Settings Form */
.settings-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 500;
  color: #333;
}

.form-group input,
.form-group textarea,
.form-group select {
  padding: 10px 12px;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  font-size: 14px;
}

.form-group input:focus,
.form-group textarea:focus,
.form-group select:focus {
  outline: none;
  border-color: #667eea;
}

.save-btn {
  padding: 12px;
  background: #667eea;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.save-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}

.modal {
  background: white;
  border-radius: 12px;
  width: 100%;
  max-width: 400px;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #f0f0f0;
}

.modal-header h3 {
  margin: 0;
  font-size: 18px;
}

.modal-body {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.modal-actions {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.cancel-btn {
  flex: 1;
  padding: 12px;
  background: #f5f5f5;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
}

.submit-btn {
  flex: 1;
  padding: 12px;
  background: #667eea;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .teams-page {
    padding: 16px;
  }

  .team-detail-panel {
    width: 100%;
  }
}
</style>