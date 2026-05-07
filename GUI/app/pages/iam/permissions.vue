<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useIamApi } from '@/composables/iam/api'
import { useIamAuth } from '@/composables/iam/auth'
import { initMenu } from '@/composables/iam/SideNav'

definePageMeta({
  title: 'IAM - Permissions',
})

initMenu(3)
const router = useRouter()
const iamApi = useIamApi()
const iamAuth = useIamAuth()

// ─── Types ───────────────────────────────────────────────────────────────────
interface PolicyRule { role: string, resource: string, action: string }
interface PermResource { resource: string, description: string, actions: string[] }

// ─── State ───────────────────────────────────────────────────────────────────
const policies = ref<PolicyRule[]>([])
const knownResources = ref<PermResource[]>([])
const configRoles = ref<string[]>([])
const loading = ref(false)
const loadError = ref('')
const successMsg = ref('')
const activeRoleName = ref<string>('')

/** Keys of in-flight toggles → show busy indicator */
const pendingToggles = ref<Set<string>>(new Set())

const BUILTIN_ROLES = new Set(['super_admin', 'admin'])

// ─── Computed ────────────────────────────────────────────────────────────────
const activeRolePolicies = computed(() =>
  policies.value.filter(p => p.role === activeRoleName.value),
)

const isSuperAdmin = computed(() => {
  const roles: string[] = iamAuth.user.value?.roles ?? []
  return roles.includes('super_admin')
})

// ─── Helpers ─────────────────────────────────────────────────────────────────
function hasPermission(role: string, resource: string, action: string): boolean {
  return policies.value.some(
    p => p.role === role && p.resource === resource && p.action === action,
  )
}

function toggleKey(role: string, resource: string, action: string): string {
  return `${role}::${resource}::${action}`
}

function handleApiError(err: any) {
  const status = Number(err?.statusCode ?? err?.status ?? 0)
  if (status === 401 || status === 403) {
    iamAuth.clearSession()
    router.replace('/iam/login')
    return
  }
  loadError.value = String(
    err?.data?.error ?? err?.message ?? 'Operation failed',
  )
}

// ─── Data Loading ─────────────────────────────────────────────────────────────
async function loadData() {
  loading.value = true
  loadError.value = ''
  successMsg.value = ''
  try {
    const [pols, resources, roles] = await Promise.all([
      iamApi.request<PolicyRule[]>('/permissions').catch(() => [] as PolicyRule[]),
      iamApi.request<PermResource[]>('/permissions/resources').catch(() => [] as PermResource[]),
      iamApi.request<any[]>('/roles').catch(() => [] as any[]),
    ])

    policies.value = pols ?? []
    knownResources.value = resources ?? []

    configRoles.value = (roles ?? [])
      .map((r: any) => String(r.name ?? ''))
      .filter(name => !BUILTIN_ROLES.has(name))

    // Select first configurable role by default
    if (configRoles.value.length > 0 && !activeRoleName.value) {
      activeRoleName.value = configRoles.value[0]!
    }
  }
  catch (err: any) {
    handleApiError(err)
  }
  finally {
    loading.value = false
  }
}

// ─── Toggle Permission ────────────────────────────────────────────────────────
async function togglePermission(role: string, resource: string, action: string) {
  const key = toggleKey(role, resource, action)
  if (pendingToggles.value.has(key))
    return

  pendingToggles.value = new Set([...pendingToggles.value, key])
  successMsg.value = ''
  loadError.value = ''

  const granting = !hasPermission(role, resource, action)
  try {
    if (granting) {
      await iamApi.request('/permissions', {
        method: 'POST',
        body: { role, resource, action },
      })
      policies.value = [...policies.value, { role, resource, action }]
      successMsg.value = `Granted "${action}" on "${resource}" to ${role}`
    }
    else {
      await iamApi.request('/permissions', {
        method: 'DELETE',
        body: { role, resource, action },
      })
      policies.value = policies.value.filter(
        p => !(p.role === role && p.resource === resource && p.action === action),
      )
      successMsg.value = `Revoked "${action}" on "${resource}" from ${role}`
    }
  }
  catch (err: any) {
    handleApiError(err)
  }
  finally {
    const next = new Set(pendingToggles.value)
    next.delete(key)
    pendingToggles.value = next
  }
}

// ─── Lifecycle ────────────────────────────────────────────────────────────────
onMounted(async () => {
  if (!iamAuth.isLoggedIn.value) {
    router.replace('/iam/login')
    return
  }
  await loadData()
})
</script>

