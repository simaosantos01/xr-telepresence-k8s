import asyncio
import json
import logging
import queue
import ssl
from threading import Thread

import aiohttp_cors
from aiohttp import web
from aiortc import MediaStreamTrack, RTCPeerConnection, RTCSessionDescription, RTCDataChannel
from av import VideoFrame

from remote_object_detection.obj_detection import detect_objects

ssl_context = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
ssl_context.load_cert_chain(certfile="cert.pem", keyfile="key.pem")

logging.basicConfig(level=logging.INFO)

frame_queue = queue.Queue()
results_queue = queue.Queue()


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
    pc = RTCPeerConnection()

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
    Thread(target=detect_objects, args=(frame_queue, results_queue)).start()
    web.run_app(app, port=8181, ssl_context=ssl_context)
