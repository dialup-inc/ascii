#import "cam.h"

#import <AVFoundation/AVFoundation.h>

// image_src is the source image, image_dst is the converted image
void NV12_YUV420P(const unsigned char *image_src, unsigned char *image_dst,
                  int image_width, int image_height) {
  unsigned char *p = image_dst;
  memcpy(p, image_src, image_width * image_height * 3 / 2);
  const unsigned char *pNV = image_src + image_width * image_height;
  unsigned char *pU = p + image_width * image_height;
  unsigned char *pV =
      p + image_width * image_height + ((image_width * image_height) >> 2);
  for (int i = 0; i < (image_width * image_height) / 2; i++) {
    if ((i % 2) == 0)
      *pU++ = *(pNV + i);
    else
      *pV++ = *(pNV + i);
  }
}

@interface FrameDelegate
    : NSObject <AVCaptureVideoDataOutputSampleBufferDelegate> {
  FrameCallback mCallback;
  void *mUserdata;
}

- (void)captureOutput:(AVCaptureOutput *)captureOutput
    didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer
           fromConnection:(AVCaptureConnection *)connection;

@end

@implementation FrameDelegate

- (id)initWithCallback:(FrameCallback)callback Userdata:(void *)userdata {
  [super init];

  self->mCallback = callback;
  self->mUserdata = userdata;

  return self;
}

- (void)captureOutput:(AVCaptureOutput *)captureOutput
    didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer
           fromConnection:(AVCaptureConnection *)connection {

  if (CMSampleBufferGetNumSamples(sampleBuffer) != 1 ||
      !CMSampleBufferIsValid(sampleBuffer) ||
      !CMSampleBufferDataIsReady(sampleBuffer)) {
    return;
  }

  CVImageBufferRef image_buffer = CMSampleBufferGetImageBuffer(sampleBuffer);
  if (image_buffer == NULL) {
    return;
  }

  image_buffer = CVBufferRetain(image_buffer);

  CVReturn ret =
      CVPixelBufferLockBaseAddress(image_buffer, kCVPixelBufferLock_ReadOnly);
  if (ret != kCVReturnSuccess) {
    return;
  }
  static size_t const kYPlaneIndex = 0;
  static size_t const kUVPlaneIndex = 1;
  uint8_t *y_plane_address = static_cast<uint8_t *>(
      CVPixelBufferGetBaseAddressOfPlane(image_buffer, kYPlaneIndex));
  size_t y_plane_height =
      CVPixelBufferGetHeightOfPlane(image_buffer, kYPlaneIndex);
  size_t y_plane_width =
      CVPixelBufferGetWidthOfPlane(image_buffer, kYPlaneIndex);
  size_t y_plane_bytes_per_row =
      CVPixelBufferGetBytesPerRowOfPlane(image_buffer, kYPlaneIndex);
  size_t uv_plane_height =
      CVPixelBufferGetHeightOfPlane(image_buffer, kUVPlaneIndex);
  size_t uv_plane_bytes_per_row =
      CVPixelBufferGetBytesPerRowOfPlane(image_buffer, kUVPlaneIndex);
  size_t frame_size = y_plane_bytes_per_row * y_plane_height +
                      uv_plane_bytes_per_row * uv_plane_height;

  // TODO(maxhawkins): is this slow?
  unsigned char *i420_image = (unsigned char *)malloc(frame_size);
  NV12_YUV420P(y_plane_address, i420_image, y_plane_width, y_plane_height);

  self->mCallback(self->mUserdata, i420_image, frame_size);

  free(i420_image);

  CVPixelBufferUnlockBaseAddress(image_buffer, 0);
  CVBufferRelease(image_buffer);
}

@end

class Capture {
public:
  Capture(FrameCallback callback, void *userdata);
  ~Capture();

  int start(int cam_id, int width, int height, float framerate);

private:
  AVCaptureSession *mSession;
  AVCaptureDeviceInput *mInput;
  AVCaptureVideoDataOutput *mOutput;
  AVCaptureDevice *mCamera;
  FrameDelegate *mDelegate;

  FrameCallback mCallback;
  void *mUserdata;
};

Capture::Capture(FrameCallback callback, void *userdata) {
  mCallback = callback;
  mUserdata = userdata;
}

void configureCamera(AVCaptureDevice *videoDevice, float desiredRate) {
  float rate = 0;
  AVCaptureDeviceFormat *currentFormat = [videoDevice activeFormat];
  for (AVFrameRateRange *range in currentFormat.videoSupportedFrameRateRanges) {
    rate = range.minFrameRate;
    if (rate <= desiredRate) {
      break;
    }
  }

  if (rate <= 0) {
    return;
  }

  if ([videoDevice lockForConfiguration:NULL]) {
    videoDevice.activeVideoMaxFrameDuration = CMTimeMake(1, rate);
    videoDevice.activeVideoMinFrameDuration = CMTimeMake(1, rate);
    [videoDevice unlockForConfiguration];
  }
}

int Capture::start(int cam_id, int width, int height, float framerate) {
  NSAutoreleasePool *pool = [[NSAutoreleasePool alloc] init];

  NSArray *cameras = [[AVCaptureDevice devicesWithMediaType:AVMediaTypeVideo]
      arrayByAddingObjectsFromArray:[AVCaptureDevice
                                        devicesWithMediaType:AVMediaTypeVideo]];

  if (cam_id < 0 || cam_id >= int(cameras.count)) {
    [pool drain];
    return E_CAMERA_NOT_FOUND;
  }
  mCamera = cameras[cam_id];
  if (!mCamera) {
    [pool drain];
    return E_CAMERA_NOT_FOUND;
  }

  configureCamera(mCamera, framerate);

  NSError *error = nil;
  mInput = [[AVCaptureDeviceInput alloc] initWithDevice:mCamera error:&error];
  if (error) {
    NSLog(@"Camera init failed: %@", error.localizedDescription);
    [pool drain];
    return E_OPEN_CAMERA_FAILED;
  }

  mDelegate = [[FrameDelegate alloc] initWithCallback:mCallback
                                             Userdata:mUserdata];

  mOutput = [[AVCaptureVideoDataOutput alloc] init];
  dispatch_queue_t queue =
      dispatch_queue_create("captureQueue", DISPATCH_QUEUE_SERIAL);
  [mOutput setSampleBufferDelegate:mDelegate queue:queue];
  dispatch_release(queue);

  mOutput.videoSettings = @{
    (id)kCVPixelBufferWidthKey : @(1.0 * width),
    (id)kCVPixelBufferHeightKey : @(1.0 * height),
    (id)kCVPixelBufferPixelFormatTypeKey :
        @(kCVPixelFormatType_420YpCbCr8BiPlanarFullRange)
  };
  mOutput.alwaysDiscardsLateVideoFrames = YES;

  mSession = [[AVCaptureSession alloc] init];
  mSession.sessionPreset = AVCaptureSessionPresetMedium;
  [mSession addInput:mInput];
  [mSession addOutput:mOutput];

  [mSession startRunning];

  [pool drain];
  return E_OK;
}

extern "C" {

// cam_init allocates a new Camera and sets its frame callback
int cam_init(Camera *cam, FrameCallback callback, void *userdata) {
  if (!callback) {
    return E_INIT_FAILED;
  }

  Capture *capture = new Capture(callback, userdata);
  *cam = capture;

  return E_OK;
};

// cam_start makes a Camera begin reading frames from the device
int cam_start(Camera cam, int cam_id, int width, int height, float framerate) {
  Capture *capture = (Capture *)cam;
  return capture->start(cam_id, width, height, framerate);
};
}
