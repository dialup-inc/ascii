#ifndef _CAM_H_
#define _CAM_H_

#include <libuvc/libuvc.h>

class Camera {
public:
    Camera();
    int read(char *ret);
    int start(int cam_id, int width, int height);
private:
    uvc_context_t *ctx;
};

#endif
