<template>
  <div class="page-container">
    <div class="page-header">
      <h2 class="page-title">Статистика</h2>
    </div>
    <div class="card-panel">
      <div class="form-group mb-0">
        <label class="form-label">Выберите счетчики:</label>
        <div v-if="loading && counters.length === 0" class="text-muted">Загрузка счетчиков...</div>
        <div v-else-if="counters.length === 0" class="text-muted">Нет доступных счетчиков</div>
        <div v-else class="chips-container">
          <button
            type="button"
            class="chip-btn select-all-btn"
            :class="{ active: isAllSelected }"
            @click="selectAllCounters"
          >
            Выбрать все
          </button>
          <button
            type="button"
            v-for="counter in counters"
            :key="counter.id"
            class="chip-btn"
            :class="{ active: selectedCounterIds.includes(counter.id) }"
            @click="toggleCounter(counter.id)"
          >
            {{ counter.name }}
          </button>
        </div>
      </div>
    </div>
    <div class="card-panel">
      <div class="filters-row">
        <div class="form-group mb-0 date-input">
          <label class="form-label">С (начало):</label>
          <input type="datetime-local" v-model="dateStart" class="form-input" />
        </div>
        <div class="form-group mb-0 date-input">
          <label class="form-label">По (конец):</label>
          <input type="datetime-local" v-model="dateEnd" class="form-input" />
        </div>
        <div class="form-group mb-0 period-group">
          <label class="form-label">Период агрегации:</label>
          <div class="period-selector">
            <button
                type="button"
                v-for="p in periods"
                :key="p.value"
                class="period-btn"
                :class="{ active: currentPeriod === p.value }"
                @click="currentPeriod = p.value"
            >
              {{ p.label }}
            </button>
          </div>
        </div>
      </div>
    </div>
    <template v-if="tableData.length === 0">
      <div v-if="loading" class="loading">
        Загрузка данных...
      </div>
      <div v-else-if="selectedCounterIds.length === 0" class="empty-state">
        Выберите хотя бы один счетчик для отображения статистики.
      </div>
      <div v-else-if="!dateStart || !dateEnd" class="empty-state">
        Укажите начальную и конечную дату для загрузки и отображения статистики.
      </div>
      <div v-else class="empty-state">
        Нет данных для отображения за выбранный период.
      </div>
    </template>    
    <template v-else>
      <div class="data-wrapper" :class="{ 'is-updating': loading }">
        <div class="chart-container card-panel">
          <Line :data="chartData" :options="chartOptions" />
        </div>
        <div class="table-container">
          <table>
            <thead>
            <tr>
              <th>Период</th>
              <template v-for="counter in selectedCounters" :key="'th-c-' + counter.id">
                <th v-for="group in counter.groups" :key="'th-g-' + counter.id + '-' + group.id" class="text-right">
                  {{ counter.name }} ({{ group.description || `Группа ${group.id}` }})
                </th>
              </template>
            </tr>
            </thead>
            <tbody>
            <tr v-for="row in tableData" :key="row.period">
              <td class="font-medium">{{ row.formattedPeriod }}</td>
              <template v-for="counter in selectedCounters" :key="'td-c-' + counter.id">
                <td v-for="group in counter.groups" :key="'td-g-' + counter.id + '-' + group.id" class="text-right">
                  {{ row.groups[`${counter.id}_${group.id}`] || 0 }}
                </td>
              </template>
            </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
import api from '../api'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js'

ChartJS.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend
)

