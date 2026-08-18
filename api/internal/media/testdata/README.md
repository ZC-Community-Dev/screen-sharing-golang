# Deterministic WebM fixture

The unit and WebSocket integration tests generate their minimal VP8-only WebM
fixture in Go (`deterministicWebMFixture`). Keeping the source bytes in test
code makes the fixture deterministic on Windows and Linux without requiring
FFmpeg or a browser in CI.

The fixture contains one EBML header, one Segment, one VP8 video TrackEntry,
and one Cluster with a keyframe SimpleBlock. It intentionally contains no
audio track. Real Chrome/Edge MediaRecorder output remains a required manual
compatibility check because this synthetic fixture does not model every EBML
element emitted by browsers.
