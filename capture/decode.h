#ifndef _DECODE_H_
#define _DECODE_H_

#ifdef __cplusplus
extern "C" {
#endif

int vpx_init_dec();
int vpx_decode(const char* frame, int frame_len, char* yv12_frame, int yv12_len);
int vpx_cleanup_dec();

#ifdef __cplusplus
}
#endif

#endif
