import uvicorn
from anyio import sleep
from fastapi import FastAPI, HTTPException, Query, Depends, Security
from fastapi.security.api_key import APIKeyHeader
from fastapi.responses import StreamingResponse
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List, Optional, Tuple, Dict
import threading
import time
from datetime import datetime, timedelta
import duckdb
import json
import configparser
import os
import logging
from contextlib import asynccontextmanager
from people_counter_cpu import PeopleCounter


class LineConfig(BaseModel):
    id: int
    name: str
    start: Tuple[int, int]
    end: Tuple[int, int]
    group_id: int


class LinesGroup(BaseModel):
    id: int
    description: Optional[str] = None


class CounterConfig(BaseModel):
    id: int
    name: str
    description: Optional[str] = None
    url: str
    vid_stride: int
    lines: List[LineConfig]
    groups: List[LinesGroup]


class DuckDBLogHandler(logging.Handler):
    def __init__(self, connection):
        super().__init__()
        self.connection = connection

    def emit(self, record):
        try:
            log_message = self.format(record)
            now = datetime.now()
            self.connection.execute(
                "INSERT INTO logs (timestamp, message) VALUES (?, ?)",
                (now, log_message)
            )
        except Exception:
            self.handleError(record)


started_at = datetime.now()

CONFIG_FILE = "config.ini"
config_parser = configparser.ConfigParser()

if not os.path.exists(CONFIG_FILE):
    config_parser['SERVICE'] = {
        'port': '8000',
        'api_key': 'secretkey123',
        'keep_logs_days': '60'
    }
    with open(CONFIG_FILE, 'w') as configfile:
        config_parser.write(configfile)

config_parser.read(CONFIG_FILE)
SERVICE_PORT = int(config_parser.get('SERVICE', 'port', fallback='8000'))
API_KEY = config_parser.get('SERVICE', 'api_key', fallback='secretkey123')

API_KEY_NAME = "X-Api-Key"
api_key_header = APIKeyHeader(name=API_KEY_NAME, auto_error=False)
KEEP_LOGS_DAYS = int(config_parser.get('SERVICE', 'keep_logs_days', fallback='60'))

async def get_api_key(
    api_key_header: Optional[str] = Security(api_key_header),
    api_key_query: Optional[str] = Query(None, alias="api_key")
):
    if api_key_header == API_KEY:
        return api_key_header
    if api_key_query == API_KEY:
        return api_key_query
    raise HTTPException(status_code=403, detail="Неверный API ключ")

DB_FILE = "counters.duckdb"
db_conn = duckdb.connect(DB_FILE)

db_conn.execute("""
CREATE TABLE IF NOT EXISTS passes (
    counter_id VARCHAR,
    group_id INTEGER,
    timestamp TIMESTAMP,
    pass_count INTEGER
)
""")

db_conn.execute("""
CREATE TABLE IF NOT EXISTS counters_config (
    id INTEGER PRIMARY KEY,
    name VARCHAR,
    description VARCHAR,
    url VARCHAR,
    vid_stride INTEGER,
    lines VARCHAR,
    groups VARCHAR
)
""")

db_conn.execute("""
CREATE TABLE IF NOT EXISTS logs (
    timestamp TIMESTAMP,
    message VARCHAR
)
""")

counters: Dict[int, PeopleCounter] = {}
last_counts: Dict[int, Dict[int, int]] = {}
lock = threading.Lock()

logger = logging.getLogger("PeopleCounterServiceLogger")
logger.setLevel(logging.INFO)
logger.propagate = False

if logger.hasHandlers():
    logger.handlers.clear()

console_handler = logging.StreamHandler()
console_handler.setFormatter(logging.Formatter('%(asctime)s - %(levelname)s - %(message)s'))
logger.addHandler(console_handler)

db_handler = DuckDBLogHandler(db_conn)
db_handler.setFormatter(logging.Formatter('%(message)s'))
logger.addHandler(db_handler)


def run_counter(counter: PeopleCounter):
    counter.start()


