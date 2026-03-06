<template>
  <div class="page-container">
    <div class="page-header">
      <div class="header-main">
        <h1 class="page-title">{{ t('settings.skills.title') }}</h1>
        <p class="page-description">{{ t('settings.skills.description') }}</p>
      </div>
      <div class="header-actions">
        <n-button
          :loading="reloading"
          @click="handleReload"
          secondary
        >
          <template #icon><n-icon><RefreshOutline /></n-icon></template>
          {{ t('settings.skills.reload') }}
        </n-button>
      </div>
    </div>

    <div class="skills-list" v-loading="loading" :class="{ 'is-loading': loading }">
      <n-empty v-if="skills.length === 0 && !loading" :description="t('settings.skills.noSkills')" size="large" class="page-empty">
        <template #extra>
          <p class="empty-tip">{{ t('settings.skills.noSkillsTip') }}</p>
        </template>
      </n-empty>

      <div v-else class="skill-grid">
        <div v-for="skill in skills" :key="skill.name" class="skill-card">
          <div class="skill-header">
            <n-switch
              :value="skill.enabled"
              :loading="loadingSkills[skill.name]"
              @update:value="(val: boolean) => handleToggle(skill.name, val)"
              size="large"
              class="skill-switch"
            />
            <div class="skill-info">
              <div class="skill-title">
                <h3 class="skill-name">{{ skill.display_name }}</h3>
                <n-tag size="small" type="info" round>{{ skill.name }}</n-tag>
              </div>
              <p class="skill-desc">{{ skill.description }}</p>
            </div>
          </div>
          
          <div class="skill-meta">
            <div class="meta-item">
              <span class="meta-label">{{ t('settings.skills.version') }}</span>
              <span class="meta-value">v{{ skill.version }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">{{ t('settings.skills.level') }}</span>
              <span class="meta-value">{{ skill.level }}</span>
            </div>
            <div v-if="skill.author" class="meta-item">
              <span class="meta-label">{{ t('settings.skills.author') }}</span>
              <span class="meta-value">{{ skill.author }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, reactive, onMounted } from 'vue'
import {
  NTag, NSwitch, NEmpty, NButton, NIcon, useMessage
} from 'naive-ui'
import { RefreshOutline } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { getSkills, setSkillEnabled, reloadSkills, type Skill } from '@/api/settings'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const reloading = ref(false)
const skills = ref<Skill[]>([])
const loadingSkills = reactive<Record<string, boolean>>({})

// 中文：加载技能列表
// English: Load skills list
async function loadSkills() {
  loading.value = true
  try {
    skills.value = await getSkills()
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    loading.value = false
  }
}

// 重新加载技能目录
async function handleReload() {
  reloading.value = true
  try {
    await reloadSkills()
    await loadSkills()
    message.success(t('settings.skills.reloadSuccess'))
  } catch {
    message.error(t('common.error'))
  } finally {
    reloading.value = false
  }
}

// 中文：切换技能启用状态
// English: Toggle skill enabled state
async function handleToggle(name: string, enabled: boolean) {
  loadingSkills[name] = true
  try {
    await setSkillEnabled(name, enabled)
    // 中文：更新本地状态 / Update local state
    const skill = skills.value.find(s => s.name === name)
    if (skill) {
      skill.enabled = enabled
    }
    message.success(t('common.success'))
  } catch (error) {
    message.error(t('common.error'))
  } finally {
    loadingSkills[name] = false
  }
}

onMounted(() => {
  loadSkills()
})
</script>

<style scoped lang="scss">
@use '@/styles/variables.scss' as *;
@use '@/styles/page-layout' as *;

.skills-list {
  min-height: 200px;

  &.is-loading {
    opacity: 0.7;
  }
}

.skill-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: $spacing-5;
}

.skill-card {
  border: 1px solid $color-border-light;
  border-radius: $radius-xl;
  padding: $spacing-6;
  transition: $transition-normal;
  animation: slideUp 0.4s ease-out;
  animation-fill-mode: both;

  @for $i from 1 through 12 {
    &:nth-child(#{$i}) {
      animation-delay: #{$i * 0.05}s;
    }
  }

  &:hover {
    transform: translateY(-2px);
    box-shadow: $shadow-hover;
    border-color: $color-primary-light;
  }

  @keyframes slideUp {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
}

.skill-header {
  display: flex;
  gap: $spacing-4;
  margin-bottom: $spacing-4;

  .skill-info {
    flex: 1;
  }

  .skill-switch {
    transition: all 0.3s ease;

    &:hover {
      transform: scale(1.05);
    }
  }
}

.skill-title {
  display: flex;
  align-items: center;
  gap: $spacing-2;
  margin-bottom: $spacing-2;

  .skill-name {
    margin: 0;
    font-weight: $font-weight-semibold;
    font-size: $font-size-h4;
    color: $color-text-primary;
  }
}

.skill-desc {
  margin: 0;
  font-size: $font-size-sm;
  color: $color-text-secondary;
  line-height: $line-height-normal;
}

.skill-meta {
  display: flex;
  gap: $spacing-6;
  padding-top: $spacing-4;
  border-top: 1px solid $color-border-light;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: $spacing-1;

  .meta-label {
    font-size: $font-size-xs;
    color: $color-text-tertiary;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .meta-value {
    font-size: $font-size-sm;
    color: $color-text-secondary;
    font-weight: $font-weight-medium;
  }
}
</style>
