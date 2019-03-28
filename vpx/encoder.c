#include <memory.h>
#include <stdio.h>
#include <stdlib.h>

#define VPX_CODEC_DISABLE_COMPAT 1
#include <vpx/vp8cx.h>
#include <vpx/vpx_encoder.h>

int vpx_init_enc(vpx_codec_ctx_t *codec, vpx_image_t **raw, int width,
                 int height) {
  vpx_codec_iface_t *interface = vpx_codec_vp8_cx();

  vpx_image_t *img = vpx_img_alloc(NULL, VPX_IMG_FMT_I420, width, height, 1);
  if (!img) {
    return VPX_CODEC_MEM_ERROR;
  }
  *raw = img;

  vpx_codec_enc_cfg_t cfg;

  // Populate encoder configuration
  vpx_codec_err_t err = vpx_codec_enc_config_default(interface, &cfg, 0);
  if (err) {
    return err;
  }

  // Update the default configuration with our settings
  cfg.rc_target_bitrate =
      width * height * cfg.rc_target_bitrate / cfg.g_w / cfg.g_h;
  cfg.g_w = width;
  cfg.g_h = height;
  cfg.g_lag_in_frames = 0;
  cfg.g_pass = VPX_RC_ONE_PASS;
  cfg.rc_end_usage = VPX_CBR;
  cfg.kf_mode = VPX_KF_AUTO;
  cfg.kf_max_dist = 1000;
  cfg.g_error_resilient =
      VPX_ERROR_RESILIENT_DEFAULT | VPX_ERROR_RESILIENT_PARTITIONS;

  // Initialize codec
  err = vpx_codec_enc_init(codec, interface, &cfg, 0);
  if (err) {
    return err;
  }

  return 0;
}

int vpx_encode(vpx_codec_ctx_t *ctx, vpx_image_t *raw, const char *yv12_frame,
               int yv12_len, char *encoded, int encoded_cap, int *encoded_len,
               int pts, int force_key_frame) {
  *encoded_len = 0;

  int flags = 0;
  if (force_key_frame) {
    flags |= VPX_EFLAG_FORCE_KF;
  }

  // This does not work correctly (only the 1st plane is encoded), why?
  // raw->planes[0] = (unsigned char *)yv12_frame;
  memcpy(raw->planes[0], yv12_frame, yv12_len);

  vpx_codec_err_t err =
      vpx_codec_encode(ctx, raw, pts, 1, flags, VPX_DL_REALTIME);
  if (err) {
    return err;
  }

  vpx_codec_iter_t iter = NULL;
  const vpx_codec_cx_pkt_t *pkt;
  while ((pkt = vpx_codec_get_cx_data(ctx, &iter))) {
    if (pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
      size_t frame_size = pkt->data.frame.sz;
      if (frame_size + *encoded_len > encoded_cap) {
        return VPX_CODEC_MEM_ERROR;
      }

      memcpy(encoded, pkt->data.frame.buf, pkt->data.frame.sz);
      *encoded_len += frame_size;
      encoded += frame_size;

      // TODO(maxhawkins): should we be concatenating packets instead of
      // returning early?
      return 0;
    }
  }

  return 0;
}

int vpx_cleanup_enc(vpx_codec_ctx_t *codec, vpx_image_t *raw) {
  vpx_img_free(raw);

  vpx_codec_err_t err = vpx_codec_destroy(codec);
  if (err) {
    return err;
  }

  return 0;
}
