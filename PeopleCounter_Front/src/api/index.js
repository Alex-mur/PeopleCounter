export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:9000/';

let isRefreshing = false;

async function request(endpoint, options = {}) {
    let token = localStorage.getItem('accessToken');
    const headers = {
        'Content-Type': 'application/json',
        ...(options.headers || {}),
    };

    const isAuthRoute = endpoint.includes('login') || endpoint.includes('refresh');

    if (token && !isAuthRoute) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    try {
        let response = await fetch(`${API_BASE_URL}${endpoint}`, {
            ...options,
            headers,
        });

        if (response.status === 401 && !isAuthRoute) {
            if (!isRefreshing) {
                isRefreshing = true;
                try {
                    const refreshed = await refreshToken();

                    if (refreshed) {
                        token = localStorage.getItem('accessToken');
                        headers['Authorization'] = `Bearer ${token}`;

                        response = await fetch(`${API_BASE_URL}${endpoint}`, {
                            ...options,
                            headers,
                        });
                    } else {
                        forceLogout();
                        throw new Error('Сессия истекла. Пожалуйста, войдите снова.');
                    }
                } finally {
                    isRefreshing = false;
                }
            } else {
                throw new Error('Обновление токена уже в процессе');
            }
        }

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || 'Request failed');
        }

        if (response.status === 204) {
            return null;
        }

        return await response.json();
    } catch (error) {
        console.error(`API Error [${endpoint}]:`, error);
        throw error;
    }
}

async function refreshToken() {
    const rToken = localStorage.getItem('refreshToken');
    if (!rToken) return false;

    try {
        const response = await fetch(`${API_BASE_URL}api/refresh`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ refresh_token: rToken }),
        });

        if (!response.ok) {
            return false;
        }

        const data = await response.json();
        localStorage.setItem('accessToken', data.access_token);
        localStorage.setItem('refreshToken', data.refresh_token);
        return true;
    } catch (e) {
        return false;
    }
}

function forceLogout() {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    window.location.href = '/login';
}

const api = {
    login: async (loginStr, passwordStr) => {
        const data = await request('api/login', {
            method: 'POST',
            body: JSON.stringify({ login: loginStr, password: passwordStr }),
        });
        localStorage.setItem('accessToken', data.access_token);
        localStorage.setItem('refreshToken', data.refresh_token);
        return data;
    },

    // Пользователи
    getCurrentUser: () => request('api/user'),
    getUsers: () => request('api/users'),
    createUser: (user) => request('api/users', { method: 'POST', body: JSON.stringify(user) }),
    updateUser: (id, user) => request(`api/users/${id}`, { method: 'PUT', body: JSON.stringify(user) }),
    deleteUser: (id) => request(`api/users/${id}`, { method: 'DELETE' }),

    // Cчетчики пользователей
    getUserCounters: (id) => request(`api/users/${id}/counters`),
    setUserCounters: (id, counterIds) => request(`api/users/${id}/counters`, {
        method: 'POST',
        body: JSON.stringify({ counter_ids: counterIds })
    }),

    // Сервисы
    getServices: () => request('api/services'),
    createService: (service) => request('api/services', { method: 'POST', body: JSON.stringify(service) }),
    updateService: (id, service) => request(`api/services/${id}`, { method: 'PUT', body: JSON.stringify(service) }),
    deleteService: (id) => request(`api/services/${id}`, { method: 'DELETE' }),

    // Счетчики
    getCounters: () => request('api/counters'),
    createCounter: (counter) => request('api/counters', { method: 'POST', body: JSON.stringify(counter) }),
    updateCounter: (id, counter) => request(`api/counters/${id}`, { method: 'PUT', body: JSON.stringify(counter) }),
    deleteCounter: (id) => request(`api/counters/${id}`, { method: 'DELETE' }),

    // Статистика
    getCounterStats: (id, period = 'day', dateStart, dateEnd) => {
        let url = `api/counters/${id}/stats?period=${period}`;
        if (dateStart) {
            url += `&date_start=${encodeURIComponent(dateStart)}`;
        }
        if (dateEnd) {
            url += `&date_end=${encodeURIComponent(dateEnd)}`;
        }
        return request(url);
    }
};

export default api;