#!/bin/bash

# ==========================================
# --- ОСНОВНЫЕ НАСТРОЙКИ СИСТЕМЫ ---
# ==========================================
SERVER_IP="192.168.1.104"      # IP-адрес хоста (сервера) на котором запущены контейнеры.

# Настройки портов сервисов
FRONT_PORT="8080"              # Порт Frontend
MANAGER_PORT="9000"            # Порт Manager API
WORKER_PORT="8000"             # Порт Worker (YOLO26)
WORKER_API_KEY="my_super_secret_api_key" # API Ключ для Worker

# Настройки PostgreSQL
DB_USER="people_admin"         # Имя пользователя БД
DB_PASS="SecurePassword123!"   # Пароль БД
DB_NAME="people_counter_db"    # Название базы данных
DB_PORT="5434"                 # Порт БД на хосте
# ==========================================

MANAGER_CONF="./PeopleCounter_Manager/config.json"
MANAGER_EX="./PeopleCounter_Manager/example.config.json"
WORKER_CONF="./PeopleCounter_WorkerYOLO26/config.ini"
WORKER_EX="./PeopleCounter_WorkerYOLO26/config.ini.example"
ENV_FILE=".env"

echo "=== Подготовка окружения PeopleCounter ==="

# 0. Создаем .env файл для Docker Compose
echo "[+] Генерация .env файла для docker-compose"
cat <<EOF > $ENV_FILE
SERVER_IP=${SERVER_IP}
FRONT_PORT=${FRONT_PORT}
MANAGER_PORT=${MANAGER_PORT}
DB_USER=${DB_USER}
DB_PASS=${DB_PASS}
DB_NAME=${DB_NAME}
DB_PORT=${DB_PORT}
EOF

# 1. Защита от бага Docker (удаляем случайно созданные директории)
if [ -d "$MANAGER_CONF" ]; then
    rm -rf "$MANAGER_CONF"
fi
if [ -d "$WORKER_CONF" ]; then
    rm -rf "$WORKER_CONF"
fi

# 2. Создаем и настраиваем конфиг Менеджера
if [ ! -f "$MANAGER_CONF" ]; then
    cp "$MANAGER_EX" "$MANAGER_CONF"
    echo "[+] Создан $MANAGER_CONF"

    DB_CONN="postgres://${DB_USER}:${DB_PASS}@localhost:${DB_PORT}/${DB_NAME}?sslmode=disable"

    # Меняем строку подключения к БД
    sed -i -E "s|\"db_conn_string\":.*|\"db_conn_string\": \"${DB_CONN}\"|" "$MANAGER_CONF"

    # Меняем порт менеджера
    sed -i -E "s|\"api_port\":.*|\"api_port\": ${MANAGER_PORT},|" "$MANAGER_CONF"

    echo "[+] Настроен config.json (БД: ${DB_PORT}, API Port: ${MANAGER_PORT})"
else
    echo "[i] Файл $MANAGER_CONF уже существует, пропускаем создание."
fi

# 3. Создаем и настраиваем конфиг Воркера
if [ ! -f "$WORKER_CONF" ]; then
    cp "$WORKER_EX" "$WORKER_CONF"
    echo "[+] Создан $WORKER_CONF"

    # В .ini файле меняем значения port и api_key
    sed -i -E "s|^port[ \t]*=.*|port = ${WORKER_PORT}|" "$WORKER_CONF"
    sed -i -E "s|^api_key[ \t]*=.*|api_key = ${WORKER_API_KEY}|" "$WORKER_CONF"

    echo "[+] Настроен config.ini (Worker Port: ${WORKER_PORT}, API Key задан)"
else
    echo "[i] Файл $WORKER_CONF уже существует, пропускаем создание."
fi

# 4. Сборка и запуск контейнеров
echo "=== Запуск сборки Docker ==="
docker-compose up -d --build

# 5. Вывод информации
echo "======================================"
echo "✅ Проект успешно запущен!"
echo "Frontend:             http://${SERVER_IP}:${FRONT_PORT}"
echo "Manager Swagger API:  http://${SERVER_IP}:${MANAGER_PORT}/swagger/index.html"
echo "Worker YOLO API:      http://${SERVER_IP}:${WORKER_PORT}"
echo "PostgreSQL:  localhost:${DB_PORT} (БД: ${DB_NAME})"
echo "======================================"
