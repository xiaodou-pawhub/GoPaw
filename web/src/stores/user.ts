// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface UserInfo {
  id: string
  username: string
  role: string
}

export const useUserStore = defineStore('user', () => {
  const user = ref<UserInfo | null>(null)

  const isLoggedIn = computed(() => !!user.value && !!localStorage.getItem('access_token'))
  const displayName = computed(() => user.value?.username || '用户')

  function setUser(u: UserInfo) {
    user.value = u
  }

  function clearUser() {
    user.value = null
    localStorage.removeItem('access_token')
  }

  return {
    user,
    isLoggedIn,
    displayName,
    setUser,
    clearUser,
  }
})
