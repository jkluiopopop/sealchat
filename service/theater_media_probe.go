package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"
	"time"

	"sealchat/utils"
)

type theaterMediaMetadata struct {
	Kind       string
	MimeType   string
	Width      int
	Height     int
	DurationMS int64
	FrameCount int
	FrameRate  float64
	Container  string
	VideoCodec string
	AudioCodec string
}

func detectTheaterMediaType(head []byte) (string, string) {
	if len(head) >= 8 && bytes.Equal(head[:8], []byte("\x89PNG\r\n\x1a\n")) {
		if bytes.Contains(head, []byte("acTL")) {
			return "image/apng", "animated_image"
		}
		return "image/png", "static_image"
	}
	if len(head) >= 3 && bytes.Equal(head[:3], []byte("\xff\xd8\xff")) {
		return "image/jpeg", "static_image"
	}
	if len(head) >= 6 && (string(head[:6]) == "GIF87a" || string(head[:6]) == "GIF89a") {
		return "image/gif", "animated_image"
	}
	if len(head) >= 16 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP" {
		if bytes.Contains(head, []byte("ANIM")) || (len(head) > 20 && string(head[12:16]) == "VP8X" && head[20]&0x02 != 0) {
			return "image/webp", "animated_image"
		}
		return "image/webp", "static_image"
	}
	if len(head) >= 12 && string(head[4:8]) == "ftyp" {
		return "video/mp4", "video"
	}
	if len(head) >= 4 && bytes.Equal(head[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return "video/webm", "video"
	}
	return "", ""
}

func probeTheaterMedia(ctx context.Context, path, kind, mimeType string, config utils.TheaterMediaConfig, toolchain MediaToolchain, runner MediaCommandRunner) (theaterMediaMetadata, error) {
	switch kind {
	case "static_image":
		file, err := os.Open(path)
		if err != nil {
			return theaterMediaMetadata{}, err
		}
		defer file.Close()
		decoded, _, err := image.DecodeConfig(file)
		if err != nil {
			if mimeType == "image/webp" {
				width, height, animated, parseErr := parseWebPMetadata(path)
				if parseErr == nil && !animated {
					return validateTheaterMediaMetadata(theaterMediaMetadata{Kind: kind, MimeType: mimeType, Width: width, Height: height, FrameCount: 1}, config)
				}
			}
			return theaterMediaMetadata{}, fmt.Errorf("IMAGE_DECODE_FAILED: %w", err)
		}
		return validateTheaterMediaMetadata(theaterMediaMetadata{Kind: kind, MimeType: mimeType, Width: decoded.Width, Height: decoded.Height, FrameCount: 1}, config)
	case "animated_image":
		metadata, err := probeAnimatedImage(path, mimeType)
		if err != nil {
			return theaterMediaMetadata{}, err
		}
		return validateTheaterMediaMetadata(metadata, config)
	case "video":
		if !toolchain.FFprobeAvailable() {
			return theaterMediaMetadata{}, errors.New(TheaterMediaErrorProcessorUnavailable)
		}
		probeCtx, cancel := context.WithTimeout(ctx, time.Duration(config.ProbeTimeoutSeconds)*time.Second)
		defer cancel()
		output, err := runner.Run(probeCtx, toolchain.FFprobePath, "-v", "error", "-show_format", "-show_streams", "-of", "json", path)
		if err != nil {
			return theaterMediaMetadata{}, fmt.Errorf("%s: %w", TheaterMediaErrorProbeFailed, err)
		}
		metadata, err := parseFFprobeMetadata(output, mimeType)
		if err != nil {
			return theaterMediaMetadata{}, err
		}
		return validateTheaterMediaMetadata(metadata, config)
	default:
		return theaterMediaMetadata{}, errors.New(TheaterMediaErrorUnsupported)
	}
}

func probeAnimatedImage(path, mimeType string) (theaterMediaMetadata, error) {
	switch mimeType {
	case "image/gif":
		file, err := os.Open(path)
		if err != nil {
			return theaterMediaMetadata{}, err
		}
		defer file.Close()
		decoded, err := gif.DecodeAll(file)
		if err != nil {
			return theaterMediaMetadata{}, fmt.Errorf("IMAGE_DECODE_FAILED: %w", err)
		}
		duration := int64(0)
		for _, delay := range decoded.Delay {
			if delay < 2 {
				delay = 2
			}
			duration += int64(delay) * 10
		}
		return theaterMediaMetadata{Kind: "animated_image", MimeType: mimeType, Width: decoded.Config.Width, Height: decoded.Config.Height, FrameCount: len(decoded.Image), DurationMS: duration}, nil
	case "image/apng":
		return parseAPNGMetadata(path)
	case "image/webp":
		width, height, animated, err := parseWebPMetadata(path)
		if err != nil || !animated {
			return theaterMediaMetadata{}, errors.New("IMAGE_DECODE_FAILED: WebP animation metadata invalid")
		}
		data, _ := os.ReadFile(path)
		frameCount := bytes.Count(data, []byte("ANMF"))
		if frameCount == 0 {
			frameCount = 1
		}
		return theaterMediaMetadata{Kind: "animated_image", MimeType: mimeType, Width: width, Height: height, FrameCount: frameCount}, nil
	default:
		return theaterMediaMetadata{}, errors.New(TheaterMediaErrorUnsupported)
	}
}

