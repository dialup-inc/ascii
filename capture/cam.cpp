#include <libuvc/libuvc.h>

#include "errors.h"
#include "cam.h"

#include <exception>
#include <sstream>
#include <string>

Camera::Camera() {
  uvc_error_t res;
  
  res = uvc_init(&this->ctx, NULL);
  if (res < 0) {
    const std::string err = uvc_strerror(res);
    std::ostringstream ss;
    ss << "uvc error: " << err;
    
    throw std::runtime_error(ss.str());
  }
}

int Camera::start(int cam_id, int width, int height) {
  uvc_error_t res;

  uvc_device_t *dev;
  res = uvc_find_device(ctx, &dev, 0, 0, NULL);
  if (res < 0) {
    uvc_perror(res, "find_device");
    return E_OPEN_CAMERA_FAILED;
  }

  uvc_device_handle_t *devh;
  res = uvc_open(dev, &devh);
  if (res < 0) {
    uvc_perror(res, "open");
    return E_OPEN_CAMERA_FAILED;
  }

  uvc_stream_ctrl_t ctrl;
  res = uvc_get_stream_ctrl_format_size(
    devh, &ctrl,
    UVC_FRAME_FORMAT_YUYV,
    width, height, 30);
  if (res < 0) {
    uvc_perror(res, "get_stream");
    return E_OPEN_CAMERA_FAILED;
  }

  return 0;
}

int Camera::read(char *ret) {
  return 0;
}

// int cam_open(int cam_id, int width, int height) {
//   uvc_error_t res;




//   res = uvc_start_streaming(devh, &ctrl, cb, 12345, 0);
//   if (res < 0) {
//     return E_OPEN_CAMERA_FAILED;
//   }

//   uvc_set_ae_mode(devh, 1);

//   return 0;
// }

// int cam_yv12_frame(char* ret) {

//   return 0;
// }
