<template>
  <div class="login-container">
    <div class="login-card">
      <h1 class="login-title">People Counter</h1>
      <div v-if="error" class="error-message">
        {{ error }}
      </div>

      <form @submit.prevent="handleLogin">
        <div class="form-group">
          <label class="form-label">Логин</label>
          <input
              v-model="loginForm.login"
              type="text"
              class="form-input"
              required
              autocomplete="username"
          />
        </div>

        <div class="form-group">
          <label class="form-label">Пароль</label>
          <input
              v-model="loginForm.password"
              type="password"
              class="form-input"
              required
              autocomplete="current-password"
          />
        </div>

        <button
            type="submit"
            class="btn-primary"
            :disabled="loading"
        >
          {{ loading ? 'Вход...' : 'Войти' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script>
import { mapActions } from 'pinia'
import { useAuthStore } from '../stores/auth'

export default {
  name: 'LoginView',
  data() {
    return {
      loginForm: {
        login: '',
        password: ''
      },
      error: null,
      loading: false
    }
  },
  methods: {
    ...mapActions(useAuthStore, ['login']),
    
    async handleLogin() {
      this.error = null
      this.loading = true

      try {
        await this.login(this.loginForm.login, this.loginForm.password)
        await this.$router.push({ name: 'Statistics' })
      } catch (err) {
        this.error = err.message || 'Ошибка авторизации. Проверьте логин и пароль.'
        console.error('Login failed:', err)
      } finally {
        this.loading = false
      }
    }
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 1rem;
}

.login-card {
  width: 100%;
  max-width: 420px;
}

.login-title {
  text-align: center;
  margin-top: 0;
  margin-bottom: 2rem;
  color: var(--color-primary);
}

button[type="submit"] {
  width: 100%;
  margin-top: 1.5rem;
}
</style>