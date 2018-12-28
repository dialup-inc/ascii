#ifndef _ENCODE_H_
#define _ENCODE_H_

int vpx_init(int width, int height);

/** encoded will be filled with encoded data; it must be big enough */
int vpx_encode(const char* yv12_frame, char* encoded, int* size, bool force_key_frame);

int vpx_cleanup();

#endif
