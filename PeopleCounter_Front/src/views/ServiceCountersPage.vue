<template>
  <div class="page-container">
    <div class="page-header">
      <h2 class="section-title">Счетчики сервиса: {{ serviceName }}</h2>
      <button class="btn-small back-btn" @click="goBack">← Назад к сервисам</button>
    </div>
    <div v-if="loading" class="loading">Загрузка...</div>
    <div v-else-if="!selectedCounter">
      <h3 class="subsection-title">Все счетчики</h3>
      <button class="btn-small btn-add mb-4" @click="createNewCounter">+ Создать счетчик</button>

      <div v-if="serviceCounters.length === 0" class="empty-state">
        Нет счетчиков
      </div>
      <div v-else class="table-container">
        <table>
          <thead>
          <tr>
            <th>ID</th><th>Название</th><th>URL</th><th>Описание</th><th>Действия</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="counter in serviceCounters" :key="counter.id">
            <td>{{ counter.id }}</td>
            <td>{{ counter.name }}</td>
            <td>{{ counter.url }}</td>
            <td>{{ counter.description }}</td>
            <td>
              <button class="btn-small btn-edit mr-2" @click="selectCounter(counter)">Настроить</button>
              <button class="btn-small btn-delete" @click="deleteCounter(counter.id)">Удалить</button>
            </td>
          </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-else>
      <div class="header-action mb-4">
        <h3 class="subsection-title">Настройка: {{ selectedCounter.id ? selectedCounter.name : 'Новый счетчик' }}</h3>
        <button class="btn-small back-btn" @click="selectedCounter = null">Закрыть</button>
      </div>

      <div v-if="selectedCounter.id" class="card-panel mb-4">
        <div class="header-action mb-4">
          <h4 class="m-0">Видео поток</h4>
          <button
              type="button"
              class="btn-small"
              :class="isEditMode ? 'btn-success' : 'btn-primary'"
              @click="isEditMode = !isEditMode"
          >
            {{ isEditMode ? 'Сохранить разметку' : 'Редактировать линии' }}
          </button>
        </div>

        <div class="stream-wrapper">
          <img
              v-if="!isEditMode"
              :src="liveStreamUrl"
              @error="handleStreamError"
              class="stream-element"
              alt="Video stream"
          />
          <canvas
              v-if="isEditMode"
              ref="streamCanvas"
              @mousedown="handleCanvasMouseDown"
              @mousemove="handleCanvasMouseMove"
              @mouseup="handleCanvasMouseUp"
              @mouseleave="handleCanvasMouseUp"
              class="stream-element canvas-edit"
          ></canvas>
        </div>
        <p v-if="isEditMode" class="hint-text">
          Перетаскивайте концы линий на видео, чтобы изменить зону подсчета.
        </p>
      </div>
      <div class="card-panel">
        <form @submit.prevent="saveCounter">
          <div class="form-group">
            <label class="form-label">Название</label>
            <input v-model="selectedCounter.name" type="text" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">Описание</label>
            <input v-model="selectedCounter.description" type="text" class="form-input" />
          </div>
          <div class="form-group">
            <label class="form-label">URL (RTSP/HTTP)</label>
            <input v-model="selectedCounter.url" type="text" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">Vid Stride ({{ selectedCounter.vid_stride || 1 }})</label>
            <input v-model.number="selectedCounter.vid_stride" type="range" class="form-input" min="1" max="10" />
          </div>
          <hr class="separator" />
          <h4 class="subsection-title mb-4">Группы подсчета</h4>
          <div class="table-container mb-4">
            <table>
              <thead>
              <tr><th>ID</th><th>Описание</th><th>Действия</th></tr>
              </thead>
              <tbody>
              <tr v-for="(group, idx) in selectedCounter.groups" :key="idx">
                <td>
                  <input v-model.number="group.id" type="number" class="form-input w-80" />
                </td>
                <td>
                  <input v-model="group.description" type="text" class="form-input" />
                </td>
                <td>
                  <button type="button" class="btn-small btn-delete" @click="removeGroup(idx)">Удалить</button>
                </td>
              </tr>
              </tbody>
            </table>
          </div>
          <button type="button" class="btn-small btn-add mb-4" @click="addGroup">+ Добавить группу</button>
          <hr class="separator" />
          <h4 class="subsection-title mb-4">Линии (Зоны детекции)</h4>
          <div class="table-container mb-4">
            <table>
              <thead>
              <tr>
                <th>Название</th><th>Группа</th><th>Start X</th><th>Start Y</th><th>End X</th><th>End Y</th><th>Действия</th>
              </tr>
              </thead>
              <tbody>
              <tr v-for="(line, idx) in selectedCounter.lines" :key="idx">
                <td><input v-model="line.name" type="text" class="form-input w-120" /></td>
                <td>
                  <select v-model.number="line.group_id" class="form-input w-100">
                    <option v-for="g in selectedCounter.groups" :key="g.id" :value="g.id">{{ g.id }}</option>
                  </select>
                </td>
                <td><input v-model.number="line.start[0]" type="number" class="form-input w-80" /></td>
                <td><input v-model.number="line.start[1]" type="number" class="form-input w-80" /></td>
                <td><input v-model.number="line.end[0]" type="number" class="form-input w-80" /></td>
                <td><input v-model.number="line.end[1]" type="number" class="form-input w-80" /></td>
                <td>
                  <button type="button" class="btn-small btn-delete" @click="removeLine(idx)">Удалить</button>
                </td>
              </tr>
              </tbody>
            </table>
          </div>
          <button type="button" class="btn-small btn-add mb-4" @click="addLine">+ Добавить линию</button>

          <div class="form-actions">
            <button type="submit" class="btn-primary flex-1">Сохранить всё</button>
            <button type="button" class="btn-cancel" @click="selectedCounter = null">Отмена</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import api, { API_BASE_URL } from '../api'

