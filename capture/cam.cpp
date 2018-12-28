#include <opencv2/opencv.hpp>

#include "errors.h"
#include "cam.h"

using namespace cv;

typedef Vec<uchar, 1> Vec1b;

static VideoCapture cap;
static Mat          rgb, yv12;

int cam_open(int cam_id, int width, int height) {
  if (!cap.open(cam_id)) {
    return E_OPEN_CAMERA_FAILED;
  }

  cap.set(CV_CAP_PROP_FRAME_WIDTH,  width);
  cap.set(CV_CAP_PROP_FRAME_HEIGHT, height);
  cap.set(CV_CAP_PROP_FPS,          30);

  return 0;
}

int cam_yv12_frame(char* ret) {
  cap.read(rgb);
  if (!rgb.data) return E_NO_CAMERA_DATA;

  cvtColor(rgb, yv12, CV_RGB2YUV_YV12);

  int rows = yv12.rows;
  int cols = yv12.cols;

  char* b = ret;
  for (int r = 0; r < rows; r++) {
    for (int c = 0; c < cols; c++) {
      *b = yv12.at<Vec1b>(r, c)[0];
      b++;
    }
  }

  return 0;
}
