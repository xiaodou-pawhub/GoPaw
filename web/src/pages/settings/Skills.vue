<template>
  <div class="skills-page">
    <n-space vertical :size="24">
      <div class="page-header">
        <n-h2>{{ t('settings.skills.title') }}</n-h2>
        <n-text depth="3">{{ t('settings.skills.description') }}</n-text>
      </div>

      <n-card bordered class="list-card" :loading="loading">
        <n-space vertical :size="16">
          <n-alert v-if="skills.length === 0" type="info">
            {{ t('settings.skills.noSkills') }}
          </n-alert>

          <n-list v-else hoverable>
            <n-list-item v-for="skill in skills" :key="skill.name" class="skill-item">
              <template #prefix>
                <n-switch
                  :value="skill.enabled"
                  :loading="loadingSkills[skill.name]"
                  @update:value="(val: boolean) => handleToggle(skill.name, val)"
                />
              </template>

              <n-space vertical :size="4">
                <div class="skill-header">
                  <span class="skill-name">{{ skill.display_name }}</span>
                  <n-tag size="small" quaternary type="info">{{ skill.name }}</n-tag>
                </div>
                <n-text depth="3" class="skill-desc">{{ skill.description }}</n-text>
                <n-space :size="8">
                  <n-text depth="3" style="font-size: 12px">
                    v{{ skill.version }}
                  </n-text>
                  <n-divider vertical />
                  <n-text depth="3" style="font-size: 12px">
                    {{ t('settings.skills.level') }}: {{ skill.level }}
                  </n-text>
                  <template v-if="skill.author">
                    <n-divider vertical />
                    <n-text depth="3" style="font-size: 12px">
                      {{ skill.author }}
                    </n-text>
                  </template>
                </n-space>
              </n-space>
            </n-list-item>
          </n-list>
        </n-space>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
// 中文：导入必要的依赖
// English: Import necessary dependencies
import { ref, reactive, onMounted } from 'vue'
import {
  NCard, NList, NListItem, NSpace, NTag, NText, NSwitch,
  NAlert, NDivider, NH2, useMessage
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { getSkills, setSkillEnabled, type Skill } from '@/api/settings'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
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
.skills-page {
  padding: 12px;
}

.page-header {
  margin-bottom: 8px;
}

.list-card {
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
}

.skill-item {
  padding: 16px 0;
}

.skill-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.skill-name {
  font-weight: 700;
  font-size: 17px;
}

.skill-desc {
  font-size: 14px;
}
</style>
