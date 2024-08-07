package bufseekio

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestNewReadSeekerSize(t *testing.T) {
	buf := bytes.NewReader(make([]byte, 100))

	// Test custom buffer size.
	if rs := NewReadSeekerSize(buf, 20); len(rs.buf) != 20 {
		t.Fatalf("want %d got %d", 20, len(rs.buf))
	}

	// Test too small buffer size.
	if rs := NewReadSeekerSize(buf, 1); len(rs.buf) != minReadBufferSize {
		t.Fatalf("want %d got %d", minReadBufferSize, len(rs.buf))
	}

	// Test reuse existing ReadSeeker.
	rs := NewReadSeekerSize(buf, 20)
	if rs2 := NewReadSeekerSize(rs, 5); rs != rs2 {
		t.Fatal("expected ReadSeeker to be reused but got a different ReadSeeker")
	}
}

func TestNewReadSeeker(t *testing.T) {
	buf := bytes.NewReader(make([]byte, 100))
	if rs := NewReadSeeker(buf); len(rs.buf) != defaultBufSize {
		t.Fatalf("want %d got %d", defaultBufSize, len(rs.buf))
	}
}

func TestReadSeeker_Read(t *testing.T) {
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	rs := NewReadSeekerSize(bytes.NewReader(data), 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	// Test small read.
	got := make([]byte, 5)
	if n, err := rs.Read(got); err != nil || n != 5 || !reflect.DeepEqual(got, []byte{0, 1, 2, 3, 4}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 5, n, []byte{0, 1, 2, 3, 4}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 5 {
		t.Fatalf("want %d got %d, err=%v", 5, p, err)
	}

	// Test big read with initially filled buffer.
	got = make([]byte, 25)
	if n, err := rs.Read(got); err != nil || n != 15 || !reflect.DeepEqual(got, []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 15, n, []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 20 {
		t.Fatalf("want %d got %d, err=%v", 20, p, err)
	}

	// Test big read with initially empty buffer.
	if n, err := rs.Read(got); err != nil || n != 25 || !reflect.DeepEqual(got, []byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 25, n, []byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 45 {
		t.Fatalf("want %d got %d, err=%v", 45, p, err)
	}

	// Test EOF.
	if p, err := rs.Seek(98, io.SeekStart); err != nil || p != 98 {
		t.Fatalf("want %d got %d, err=%v", 98, p, err)
	}
	got = make([]byte, 5)
	if n, err := rs.Read(got); err != nil || n != 2 || !reflect.DeepEqual(got, []byte{98, 99, 0, 0, 0}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 2, n, []byte{98, 99, 0, 0, 0}, got, err)
	}
	if n, err := rs.Read(got); err != io.EOF || n != 0 {
		t.Fatalf("want n read %d got %d, err=%v", 0, n, err)
	}

	// Test source that returns bytes and an error at the same time.
	rs = NewReadSeekerSize(&readAndError{bytes: []byte{2, 3, 5}}, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}
	got = make([]byte, 5)
	if n, err := rs.Read(got); err != nil || n != 3 || !reflect.DeepEqual(got, []byte{2, 3, 5, 0, 0}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 3, n, []byte{2, 3, 5, 0, 0}, got, err)
	}
	if n, err := rs.Read(got); err != expectedErr || n != 0 {
		t.Fatalf("want n read %d got %d, want error %v, got %v", 0, n, expectedErr, err)
	}

	// Test read nothing with an empty buffer and a queued error.
	rs = NewReadSeekerSize(&readAndError{bytes: []byte{2, 3, 5}}, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}
	got = make([]byte, 3)
	if n, err := rs.Read(got); err != nil || n != 3 || !reflect.DeepEqual(got, []byte{2, 3, 5}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 3, n, []byte{2, 3, 5}, got, err)
	}
	if n, err := rs.Read(nil); err != expectedErr || n != 0 {
		t.Fatalf("want n read %d got %d, want error %v, got %v", 0, n, expectedErr, err)
	}
	if n, err := rs.Read(nil); err != nil || n != 0 {
		t.Fatalf("want n read %d got %d, err=%v", 0, n, err)
	}

	// Test read nothing with a non-empty buffer and a queued error.
	rs = NewReadSeekerSize(&readAndError{bytes: []byte{2, 3, 5}}, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}
	got = make([]byte, 1)
	if n, err := rs.Read(got); err != nil || n != 1 || !reflect.DeepEqual(got, []byte{2}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 1, n, []byte{}, got, err)
	}
	if n, err := rs.Read(nil); err != nil || n != 0 {
		t.Fatalf("want n read %d got %d, err=%v", 0, n, err)
	}
}

var expectedErr = errors.New("expected error")

type readAndError struct {
	bytes []byte
}

func (r *readAndError) Read(p []byte) (n int, err error) {
	for i, b := range r.bytes {
		p[i] = b
	}
	return len(r.bytes), expectedErr
}

func (r *readAndError) Seek(offset int64, whence int) (int64, error) {
	panic("not implemented")
}

