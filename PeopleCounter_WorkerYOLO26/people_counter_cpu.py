import cv2
import numpy as np
import time
import math
import logging
from collections import deque, defaultdict
from ultralytics import YOLO


class YOLOStreamWarningFilter(logging.Filter):
    def filter(self, record):
        msg = record.getMessage()
        if "Waiting for stream" in msg or "Video stream unresponsive" in msg:
            return False
        return True

logging.getLogger("ultralytics").addFilter(YOLOStreamWarningFilter())


class PeopleCounter:
    def __init__(self, input_source=None, confidence=0.1, vid_stride=4, lines=None, on_connected_callback=None):
        self.input_source = input_source
        self.confidence = confidence
        self.on_connected_callback = on_connected_callback
        self.vid_stride = vid_stride

        raw_lines = lines if lines is not None else []
        self.line_stats = []

        self.track_line_state = defaultdict(lambda: defaultdict(bool))
        self.group_stats = {}

        for i, line_data in enumerate(raw_lines[:6]):
            group_id = line_data.get('group_id', i)
            self.line_stats.append({
                'id': line_data.get('id', i),
                'name': line_data.get('name', f'Line {i + 1}'),
                'start': line_data.get('start', (0, 0)),
                'end': line_data.get('end', (640, 360)),
                'group_id': group_id,
            })

            if group_id not in self.group_stats:
                self.group_stats[group_id] = {
                    'count': 0,
                    'lines': []
                }
            self.group_stats[group_id]['lines'].append(line_data.get('id', i))

        self.model = YOLO('model/yolo26n_openvino_model', task='detect')

        self.running = False
        self.current_frame = None
        self.track_history = defaultdict(lambda: deque(maxlen=40))


    def start(self):
        self.running = True
        try:
            temp_vs = cv2.VideoCapture(self.input_source)
            fps_input = temp_vs.get(cv2.CAP_PROP_FPS)
            if not fps_input or fps_input <= 0:
                fps_input = 30.0
            temp_vs.release()

            results_generator = self.model.track(
                source=self.input_source,
                stream=True,
                persist=True,
                tracker="bytetrack.yaml",
                classes=[0],
                conf=self.confidence,
                iou=0.8,
                vid_stride=self.vid_stride,
                verbose=False
            )

            last_process_time = time.time()
            first_frame_received = False

            for result in results_generator:
                if not self.running:
                    break

                if not first_frame_received:
                    first_frame_received = True
                    if self.on_connected_callback:
                        self.on_connected_callback()

                current_time = time.time()
                time_diff = current_time - last_process_time
                user_fps = 1.0 / time_diff if time_diff > 0 else 0.0
                last_process_time = current_time

                frame = result.orig_img.copy()
                orig_H, orig_W = frame.shape[:2]

                scale = 1.0
                if orig_W > 640:
                    scale = 640.0 / orig_W
                    new_w = 640
                    new_h = int(orig_H * scale)
                    frame = cv2.resize(frame, (new_w, new_h))

                cv2.putText(frame, f"FPS: {user_fps:.1f}", (10, 30), cv2.FONT_HERSHEY_SIMPLEX, 0.6, (0, 0, 255), 1)
                cv2.putText(frame, f"vid_stride: {self.vid_stride}", (10, 60), cv2.FONT_HERSHEY_SIMPLEX, 0.6,
                            (0, 0, 255), 1)

                for l_stat in self.line_stats:
                    p_start = l_stat['start']
                    p_end = l_stat['end']
                    name = l_stat['name']
                    group_id = l_stat['group_id']
                    group_count = self.group_stats[group_id]['count']

                    # Рисуем базовую линию
                    cv2.line(frame, p_start, p_end, (0, 255, 255), 2)

                    dx = p_end[0] - p_start[0]
                    dy = p_end[1] - p_start[1]

                    cx_line = (p_start[0] + p_end[0]) // 2
                    cy_line = (p_start[1] + p_end[1]) // 2

                    # Вектор нормали (Направление IN)
                    in_dx, in_dy = -dy, dx
                    length = math.hypot(in_dx, in_dy)

                    if length > 0:
                        in_dx /= length
                        in_dy /= length
                        arrow_len = 30
                        arrow_end = (int(cx_line + in_dx * arrow_len), int(cy_line + in_dy * arrow_len))

                        # Стрелка IN
                        cv2.arrowedLine(frame, (cx_line, cy_line), arrow_end, (0, 0, 255), 2, tipLength=0.3)

                    # Пишем имя линии и счетчик над началом линии
                    text = f"{name}: {group_count}"
                    cv2.putText(frame, text, (cx_line, cy_line),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.4, (0, 0, 255), 1)

                if result.boxes is not None and result.boxes.id is not None:
                    boxes = result.boxes.xyxy.cpu().numpy() * scale
                    track_ids = result.boxes.id.int().cpu().numpy()

                    for box, tid in zip(boxes, track_ids):
                        x1, y1, x2, y2 = map(int, box)
                        cx, cy = (x1 + x2) // 2, (y1 + y2) // 2

                        np.random.seed(int(tid) % 1000)
                        color = tuple(int(c) for c in np.random.randint(100, 230, 3))

                        cv2.rectangle(frame, (x1, y1), (x2, y2), color, 2)
                        cv2.putText(frame, f'ID:{tid}', (x1, max(14, y1 - 8)),
                                    cv2.FONT_HERSHEY_SIMPLEX, 0.45, color, 2)

                        track = self.track_history[tid]
                        if track:
                            prev_cx, prev_cy = track[-1]

                            # Проверяем пересечение с каждой линией индивидуально
                            for l_stat in self.line_stats:
                                line_id = l_stat['id']
                                group_id = l_stat['group_id']
                                p_start = l_stat['start']
                                p_end = l_stat['end']

                                dx = p_end[0] - p_start[0]
                                dy = p_end[1] - p_start[1]

                                prev_pos = dx * (prev_cy - p_start[1]) - dy * (prev_cx - p_start[0])
                                curr_pos = dx * (cy - p_start[1]) - dy * (cx - p_start[0])

                                # Интересует только переход в направлении "In" (от отрицательного положения к положительному)
                                if prev_pos < 0 and curr_pos >= 0:
                                    vx = cx - prev_cx
                                    vy = cy - prev_cy

                                    pos_start = vx * (p_start[1] - prev_cy) - vy * (p_start[0] - prev_cx)
                                    pos_end = vx * (p_end[1] - prev_cy) - vy * (p_end[0] - prev_cx)

                                    # Если отрезки (вектор движения и линия) пересекаются
                                    if pos_start * pos_end <= 0:
                                        # Mark this line as crossed for this track id
                                        self.track_line_state[tid][line_id] = True

                                        # Check if all lines in the group have been crossed
                                        all_crossed = True
                                        for g_line_id in self.group_stats[group_id]['lines']:
                                            if not self.track_line_state[tid].get(g_line_id, False):
                                                all_crossed = False
                                                break

                                        if all_crossed:
                                            self.group_stats[group_id]['count'] += 1
                                            # Reset track state for this group so they can be counted again
                                            for g_line_id in self.group_stats[group_id]['lines']:
                                                self.track_line_state[tid][g_line_id] = False

                        track.append((cx, cy))

                        pts = np.array(track).reshape(-1, 1, 2)
                        cv2.polylines(frame, [pts], False, color, 2)
                        cv2.circle(frame, (cx, cy), 4, (0, 0, 255), -1)

                active_ids = set(track_ids) if result.boxes is not None and result.boxes.id is not None else set()
                # To avoid memory leak, we would clean up track_line_state for IDs no longer present.
                # However, an ID might briefly disappear. A robust system would clean up after a timeout.
                # Here we just keep it simple.

                self.current_frame = frame

        except Exception as e:
            logging.error(f"Error in people counter: {e}")
        finally:
            self.running = False


    def get_stats(self):
        # Return stats by group instead of by line
        return [{'group_id': gid, 'count': stats['count']} for gid, stats in self.group_stats.items()]


    def get_current_frame(self):
        if self.current_frame is not None:
            ok, jpeg = cv2.imencode('.jpg', self.current_frame, [cv2.IMWRITE_JPEG_QUALITY, 85])
            return jpeg.tobytes() if ok else None
        return None


    def stop(self):
        self.running = False


    def is_running(self):
        return self.running


    def cleanup(self):
        self.running = False
        cv2.destroyAllWindows()
