package capture

import "fmt"

type CaptureError int

func (e CaptureError) Error() string {
	var d string
	switch e {
	case -1:
		d = "E_OPEN_CAMERA_FAILED"
	case -2:
		d = "E_NO_CAMERA_DATA"
	case -3:
		d = "E_IMG_ALLOC"
	case -4:
		d = "E_ENCODER_CONFIG"
	case -5:
		d = "E_VPX_INIT"
	case -6:
		d = "E_ENCODE_FAILED"
	case -7:
		d = "E_CODEC_DESTROY"
	case -8:
		d = "E_BAD_SIZE"
	case -9:
		d = "E_DECODE_FAILED"
	default:
		d = fmt.Sprintf("%d", e)
	}
	return fmt.Sprintf("capture error: %s", d)
}
