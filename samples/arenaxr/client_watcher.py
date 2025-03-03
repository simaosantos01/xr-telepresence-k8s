from arena import *

scene = Scene(host="arenaxr.org", namespace="simaosantos", scene="my-test")


def user_join_callback(scene, camera, msg):
    print(f"User found: {camera.displayName} [object_id={camera.object_id}]")
    ##Get access to user state
    # camera is a Camera class instance (see Objects)
    # etc.


def user_left_callback(scene, camera, msg):
    print(f"User left: {camera.displayName} [object_id={camera.object_id}]")
    # Get access to user state
    # camera is a Camera class instance (see Objects)
    # etc.


scene.user_join_callback = user_join_callback
scene.user_left_callback = user_left_callback

scene.run_tasks()
