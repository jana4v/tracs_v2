<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useIamApi } from '@/composables/iam/api'
import { useIamAuth } from '@/composables/iam/auth'
import { initMenu } from '@/composables/iam/SideNav'

definePageMeta({
  title: 'IAM - Users',
})

initMenu(1)
const router = useRouter()
const iamApi = useIamApi()
const iamAuth = useIamAuth()

interface UserRecord {
  id: string
  username: string
  fullName: string
  email: string
  isActive: boolean
  status: string
  roles: string[]
}

const users = ref<UserRecord[]>([])
const roleOptions = ref<string[]>([])
const loading = ref(false)
const saving = ref(false)
const search = ref('')
const loadError = ref('')
const actionMessage = ref('')

const createDialogVisible = ref(false)
const editDialogVisible = ref(false)
const createForm = ref({
  username: '',
  fullName: '',
  email: '',
  password: '',
  roles: [] as string[],
})
const editForm = ref({
  id: '',
  username: '',
  fullName: '',
  email: '',
  isActive: true,
  roles: [] as string[],
})

const filteredUsers = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q)
    return users.value
  return users.value.filter(u =>
    [u.username, u.fullName, u.email, u.status, ...(u.roles || [])].join(' ').toLowerCase().includes(q),
  )
})

async function loadUsers() {
  loadError.value = ''
  loading.value = true
  try {
    const data = await iamApi.request<any[]>('/users').catch(() => [])
    users.value = (data || []).map((u: any, idx: number) => ({
      id: String(u.id ?? u.userId ?? idx),
      username: String(u.username ?? u.name ?? ''),
      fullName: String(u.full_name ?? u.fullName ?? ''),
      email: String(u.email ?? ''),
      isActive: Boolean(u.is_active ?? (u.status ? String(u.status).toLowerCase() === 'active' : true)),
      status: String(u.status ?? (u.is_active === false ? 'inactive' : 'active')),
      roles: Array.isArray(u.roles) ? u.roles : [],
    }))
  }
  catch (err: any) {
    const status = Number(err?.statusCode ?? err?.status ?? 0)
    if (status === 401 || status === 403) {
      iamAuth.clearSession()
      router.replace('/iam/login')
      return
    }
    loadError.value = String(err?.data?.error ?? err?.message ?? 'Failed to load users')
  }
  finally {
    loading.value = false
  }
}

async function loadRoles() {
  try {
    const data = await iamApi.request<any[]>('/roles').catch(() => [])
    roleOptions.value = (data || [])
      .map((r: any) => String(r.name ?? '').trim())
      .filter((name: string) => Boolean(name))
  }
  catch {
    roleOptions.value = []
  }
}

function resetCreateForm() {
  createForm.value = {
    username: '',
    fullName: '',
    email: '',
    password: '',
    roles: [],
  }
}

function openCreateDialog() {
  actionMessage.value = ''
  resetCreateForm()
  createDialogVisible.value = true
}

function openEditDialog(user: UserRecord) {
  actionMessage.value = ''
  editForm.value = {
    id: user.id,
    username: user.username,
    fullName: user.fullName,
    email: user.email,
    isActive: user.isActive,
    roles: [...(user.roles || [])],
  }
  editDialogVisible.value = true
}

async function createUser() {
  const username = createForm.value.username.trim()
  const fullName = createForm.value.fullName.trim()
  const email = createForm.value.email.trim()
  const password = createForm.value.password

  if (!username || !fullName || !email || !password) {
    actionMessage.value = 'Username, full name, email, and password are required.'
    return
  }

  saving.value = true
  loadError.value = ''
  actionMessage.value = ''
  try {
    await iamApi.request('/users', {
      method: 'POST',
      body: {
        username,
        full_name: fullName,
        email,
        password,
        roles: createForm.value.roles,
      },
    })
    createDialogVisible.value = false
    actionMessage.value = 'User created successfully.'
    await loadUsers()
  }
  catch (err: any) {
    const status = Number(err?.statusCode ?? err?.status ?? 0)
    if (status === 401 || status === 403) {
      iamAuth.clearSession()
      router.replace('/iam/login')
      return
    }
    loadError.value = String(err?.data?.error ?? err?.message ?? 'Failed to create user')
  }
  finally {
    saving.value = false
  }
}

async function updateUser() {
  if (!editForm.value.id)
    return

  const fullName = editForm.value.fullName.trim()
  const email = editForm.value.email.trim()
  if (!fullName || !email) {
    actionMessage.value = 'Full name and email are required.'
    return
  }

  saving.value = true
  loadError.value = ''
  actionMessage.value = ''
  try {
    await iamApi.request(`/users/${editForm.value.id}`, {
      method: 'PUT',
      body: {
        full_name: fullName,
        email,
        is_active: editForm.value.isActive,
      },
    })

    await iamApi.request(`/users/${editForm.value.id}/roles`, {
      method: 'PUT',
      body: {
        roles: editForm.value.roles,
      },
    })

    editDialogVisible.value = false
    actionMessage.value = 'User updated successfully.'
    await loadUsers()
  }
  catch (err: any) {
    const status = Number(err?.statusCode ?? err?.status ?? 0)
    if (status === 401 || status === 403) {
      iamAuth.clearSession()
      router.replace('/iam/login')
      return
    }
    loadError.value = String(err?.data?.error ?? err?.message ?? 'Failed to update user')
  }
  finally {
    saving.value = false
  }
}

