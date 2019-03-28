#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define VPX_CODEC_DISABLE_COMPAT 1
#include "vpx/vp8dx.h"
#include "vpx/vpx_decoder.h"

int vpx_init_dec(vpx_codec_ctx_t *ctx) {
  vpx_codec_iface_t *interface = vpx_codec_vp8_dx();

  // Initialize codec
  int flags = 0;
  vpx_codec_err_t err = vpx_codec_dec_init(ctx, interface, NULL, flags);
  if (err) {
    return err;
  }

  return 0;
}

int vpx_decode(vpx_codec_ctx_t *ctx, const char *frame, int frame_len,
               char *yv12_frame, int yv12_cap, int *yv12_len) {
  // Decode the frame
  vpx_codec_err_t err =
      vpx_codec_decode(ctx, (const unsigned char *)frame, frame_len, NULL, 0);
  if (err) {
    return err;
  }

  // Write decoded data to yv12_frame
  vpx_codec_iter_t iter = NULL;
  vpx_image_t *img;

  while ((img = vpx_codec_get_frame(ctx, &iter))) {
    for (int plane = 0; plane < 3; plane++) {
      unsigned char *buf = img->planes[plane];
      for (int y = 0; y < (plane ? (img->d_h + 1) >> 1 : img->d_h); y++) {
        if (*yv12_len > yv12_cap) {
          return VPX_CODEC_MEM_ERROR;
        }

        int len = (plane ? (img->d_w + 1) >> 1 : img->d_w);
        memcpy(yv12_frame + *yv12_len, buf, len);
        buf += img->stride[plane];
        *yv12_len += len;
      }
    }
  }

  return 0;
}

int vpx_cleanup_dec(vpx_codec_ctx_t *ctx) {
  vpx_codec_err_t err = vpx_codec_destroy(ctx);
  if (err) {
    return err;
  }
  return 0;
}
