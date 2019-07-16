package camera

import "image"

type FrameCallback func(image.Image, error)
