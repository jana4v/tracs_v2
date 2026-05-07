<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useIamApi } from '@/composables/iam/api'
import { useIamAuth } from '@/composables/iam/auth'
import { initMenu } from '@/composables/iam/SideNav'

definePageMeta({
  title: 'IAM - Overview',
})

initMenu(0)
const router = useRouter()
const iamApi = useIamApi()
const iamAuth = useIamAuth()

interface UserRecord {
  id: string
  last_login?: string
}

interface RoleRecord {
  id: string
  permissions?: string[]
}

const stats = ref({
  users: 0,
  roles: 0,
  permissions: 0,
  recentAudit: 0,
})

const loading = ref(false)
const loadError = ref('')

async function loadStats() {
  loadError.value = ''
  loading.value = true
  try {
    const [users, roles] = await Promise.all([
      iamApi.request<UserRecord[]>('/users').catch(() => []),
      iamApi.request<RoleRecord[]>('/roles').catch(() => []),
    ])

    const permissionsSet = new Set<string>()
    for (const role of roles || []) {
      for (const permission of role.permissions || []) {
        permissionsSet.add(permission)
      }
    }

    const now = Date.now()
    const last24Hours = 24 * 60 * 60 * 1000
    const recentSignIns = (users || []).filter((u) => {
      if (!u.last_login)
        return false
      const ts = new Date(u.last_login).getTime()
      return !Number.isNaN(ts) && now - ts <= last24Hours
    }).length

    stats.value = {
      users: users.length,
      roles: roles.length,
      permissions: permissionsSet.size,
      recentAudit: recentSignIns,
    }
  }
  catch (err: any) {
    const status = Number(err?.statusCode ?? err?.status ?? 0)
    if (status === 401 || status === 403) {
      iamAuth.clearSession()
      router.replace('/iam/login')
      return
    }
    loadError.value = String(err?.data?.error ?? err?.message ?? 'Failed to load IAM data')
  }
  finally {
    loading.value = false
  }
}

async function doLogout() {
  await iamAuth.logout()
  router.replace('/iam/login')
}

onMounted(async () => {
  if (!iamAuth.isLoggedIn.value) {
    router.replace('/iam/login')
    return
  }
  await loadStats()
})
</script>

<template>
  <div class="content iam-page">
    <AppName appname="Identity &amp; Access Management" />

    <div class="iam-header">
      <p class="iam-subtitle">
        Manage users, assign roles, control permissions, and review audit trails.
      </p>
      <div class="header-actions">
        <Button label="Refresh" icon="pi pi-refresh" :loading="loading" @click="loadStats" />
        <Button label="Logout" icon="pi pi-sign-out" severity="secondary" @click="doLogout" />
      </div>
    </div>

    <Message v-if="loadError" severity="error" :closable="false" class="mb-3">
      {{ loadError }}
    </Message>

    <div class="iam-grid">
      <Card>
        <template #title>
          Total Users
        </template>
        <template #content>
          <div class="iam-stat">
            {{ stats.users }}
          </div>
        </template>
      </Card>

      <Card>
        <template #title>
          Total Roles
        </template>
        <template #content>
          <div class="iam-stat">
            {{ stats.roles }}
          </div>
        </template>
      </Card>

      <Card>
        <template #title>
          Total Permissions
        </template>
        <template #content>
          <div class="iam-stat">
            {{ stats.permissions }}
          </div>
        </template>
      </Card>

      <Card>
        <template #title>
          Recent Sign-ins (24h)
        </template>
        <template #content>
          <div class="iam-stat">
            {{ stats.recentAudit }}
          </div>
        </template>
      </Card>
    </div>

    <Card class="iam-quick-actions">
      <template #title>
        Quick Actions
      </template>
      <template #content>
        <div class="action-row">
          <NuxtLink to="/iam/users">
            <Button label="Manage Users" icon="pi pi-users" />
          </NuxtLink>
          <NuxtLink to="/iam/roles">
            <Button label="Manage Roles" icon="pi pi-id-card" severity="secondary" />
          </NuxtLink>
          <NuxtLink to="/iam/permissions">
            <Button label="Manage Permissions" icon="pi pi-lock" severity="contrast" />
          </NuxtLink>
          <NuxtLink to="/iam/audit">
            <Button label="View Audit Logs" icon="pi pi-history" severity="help" />
          </NuxtLink>
        </div>
      </template>
    </Card>
  </div>
</template>

<style scoped>
.iam-page {
  padding: 1rem;
}

.iam-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.iam-subtitle {
  margin: 0;
  color: var(--p-text-muted-color);
}

.iam-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.iam-stat {
  font-size: 2rem;
  font-weight: 700;
  line-height: 1;
}

.iam-quick-actions {
  margin-top: 1rem;
}

.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}
</style>