const formatDatetimeLocal = (d) => {
  const pad = (n) => n.toString().padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const lineColors = [
  '#8a5021', '#4e628c', '#5e6135', '#ba1a1a', '#755945',
  '#2a9d8f', '#e9c46a', '#f4a261', '#e76f51', '#264653'
]

export default {
  name: 'StatsView',
  components: {
    Line
  },
  data() {
    const now = new Date()
    const sevenDaysAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)

    return {
      counters: [],
      selectedCounterIds: [],
      statsData: [],
      loading: true,
      fetchId: 0, 
      currentPeriod: 'day',
      periods: [
        { value: 'hour', label: 'Час' },
        { value: 'day', label: 'День' },
        { value: 'week', label: 'Неделя' },
        { value: 'month', label: 'Месяц' },
        { value: 'year', label: 'Год' }
      ],
      dateStart: formatDatetimeLocal(sevenDaysAgo),
      dateEnd: formatDatetimeLocal(now)
    }
  },
  computed: {
    selectedCounters() {
      return this.counters.filter(c => this.selectedCounterIds.includes(c.id))
    },
    isAllSelected() {
      return this.counters.length > 0 && this.selectedCounterIds.length === this.counters.length
    },
    tableData() {
      if (!this.statsData.length) return []

      const grouped = this.statsData.reduce((acc, curr) => {
        if (!acc[curr.period]) acc[curr.period] = {}
        const key = `${curr.counter_id}_${curr.group_id}`
        acc[curr.period][key] = curr.passes
        return acc
      }, {})

      return Object.keys(grouped)
          .sort((a, b) => new Date(b).getTime() - new Date(a).getTime())
          .map(period => ({
            period,
            formattedPeriod: this.formatPeriod(period, this.currentPeriod),
            groups: grouped[period]
          }))
    },
    chartData() {
      const sortedKeys = [...this.tableData].reverse()
      const labels = sortedKeys.map(row => row.formattedPeriod)
      const datasets = []

      let colorIndex = 0
      this.selectedCounters.forEach(counter => {
        (counter.groups || []).forEach(group => {
          const color = lineColors[colorIndex % lineColors.length]
          datasets.push({
            label: `${counter.name} (${group.description || `Группа ${group.id}`})`,
            backgroundColor: color,
            borderColor: color,
            data: sortedKeys.map(row => row.groups[`${counter.id}_${group.id}`] || 0),
            tension: 0.3,
            borderWidth: 2,
            pointRadius: 4,
            pointHoverRadius: 6
          })
          colorIndex++
        })
      })

      return { labels, datasets }
    },
    chartOptions() {
      return {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            position: 'top',
            labels: {
              usePointStyle: true,
              font: { family: 'Roboto', size: 13 }
            }
          },
          tooltip: {
            mode: 'index',
            intersect: false,
            backgroundColor: 'rgba(34, 26, 21, 0.9)'
          }
        },
        scales: {
          y: {
            beginAtZero: true,
            grid: { color: 'rgba(132, 116, 106, 0.1)' },
            ticks: { precision: 0 }
          },
          x: {
            grid: { display: false }
          }
        },
        interaction: {
          mode: 'nearest',
          axis: 'x',
          intersect: false
        }
      }
    }
  },
  watch: {
    selectedCounterIds: {
      handler: 'loadStats',
      deep: true
    },
    currentPeriod: 'loadStats',
    dateStart: 'loadStats',
    dateEnd: 'loadStats'
  },
  async mounted() {
    await this.loadCounters()
    if (this.counters.length > 0) {
      this.selectedCounterIds = [this.counters[0].id]
    } else {
      this.loading = false
    }
  },
  methods: {
    toggleCounter(id) {
      const index = this.selectedCounterIds.indexOf(id)
      if (index > -1) {
        this.selectedCounterIds.splice(index, 1)
      } else {
        this.selectedCounterIds.push(id)
      }
    },
    selectAllCounters() {
      if (this.isAllSelected) {
        this.selectedCounterIds = []
      } else {
        this.selectedCounterIds = this.counters.map(c => c.id)
      }
    },
    async loadCounters() {
      this.loading = true
      try {
        this.counters = await api.getCounters()
      } catch (err) {
        console.error('Failed to load counters:', err)
      }
    },
    async loadStats() {
      if (this.selectedCounterIds.length === 0 || !this.dateStart || !this.dateEnd) {
        this.statsData = []
        this.loading = false
        return
      }

      const reqId = ++this.fetchId
      this.loading = true
      
      try {
        const startIso = new Date(this.dateStart).toISOString()
        const endIso = new Date(this.dateEnd).toISOString()

        const promises = this.selectedCounterIds.map(async (id) => {
          const response = await api.getCounterStats(
              id,
              this.currentPeriod,
              startIso,
              endIso
          )
          return (response.data || []).map(item => ({ ...item, counter_id: id }))
        })

        const results = await Promise.all(promises)

        if (reqId === this.fetchId) {
          this.statsData = results.flat()
        }
      } catch (err) {
        console.error('Failed to load stats:', err)
        if (reqId === this.fetchId) {
          this.statsData = []
        }
      } finally {
        if (reqId === this.fetchId) {
          this.loading = false
        }
      }
    },
    formatPeriod(dateStr, periodType) {
      const d = new Date(dateStr)
      if (periodType === 'hour') {
        return d.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
      }
      if (periodType === 'month') {
        return d.toLocaleString('ru-RU', { month: 'long', year: 'numeric' })
      }
      return d.toLocaleDateString('ru-RU')
    }
  }
}
</script>

