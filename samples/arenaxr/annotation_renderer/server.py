import asyncio
import json
import logging
import queue
from threading import Thread

import aiohttp_cors
import cv2
from aiohttp import web
from aiortc import MediaStreamTrack, RTCPeerConnection, RTCSessionDescription
from av.video.frame import VideoFrame

from annotation_renderer.obj_detection import detect_objects

logging.basicConfig(level=logging.INFO)

frame_queue = queue.Queue()


async def handle_video_track(track: MediaStreamTrack):
    frame_count = 0

    while True:
        try:
            frame_count += 1
            frame = await asyncio.wait_for(track.recv(), timeout=30)
            #if isinstance(frame, VideoFrame):
            frame = frame.to_ndarray(format="bgr24")

            frame_queue.put(frame)
            logging.info(f"Enqueued frame {frame_count}")
            # cv2.imwrite(f"imgs/received_frame_{frame_count}.jpg", frame)

        except asyncio.TimeoutError:
            logging.warning("Timeout: No frames received, closing stream.")
            break
        except Exception as e:
            logging.error(f"Stream error: {e}")
            break


async def handle_offer(request):
    body = await request.json()
    pc = RTCPeerConnection()

    @pc.on("track")
    def on_track(track):
        if isinstance(track, MediaStreamTrack):
            logging.info(f"Receiving {track.kind} track")
            asyncio.ensure_future(handle_video_track(track))

    @pc.on("datachannel")
    def on_datachannel(channel):
        logging.info(f"Data channel established: {channel.label}")

    @pc.on("connectionstatechange")
    async def on_connectionstatechange():
        logging.info(f"Connection state is {pc.connectionState}")

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
    Thread(target=detect_objects, args=(frame_queue,)).start()
    web.run_app(app, port=8181)
