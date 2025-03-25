import argparse
import asyncio
import json
import logging
import os
import queue
from threading import Thread

import aiohttp_cors
from aiohttp import web
from aiortc import MediaStreamTrack, RTCPeerConnection, RTCSessionDescription, RTCDataChannel, RTCConfiguration, \
    RTCIceServer
from av import VideoFrame

from obj_detection import dummy_detect_objects, detect_objects

logging.basicConfig(level=logging.INFO)

frame_queue = queue.Queue()
results_queue = queue.Queue()

config = RTCConfiguration([
    RTCIceServer(urls=f"turn:{os.getenv("TURN_URL")}",
                 username=os.getenv("TURN_USERNAME"),
                 credential=os.getenv("TURN_PASSWORD")),
])


async def handle_video_track(track: MediaStreamTrack):
    frame_count = 0

    while True:
        try:
            frame_count += 1
            frame = await asyncio.wait_for(track.recv(), timeout=30)
            if isinstance(frame, VideoFrame):
                frame = frame.to_ndarray(format="bgr24")

            frame_queue.put(frame)
            logging.info(f"Enqueued frame {frame_count}")
        except asyncio.TimeoutError:
            logging.warning("Timeout: No frames received, closing stream.")
            break
        except Exception as e:
            logging.error(f"Stream error: {e}")
            break


async def send_obj_coordinates(channel: RTCDataChannel):
    while True:
        await asyncio.sleep(0.1)
        if not results_queue.empty():
            json_data = json.dumps(results_queue.get())
            try:
                channel.send(json_data)
                logging.info(f"Sent data: {json_data}")
            except Exception as e:
                logging.error(f"Error sending data: {e}")
                break


async def handle_offer(request):
    body = await request.json()
    pc = RTCPeerConnection(config)

    channel = pc.createDataChannel("obj_detection")

    @pc.on("track")
    def on_track(track):
        if isinstance(track, MediaStreamTrack):
            logging.info(f"Receiving {track.kind} track")
            asyncio.ensure_future(handle_video_track(track))

    @pc.on("connectionstatechange")
    async def on_connectionstatechange():
        logging.info(f"Connection state is {pc.connectionState}")

    @channel.on("open")
    def on_channel():
        logging.info(f"Data channel established: {channel.label}")
        asyncio.ensure_future(send_obj_coordinates(channel))

    offer = RTCSessionDescription(sdp=body["sdp"], type=body["type"])
    await pc.setRemoteDescription(offer)

    answer = await pc.createAnswer()
    await pc.setLocalDescription(answer)

    return web.Response(content_type="application/json", text=json.dumps(
        {"sdp": pc.localDescription.sdp, "type": pc.localDescription.type}
    ), )


app = web.Application()
cors = aiohttp_cors.setup(app, defaults={
    "*": aiohttp_cors.ResourceOptions(
        allow_credentials=True,
        expose_headers="*",
        allow_headers="*",
    )
})

route = app.router.add_post("/offer", handle_offer)
cors.add(route)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="A script with a object detection flag.")
    parser.add_argument("--no_detection", action="store_false", help="Disable object detection.")

    parser.set_defaults(no_detection=True)
    args = parser.parse_args()

    logging.info(f"Turn server:{config.iceServers[0].urls}")

    if args.no_detection:
        logging.info("Running with object detection")
        Thread(target=detect_objects, args=(frame_queue, results_queue)).start()
    else:
        logging.info("Running with no object detection")
        Thread(target=dummy_detect_objects, args=(frame_queue, results_queue)).start()
    web.run_app(app, port=8181)
