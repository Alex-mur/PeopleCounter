<template>
  <div class="app-wrapper">
    <div v-if="isLoading" class="loading-overlay">
      Загрузка...
    </div>

    <div v-else class="app-container">
      <header class="header">
        <div class="header-title">People Counter</div>

        <div class="user-info">
          <span class="user-name">
            {{ user?.name || user?.login }}
          </span>
          <span class="user-role" v-if="user?.role">
            {{ user.role === 'admin' ? 'Администратор' : 'Пользователь' }}
          </span>
          <button class="btn-logout" @click="handleLogout">Выйти</button>
        </div>
      </header>

      <nav class="nav">
        <router-link :to="{ name: 'Statistics' }" class="nav-link" active-class="active">
          Статистика
        </router-link>

        <router-link
            v-if="isAdmin"
            :to="{ name: 'Management' }"
            class="nav-link"
            active-class="active"
        >
          Управление
        </router-link>
      </nav>
      <main class="main-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useAuthStore } from '../stores/auth'

export default {
  name: 'AppLayout',
  data() {
    return {
      isLoading: true
    }
  },
  computed: {
    ...mapState(useAuthStore, ['user', 'isAdmin'])
  },
  async mounted() {
    try {
      if (!this.user) {
        await this.fetchUser()
      }
    } catch (error) {
      console.error('Failed to load user session:', error)
      this.$router.push({ name: 'Login' })
    } finally {
      this.isLoading = false
    }
  },
  methods: {
    ...mapActions(useAuthStore, ['fetchUser', 'logout']),

    handleLogout() {
      this.logout()
      this.$router.push({ name: 'Login' })
    }
  }
}
</script>

<style scoped>
.app-wrapper {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
}

.app-container {
  display: flex;
  flex-direction: column;
  flex: 1;
}

.loading-overlay {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  font-size: 1.5rem;
  color: var(--color-text-secondary);
}

.header {
  background: var(--color-bg-secondary);
  padding: 1rem 2rem;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--color-primary);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.user-name {
  color: var(--color-text-primary);
  font-weight: 500;
}

.user-role {
  background: var(--color-primary);
  color: var(--color-bg-primary);
  padding: 0.25rem 0.75rem;
  border-radius: 1rem;
  font-size: 0.875rem;
  font-weight: 600;
}

.btn-logout {
  background: transparent;
  border: 1px solid var(--color-border);
  color: var(--color-text-secondary);
  padding: 0.5rem 1rem;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-logout:hover {
  background: var(--color-bg-card);
  color: var(--color-text-primary);
}

.nav {
  background: var(--color-bg-secondary);
  padding: 0 2rem;
  display: flex;
  gap: 0.5rem;
  border-bottom: 1px solid var(--color-border);
}

.nav-link {
  padding: 1rem 1.5rem;
  color: var(--color-text-secondary);
  text-decoration: none;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.nav-link:hover {
  color: var(--color-text-primary);
}

.nav-link.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

.main-content {
  flex: 1;
  padding: 2rem;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
}
</style>