package bufseekio

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestReadSeeker_Seek_SeekCurrent(t *testing.T) {
	recorder := &readSeekRecorder{rs: bytes.NewReader(make([]byte, 100))}

	rs := NewReadSeekerSize(recorder, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	// Seek forwards.
	p, err := rs.Seek(10, io.SeekCurrent)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	if p != 10 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 10, p)
	}
	recorder.assertSeeks(t, []seekRecord{{offset: 10, whence: io.SeekCurrent}})

	// Get position without moving.
	p, err = rs.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	if p != 10 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 10, p)
	}
	recorder.assertSeeks(t, nil)

	// Move backwards.
	p, err = rs.Seek(-5, io.SeekCurrent)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	if p != 5 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 5, p)
	}
	recorder.assertSeeks(t, []seekRecord{{offset: -5, whence: io.SeekCurrent}})
}

func TestReadSeeker_Seek_SeekEnd(t *testing.T) {
	recorder := &readSeekRecorder{rs: bytes.NewReader(make([]byte, 100))}

	rs := NewReadSeekerSize(recorder, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	// Seek from end.
	p, err := rs.Seek(-10, io.SeekEnd)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	if p != 90 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 90, p)
	}
	recorder.assertSeeks(t, []seekRecord{{offset: -10, whence: io.SeekEnd}})

	// Seek from end again.
	p, err = rs.Seek(-10, io.SeekEnd)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	if p != 90 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 90, p)
	}
	// It will always seek again because it only keeps track of the position from the start.
	recorder.assertSeeks(t, []seekRecord{{offset: -10, whence: io.SeekEnd}})
}

func TestReadSeeker_Seek_BufferReuse(t *testing.T) {
	recorder := &readSeekRecorder{rs: bytes.NewReader(make([]byte, 100))}

	rs := NewReadSeekerSize(recorder, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	// Seek to some random position.
	p, err := rs.Seek(10, io.SeekStart)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	if p != 10 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 10, p)
	}
	recorder.assertSeeks(t, []seekRecord{{offset: 10, whence: io.SeekStart}})

	// Read some bytes to fill the internal buffer.
	// Buffer should span [10, 30).
	n, err := rs.Read(make([]byte, 10))
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if n != 10 {
		t.Fatalf("mismatch in # of bytes read: expected: %d, got %d", 10, n)
	}
	if rs.r != 10 {
		t.Fatalf("buffer read position mismatch: expected: %d, got %d", 10, rs.r)
	}
	if rs.w != 20 {
		t.Fatalf("buffer write position mismatch: expected: %d, got %d", 20, rs.w)
	}
	recorder.assertReads(t, []readRecord{{requested: 20}})

	// Seek to an earlier position within the buffer.
	p, err = rs.Seek(-10, io.SeekCurrent)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	if p != 10 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 10, p)
	}
	recorder.assertSeeks(t, nil) // no seeks

	// Seek to a later position within the buffer.
	p, err = rs.Seek(25, io.SeekStart)
	if err != nil {
		t.Fatalf("seek error: %v", err)
	}
	if p != 25 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 25, p)
	}
	recorder.assertSeeks(t, nil) // no seeks

	// Read more than is within the buffer.
	n, err = rs.Read(make([]byte, 10))
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if n != 5 { // should only have returned the bytes in the buffer.
		t.Fatalf("mismatch in # of bytes read: expected: %d, got %d", 5, n)
	}
	if rs.r != 20 {
		t.Fatalf("buffer read position mismatch: expected: %d, got %d", 20, rs.r)
	}
	if rs.w != 20 {
		t.Fatalf("buffer write position mismatch: expected: %d, got %d", 20, rs.w)
	}
	recorder.assertReads(t, nil) // no reads

	// Read again. This will fill a new buffer.
	n, err = rs.Read(make([]byte, 10))
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if n != 10 {
		t.Fatalf("mismatch in # of bytes read: expected: %d, got %d", 10, n)
	}
	if rs.pos != 30 {
		t.Fatalf("buffer start position mismatch: expected: %d, got %d", 30, rs.pos)
	}
	if rs.r != 10 {
		t.Fatalf("buffer read position mismatch: expected: %d, got %d", 10, rs.r)
	}
	if rs.w != 20 {
		t.Fatalf("buffer write position mismatch: expected: %d, got %d", 20, rs.w)
	}
	recorder.assertReads(t, []readRecord{{requested: 20}})
}

