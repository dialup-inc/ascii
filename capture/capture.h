#ifndef CAPTURE_H
#define CAPTURE_H

#ifdef __cplusplus
#include <opencv2/opencv.hpp>
extern "C" {
#endif

int capture_start(int cam_id, int width, int height);
int capture_read(char *ret, int len, int force_key_frame);
int capture_stop();

#ifdef __cplusplus
}
#endif

#endif