func parseAPNGMetadata(path string) (theaterMediaMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return theaterMediaMetadata{}, err
	}
	if len(data) < 33 || !bytes.Contains(data, []byte("acTL")) {
		return theaterMediaMetadata{}, errors.New("IMAGE_DECODE_FAILED: invalid APNG")
	}
	width := int(binary.BigEndian.Uint32(data[16:20]))
	height := int(binary.BigEndian.Uint32(data[20:24]))
	index := bytes.Index(data, []byte("acTL"))
	frameCount := 1
	if index >= 4 && index+8 <= len(data) {
		frameCount = int(binary.BigEndian.Uint32(data[index+4 : index+8]))
	}
	duration := int64(0)
	for offset := 0; ; {
		index := bytes.Index(data[offset:], []byte("fcTL"))
		if index < 0 {
			break
		}
		index += offset
		if index+26 <= len(data) {
			numerator := binary.BigEndian.Uint16(data[index+20 : index+22])
			denominator := binary.BigEndian.Uint16(data[index+22 : index+24])
			if denominator == 0 {
				denominator = 100
			}
			duration += int64(numerator) * 1000 / int64(denominator)
		}
		offset = index + 4
	}
	return theaterMediaMetadata{Kind: "animated_image", MimeType: "image/apng", Width: width, Height: height, FrameCount: frameCount, DurationMS: duration}, nil
}

func parseWebPMetadata(path string) (int, int, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, false, err
	}
	if len(data) < 30 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, false, errors.New("invalid WebP")
	}
	animated := bytes.Contains(data, []byte("ANIM"))
	index := bytes.Index(data, []byte("VP8X"))
	if index >= 0 && index+18 <= len(data) {
		width := 1 + int(data[index+12]) + int(data[index+13])<<8 + int(data[index+14])<<16
		height := 1 + int(data[index+15]) + int(data[index+16])<<8 + int(data[index+17])<<16
		return width, height, animated, nil
	}
	return 0, 0, animated, errors.New("WebP dimensions unavailable")
}

func parseFFprobeMetadata(raw []byte, mimeType string) (theaterMediaMetadata, error) {
	var document struct {
		Streams []struct {
			CodecType    string `json:"codec_type"`
			CodecName    string `json:"codec_name"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			AvgFrameRate string `json:"avg_frame_rate"`
		} `json:"streams"`
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return theaterMediaMetadata{}, fmt.Errorf("%s: %w", TheaterMediaErrorProbeFailed, err)
	}
	metadata := theaterMediaMetadata{Kind: "video", MimeType: mimeType, Container: document.Format.FormatName}
	videoStreams := 0
	for _, stream := range document.Streams {
		switch stream.CodecType {
		case "video":
			videoStreams++
			metadata.VideoCodec = stream.CodecName
			metadata.Width = stream.Width
			metadata.Height = stream.Height
			metadata.FrameRate = parseFrameRate(stream.AvgFrameRate)
		case "audio":
			if metadata.AudioCodec == "" {
				metadata.AudioCodec = stream.CodecName
			}
		}
	}
	if videoStreams != 1 {
		return theaterMediaMetadata{}, errors.New("MEDIA_PROBE_FAILED: video stream count invalid")
	}
	duration, _ := strconv.ParseFloat(document.Format.Duration, 64)
	metadata.DurationMS = int64(duration * 1000)
	return metadata, nil
}

func parseFrameRate(value string) float64 {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		result, _ := strconv.ParseFloat(value, 64)
		return result
	}
	numerator, _ := strconv.ParseFloat(parts[0], 64)
	denominator, _ := strconv.ParseFloat(parts[1], 64)
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func validateTheaterMediaMetadata(metadata theaterMediaMetadata, config utils.TheaterMediaConfig) (theaterMediaMetadata, error) {
	if metadata.Width <= 0 || metadata.Height <= 0 || metadata.Width > config.MaxDimension || metadata.Height > config.MaxDimension || int64(metadata.Width)*int64(metadata.Height) > 64000000 {
		return theaterMediaMetadata{}, errors.New(TheaterMediaErrorLimitExceeded + ": dimensions")
	}
	if metadata.Kind == "animated_image" {
		if metadata.FrameCount <= 1 || metadata.FrameCount > config.MaxAnimatedFrames || metadata.DurationMS > config.MaxAnimatedDurationMS || int64(metadata.Width)*int64(metadata.Height)*int64(metadata.FrameCount) > config.MaxAnimatedPixelFrames {
			return theaterMediaMetadata{}, errors.New(TheaterMediaErrorLimitExceeded + ": animation")
		}
	}
	if metadata.Kind == "video" {
		if metadata.Width > config.VideoMaxWidth || metadata.Height > config.VideoMaxHeight || metadata.DurationMS > config.VideoMaxDurationMS || metadata.FrameRate > float64(config.VideoMaxFrameRate) {
			return theaterMediaMetadata{}, errors.New(TheaterMediaErrorLimitExceeded + ": video")
		}
	}
	return metadata, nil
}