<style scoped>

.data-wrapper {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  transition: opacity 0.2s ease-in-out;
}
.data-wrapper.is-updating {
  opacity: 0.5;
  pointer-events: none;
}

.chart-container { 
  height: 400px; 
  position: relative; 
  width: 100%; 
}

.chips-container { 
  display: flex; 
  flex-wrap: wrap; 
  gap: 0.75rem; 
  margin-top: 0.75rem; 
}
.chip-btn { 
  padding: 8px 16px; 
  background-color: var(--md-sys-color-surface-variant); 
  border: 1px solid var(--md-sys-color-outline-variant); 
  color: var(--md-sys-color-on-surface); 
  border-radius: 20px; 
  font-family: 'Roboto', sans-serif;
  font-weight: 500; 
  font-size: 0.875rem;
  cursor: pointer; 
  transition: all 0.2s; 
  white-space: nowrap; 
}
.chip-btn:hover { 
  background-color: var(--md-sys-color-surface-container-highest); 
}
.chip-btn.active { 
  background-color: var(--color-primary); 
  color: var(--md-sys-color-on-primary); 
  border-color: var(--color-primary); 
  box-shadow: var(--md-elevation-1);
}
.select-all-btn { 
  border-style: dashed; 
}
.text-muted {
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  margin-top: 0.5rem;
}

.filters-row { 
  display: flex; 
  flex-wrap: wrap; 
  gap: 1.5rem; 
  align-items: flex-end; 
}
.period-group { 
  display: flex; 
  flex-direction: column; 
  gap: 4px; 
}
.period-selector { 
  display: inline-flex; 
  gap: 0.25rem; 
  background-color: var(--md-sys-color-surface-variant); 
  padding: 0.25rem; 
  border-radius: 20px; 
  border: 1px solid var(--md-sys-color-outline-variant); 
}
.period-btn { 
  padding: 6px 16px; 
  background: transparent; 
  border: none; 
  color: var(--color-text-secondary); 
  border-radius: 16px; 
  font-weight: 500; 
  font-family: 'Roboto', sans-serif;
  font-size: 0.875rem;
  cursor: pointer; 
  transition: all 0.2s; 
}
.period-btn.active { 
  background-color: var(--color-primary); 
  color: var(--md-sys-color-on-primary); 
}

.date-input { 
  min-width: 220px; 
}

@media (max-width: 768px) {
  .chart-container { height: 300px; }
  .filters-row { flex-direction: column; align-items: stretch; gap: 1rem; }
  .period-selector { width: 100%; justify-content: space-between; }
  .period-btn { flex: 1; text-align: center; padding: 6px 4px; }
}
</style>