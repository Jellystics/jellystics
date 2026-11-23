package jellyfin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

const sessionPayload = `[{
  "Id":"s1","UserId":"u1","RemoteEndPoint":"203.0.113.9",
  "NowPlayingItem":{"Id":"i1","Name":"Film","Type":"Movie","MediaStreams":[
    {"Index":0,"Type":"Video","Codec":"hevc","VideoRange":"HDR","VideoRangeType":"DOVIWithHDR10","IsExternal":false},
    {"Index":1,"Type":"Audio","Language":"fre","IsDefault":true},
    {"Index":2,"Type":"Subtitle","Language":"eng","IsExternal":true}]},
  "PlayState":{"PositionTicks":100,"IsPaused":false,"PlayMethod":"Transcode","AudioStreamIndex":1,"SubtitleStreamIndex":-1},
  "TranscodingInfo":{"VideoCodec":"h264","IsVideoDirect":false,"IsAudioDirect":true,"Bitrate":8000000,
    "Width":1920,"Height":1080,"HardwareAccelerationType":"vaapi",
    "TranscodeReasons":%s}
}]`

func decodeSessions(t *testing.T, reasons string) SessionInfo {
	t.Helper()
	var out []SessionInfo
	body := []byte(fmt.Sprintf(sessionPayload, reasons))
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d sessions", len(out))
	}
	return out[0]
}

func TestDecodeSession_CaptureFields_ArrayReasons(t *testing.T) {
	s := decodeSessions(t, `["VideoCodecNotSupported","ContainerNotSupported"]`)
	ps, ti := s.PlayState, s.TranscodingInfo
	if *ps.AudioStreamIndex != 1 || *ps.SubtitleStreamIndex != -1 {
		t.Fatalf("play state: %+v", ps)
	}
	if !reflect.DeepEqual([]string(ti.TranscodeReasons), []string{"VideoCodecNotSupported", "ContainerNotSupported"}) {
		t.Fatalf("reasons: %v", ti.TranscodeReasons)
	}
	if *ti.Bitrate != 8000000 || *ti.Width != 1920 || *ti.Height != 1080 || ti.HardwareAccelerationType != "vaapi" {
		t.Fatalf("transcoding info: %+v", ti)
	}
	v := s.NowPlayingItem.MediaStreams[0]
	if v.VideoRange != "HDR" || v.VideoRangeType != "DOVIWithHDR10" || v.Index != 0 {
		t.Fatalf("video stream: %+v", v)
	}
	if !s.NowPlayingItem.MediaStreams[2].IsExternal || s.NowPlayingItem.MediaStreams[2].Index != 2 {
		t.Fatalf("subtitle stream: %+v", s.NowPlayingItem.MediaStreams[2])
	}
}

func TestDecodeSession_CommaSeparatedReasons(t *testing.T) {
	s := decodeSessions(t, `"ContainerNotSupported, AudioCodecNotSupported"`)
	want := []string{"ContainerNotSupported", "AudioCodecNotSupported"}
	if !reflect.DeepEqual([]string(s.TranscodingInfo.TranscodeReasons), want) {
		t.Fatalf("reasons: %v", s.TranscodingInfo.TranscodeReasons)
	}
}

func TestDecodeSession_TolerantToOddShapes(t *testing.T) {
	for _, r := range []string{`null`, `0`, `"None"`, `""`, `{"a":1}`, `[]`} {
		s := decodeSessions(t, r)
		if len(s.TranscodingInfo.TranscodeReasons) != 0 {
			t.Fatalf("%s: reasons %v", r, s.TranscodingInfo.TranscodeReasons)
		}
		if s.TranscodingInfo.HardwareAccelerationType != "vaapi" {
			t.Fatalf("%s: session not fully decoded", r)
		}
	}
	var ti TranscodingInfo
	if err := json.Unmarshal([]byte(`{"HardwareAccelerationType":3,"TranscodeReasons":7}`), &ti); err != nil {
		t.Fatalf("numeric enums must not fail: %v", err)
	}
	if ti.HardwareAccelerationType != "3" {
		t.Fatalf("hw = %q", ti.HardwareAccelerationType)
	}
}