func TestReadSeeker_Seek(t *testing.T) {
	type test struct {
		seekTo  int64
		bytes   []byte
		readErr error
	}

	// The test is going to read 2 bytes at specified seek positions
	// out of a buffer of size 100.
	tests := []test{
		// Start
		{seekTo: 0, bytes: []byte{0, 1}},

		// Overlapping positions within a buffer.
		{seekTo: 10, bytes: []byte{10, 11}},
		{seekTo: 20, bytes: []byte{20, 21}},

		// End
		{seekTo: 99, bytes: []byte{99}, readErr: nil},
		{seekTo: 100, bytes: []byte{}, readErr: io.EOF},
	}

	// Test seeking to one position, then another.
	for _, test1 := range tests {
		for _, test2 := range tests {
			t.Run(fmt.Sprintf("seek_to_%d_and_%d", test1.seekTo, test2.seekTo), func(t *testing.T) {
				bs := make([]byte, 100)
				for i := range bs {
					bs[i] = byte(i)
				}
				recorder := &readSeekRecorder{rs: bytes.NewReader(bs)}

				rs := NewReadSeekerSize(recorder, 20)
				if len(rs.buf) != 20 {
					t.Fatal("the buffer size was changed and the validity of this test has become unknown")
				}

				// Seek to the first position.
				p, err := rs.Seek(test1.seekTo, io.SeekStart)
				if err != nil {
					t.Fatalf("seek error: %v", err)
				}
				if p != test1.seekTo {
					t.Fatalf("seek position mismatch: expected %d, got %d", test1.seekTo, p)
				}

				// Read to trigger a buffer read.
				_, _ = rs.Read([]byte{0x00})
				if err != nil && err != io.EOF {
					t.Fatalf("seek error: %v", err)
				}

				// Seek to the second position.
				p, err = rs.Seek(test2.seekTo, io.SeekStart)
				if err != nil {
					t.Fatalf("seek error: %v", err)
				}
				if p != test2.seekTo {
					t.Fatalf("seek position mismatch: expected %d, got %d", test2.seekTo, p)
				}

				// Check a subsequent read works as expected.
				got := make([]byte, 2)
				n, err := rs.Read(got)
				if err != test2.readErr {
					t.Fatalf("error mismatch: expected %v, got %v", test2.readErr, err)
				}
				got = got[:n]
				if !reflect.DeepEqual(test2.bytes, got) {
					t.Fatalf("mismatch bytes returned by Read(): expected %#v, got %#v", test2.bytes, got)
				}
			})
		}
	}
}

func Test_Read_BigBuffer(t *testing.T) {
	recorder := &readSeekRecorder{rs: bytes.NewReader(make([]byte, 100))}

	rs := NewReadSeekerSize(recorder, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	got := make([]byte, 50)
	n, err := rs.Read(got)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if n != 50 {
		t.Fatalf("mismatch in # of bytes read: expected: %d, got %d", 50, n)
	}
	recorder.assertReads(t, []readRecord{{requested: 50}})

	p, err := rs.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if p != 50 {
		t.Fatalf("seek position mismatch: expected %d, got %d", 50, p)
	}
}

type readRecord struct {
	requested int // number of bytes requested
}

type seekRecord struct {
	offset int64
	whence int
}

type readSeekRecorder struct {
	rs    io.ReadSeeker
	reads []readRecord
	seeks []seekRecord
}

func (r *readSeekRecorder) Read(p []byte) (n int, err error) {
	r.reads = append(r.reads, readRecord{requested: len(p)})
	return r.rs.Read(p)
}

func (r *readSeekRecorder) Seek(offset int64, whence int) (int64, error) {
	r.seeks = append(r.seeks, seekRecord{offset: offset, whence: whence})
	return r.rs.Seek(offset, whence)
}

func (r *readSeekRecorder) assertReads(t *testing.T, expected []readRecord) {
	t.Helper()

	if !reflect.DeepEqual(expected, r.reads) {
		t.Fatalf("read mismatch; expected %#v, got %#v", expected, r.reads)
	}
	// Clear reads
	r.reads = nil
}

func (r *readSeekRecorder) assertSeeks(t *testing.T, expected []seekRecord) {
	t.Helper()

	if !reflect.DeepEqual(expected, r.seeks) {
		t.Fatalf("seek mismatch; expected %#v, got %#v", expected, r.seeks)
	}
	// Clear seeks
	r.seeks = nil
}
