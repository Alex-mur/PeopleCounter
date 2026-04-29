<template>
  <div class="page-container">

    <div class="section-block">
      <div class="section-header">
        <h2 class="section-title">Сервисы</h2>
        <button class="btn-small btn-add" @click="openServiceModal()">+ Добавить сервис</button>
      </div>

      <div v-if="loadingServices" class="loading">Загрузка...</div>
      <div v-else class="table-container">
        <table>
          <thead>
          <tr>
            <th>ID</th>
            <th>Название</th>
            <th>API URL</th>
            <th>Описание</th>
            <th>Действия</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="service in services" :key="service.id">
            <td>{{ service.id }}</td>
            <td>{{ service.name }}</td>
            <td>{{ service.api_url }}</td>
            <td>{{ service.description }}</td>
            <td class="actions-cell">
              <button class="btn-small btn-edit" @click="openServiceModal(service)">Изменить</button>
              <button class="btn-small btn-delete" @click="deleteService(service.id)">Удалить</button>
              <button class="btn-small btn-manage" @click="manageServiceCounters(service.id)">Счетчики</button>
            </td>
          </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="section-block">
      <div class="section-header">
        <h2 class="section-title">Пользователи</h2>
        <button class="btn-small btn-add" @click="openUserModal()">+ Добавить пользователя</button>
      </div>

      <div v-if="loadingUsers" class="loading">Загрузка...</div>
      <div v-else class="table-container">
        <table>
          <thead>
          <tr>
            <th>ID</th>
            <th>Логин</th>
            <th>Имя</th>
            <th>Email</th>
            <th>Роль</th>
            <th>Действия</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.id }}</td>
            <td>{{ user.login }}</td>
            <td>{{ user.name }}</td>
            <td>{{ user.email }}</td>
            <td>{{ user.role === 'admin' ? 'Администратор' : 'Пользователь' }}</td>
            <td class="actions-cell">
              <button class="btn-small btn-edit" @click="openUserModal(user)">Изменить</button>
              <button class="btn-small btn-delete" @click="deleteUser(user.id)">Удалить</button>
              <button
                  v-if="user.role === 'viewer'"
                  class="btn-small btn-manage"
                  @click="openUserCountersModal(user)">
                Счетчики
              </button>
            </td>
          </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="showServiceModal" class="modal-overlay" @click.self="closeServiceModal">
      <div class="modal-content">
        <h3 class="modal-title">{{ editingService ? 'Редактировать сервис' : 'Новый сервис' }}</h3>
        <form @submit.prevent="saveService">
          <div class="form-group">
            <label class="form-label">Название</label>
            <input v-model="serviceForm.name" type="text" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">API URL</label>
            <input v-model="serviceForm.api_url" type="text" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">API Key</label>
            <input v-model="serviceForm.api_key" type="text" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">Описание</label>
            <input v-model="serviceForm.description" type="text" class="form-input" />
          </div>
          <div class="modal-buttons">
            <button type="button" class="btn-cancel" @click="closeServiceModal">Отмена</button>
            <button type="submit" class="btn-submit">Сохранить</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showUserModal" class="modal-overlay" @click.self="closeUserModal">
      <div class="modal-content">
        <h3 class="modal-title">{{ editingUser ? 'Редактировать пользователя' : 'Новый пользователь' }}</h3>
        <form @submit.prevent="saveUser">
          <div class="form-group">
            <label class="form-label">Логин</label>
            <input v-model="userForm.login" type="text" class="form-input" required :disabled="!!editingUser" />
          </div>
          <div class="form-group">
            <label class="form-label">Имя</label>
            <input v-model="userForm.name" type="text" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">Email</label>
            <input v-model="userForm.email" type="email" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">Пароль</label>
            <input
                v-model="userForm.password"
                type="password"
                class="form-input"
                :required="!editingUser"
                :placeholder="editingUser ? 'Оставьте пустым, чтобы не менять' : 'Введите пароль'"
            />
          </div>
          <div class="form-group">
            <label class="form-label">Роль</label>
            <select v-model="userForm.role" class="form-input" required>
              <option value="admin">Администратор</option>
              <option value="viewer">Зритель</option>
            </select>
          </div>
          <div class="modal-buttons">
            <button type="button" class="btn-cancel" @click="closeUserModal">Отмена</button>
            <button type="submit" class="btn-submit">Сохранить</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showUserCountersModal" class="modal-overlay" @click.self="closeUserCountersModal">
      <div class="modal-content">
        <h3 class="modal-title">Счетчики пользователя: {{ selectedUserForCounters?.name }}</h3>
        <form @submit.prevent="saveUserCounters">
          <div class="form-group">
            <label class="form-label">Выберите доступные счетчики:</label>

            <div class="counters-list">
              <div v-if="allCounters.length === 0" class="empty-text">
                Нет доступных счетчиков в системе.
              </div>
              <label
                  v-for="counter in allCounters"
                  :key="counter.id"
                  class="checkbox-item"
              >
                <input
                    type="checkbox"
                    :value="counter.id"
                    v-model="userCounterIds"
                />
                <span class="checkbox-label">
                  {{ counter.name }} <small>(Сервис: {{ counter.service_id }})</small>
                </span>
              </label>
            </div>
          </div>

          <div class="modal-buttons">
            <button type="button" class="btn-cancel" @click="closeUserCountersModal">Отмена</button>
            <button type="submit" class="btn-submit">Сохранить</button>
          </div>
        </form>
      </div>
    </div>

  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'ManagementView',
  data() {
    return {
      users: [],
      services: [],
      loadingUsers: true,
      loadingServices: true,
      showUserModal: false,
      showServiceModal: false,
      showUserCountersModal: false,
      editingUser: null,
      editingService: null,
      selectedUserForCounters: null,
      userForm: {},
      serviceForm: {},
      allCounters: [],
      userCounterIds: []
    }
  },
  async mounted() {
    await Promise.all([this.loadServices(), this.loadUsers()])
  },
  methods: {
    async loadServices() {
      this.loadingServices = true
      try {
        this.services = await api.getServices()
      } catch (err) {
        console.error('Ошибка загрузки сервисов:', err)
      } finally {
        this.loadingServices = false
      }
    },

    openServiceModal(service = null) {
      if (service) {
        this.editingService = service
        this.serviceForm = { ...service }
      } else {
        this.editingService = null
        this.serviceForm = { name: '', api_url: '', api_key: '', description: '' }
      }
      this.showServiceModal = true
    },

    closeServiceModal() {
      this.showServiceModal = false
      this.editingService = null
    },

    async saveService() {
      try {
        if (this.editingService && this.editingService.id) {
          await api.updateService(this.editingService.id, this.serviceForm)
        } else {
          await api.createService(this.serviceForm)
        }
        await this.loadServices()
        this.closeServiceModal()
      } catch (err) {
        alert('Ошибка сохранения сервиса: ' + err.message)
      }
    },

    async deleteService(id) {
      if (confirm('Вы уверены, что хотите удалить этот сервис?')) {
        try {
          await api.deleteService(id)
          await this.loadServices()
        } catch (err) {
          alert('Ошибка удаления: ' + err.message)
        }
      }
    },

    manageServiceCounters(serviceId) {
      this.$router.push({ name: 'ServiceCounters', params: { serviceId: serviceId.toString() } })
    },

    async loadUsers() {
      this.loadingUsers = true
      try {
        this.users = await api.getUsers()
      } catch (err) {
        console.error('Ошибка загрузки пользователей:', err)
      } finally {
        this.loadingUsers = false
      }
    },

    openUserModal(user = null) {
      if (user) {
        this.editingUser = user
        this.userForm = { ...user, password: '' }
      } else {
        this.editingUser = null
        this.userForm = { login: '', name: '', email: '', password: '', role: 'viewer', description: '' }
      }
      this.showUserModal = true
    },

    closeUserModal() {
      this.showUserModal = false
      this.editingUser = null
    },

    async saveUser() {
      try {
        const payload = { ...this.userForm }
        if (this.editingUser && !payload.password) {
          delete payload.password
        }

        if (this.editingUser && this.editingUser.id) {
          await api.updateUser(this.editingUser.id, payload)
        } else {
          await api.createUser(payload)
        }
        await this.loadUsers()
        this.closeUserModal()
      } catch (err) {
        alert('Ошибка сохранения пользователя: ' + err.message)
      }
    },

    async deleteUser(id) {
      if (confirm('Вы уверены, что хотите удалить этого пользователя?')) {
        try {
          await api.deleteUser(id)
          await this.loadUsers()
        } catch (err) {
          alert('Ошибка удаления: ' + err.message)
        }
      }
    },

    async openUserCountersModal(user) {
      this.selectedUserForCounters = user
      this.userCounterIds = []

      try {
        const [countersRes, userCountersRes] = await Promise.all([
          api.getCounters(),
          api.getUserCounters(user.id)
        ])

        this.allCounters = countersRes || []
        this.userCounterIds = userCountersRes.counter_ids || []
        this.showUserCountersModal = true
      } catch (err) {
        alert('Ошибка загрузки счетчиков пользователя: ' + err.message)
      }
    },

    closeUserCountersModal() {
      this.showUserCountersModal = false
      this.selectedUserForCounters = null
    },

    async saveUserCounters() {
      if (!this.selectedUserForCounters) return

      try {
        await api.setUserCounters(this.selectedUserForCounters.id, this.userCounterIds)
        this.closeUserCountersModal()
        alert('Права доступа к счетчикам успешно обновлены')
      } catch (err) {
        alert('Ошибка сохранения счетчиков: ' + err.message)
      }
    }
  }
}
</script>

<style scoped>
</style>