async function removeUser(user: UserRecord) {
  const confirmed = window.confirm(`Delete user ${user.username}? This cannot be undone.`)
  if (!confirmed)
    return

  saving.value = true
  loadError.value = ''
  actionMessage.value = ''
  try {
    await iamApi.request(`/users/${user.id}`, {
      method: 'DELETE',
    })
    actionMessage.value = `User ${user.username} deleted.`
    await loadUsers()
  }
  catch (err: any) {
    const status = Number(err?.statusCode ?? err?.status ?? 0)
    if (status === 401 || status === 403) {
      iamAuth.clearSession()
      router.replace('/iam/login')
      return
    }
    loadError.value = String(err?.data?.error ?? err?.message ?? 'Failed to delete user')
  }
  finally {
    saving.value = false
  }
}

onMounted(async () => {
  if (!iamAuth.isLoggedIn.value) {
    router.replace('/iam/login')
    return
  }
  await loadRoles()
  await loadUsers()
})
</script>

<template>
  <div class="content iam-page">
    <AppName appname="IAM - User Accounts" />

    <div class="toolbar">
      <IconField icon-position="left" class="search-field">
        <InputIcon class="pi pi-search" />
        <InputText v-model="search" placeholder="Search users" class="w-full" />
      </IconField>
      <div class="toolbar-actions">
        <Button label="Add User" icon="pi pi-user-plus" @click="openCreateDialog" />
        <Button label="Refresh" icon="pi pi-refresh" :loading="loading" @click="loadUsers" />
      </div>
    </div>

    <Message v-if="loadError" severity="error" :closable="false" class="mb-3">
      {{ loadError }}
    </Message>

    <Message v-if="actionMessage" severity="success" :closable="false" class="mb-3">
      {{ actionMessage }}
    </Message>

    <DataTable :value="filteredUsers" striped-rows paginator :rows="10" size="small" :loading="loading">
      <Column field="username" header="Username" sortable />
      <Column field="fullName" header="Full Name" sortable />
      <Column field="email" header="Email" sortable />
      <Column field="status" header="Status" sortable>
        <template #body="slotProps">
          <Tag :value="slotProps.data.status" :severity="slotProps.data.status === 'active' ? 'success' : 'danger'" />
        </template>
      </Column>
      <Column field="roles" header="Roles">
        <template #body="slotProps">
          <span>{{ (slotProps.data.roles || []).join(', ') || '-' }}</span>
        </template>
      </Column>
      <Column header="Actions" style="width: 11rem">
        <template #body="slotProps">
          <div class="row-actions">
            <Button size="small" text icon="pi pi-pencil" @click="openEditDialog(slotProps.data)" />
            <Button size="small" text severity="danger" icon="pi pi-trash" :loading="saving" @click="removeUser(slotProps.data)" />
          </div>
        </template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="createDialogVisible" modal header="Add User" :style="{ width: '34rem' }">
      <div class="form-grid">
        <label class="field">
          <span>Username</span>
          <InputText v-model="createForm.username" />
        </label>
        <label class="field">
          <span>Full Name</span>
          <InputText v-model="createForm.fullName" />
        </label>
        <label class="field">
          <span>Email</span>
          <InputText v-model="createForm.email" />
        </label>
        <label class="field">
          <span>Password</span>
          <Password v-model="createForm.password" :feedback="false" toggle-mask />
        </label>
        <label class="field">
          <span>Roles</span>
          <MultiSelect v-model="createForm.roles" :options="roleOptions" placeholder="Select roles" display="chip" />
        </label>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="createDialogVisible = false" />
        <Button label="Create" icon="pi pi-check" :loading="saving" @click="createUser" />
      </template>
    </Dialog>

    <Dialog v-model:visible="editDialogVisible" modal header="Update User" :style="{ width: '34rem' }">
      <div class="form-grid">
        <label class="field">
          <span>Username</span>
          <InputText v-model="editForm.username" disabled />
        </label>
        <label class="field">
          <span>Full Name</span>
          <InputText v-model="editForm.fullName" />
        </label>
        <label class="field">
          <span>Email</span>
          <InputText v-model="editForm.email" />
        </label>
        <label class="field checkbox-field">
          <Checkbox v-model="editForm.isActive" binary input-id="isActive" />
          <span>Active account</span>
        </label>
        <label class="field">
          <span>Roles</span>
          <MultiSelect v-model="editForm.roles" :options="roleOptions" placeholder="Select roles" display="chip" />
        </label>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="editDialogVisible = false" />
        <Button label="Save" icon="pi pi-save" :loading="saving" @click="updateUser" />
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.iam-page {
  padding: 1rem;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.search-field {
  width: min(420px, 100%);
}

.toolbar-actions {
  display: flex;
  gap: 0.5rem;
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.form-grid {
  display: grid;
  gap: 0.9rem;
}

.field {
  display: grid;
  gap: 0.35rem;
}

.checkbox-field {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
</style>
