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

asciirtc: camera/libcam.a *.go
	go build -o $@ .

camera/libcam.a: camera/cam_avfoundation.o
	$(AR) -cr $@ $<

camera/cam_avfoundation.o: camera/cam_avfoundation.mm
	$(CC) $(CFLAGS) -o $@ -c $<

clean:
	rm -f camera/libcam.a camera/cam_avfoundation.o asciirtc
