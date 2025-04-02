FROM python:3.12-slim

WORKDIR /usr/src/app

COPY . .

RUN apt-get update && apt-get install -y \
    python3-dev pkg-config \
    libavformat-dev libavcodec-dev libavdevice-dev \
    libavutil-dev libswscale-dev libswresample-dev libavfilter-dev \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install --no-cache-dir -r requirements.txt

ENTRYPOINT ["python3", "server.py"]