export default {
  name: 'CounterManager',
  data() {
    return {
      serviceId: parseInt(this.$route.params.serviceId),
      serviceName: '',
      serviceCounters: [],
      selectedCounter: null,
      loading: true,
      isEditMode: false,
      liveStreamUrl: '',
      dragState: {
        isDragging: false,
        lineIndex: null,
        pointType: null,
        canvasScale: { x: 1, y: 1 }
      },
      snapshotImage: new Image()
    }
  },
  computed: {
    accessToken() {
      return localStorage.getItem('accessToken') || ''
    }
  },
  watch: {
    async isEditMode(newMode, oldMode) {
      if (newMode && this.selectedCounter?.id) {
        this.liveStreamUrl = ''
        setTimeout(this.loadSnapshotToCanvas, 50)
      } else if (!newMode && oldMode && this.selectedCounter?.id) {
        try {
          await api.updateCounter(this.selectedCounter.id, this.selectedCounter)
        } catch (err) {
          alert('Ошибка при выходе из режима редактирования: ' + err.message)
        }
        this.liveStreamUrl = `${API_BASE_URL}api/counters/${this.selectedCounter.id}/stream?jwt=${this.accessToken}&t=${Date.now()}`
      }
    },
    selectedCounter(newVal) {
      if (!newVal) {
        this.isEditMode = false
        this.liveStreamUrl = ''
        this.snapshotImage.src = ''
      }
    }
  },
  async mounted() {
    await Promise.all([this.loadServiceInfo(), this.loadCounters()])
  },
  methods: {
    handleStreamError(event) {
      console.error('Stream loading error', event)
    },
    async loadServiceInfo() {
      try {
        const services = await api.getServices()
        const service = services.find(s => s.id === this.serviceId)
        if (service) this.serviceName = service.name
      } catch (err) {
        console.error('Failed to load service info:', err)
      }
    },
    async loadCounters() {
      this.loading = true
      try {
        const allCounters = await api.getCounters()
        this.serviceCounters = allCounters.filter(c => c.service_id === this.serviceId)
      } catch (err) {
        console.error('Failed to load counters:', err)
      } finally {
        this.loading = false
      }
    },
    createNewCounter() {
      this.selectedCounter = {
        name: '',
        description: '',
        url: '',
        service_id: this.serviceId,
        vid_stride: 1,
        groups: [{ id: 1, description: 'Группа 1' }],
        lines: [{ id: 1, name: 'Линия 1', group_id: 1, start: [100, 200], end: [500, 200] }]
      }
      this.liveStreamUrl = ''
    },
    selectCounter(counter) {
      // Глубокое копирование для безопасного редактирования
      this.selectedCounter = JSON.parse(JSON.stringify(counter))
      this.liveStreamUrl = `${API_BASE_URL}api/counters/${this.selectedCounter.id}/stream?jwt=${this.accessToken}&t=${Date.now()}`
    },
    addGroup() {
      if (!this.selectedCounter) return
      const currentGroups = this.selectedCounter.groups || []
      const maxId = currentGroups.length > 0 ? Math.max(...currentGroups.map(g => g.id)) : 0
      this.selectedCounter.groups.push({ id: maxId + 1, description: `Группа ${maxId + 1}` })
    },
    removeGroup(index) {
      if (!this.selectedCounter) return
      this.selectedCounter.groups.splice(index, 1)
    },
    addLine() {
      if (!this.selectedCounter) return
      const defaultGroupId = this.selectedCounter.groups && this.selectedCounter.groups.length > 0
          ? this.selectedCounter.groups[0].id
          : 1
      const lineNum = (this.selectedCounter.lines?.length || 0) + 1
      this.selectedCounter.lines.push({
        id: lineNum, name: `Линия ${lineNum}`, group_id: defaultGroupId, start: [100, 200], end: [300, 200]
      })
    },
    removeLine(index) {
      if (!this.selectedCounter) return
      this.selectedCounter.lines.splice(index, 1)
    },
    async saveCounter() {
      if (!this.selectedCounter) return
      try {
        if (this.selectedCounter.id) {
          await api.updateCounter(this.selectedCounter.id, this.selectedCounter)
        } else {
          await api.createCounter(this.selectedCounter)
        }
        alert('Сохранено успешно!')
        this.isEditMode = false
        this.selectedCounter = null
        await this.loadCounters()
      } catch (err) {
        alert('Ошибка сохранения: ' + err.message)
      }
    },
    async deleteCounter(id) {
      if (!id) return
      if (confirm('Удалить счетчик?')) {
        try {
          await api.deleteCounter(id)
          await this.loadCounters()
        } catch (err) {
          alert('Ошибка удаления: ' + err.message)
        }
      }
    },
    goBack() {
      this.$router.push({ name: 'Management' })
    },
    drawCanvas() {
      const canvas = this.$refs.streamCanvas
      if (!canvas || !this.isEditMode || !this.selectedCounter) return

      const ctx = canvas.getContext('2d')
      if (!ctx) return

      ctx.clearRect(0, 0, canvas.width, canvas.height)

      if (this.snapshotImage.width > 0 && this.snapshotImage.height > 0) {
        try {
          ctx.drawImage(this.snapshotImage, 0, 0)
        } catch(e) {}
      } else {
        ctx.fillStyle = '#000'
        ctx.fillRect(0, 0, canvas.width, canvas.height)
      }

      this.dragState.canvasScale = {
        x: canvas.width / canvas.offsetWidth,
        y: canvas.height / canvas.offsetHeight
      }

      if (!this.selectedCounter.lines) return

      this.selectedCounter.lines.forEach((line, index) => {
        // Получение координат
        const startX = line.start[0]
        const startY = line.start[1]
        const endX = line.end[0]
        const endY = line.end[1]

        const color = `hsl(${(index * 360) / this.selectedCounter.lines.length}, 70%, 60%)`
        ctx.strokeStyle = color
        ctx.lineWidth = 3

        // Рисуем линию
        ctx.beginPath()
        ctx.moveTo(startX, startY)
        ctx.lineTo(endX, endY)
        ctx.stroke()

        // Точки на концах
        ctx.fillStyle = color
        ctx.beginPath(); ctx.arc(startX, startY, 6, 0, 2 * Math.PI); ctx.fill()
        ctx.beginPath(); ctx.arc(endX, endY, 6, 0, 2 * Math.PI); ctx.fill()

        // Рисуем стрелку направления (вектор перпендикуляра)
        const midX = (startX + endX) / 2
        const midY = (startY + endY) / 2
        const dx = endX - startX
        const dy = endY - startY
        const perpX = -dy
        const perpY = dx
        const perpLength = Math.sqrt(perpX * perpX + perpY * perpY)
        const arrowLength = 20
        const arrowHeadSize = 8

        if (perpLength > 0) {
          const normPerpX = (perpX / perpLength) * arrowLength
          const normPerpY = (perpY / perpLength) * arrowLength
          const arrowEndX = midX + normPerpX
          const arrowEndY = midY + normPerpY

          ctx.strokeStyle = color; ctx.fillStyle = color; ctx.lineWidth = 2
          ctx.beginPath()
          ctx.moveTo(midX, midY)
          ctx.lineTo(arrowEndX, arrowEndY)
          ctx.stroke()

          const angle = Math.atan2(normPerpY, normPerpX)
          ctx.beginPath()
          ctx.moveTo(arrowEndX, arrowEndY)
          ctx.lineTo(arrowEndX - arrowHeadSize * Math.cos(angle - Math.PI/6), arrowEndY - arrowHeadSize * Math.sin(angle - Math.PI/6))
          ctx.lineTo(arrowEndX - arrowHeadSize * Math.cos(angle + Math.PI/6), arrowEndY - arrowHeadSize * Math.sin(angle + Math.PI/6))
          ctx.closePath()
          ctx.fill()
        }
      })
    },
    loadSnapshotToCanvas() {
      if (!this.selectedCounter?.id || !this.$refs.streamCanvas) return

      this.snapshotImage.onload = () => {
        const canvas = this.$refs.streamCanvas
        if (canvas) {
          canvas.width = this.snapshotImage.width
          canvas.height = this.snapshotImage.height
          this.drawCanvas()
        }
      }

      this.snapshotImage.onerror = this.handleStreamError
      this.snapshotImage.crossOrigin = 'anonymous'
      this.snapshotImage.src = `${API_BASE_URL}api/counters/${this.selectedCounter.id}/stream?jwt=${this.accessToken}&snapshot=${Date.now()}`
    },
    getCanvasCoordinates(event) {
      const canvas = this.$refs.streamCanvas
      if (!canvas) return { x: 0, y: 0 }
      const rect = canvas.getBoundingClientRect()
      return {
        x: (event.clientX - rect.left) * this.dragState.canvasScale.x,
        y: (event.clientY - rect.top) * this.dragState.canvasScale.y
      }
    },
    findNearestPoint(x, y, threshold = 15) {
      if (!this.selectedCounter?.lines) return null
      for (let i = 0; i < this.selectedCounter.lines.length; i++) {
        const line = this.selectedCounter.lines[i]

        const distStart = Math.sqrt(Math.pow(x - line.start[0], 2) + Math.pow(y - line.start[1], 2))
        if (distStart <= threshold) return { lineIndex: i, pointType: 'start' }

        const distEnd = Math.sqrt(Math.pow(x - line.end[0], 2) + Math.pow(y - line.end[1], 2))
        if (distEnd <= threshold) return { lineIndex: i, pointType: 'end' }
      }
      return null
    },
    handleCanvasMouseDown(event) {
      if (!this.isEditMode) return
      const coords = this.getCanvasCoordinates(event)
      const nearest = this.findNearestPoint(coords.x, coords.y)
      if (nearest) {
        this.dragState.isDragging = true
        this.dragState.lineIndex = nearest.lineIndex
        this.dragState.pointType = nearest.pointType
      }
    },
    handleCanvasMouseMove(event) {
      if (!this.dragState.isDragging || !this.isEditMode || !this.selectedCounter?.lines) return
      if (this.dragState.lineIndex === null || this.dragState.pointType === null) return

      const coords = this.getCanvasCoordinates(event)
      const line = this.selectedCounter.lines[this.dragState.lineIndex]

      if (this.dragState.pointType === 'start') {
        line.start[0] = Math.round(coords.x)
        line.start[1] = Math.round(coords.y)
      } else {
        line.end[0] = Math.round(coords.x)
        line.end[1] = Math.round(coords.y)
      }
      this.drawCanvas()
    },
    handleCanvasMouseUp() {
      this.dragState.isDragging = false
      this.dragState.lineIndex = null
      this.dragState.pointType = null
    }
  }
}
</script>

<style scoped>
.stream-wrapper {
  background: #000;
  border-radius: 0.5rem;
  overflow: hidden;
  position: relative;
  width: 100%;
}
.stream-element {
  width: 100%;
  height: auto;
  display: block;
}
.canvas-edit {
  cursor: crosshair;
}
.hint-text {
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  margin-top: 0.5rem;
}
</style>