<template>
  <div class="content iam-page">
    <AppName appname="IAM - Permissions" />

    <div class="toolbar">
      <Button label="Refresh" icon="pi pi-refresh" :loading="loading" @click="loadData" />
    </div>

    <Message v-if="loadError" severity="error" :closable="false" class="mb-3">
      {{ loadError }}
    </Message>
    <Message v-if="successMsg" severity="success" :closable="true" class="mb-3" @close="successMsg = ''">
      {{ successMsg }}
    </Message>

    <!-- Super-admin notice for non-super-admins -->
    <Message v-if="!isSuperAdmin" severity="info" :closable="false" class="mb-3">
      You have read-only access. Only super_admin users can modify permissions.
    </Message>

    <!-- ── Built-in Roles ─────────────────────────────────────────────── -->
    <div class="section-card mb-3">
      <div class="section-header">
        Built-in Roles (Fixed)
      </div>
      <div class="builtin-roles">
        <div class="builtin-role-card">
          <Tag value="super_admin" severity="danger" />
          <div class="builtin-role-info">
            <strong>IAM Administrator</strong>
            <span>Manages IAM roles and endpoint permissions. Full IAM resource access.</span>
          </div>
        </div>
        <div class="builtin-role-card">
          <Tag value="admin" severity="primary" />
          <div class="builtin-role-info">
            <strong>System Admin</strong>
            <span>Full access to all service endpoints. Cannot modify IAM permissions.</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Configurable Roles ─────────────────────────────────────────── -->
    <div class="section-card">
      <div class="section-header">
        Configurable Role Permissions
      </div>

      <p class="hint-text mb-3">
        Enable or disable endpoint resources for each role. Changes take effect immediately.
      </p>

      <div v-if="loading" class="empty-state">
        Loading&hellip;
      </div>

      <div v-else-if="configRoles.length === 0" class="empty-state">
        No configurable roles found.
      </div>

      <template v-else>
        <Tabs v-model:value="activeRoleName">
          <TabList>
            <Tab v-for="role in configRoles" :key="role" :value="role">
              {{ role }}
            </Tab>
          </TabList>

          <TabPanels>
            <TabPanel v-for="role in configRoles" :key="role" :value="role">
              <DataTable
                :value="knownResources"
                class="perm-table"
                size="small"
                striped-rows
              >
                <Column field="resource" header="Resource" style="min-width:180px" />
                <Column field="description" header="Description" />

                <!-- Read column -->
                <Column header="Read" style="width:100px; text-align:center">
                  <template #body="{ data }">
                    <div class="toggle-cell">
                      <Checkbox
                        v-if="data.actions.includes('read')"
                        :model-value="hasPermission(role, data.resource, 'read')"
                        binary
                        :disabled="!isSuperAdmin || pendingToggles.has(toggleKey(role, data.resource, 'read'))"
                        @change="togglePermission(role, data.resource, 'read')"
                      />
                      <span v-else class="na-mark">—</span>
                    </div>
                  </template>
                </Column>

                <!-- Write column -->
                <Column header="Write" style="width:100px; text-align:center">
                  <template #body="{ data }">
                    <div class="toggle-cell">
                      <Checkbox
                        v-if="data.actions.includes('write')"
                        :model-value="hasPermission(role, data.resource, 'write')"
                        binary
                        :disabled="!isSuperAdmin || pendingToggles.has(toggleKey(role, data.resource, 'write'))"
                        @change="togglePermission(role, data.resource, 'write')"
                      />
                      <span v-else class="na-mark">—</span>
                    </div>
                  </template>
                </Column>
              </DataTable>

              <!-- Summary of all granted permissions for this role -->
              <div v-if="activeRolePolicies.length > 0" class="perm-summary mt-3">
                <strong>Granted ({{ activeRolePolicies.length }}):</strong>
                <Tag
                  v-for="p in activeRolePolicies"
                  :key="`${p.resource}:${p.action}`"
                  :value="`${p.resource}:${p.action}`"
                  severity="secondary"
                  class="perm-tag"
                />
              </div>
              <div v-else class="hint-text mt-3">
                No permissions granted for {{ role }}.
              </div>
            </TabPanel>
          </TabPanels>
        </Tabs>
      </template>
    </div>
  </div>
</template>

<style scoped>
.iam-page {
  padding: 1rem;
}

.toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 1rem;
}

.mb-3 {
  margin-bottom: 0.75rem;
}

.mt-3 {
  margin-top: 0.75rem;
}

.section-card {
  border: 1px solid var(--p-content-border-color, #e2e8f0);
  border-radius: 8px;
  padding: 1rem 1.25rem;
  background: var(--p-content-background, #fff);
}

.section-header {
  font-size: 0.95rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
  color: var(--p-text-color, #1a202c);
}

.hint-text {
  font-size: 0.875rem;
  color: var(--p-text-muted-color, #6b7280);
}

.builtin-roles {
  display: flex;
  gap: 1.5rem;
  flex-wrap: wrap;
}

.builtin-role-card {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border: 1px solid var(--p-content-border-color, #e2e8f0);
  border-radius: 8px;
  min-width: 260px;
}

.builtin-role-info {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  font-size: 0.875rem;
  color: var(--p-text-muted-color, #6b7280);
}

.builtin-role-info strong {
  color: var(--p-text-color, #1a202c);
}

.perm-table {
  margin-top: 0.5rem;
}

.toggle-cell {
  display: flex;
  justify-content: center;
  align-items: center;
}

.na-mark {
  color: var(--p-text-muted-color, #9ca3af);
}

.perm-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  align-items: center;
  font-size: 0.875rem;
}

.perm-tag {
  font-size: 0.75rem;
}

.empty-state {
  padding: 2rem;
  text-align: center;
  color: var(--p-text-muted-color, #6b7280);
}
</style>
