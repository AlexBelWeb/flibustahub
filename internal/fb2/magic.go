package fb2

// ImageKind is the cover file format detected from magic bytes.
type ImageKind int

const (
	ImageNone ImageKind = iota
	ImageJPEG
	ImagePNG
	ImageGIF
	ImageWebP
)

func (k ImageKind) Ext() string {
	switch k {
	case ImageJPEG:
		return ".jpg"
	case ImagePNG:
		return ".png"
	case ImageGIF:
		return ".gif"
	case ImageWebP:
		return ".webp"
	default:
		return ""
	}
}

func (k ImageKind) MIME() string {
	switch k {
	case ImageJPEG:
		return "image/jpeg"
	case ImagePNG:
		return "image/png"
	case ImageGIF:
		return "image/gif"
	case ImageWebP:
		return "image/webp"
	default:
		return ""
	}
}

// DetectImage identifies JPEG, PNG, GIF or WebP from magic bytes.
// Unrecognized bytes are ImageNone — they are not treated as JPEG.
func DetectImage(b []byte) ImageKind {
	if len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff {
		return ImageJPEG
	}
	if len(b) >= 8 &&
		b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4e && b[3] == 0x47 &&
		b[4] == 0x0d && b[5] == 0x0a && b[6] == 0x1a && b[7] == 0x0a {
		return ImagePNG
	}
	if len(b) >= 6 &&
		b[0] == 'G' && b[1] == 'I' && b[2] == 'F' && b[3] == '8' &&
		(b[4] == '7' || b[4] == '9') && b[5] == 'a' {
		return ImageGIF
	}
	if len(b) >= 12 &&
		b[0] == 'R' && b[1] == 'I' && b[2] == 'F' && b[3] == 'F' &&
		b[8] == 'W' && b[9] == 'E' && b[10] == 'B' && b[11] == 'P' {
		return ImageWebP
	}
	return ImageNone
}
