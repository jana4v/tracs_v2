<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useIamAuth } from '@/composables/iam/auth'

definePageMeta({
  title: 'IAM - Login',
})

const router = useRouter()
const iamAuth = useIamAuth()

const username = ref('')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')

async function submit() {
  errorMessage.value = ''
  loading.value = true
  try {
    await iamAuth.login(username.value.trim(), password.value)
    router.replace('/iam')
  }
  catch (err: any) {
    console.error('[Login] Error:', err)
    // Extract error message from various possible error structures
    let message = 'Login failed'
    if (err?.data?.message)
      message = err.data.message
    else if (err?.data?.error)
      message = err.data.error
    else if (err?.message)
      message = err.message
    else if (err?.statusMessage)
      message = err.statusMessage
    else if (typeof err === 'string')
      message = err

    errorMessage.value = message
    console.error('[Login] Display error:', message)
  }
  finally {
    loading.value = false
  }
}

onMounted(() => {
  if (iamAuth.isLoggedIn.value) {
    router.replace('/iam')
  }
})
</script>

<template>
  <div class="iam-login-page">
    <Card class="login-card">
      <template #title>
        <div class="title">
          IAM Login
        </div>
      </template>
      <template #content>
        <p class="subtitle">
          Sign in to manage users, roles, and permissions.
        </p>

        <Message v-if="errorMessage" severity="error" :closable="false" class="mb-3">
          {{ errorMessage }}
        </Message>

        <div class="form-grid">
          <label class="field">
            <span class="field-label">Username</span>
            <InputText v-model="username" placeholder="Enter username" autocomplete="username" />
          </label>

          <label class="field">
            <span class="field-label">Password</span>
            <Password
              v-model="password"
              placeholder="Enter password"
              :feedback="false"
              toggle-mask
              autocomplete="current-password"
            />
          </label>

          <Button
            label="Sign In"
            icon="pi pi-sign-in"
            :loading="loading"
            @click="submit"
          />
        </div>
      </template>
    </Card>
  </div>
</template>

<style scoped>
.iam-login-page {
  min-height: calc(100vh - 6rem);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.login-card {
  width: min(460px, 100%);
}

.title {
  font-size: 1.25rem;
  font-weight: 700;
}

.subtitle {
  margin: 0 0 1rem;
  color: var(--p-text-muted-color);
}

.form-grid {
  display: grid;
  gap: 0.9rem;
}

.field {
  display: grid;
  gap: 0.35rem;
}

.field-label {
  font-size: 0.9rem;
  color: var(--p-text-muted-color);
}
</style>
