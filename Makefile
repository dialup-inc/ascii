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

ifeq ($(shell uname -s), Darwin)
	libs = camera/libcam.a
else
	libs = 
endif

.PHONY: run
run: ascii_roulette
	./ascii_roulette

ascii_roulette: $(libs) *.go
	go build -o $@ ./cmd/ascii_roulette

camera/libcam.a: camera/cam_avfoundation.o
	$(AR) -cr $@ $<

camera/cam_avfoundation.o: camera/cam_avfoundation.mm
	$(CC) $(CFLAGS) -o $@ -c $<

clean:
	rm -f camera/libcam.a camera/cam_avfoundation.o ascii_roulette
