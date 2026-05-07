<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useIamApi } from '@/composables/iam/api'
import { useIamAuth } from '@/composables/iam/auth'
import { initMenu } from '@/composables/iam/SideNav'

definePageMeta({
  title: 'IAM - Roles',
})

initMenu(2)
const router = useRouter()
const iamApi = useIamApi()
const iamAuth = useIamAuth()

interface RoleRecord {
  id: string
  name: string
  description: string
  permissionsCount: number
}

const roles = ref<RoleRecord[]>([])
const loading = ref(false)
const loadError = ref('')

const gatewayHost = import.meta.client ? window.location.hostname : ''
const apiBase = computed(() => (gatewayHost ? `http://${gatewayHost}/iam/api/v1` : ''))

async function loadRoles() {
  if (!apiBase.value)
    return
  loadError.value = ''
  loading.value = true
  try {
    const data = await iamApi.request<any[]>('/roles').catch(() => [])
    roles.value = (data || []).map((r: any, idx: number) => ({
      id: String(r.id ?? idx),
      name: String(r.name ?? r.role ?? ''),
      description: String(r.description ?? ''),
      permissionsCount: Array.isArray(r.permissions) ? r.permissions.length : Number(r.permissionsCount ?? 0),
    }))
  }
  catch (err: any) {
    const status = Number(err?.statusCode ?? err?.status ?? 0)
    if (status === 401 || status === 403) {
      iamAuth.clearSession()
      router.replace('/iam/login')
      return
    }
    loadError.value = String(err?.data?.error ?? err?.message ?? 'Failed to load roles')
  }
  finally {
    loading.value = false
  }
}

onMounted(async () => {
  if (!iamAuth.isLoggedIn.value) {
    router.replace('/iam/login')
    return
  }
  await loadRoles()
})
</script>

<template>
  <div class="content iam-page">
    <AppName appname="IAM - Roles" />

    <div class="toolbar">
      <Button label="Refresh" icon="pi pi-refresh" :loading="loading" @click="loadRoles" />
    </div>

    <Message v-if="loadError" severity="error" :closable="false" class="mb-3">
      {{ loadError }}
    </Message>

    <DataTable :value="roles" striped-rows paginator :rows="10" size="small" :loading="loading">
      <Column field="name" header="Role" sortable />
      <Column field="description" header="Description" />
      <Column field="permissionsCount" header="Permissions" sortable />
    </DataTable>
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
</style>
