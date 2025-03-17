import time
from queue import Queue

import cv2
from ultralytics import YOLO


def detect_objects(queue: Queue):
    model = YOLO("yolov8n.pt").to("cuda").half()
    frame_count = 0

    while True:
        time.sleep(0.1)
        if not queue.empty():
            frame_count += 1
            print(f"Processing frame {frame_count}")
            frame = queue.get()
            results = model(frame, imgsz=320)

            for result in results:
                for box in result.boxes:
                    x1, y1, x2, y2 = map(int, box.xyxy[0])  # Bounding box coordinates
                    label = model.names[int(box.cls[0])]  # Object class label

                    # Draw bounding box and label
                    cv2.rectangle(frame, (x1, y1), (x2, y2), (0, 255, 0), 2)
                    cv2.putText(frame, label, (x1, y1 - 10), cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 0, 255), 2)


            cv2.imwrite(f"imgs/received_frame_{frame_count}.jpg", frame)
