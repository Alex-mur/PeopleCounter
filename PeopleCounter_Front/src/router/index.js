import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginPage.vue')
  },
  {
    path: '/',
    component: () => import('@/layouts/AppLayout.vue'),
    children: [
      {
        path: '',
        redirect: '/statistics'
      },
      {
        path: 'statistics',
        name: 'Statistics',
        component: () => import('@/views/StatisticsPage.vue')
      },
      {
        path: 'management',
        name: 'Management',
        component: () => import('@/views/ManagementPage.vue'),
        meta: {
          requiresAdmin: true
        }
      },
      {
        path: 'management/service/:serviceId/counters',
        name: 'ServiceCounters',
        component: () => import('@/views/ServiceCountersPage.vue'),
        meta: {
          requiresAdmin: true
        }
      }
    ]
  }
]

const router = createRouter({
  routes: routes,
  history: createWebHistory(import.meta.env.BASE_URL)
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  const token = localStorage.getItem('accessToken')

  if (!token && to.name !== 'Login') {
    return next({ name: 'Login' })
  }

  if (token && to.name === 'Login') {
    return next({ name: 'Statistics' })
  }

  if (token && !authStore.user) {
    try {
      await authStore.fetchUser()
    } catch (e) {
      return next({ name: 'Login' })
    }
  }

  if (to.meta.requiresAdmin) {
    if (authStore.isAdmin) {
      return next()
    } else {
      return next({ name: 'Statistics' })
    }
  }

  return next()
})

export default router