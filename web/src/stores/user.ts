// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, Team } from '@/api/auth'
import { authApi, teamApi, tokenStorage } from '@/api/auth'

export const useUserStore = defineStore('user', () => {
  // State
  const user = ref<User | null>(null)
  const teams = ref<Team[]>([])
  const currentTeam = ref<Team | null>(null)
  const loading = ref(false)

  // Getters
  const isLoggedIn = computed(() => !!user.value && !!tokenStorage.getAccessToken())
  const displayName = computed(() => user.value?.display_name || user.value?.username || '用户')
  const userInitials = computed(() => {
    const name = displayName.value
    return name.charAt(0).toUpperCase()
  })

  // Actions
  function setUser(newUser: User) {
    user.value = newUser
    localStorage.setItem('user', JSON.stringify(newUser))
  }

  function clearUser() {
    user.value = null
    teams.value = []
    currentTeam.value = null
    tokenStorage.clearTokens()
    localStorage.removeItem('user')
    localStorage.removeItem('currentTeam')
  }

  async function fetchProfile() {
    try {
      const response = await authApi.getProfile()
      if (response.code === 200) {
        setUser(response.data)
      }
    } catch (error) {
      console.error('Failed to fetch profile:', error)
    }
  }

  async function fetchTeams() {
    try {
      const teamsData = await teamApi.list()
      teams.value = teamsData

      // Set current team if not set
      if (!currentTeam.value && teams.value.length > 0) {
        const savedTeamId = localStorage.getItem('currentTeam')
        const savedTeam = teams.value.find(t => t.id === savedTeamId)
        setCurrentTeam(savedTeam || teams.value[0])
      }
    } catch (error) {
      console.error('Failed to fetch teams:', error)
    }
  }

  function setCurrentTeam(team: Team | null) {
    currentTeam.value = team
    if (team) {
      localStorage.setItem('currentTeam', team.id)
    } else {
      localStorage.removeItem('currentTeam')
    }
  }

  async function createTeam(data: { name: string; description?: string }) {
    try {
      const teamData = await teamApi.create(data)
      teams.value.push(teamData)
      if (teams.value.length === 1) {
        setCurrentTeam(teamData)
      }
      return teamData
    } catch (error: any) {
      throw error
    }
  }

  async function logout() {
    clearUser()
  }

  // Initialize from localStorage
  function init() {
    const savedUser = localStorage.getItem('user')
    if (savedUser) {
      try {
        user.value = JSON.parse(savedUser)
      } catch {
        clearUser()
      }
    }

    const savedTeamId = localStorage.getItem('currentTeam')
    if (savedTeamId && user.value) {
      fetchTeams()
    }
  }

  return {
    // State
    user,
    teams,
    currentTeam,
    loading,
    // Getters
    isLoggedIn,
    displayName,
    userInitials,
    // Actions
    setUser,
    clearUser,
    fetchProfile,
    fetchTeams,
    setCurrentTeam,
    createTeam,
    logout,
    init,
  }
})