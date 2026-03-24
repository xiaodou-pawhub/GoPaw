<template>
  <div class="resource-packages-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">资源包管理</h1>
        <p class="page-desc">管理资源包和用户授权</p>
      </div>
      <div class="header-right">
        <button class="btn-primary" @click="showCreateDialog = true">
          <PlusIcon :size="16" />
          <span>新建资源包</span>
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <LoaderIcon :size="24" class="spin" />
      <span>加载中...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="packages.length === 0" class="empty-state">
      <PackageIcon :size="48" />
      <p>暂无资源包</p>
      <p class="hint">创建一个资源包开始授权管理</p>
      <button class="btn-primary" @click="showCreateDialog = true">
        <PlusIcon :size="16" />
        <span>新建资源包</span>
      </button>
    </div>

    <!-- 资源包列表 -->
    <div v-else class="packages-grid">
      <div v-for="pkg in packages" :key="pkg.id" class="package-card">
        <div class="card-header">
          <div class="card-title">
            <h3>{{ pkg.name }}</h3>
            <span v-if="pkg.is_global" class="global-badge">全局</span>
          </div>
          <div class="card-actions">
            <button class="icon-btn" title="编辑" @click="editPackage(pkg)">
              <PencilIcon :size="14" />
            </button>
            <button class="icon-btn danger" title="删除" @click="confirmDelete(pkg)">
              <TrashIcon :size="14" />
            </button>
          </div>
        </div>

        <p class="card-desc">{{ pkg.description || '暂无描述' }}</p>

        <div class="card-meta">
          <div class="meta-item">
            <span class="meta-label">创建时间</span>
            <span class="meta-value">{{ formatDateTime(pkg.created_at) }}</span>
          </div>
          <div class="meta-item">
            <span class="meta-label">包含资源</span>
            <span class="meta-value">{{ getResourceCount(pkg.id) }} 个</span>
          </div>
        </div>

        <div class="card-actions-bar">
          <button class="btn-secondary" @click="viewPackage(pkg)">
            <EyeIcon :size="14" />
            <span>查看详情</span>
          </button>
          <button class="btn-secondary" @click="manageGrants(pkg)">
            <UsersIcon :size="14" />
            <span>管理授权</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 创建/编辑资源包对话框 -->
    <div v-if="showCreateDialog || showEditDialog" class="modal-overlay" @click.self="closeDialog">
      <div class="modal-card">
        <h2 class="modal-title">{{ showCreateDialog ? '新建资源包' : '编辑资源包' }}</h2>

        <div class="form-group">
          <label>资源包名称 *</label>
          <input v-model="formData.name" type="text" placeholder="输入资源包名称" />
        </div>

        <div class="form-group">
          <label>描述</label>
          <textarea v-model="formData.description" rows="3" placeholder="输入资源包描述" />
        </div>

        <div class="form-group">
          <label class="checkbox-label">
            <input v-model="formData.is_global" type="checkbox" />
            <span>全局资源包（所有用户可用）</span>
          </label>
        </div>

        <div class="modal-actions">
          <button class="btn-ghost" @click="closeDialog">取消</button>
          <button class="btn-primary" @click="submitForm" :disabled="submitting">
            <span v-if="submitting" class="loading-dot" />
            <span v-else>{{ showCreateDialog ? '创建' : '保存' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 资源包详情对话框 -->
    <div v-if="showDetailDialog" class="modal-overlay modal-lg" @click.self="closeDialog">
      <div class="modal-card">
        <div class="modal-header">
          <h2 class="modal-title">{{ selectedPackage?.name }}</h2>
          <button class="icon-btn" @click="closeDialog">
            <XIcon :size="18" />
          </button>
        </div>

        <div class="detail-content">
          <div class="detail-section">
            <h3>基本信息</h3>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">描述</span>
                <span class="info-value">{{ selectedPackage?.description || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">类型</span>
                <span class="info-value">
                  <span v-if="selectedPackage?.is_global" class="global-badge">全局</span>
                  <span v-else>授权</span>
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">创建时间</span>
                <span class="info-value">{{ formatDateTime(selectedPackage?.created_at) }}</span>
              </div>
            </div>
          </div>

          <div class="detail-section">
            <h3>包含资源</h3>
            <div v-if="packageItems.length === 0" class="empty-hint">暂无资源</div>
            <div v-else class="items-list">
              <div v-for="item in packageItems" :key="item.resource_id" class="item-row">
                <span class="item-type">{{ getResourceTypeLabel(item.resource_type) }}</span>
                <span class="item-id">{{ item.resource_id }}</span>
                <button class="icon-btn danger" title="移除" @click="removeItem(item)">
                  <TrashIcon :size="14" />
                </button>
              </div>
            </div>

            <div v-if="selectedPackage" class="add-item-section">
              <h4>添加资源</h4>
              <div class="add-item-form">
                <select v-model="newItem.resource_type">
                  <option value="agent">数字员工</option>
                  <option value="skill">技能</option>
                  <option value="knowledge">知识库</option>
                  <option value="model">模型</option>
                </select>
                <input v-model="newItem.resource_id" type="text" placeholder="资源 ID" />
                <button class="btn-primary" @click="addItem" :disabled="!newItem.resource_id">
                  <PlusIcon :size="14" />
                  <span>添加</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  PlusIcon,
  PencilIcon,
  TrashIcon,
  EyeIcon,
  UsersIcon,
  LoaderIcon,
  PackageIcon,
  XIcon,
} from 'lucide-vue-next'
import { useToast } from 'vue-sonner'
import { resourcePackageApi, type ResourcePackage, type ResourceItem } from '@/api/resource'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const toast = useToast()
const userStore = useUserStore()

const loading = ref(false)
const submitting = ref(false)
const packages = ref<ResourcePackage[]>([])
const packageItemsMap = ref<Record<string, ResourceItem[]>>({})

const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showDetailDialog = ref(false)
const selectedPackage = ref<ResourcePackage | null>(null)
const packageItems = ref<ResourceItem[]>([])

const formData = ref({
  name: '',
  description: '',
  is_global: false,
})

const newItem = ref({
  resource_type: 'agent',
  resource_id: '',
})

onMounted(() => {
  loadPackages()
})

async function loadPackages() {
  loading.value = true
  try {
    packages.value = await resourcePackageApi.listPackages()
    // Load items for each package
    for (const pkg of packages.value) {
      try {
        const data = await resourcePackageApi.getPackage(pkg.id)
        packageItemsMap.value[pkg.id] = data.items || []
      } catch {
        packageItemsMap.value[pkg.id] = []
      }
    }
  } catch (error: any) {
    toast.error(error.response?.data?.error || '加载失败')
  } finally {
    loading.value = false
  }
}

function getResourceCount(packageId: string): number {
  return packageItemsMap.value[packageId]?.length || 0
}

function getResourceTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    agent: '数字员工',
    skill: '技能',
    knowledge: '知识库',
    model: '模型',
  }
  return labels[type] || type
}

function formatDateTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN')
}

