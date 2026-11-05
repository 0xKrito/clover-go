package macos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"

	"golang.org/x/sys/unix"
)

const (
	xattrUserTags   = "com.apple.metadata:_kMDItemUserTags"
	xattrFinderInfo = "com.apple.FinderInfo"
)

// Finder Tag Color indices
const (
	TagNone   = 0
	TagGray   = 1
	TagGreen  = 2
	TagPurple = 3
	TagBlue   = 4
	TagYellow = 5
	TagRed    = 6
	TagOrange = 7
)

var tagNames = map[int]string{
	TagRed:    "Red",
	TagOrange: "Orange",
	TagYellow: "Yellow",
	TagGreen:  "Green",
	TagBlue:   "Blue",
	TagPurple: "Purple",
	TagGray:   "Gray",
}

var tagColors = map[string]int{
	"red":    TagRed,
	"orange": TagOrange,
	"yellow": TagYellow,
	"green":  TagGreen,
	"blue":   TagBlue,
	"purple": TagPurple,
	"gray":   TagGray,
	"红色":     TagRed,
	"橙色":     TagOrange,
	"黄色":     TagYellow,
	"绿色":     TagGreen,
	"蓝色":     TagBlue,
	"紫色":     TagPurple,
	"灰色":     TagGray,
}

// ColorIndexFromTags maps Finder tag display names (English or Chinese) to
// the 1..7 color index purely in memory. Returns TagNone for unknown input.
// It exists so callers that already hold tag names (e.g. FileInfo.ColorTags
// parsed during ReadDirPage) never need to re-read xattr on the UI thread.
func ColorIndexFromTags(tags []string) int {
	colorIdx := TagNone
	for _, t := range tags {
		parts := strings.Split(t, "\n")
		name := strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			if num, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && num >= TagGray && num <= TagOrange {
				colorIdx = num
				break
			}
		}
		if idx, ok := tagColors[strings.ToLower(name)]; ok && colorIdx == TagNone {
			colorIdx = idx
		}
	}
	return colorIdx
}

// GetFileTags reads Finder color tags from extended attributes adhering to bplist00.
func GetFileTags(path string) ([]string, int, error) {
	buf := make([]byte, 2048)
	sz, err := unix.Getxattr(path, xattrUserTags, buf)
	if err != nil || sz <= 0 {
		return nil, TagNone, nil
	}

	tags := DecodeTagBplist(buf[:sz])
	colorIdx := TagNone

	for _, t := range tags {
		parts := strings.Split(t, "\n")
		name := strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			if num, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && num >= TagGray && num <= TagOrange {
				colorIdx = num
				break
			}
		}
		if idx, ok := tagColors[strings.ToLower(name)]; ok && colorIdx == TagNone {
			colorIdx = idx
		}
	}

	return tags, colorIdx, nil
}

// SetFileTag writes a Finder tag using Apple binary plist (bplist00) and sets FinderInfo.
func SetFileTag(path string, colorIdx int) error {
	name, ok := tagNames[colorIdx]
	if !ok || colorIdx == TagNone {
		return RemoveFileTags(path)
	}

	tagStr := fmt.Sprintf("%s\n%d", name, colorIdx)
	bplistData := EncodeTagBplist([]string{tagStr})

	if err := unix.Setxattr(path, xattrUserTags, bplistData, 0); err != nil {
		return fmt.Errorf("setxattr %s: %w", xattrUserTags, err)
	}

	setFinderInfoColor(path, colorIdx)
	return nil
}

// RemoveFileTags removes the Finder tags attribute.
func RemoveFileTags(path string) error {
	_ = unix.Removexattr(path, xattrUserTags)
	setFinderInfoColor(path, TagNone)
	return nil
}

func setFinderInfoColor(path string, colorIdx int) {
	buf := make([]byte, 32)
	sz, _ := unix.Getxattr(path, xattrFinderInfo, buf)
	if sz < 32 {
		buf = make([]byte, 32)
	}

	colorMask := byte((colorIdx & 0x7) << 1)
	buf[9] = (buf[9] & 0xF1) | colorMask

	if colorIdx == TagNone && bytes.Equal(buf, make([]byte, 32)) {
		_ = unix.Removexattr(path, xattrFinderInfo)
		return
	}
	_ = unix.Setxattr(path, xattrFinderInfo, buf, 0)
}

// EncodeTagBplist encodes string tags into standard Apple bplist00 format.
func EncodeTagBplist(tags []string) []byte {
	if len(tags) == 0 {
		return nil
	}
	header := []byte("bplist00")
	tag := tags[0]

	isASCII := true
	for _, r := range tag {
		if r > 127 {
			isASCII = false
			break
		}
	}

	obj0 := []byte{0xA1, 0x01} // Array of 1 element pointing to object 1
	var obj1 []byte
	if isASCII {
		raw := []byte(tag)
		obj1 = append([]byte{byte(0x50 | len(raw))}, raw...)
	} else {
		runes := []rune(tag)
		u16 := utf16.Encode(runes)
		buf := make([]byte, 1+len(u16)*2)
		buf[0] = byte(0x60 | len(u16))
		for i, v := range u16 {
			binary.BigEndian.PutUint16(buf[1+i*2:], v)
		}
		obj1 = buf
	}

	off0 := byte(len(header))
	off1 := byte(len(header) + len(obj0))
	offTable := []byte{off0, off1}
	offStart := uint64(len(header) + len(obj0) + len(obj1))

	trailer := make([]byte, 32)
	trailer[6] = 1 // offsetIntSize
	trailer[7] = 1 // objectRefSize
	binary.BigEndian.PutUint64(trailer[8:16], 2)
	binary.BigEndian.PutUint64(trailer[16:24], 0)
	binary.BigEndian.PutUint64(trailer[24:32], offStart)

	var res []byte
	res = append(res, header...)
	res = append(res, obj0...)
	res = append(res, obj1...)
	res = append(res, offTable...)
	res = append(res, trailer...)
	return res
}

// DecodeTagBplist parses bplist00 or text tag representations into string slices.
func DecodeTagBplist(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	if !bytes.HasPrefix(data, []byte("bplist00")) {
		// Fallback for plain text or newline separated formats
		content := string(data)
		var out []string
		for _, l := range strings.Split(content, "\n") {
			if t := strings.TrimSpace(l); t != "" {
				out = append(out, t)
			}
		}
		return out
	}

	// Scan string objects inside bplist00
	var results []string
	i := 8 // Skip "bplist00" header
	trailerStart := len(data) - 32
	if trailerStart < 8 {
		return nil
	}

	for i < trailerStart {
		b := data[i]
		if b >= 0x50 && b <= 0x5F { // ASCII String
			strLen := int(b & 0x0F)
			if i+1+strLen <= trailerStart {
				results = append(results, string(data[i+1:i+1+strLen]))
				i += 1 + strLen
				continue
			}
		} else if b >= 0x60 && b <= 0x6F { // Unicode (UTF-16BE) String
			charCount := int(b & 0x0F)
			byteCount := charCount * 2
			if i+1+byteCount <= trailerStart {
				u16 := make([]uint16, charCount)
				for j := 0; j < charCount; j++ {
					u16[j] = binary.BigEndian.Uint16(data[i+1+j*2:])
				}
				results = append(results, string(utf16.Decode(u16)))
				i += 1 + byteCount
				continue
			}
		}
		i++
	}
	return results
}
