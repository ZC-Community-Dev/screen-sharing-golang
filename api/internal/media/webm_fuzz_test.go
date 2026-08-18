package media

import (
	"errors"
	"testing"
)

func TestWebMRejectsAudioAndTruncation(t *testing.T) {
	audio := deterministicWebMFixture()
	// TrackType value: video(1) -> audio(2).
	audio[34] = 2
	parser := NewWebMParser(1 << 20)
	if _, err := parser.Push(audio); !errors.Is(err, ErrWebMUnsupported) {
		t.Fatalf("audio error=%v", err)
	}

	for end := 0; end < len(deterministicWebMFixture()); end++ {
		parser := NewWebMParser(1 << 20)
		if _, err := parser.Push(deterministicWebMFixture()[:end]); err != nil &&
			!errors.Is(err, ErrWebMInvalid) && !errors.Is(err, ErrWebMUnsupported) {
			t.Fatalf("truncation %d: %v", end, err)
		}
	}
}

func FuzzWebMParser(f *testing.F) {
	f.Add(deterministicWebMFixture(), uint8(7))
	f.Add([]byte{0x1a, 0x45, 0xdf, 0xa3}, uint8(1))
	f.Add([]byte("not-webm"), uint8(3))
	f.Fuzz(func(t *testing.T, input []byte, split uint8) {
		if len(input) > 1<<20 {
			t.Skip()
		}
		parser := NewWebMParser(1 << 20)
		defer parser.Close()
		at := 0
		if len(input) > 0 {
			at = int(split) % (len(input) + 1)
		}
		_, _ = parser.Push(input[:at])
		_, _ = parser.Push(input[at:])
		if len(parser.Init()) > 1<<20 {
			t.Fatal("parser exceeded configured bound")
		}
	})
}