func TestReadSeeker_Seek(t *testing.T) {
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	r := &seekRecorder{rs: bytes.NewReader(data)}
	rs := NewReadSeekerSize(r, 20)
	if len(rs.buf) != 20 {
		t.Fatal("the buffer size was changed and the validity of this test has become unknown")
	}

	got := make([]byte, 5)

	// Test with io.SeekStart
	if p, err := rs.Seek(10, io.SeekStart); err != nil || p != 10 {
		t.Fatalf("want %d got %d, err=%v", 10, p, err)
	}
	r.assertSeeked(t, []seekRecord{{10, io.SeekStart}})
	if n, err := rs.Read(got); err != nil || n != 5 || !reflect.DeepEqual(got, []byte{10, 11, 12, 13, 14}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 5, n, []byte{10, 11, 12, 13, 14}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 15 {
		t.Fatalf("want %d got %d, err=%v", 15, p, err)
	}
	r.assertSeeked(t, nil)

	// Test with io.SeekCurrent and positive offset within buffer.
	if p, err := rs.Seek(5, io.SeekCurrent); err != nil || p != 20 {
		t.Fatalf("want %d got %d, err=%v", 20, p, err)
	}
	r.assertSeeked(t, nil)
	if n, err := rs.Read(got); err != nil || n != 5 || !reflect.DeepEqual(got, []byte{20, 21, 22, 23, 24}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 5, n, []byte{20, 21, 22, 23, 24}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 25 {
		t.Fatalf("want %d got %d, err=%v", 25, p, err)
	}

	// Test with io.SeekCurrent and negative offset within buffer.
	if p, err := rs.Seek(-10, io.SeekCurrent); err != nil || p != 15 {
		t.Fatalf("want %d got %d, err=%v", 15, p, err)
	}
	r.assertSeeked(t, nil)
	if n, err := rs.Read(got); err != nil || n != 5 || !reflect.DeepEqual(got, []byte{15, 16, 17, 18, 19}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 5, n, []byte{15, 16, 17, 18, 19}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 20 {
		t.Fatalf("want %d got %d, err=%v", 20, p, err)
	}

	// Test with io.SeekCurrent and positive offset outside buffer.
	if p, err := rs.Seek(30, io.SeekCurrent); err != nil || p != 50 {
		t.Fatalf("want %d got %d, err=%v", 50, p, err)
	}
	r.assertSeeked(t, []seekRecord{{50, io.SeekStart}})
	if n, err := rs.Read(got); err != nil || n != 5 || !reflect.DeepEqual(got, []byte{50, 51, 52, 53, 54}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 5, n, []byte{50, 51, 52, 53, 54}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 55 {
		t.Fatalf("want %d got %d, err=%v", 55, p, err)
	}

	// Test seek with io.SeekEnd within buffer.
	if p, err := rs.Seek(-45, io.SeekEnd); err != nil || p != 55 {
		t.Fatalf("want %d got %d, err=%v", 55, p, err)
	}
	r.assertSeeked(t, []seekRecord{{-45, io.SeekEnd}})
	if n, err := rs.Read(got); err != nil || n != 5 || !reflect.DeepEqual(got, []byte{55, 56, 57, 58, 59}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 5, n, []byte{55, 56, 57, 58, 59}, got, err)
	}
	if p, err := rs.Seek(0, io.SeekCurrent); err != nil || p != 60 {
		t.Fatalf("want %d got %d, err=%v", 60, p, err)
	}

	// Test seek with error.
	if _, err := rs.Seek(-100, io.SeekStart); err == nil || err.Error() != "bytes.Reader.Seek: negative position" {
		t.Fatalf("want error 'bytes.Reader.Seek: negative position' got %v", err)
	}
	r.assertSeeked(t, []seekRecord{{-100, io.SeekStart}})

	// Test seek after error.
	if p, err := rs.Seek(10, io.SeekStart); err != nil || p != 10 {
		t.Fatalf("want %d got %d, err=%v", 10, p, err)
	}
	r.assertSeeked(t, []seekRecord{{10, io.SeekStart}})
	if n, err := rs.Read(got); err != nil || n != 5 || !reflect.DeepEqual(got, []byte{10, 11, 12, 13, 14}) {
		t.Fatalf("want n read %d got %d, want buffer %v got %v, err=%v", 5, n, []byte{10, 11, 12, 13, 14}, got, err)
	}
}

type seekRecord struct {
	offset int64
	whence int
}

type seekRecorder struct {
	rs    io.ReadSeeker
	seeks []seekRecord
}

func (r *seekRecorder) Read(p []byte) (n int, err error) {
	return r.rs.Read(p)
}

func (r *seekRecorder) Seek(offset int64, whence int) (int64, error) {
	r.seeks = append(r.seeks, seekRecord{offset: offset, whence: whence})
	return r.rs.Seek(offset, whence)
}

func (r *seekRecorder) assertSeeked(t *testing.T, expected []seekRecord) {
	t.Helper()

	if !reflect.DeepEqual(expected, r.seeks) {
		t.Fatalf("seek mismatch; expected %#v, got %#v", expected, r.seeks)
	}
	r.reset()
}

func (r *seekRecorder) reset() {
	r.seeks = nil
}
