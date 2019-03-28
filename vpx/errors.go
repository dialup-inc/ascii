package vpx

import "fmt"

type VPXCodecErr int

const (
	VPX_CODEC_OK VPXCodecErr = iota
	VPX_CODEC_ERROR
	VPX_CODEC_MEM_ERROR
	VPX_CODEC_ABI_MISMATCH
	VPX_CODEC_INCAPABLE
	VPX_CODEC_UNSUP_BITSTREAM
	VPX_CODEC_UNSUP_FEATURE
	VPX_CODEC_CORRUPT_FRAME
	VPX_CODEC_INVALID_PARAM
	VPX_CODEC_LIST_END
)

func (e VPXCodecErr) Error() string {
	switch e {
	case VPX_CODEC_OK:
		return "Success"
	case VPX_CODEC_ERROR:
		return "Unspecified internal error"
	case VPX_CODEC_MEM_ERROR:
		return "Memory allocation error"
	case VPX_CODEC_ABI_MISMATCH:
		return "ABI version mismatch"
	case VPX_CODEC_INCAPABLE:
		return "Codec does not implement requested capability"
	case VPX_CODEC_UNSUP_BITSTREAM:
		return "Bitstream not supported by this decoder"
	case VPX_CODEC_UNSUP_FEATURE:
		return "Bitstream required feature not supported by this decoder"
	case VPX_CODEC_CORRUPT_FRAME:
		return "Corrupt frame detected"
	case VPX_CODEC_INVALID_PARAM:
		return "Invalid parameter"
	case VPX_CODEC_LIST_END:
		return "End of iterated list"
	default:
		return fmt.Sprintf("codec error: %d", e)
	}
}
