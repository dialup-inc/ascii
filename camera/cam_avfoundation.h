#ifndef _CAM_H_
#define _CAM_H_

#ifdef __cplusplus
extern "C" {
#endif

#define E_OK 0
#define E_INIT_FAILED -1
#define E_OPEN_CAMERA_FAILED -2
#define E_CAMERA_NOT_FOUND -3

typedef void (*FrameCallback)(void *userdata, void *buf, int len);

typedef void *Camera;
int cam_init(Camera *cam, FrameCallback callback, void *userdata);
int cam_start(Camera cam, int cam_id, int width, int height);
int cam_close(Camera cam);

#ifdef __cplusplus
}
#endif

#endif