def cleanup_loop():
    while True:
        now = datetime.now()

        # Очистка старых данных счетчиков
        three_months_ago = now - timedelta(days=90)
        db_conn.execute("DELETE FROM passes WHERE timestamp < ?", (three_months_ago,))

        # Очистка старых логов
        logs_threshold = now - timedelta(days=KEEP_LOGS_DAYS)
        db_conn.execute("DELETE FROM logs WHERE timestamp < ?", (logs_threshold,))

        # Сброс счетчиков
        with lock:
            for cid, counter in list(counters.items()):
                if cid in last_counts:
                    for gid in list(last_counts[cid].keys()):
                        last_counts[cid][gid] = 0
                    if hasattr(counter, 'group_stats'):
                        for gid in counter.group_stats.keys():
                            counter.group_stats[gid]['count'] = 0

        time.sleep(3600)


def stats_collector_loop():
    while True:
        now = datetime.now()
        with lock:
            current_counters = list(counters.items())

        for cid, counter in current_counters:
            stats = counter.get_stats()
            for stat in stats:
                gid = stat['group_id']
                curr_count = stat['count']

                if cid not in last_counts:
                    last_counts[cid] = {}

                prev_count = last_counts[cid].get(gid, 0)

                if curr_count > prev_count:
                    delta = curr_count - prev_count
                    last_counts[cid][gid] = curr_count

                    db_conn.execute(
                        "INSERT INTO passes VALUES (?, ?, ?, ?)",
                        (str(cid), gid, now, delta)
                    )

        time.sleep(60)


def restart_counters_loop():
    while True:
        with lock:
            current_counters = list(counters.items())

        for cid, counter in current_counters:
            if not counter.is_running():
                logger.warning(f"Counter {cid} is not running. Restarting...")
                threading.Thread(target=run_counter, args=(counter,), daemon=True).start()

        time.sleep(60)


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Starting background tasks...")
    collector_thread = threading.Thread(target=stats_collector_loop, daemon=True)
    collector_thread.start()
    logger.info("Stats collector started.")

    restarter_thread = threading.Thread(target=restart_counters_loop, daemon=True)
    restarter_thread.start()
    logger.info("Counter restarter started.")

    cleanup_thread = threading.Thread(target=cleanup_loop, daemon=True)
    cleanup_thread.start()
    logger.info("Cleanup task started.")

    configs = db_conn.execute("SELECT id, name, description, url, vid_stride, lines FROM counters_config").fetchall()
    for row in configs:
        cid = row[0]
        url = row[3]
        vid_stride = row[4]
        lines_data = json.loads(row[5])

        on_connect = lambda c=cid, n=row[1]: logger.info(f"Counter {c} ({n}) stream connected successfully.")

        counter = PeopleCounter(input_source=url, vid_stride=vid_stride, lines=lines_data, on_connected_callback=on_connect)
        counters[cid] = counter
        last_counts[cid] = {}

        t = threading.Thread(target=run_counter, args=(counter,), daemon=True)
        t.start()
        logger.info(f"Loaded and started counter {cid} from DB.")

    yield

    logger.info("Shutting down... Stopping counters.")
    for counter in counters.values():
        counter.stop()
        counter.cleanup()


