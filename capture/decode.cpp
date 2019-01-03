#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>

#define VPX_CODEC_DISABLE_COMPAT 1
#include "vpx/vpx_decoder.h"
#include "vpx/vp8dx.h"

#include "decode.h"
#include "errors.h"

static vpx_codec_ctx_t codec;

int vpx_init_dec() {
  vpx_codec_iface_t* interface = vpx_codec_vp8_dx();

  // Initialize codec
  int flags = 0;
  if (vpx_codec_dec_init(&codec, interface, NULL, flags)) {
    return E_VPX_INIT;
  }

  return 0;
}

int vpx_decode(const char* frame, int frame_len, char* yv12_frame, int yv12_len) {
  // Decode the frame
  vpx_codec_err_t err = vpx_codec_decode(&codec, (const unsigned char*) frame, frame_len, NULL, 0);
  if (err) {
    printf("err: %s\n", vpx_codec_err_to_string(err));
    return E_DECODE_FAILED;
  }

  // Write decoded data to yv12_frame
  vpx_codec_iter_t iter = NULL;
  vpx_image_t*     img;
  int              total = 0;
  while ((img = vpx_codec_get_frame(&codec, &iter))) {
    for (int plane = 0; plane < 3; plane++) {
      unsigned char* buf = img->planes[plane];
      for (int y = 0; y < (plane ? (img->d_h + 1) >> 1 : img->d_h); y++) {
        if (total > yv12_len) {
          return E_BAD_SIZE;
        }

        int len = (plane ? (img->d_w + 1) >> 1 : img->d_w);
        memcpy(yv12_frame + total, buf, len);
        buf += img->stride[plane];
        total += len;
      }
    }
  }

  return 0;
}

int vpx_cleanup_dec() {
  if (vpx_codec_destroy(&codec)) {
      return E_CODEC_DESTROY;
  }
  return 0;
}
