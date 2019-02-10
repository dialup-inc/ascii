#include "cam.h"
#include "capture.h"
#include "encode.h"
#include "errors.h"

int capture_start(int cam_id, int width, int height) {
    int err;
    err = cam_open(cam_id, width, height);
    if (err) {
        return err;
    }
    err = vpx_init(width, height);
    if (err) {
        return err;
    }

    return 0;
}

// HACK
static char yv12[460800];

int capture_read(char* ret, int len, int force_key_frame) {
    int err = cam_yv12_frame(yv12);
    if (err) {
        return err;
    }

    int size;
    err = vpx_encode(yv12, ret, &size, force_key_frame);
    if (err) {
        return err;
    }
    if (!size) {
        return E_BAD_SIZE;
    }

    return size;
}

int capture_stop() {
    int err;
    err = vpx_cleanup();
    if (!err) {
        return err;
    }
    return 0;
}
