FROM python:3.10-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV YOLO_CONFIG_DIR=/tmp/Ultralytics

RUN apt-get update && apt-get install -y \
    libgl1 \
    libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir --default-timeout=120 -r requirements.txt

# Копирование исходного кода приложения
COPY . .

# Проброс порта, на котором работает сервис
EXPOSE 8000

# Запуск приложения через uvicorn
CMD ["uvicorn", "people_counter_service:app", "--host", "0.0.0.0", "--port", "8000"]
