#ifndef _CAM_H_
#define _CAM_H_

int cam_open(int cam_id, int width, int height);

/** yv12_frame: rows = height * 1.5, cols = width */
int cam_yv12_frame(char* yv12_frame);

#endif
