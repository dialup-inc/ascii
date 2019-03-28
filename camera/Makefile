CC = g++

CFLAGS = \
	-x objective-c++ \
	-Wall \
	-Wcast-align \
	-Wundef \
	-Wformat-security \
	-Wwrite-strings \
	-Wno-sign-compare \
	-Wno-conversion

LDFLAGS = \
	-framework Foundation \
	-framework AVFoundation

libcam.a: cam_avfoundation.o
	$(AR) -cr $@ $<

cam_avfoundation.o: cam_avfoundation.mm
	$(CC) $(CFLAGS) -o $@ -c $<

clean:
	rm -f libcam.a cam_avfoundation.o
