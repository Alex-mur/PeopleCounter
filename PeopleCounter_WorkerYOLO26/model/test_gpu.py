import torch
from ultralytics import YOLO

# Проверка PyTorch
print("CUDA доступна:", torch.cuda.is_available())
if torch.cuda.is_available():
    print("Название GPU:", torch.cuda.get_device_name(0))

# Проверка инициализации модели YOLO на GPU
model = YOLO('yolo26n.pt')
model.to('cuda') # Принудительно отправляем на GPU
print("Модель загружена на устройство:", model.device)