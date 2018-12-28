#include <stdio.h>
#include <stdlib.h>
#include <memory.h>

#define VPX_CODEC_DISABLE_COMPAT 1
#include <vpx/vpx_encoder.h>
#include <vpx/vp8cx.h>

#include "encode.h"
#include "errors.h"

static int             frame_size;
static vpx_codec_ctx_t codec;
static vpx_image_t     raw;
static int             frame_cnt = 0;

int vpx_init(int width, int height) {
  frame_size = (int) (width * height * 1.5);

  vpx_codec_iface_t* interface = vpx_codec_vp8_cx();

  if (!vpx_img_alloc(&raw, VPX_IMG_FMT_I420, width, height, 1)) {
    return E_IMG_ALLOC;
  }

  vpx_codec_enc_cfg_t cfg;

  // Populate encoder configuration
  vpx_codec_err_t res = vpx_codec_enc_config_default(interface, &cfg, 0);
  if (res) {
    return E_ENCODER_CONFIG;
  }

  // Update the default configuration with our settings
  cfg.rc_target_bitrate = width * height * cfg.rc_target_bitrate / cfg.g_w / cfg.g_h;
  cfg.g_w               = width;
  cfg.g_h               = height;
  cfg.g_lag_in_frames   = 0;
  cfg.g_pass            = VPX_RC_ONE_PASS;
  cfg.rc_end_usage      = VPX_CBR;
  cfg.kf_mode           = VPX_KF_AUTO;
  cfg.kf_max_dist       = 1000;
  cfg.g_error_resilient = VPX_ERROR_RESILIENT_DEFAULT | VPX_ERROR_RESILIENT_PARTITIONS;

  // Initialize codec
  if (vpx_codec_enc_init(&codec, interface, &cfg, 0)) {
    return E_VPX_INIT;
  }

  return 0;
}

int vpx_encode(const char* yv12_frame, char* encoded, int* size, bool force_key_frame) {
  *size = 0;

  int flags = 0;
  if (force_key_frame) {
    flags |= VPX_EFLAG_FORCE_KF;
  }

  // This does not work correctly (only the 1st plane is encoded), why?
  // raw.planes[0] = (unsigned char *) yv12_frame;
  memcpy(raw.planes[0], yv12_frame, frame_size);

  vpx_codec_err_t err = vpx_codec_encode(&codec, &raw, frame_cnt, 1, flags, VPX_DL_REALTIME);
  if (err) {
    return E_ENCODE_FAILED;
  }

  vpx_codec_iter_t iter = NULL;
  const vpx_codec_cx_pkt_t* pkt;
  while ((pkt = vpx_codec_get_cx_data(&codec, &iter))) {
    if (pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
      if (!*size) {
        *size = pkt->data.frame.sz;
        memcpy(encoded, pkt->data.frame.buf, pkt->data.frame.sz);
      }

      bool key_frame = pkt->data.frame.flags & VPX_FRAME_IS_KEY;
    }
  }
  frame_cnt++;

  return 0;
}

int vpx_cleanup() {
  vpx_img_free(&raw);

  if (vpx_codec_destroy(&codec)) {
    return E_CODEC_DESTROY;
  }

  return 0;
}