app = FastAPI(title="PeopleCounterWorker (YOLO26)", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/api/info", summary="Получить информацию о воркере")
def get_info(api_key: str = Depends(get_api_key)):
    return {
        "name": "PeopleCounterWorker (YOLO26)",
        "version": "1.0.1",
        "started_at": started_at.strftime("%Y-%m-%dT%H:%M:%S")
    }


@app.post("/api/counters", summary="Добавить новую камеру и начать подсчет",
          description="Если id уже существует, вернется ошибка 400.")
def create_counter(config: CounterConfig, api_key: str = Depends(get_api_key)):
    with lock:
        if config.id in counters:
            raise HTTPException(status_code=400, detail=f"Счетчик {config.id} уже запущен")

        existing = db_conn.execute("SELECT id FROM counters_config WHERE id = ?", (config.id,)).fetchone()
        if existing:
            raise HTTPException(status_code=400, detail=f"Счетчик {config.id} уже существует в БД")

        if config.lines and not config.groups:
            raise HTTPException(status_code=400, detail="Невозможно добавить линии без указания хотя бы одной группы.")

        valid_group_ids = {g.id for g in config.groups}
        for line in config.lines:
            if line.group_id not in valid_group_ids:
                raise HTTPException(
                    status_code=400,
                    detail=f"Линия {line.name} ссылается на несуществующую группу {line.group_id}"
                )

        lines_data = [l.model_dump() for l in config.lines]
        lines_json = json.dumps(lines_data)
        groups_data = [g.model_dump() for g in config.groups]
        groups_json = json.dumps(groups_data)

        db_conn.execute(
            "INSERT INTO counters_config (id, name, description, url, vid_stride, lines, groups) VALUES (?, ?, ?, ?, ?, ?, ?)",
            (config.id, config.name, config.description, config.url, config.vid_stride, lines_json, groups_json)
        )

        on_connect = lambda c=config.id, n=config.name: logger.info(
            f"Счетчик [{c}] {n} успешно подключился к видеопотоку.")

        counter = PeopleCounter(input_source=config.url, vid_stride=config.vid_stride, lines=lines_data, on_connected_callback=on_connect)
        counters[config.id] = counter
        last_counts[config.id] = {}

    t = threading.Thread(target=run_counter, args=(counter,), daemon=True)
    t.start()

    return {"status": "started_and_saved", "id": config.id, "name": config.name}


@app.put("/api/counters/{counter_id}", summary="Обновить настройки счетчика",
         description="Полностью обновляет параметры счетчика (включая линии и группы). Если счетчик был запущен, он перезапустится.")
def update_counter(counter_id: int, config: CounterConfig, api_key: str = Depends(get_api_key)):
    with lock:
        if counter_id != config.id:
            raise HTTPException(status_code=400, detail="ID в URL и в теле запроса не совпадают")

        existing = db_conn.execute("SELECT id FROM counters_config WHERE id = ?", (counter_id,)).fetchone()
        if not existing:
            raise HTTPException(status_code=404, detail=f"Счетчик с id {counter_id} не найден в БД.")

        if config.lines and not config.groups:
            raise HTTPException(status_code=400, detail="Невозможно добавить линии без указания хотя бы одной группы.")

        valid_group_ids = {g.id for g in config.groups} if config.groups else set()
        for line in config.lines:
            if line.group_id not in valid_group_ids:
                raise HTTPException(
                    status_code=400,
                    detail=f"Линия {line.name} ссылается на несуществующую группу {line.group_id}"
                )

        lines_data = [l.model_dump() for l in config.lines]
        lines_json = json.dumps(lines_data)

        groups_data = [g.model_dump() for g in config.groups] if config.groups else []
        groups_json = json.dumps(groups_data)

        new_group_ids = [g.id for g in config.groups] if config.groups else []

        if new_group_ids:
            placeholders = ','.join(['?'] * len(new_group_ids))
            db_conn.execute(
                f"DELETE FROM passes WHERE counter_id = ? AND group_id NOT IN ({placeholders})",
                [str(counter_id)] + new_group_ids
            )
        else:
            db_conn.execute("DELETE FROM passes WHERE counter_id = ?", (str(counter_id),))

        db_conn.execute(
            "UPDATE counters_config SET name=?, description=?, url=?, vid_stride=?, lines=?, groups=? WHERE id=?",
            (config.name, config.description, config.url, config.vid_stride, lines_json, groups_json, counter_id)
        )

        is_running = False
        if counter_id in counters:
            is_running = True
            counter = counters[counter_id]
            counter.stop()
            counter.cleanup()
            del counters[counter_id]

        if counter_id in last_counts:
            del last_counts[counter_id]

        if is_running:
            on_connect = lambda c=config.id, n=config.name: logger.info(
                f"Counter {c} ({n}) stream reconnected successfully after update."
            )
            new_counter = PeopleCounter(input_source=config.url, vid_stride=config.vid_stride, lines=lines_data, on_connected_callback=on_connect)
            counters[config.id] = new_counter
            last_counts[config.id] = {}

            t = threading.Thread(target=run_counter, args=(new_counter,), daemon=True)
            t.start()

    return {"status": "updated_and_restarted" if is_running else "updated", "id": config.id, "name": config.name}


@app.get("/api/counters", summary="Список всех счетчиков")
def list_counters(api_key: str = Depends(get_api_key)):
    configs = db_conn.execute("SELECT id, name, description, url, vid_stride, lines, groups FROM counters_config").fetchall()
    res = []

    with lock:
        for row in configs:
            cid = row[0]
            name = row[1]
            description = row[2]
            url = row[3]
            vid_stride = row[4]
            lines = json.loads(row[5]) if row[5] else []
            groups = json.loads(row[6]) if len(row) > 6 and row[6] is not None else []

            is_running = False
            if cid in counters:
                is_running = counters[cid].is_running()

            res.append({
                "id": cid,
                "name": name,
                "description": description,
                "url": url,
                "vid_stride": vid_stride,
                "lines": lines,
                "groups": groups,
                "running": is_running
            })
    return res


@app.delete("/api/counters/{counter_id}", summary="Удалить счетчик")
def delete_counter(counter_id: int, api_key: str = Depends(get_api_key)):
    with lock:
        existing = db_conn.execute("SELECT id FROM counters_config WHERE id = ?", (counter_id,)).fetchone()
        if not existing:
            raise HTTPException(status_code=404, detail=f"Счетчик с id {counter_id} не найден в БД.")

        db_conn.execute("DELETE FROM counters_config WHERE id = ?", (counter_id,))
        db_conn.execute("DELETE FROM passes WHERE counter_id = ?", (str(counter_id),))

        if counter_id in counters:
            counter = counters[counter_id]
            counter.stop()
            counter.cleanup()
            del counters[counter_id]

        if counter_id in last_counts:
            del last_counts[counter_id]

    return {"status": "stopped_and_deleted", "id": counter_id}


@app.get("/api/counters/{counter_id}/stream", summary="Получить MJPEG стрим")
def get_stream(counter_id: int, api_key: str = Depends(get_api_key)):
    with lock:
        if counter_id not in counters:
            raise HTTPException(status_code=404, detail="Counter not found")
        counter = counters[counter_id]

    def generate_frames():
        while counter.is_running():
            frame = counter.get_current_frame()
            if frame:
                yield (b'--frame\r\n'
                       b'Content-Type: image/jpeg\r\n\r\n' + frame + b'\r\n')
            time.sleep(0.04)

    return StreamingResponse(
        generate_frames(),
        media_type='multipart/x-mixed-replace; boundary=frame'
    )


@app.get("/api/counters/{counter_id}/stats", summary="Получить агрегированную статистику")
def get_stats(counter_id: int, period: str = Query('hour', description="month, week, day, hour"),
              api_key: str = Depends(get_api_key)):
    valid_periods = ['month', 'week', 'day', 'hour']
    if period not in valid_periods:
        raise HTTPException(status_code=400, detail="Invalid period. Must be month, week, day, or hour.")

    query = f"""
    SELECT
        date_trunc('{period}', timestamp) AS bucket,
        group_id,
        SUM(pass_count) AS total_passes
    FROM passes
    WHERE counter_id = ?
    GROUP BY bucket, group_id
    ORDER BY bucket DESC
    """

    realtime_query = """
    SELECT group_id, SUM(pass_count)
    FROM passes
    WHERE counter_id = ? AND timestamp >= date_trunc('hour', current_timestamp)
    GROUP BY group_id
    """

    try:
        with lock:
            results = db_conn.execute(query, (str(counter_id),)).fetchall()
            rt_results = db_conn.execute(realtime_query, (str(counter_id),)).fetchall()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

    history = []
    for row in results:
        history.append({
            "period": row[0].isoformat() if row[0] else None,
            "group_id": int(row[1]),
            "passes": int(row[2])
        })

    realtime_data = [{"group_id": int(row[0]), "passes": int(row[1])} for row in rt_results]

    return {
        "counter_id": counter_id,
        "aggregation_period": period,
        "history": history,
        "realtime_current_hour": realtime_data
    }


@app.get("/api/logs", summary="Получить системные логи")
def get_logs(
        start: datetime = Query(..., description="Начало периода (ISO 8601, например: 2026-03-04T10:00:00)"),
        end: Optional[datetime] = Query(None, description="Конец периода (По умолчанию текущее время)"),
        api_key: str = Depends(get_api_key)
):
    if end is None:
        end = datetime.now()

    if start > end:
        raise HTTPException(status_code=400, detail="Начало периода должно быть меньше конца.")

    query = """
    SELECT timestamp, message
    FROM logs
    WHERE timestamp >= ? AND timestamp <= ?
    ORDER BY timestamp DESC
    """
    try:
        results = db_conn.execute(query, (start, end)).fetchall()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

    logs = []
    for row in results:
        logs.append({
            "timestamp": row[0].isoformat() if row[0] else None,
            "message": row[1]
        })

    return {
        "start": start.isoformat(),
        "end": end.isoformat(),
        "total_records": len(logs),
        "logs": logs
    }


if __name__ == "__main__":
    uvicorn.run("people_counter_service:app", host="0.0.0.0", port=SERVICE_PORT, reload=False)
