import logging
import time
from queue import Queue

import cv2
import torch
from ultralytics import YOLO


def process_frame(last_frame, new_frame):
    if last_frame is None:
        return True

    gray_last = cv2.cvtColor(last_frame, cv2.COLOR_BGR2GRAY)
    gray_current = cv2.cvtColor(new_frame, cv2.COLOR_BGR2GRAY)
    gray_current = cv2.resize(gray_current, (gray_last.shape[1], gray_last.shape[0]))

    diff = cv2.absdiff(gray_last, gray_current)
    non_zero_count = cv2.countNonZero(diff)

    if non_zero_count / diff.size < 0.5:
        return False
    return True


def detect_objects(frame_queue: Queue, results_queue: Queue):
    device = "cuda" if torch.cuda.is_available() else "cpu"
    logging.info(f"Using {device} for object detection")

    model = YOLO("yolov8n.pt").to(device)
    last_frame = None
    frame_count = 0

    while True:
        time.sleep(0.1)
        if not frame_queue.empty():
            frame = frame_queue.get()

            if not process_frame(last_frame, frame):
                continue

            last_frame = frame
            frame_count += 1
            logging.info(f"Processing frame {frame_count}")
            results = model(frame)

            detection_result = []
            for result in results:
                for box in result.boxes:
                    x1, y1, x2, y2 = map(int, box.xyxy[0])

                    label = model.names[int(box.cls[0])]
                    detection_result.append((label, x1, y1, x2, y2))

            results_queue.put(detection_result)


def dummy_detect_objects(frame_queue: Queue, results_queue: Queue):
    while True:
        time.sleep(0.1)
        if not frame_queue.empty():
            frame_queue.get()
            results_queue.put([("dummy", 200, 200, 200, 200)])