function showCreateDialog() {
  formData.value = {
    name: '',
    description: '',
    is_global: false,
  }
  showCreateDialog.value = true
}

function editPackage(pkg: ResourcePackage) {
  selectedPackage.value = pkg
  formData.value = {
    name: pkg.name,
    description: pkg.description,
    is_global: pkg.is_global,
  }
  showEditDialog.value = true
}

async function viewPackage(pkg: ResourcePackage) {
  selectedPackage.value = pkg
  try {
    const data = await resourcePackageApi.getPackage(pkg.id)
    packageItems.value = data.items || []
  } catch {
    packageItems.value = []
  }
  showDetailDialog.value = true
}

function manageGrants(pkg: ResourcePackage) {
  router.push(`/resource-packages/${pkg.id}/grants`)
}

async function submitForm() {
  if (!formData.value.name) {
    toast.error('请输入资源包名称')
    return
  }

  submitting.value = true
  try {
    if (showCreateDialog.value) {
      await resourcePackageApi.createPackage(formData.value)
      toast.success('创建成功')
    } else {
      await resourcePackageApi.updatePackage(selectedPackage.value!.id, formData.value)
      toast.success('保存成功')
    }
    closeDialog()
    loadPackages()
  } catch (error: any) {
    toast.error(error.response?.data?.error || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function confirmDelete(pkg: ResourcePackage) {
  if (!confirm(`确定要删除资源包"${pkg.name}"吗？`)) return

  try {
    await resourcePackageApi.deletePackage(pkg.id)
    toast.success('删除成功')
    loadPackages()
  } catch (error: any) {
    toast.error(error.response?.data?.error || '删除失败')
  }
}

async function addItem() {
  if (!selectedPackage.value || !newItem.value.resource_id) return

  try {
    await resourcePackageApi.addItem(selectedPackage.value.id, newItem.value)
    toast.success('添加成功')
    newItem.value.resource_id = ''
    // Reload items
    const data = await resourcePackageApi.getPackage(selectedPackage.value.id)
    packageItems.value = data.items || []
  } catch (error: any) {
    toast.error(error.response?.data?.error || '添加失败')
  }
}

async function removeItem(item: ResourceItem) {
  if (!selectedPackage.value) return

  try {
    await resourcePackageApi.removeItem(selectedPackage.value.id, item.resource_type, item.resource_id)
    toast.success('移除成功')
    // Reload items
    const data = await resourcePackageApi.getPackage(selectedPackage.value.id)
    packageItems.value = data.items || []
  } catch (error: any) {
    toast.error(error.response?.data?.error || '移除失败')
  }
}

function closeDialog() {
  showCreateDialog.value = false
  showEditDialog.value = false
  showDetailDialog.value = false
  selectedPackage.value = null
}
</script>

<style scoped>
.resource-packages-page {
  padding: 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
  max-width: 1600px;
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
  flex-direction: column;
  gap: 4px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.page-desc {
  font-size: 14px;
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

.packages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 16px;
}

.package-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-title h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.global-badge {
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.card-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
}

.card-meta {
  display: flex;
  gap: 16px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.meta-label {
  font-size: 11px;
  color: var(--text-tertiary);
}

.meta-value {
  font-size: 13px;
  color: var(--text-primary);
}

.card-actions-bar {
  display: flex;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
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

.modal-overlay.modal-lg {
  max-width: 800px;
  margin: 0 auto;
}

.modal-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
  width: 100%;
  max-width: 480px;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-overlay.modal-lg .modal-card {
  max-width: 800px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.modal-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
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

.form-group input,
.form-group textarea,
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

.form-group textarea {
  resize: vertical;
  font-family: inherit;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: auto;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 24px;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-section h3 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px 0;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 11px;
  color: var(--text-tertiary);
}

.info-value {
  font-size: 13px;
  color: var(--text-primary);
}

.items-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
}

.item-type {
  font-size: 11px;
  padding: 2px 6px;
  background: var(--bg-overlay);
  border-radius: 4px;
  color: var(--text-secondary);
}

.item-id {
  flex: 1;
  font-size: 12px;
  font-family: monospace;
  color: var(--text-primary);
}

.add-item-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.add-item-section h4 {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px 0;
}

.add-item-form {
  display: flex;
  gap: 8px;
}

.add-item-form select,
.add-item-form input {
  padding: 6px 10px;
  background: var(--bg-app);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
}

.add-item-form input {
  flex: 1;
}
</style>
