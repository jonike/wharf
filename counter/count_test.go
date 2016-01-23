package counter_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/itchio/wharf.proto/counter"
	"github.com/stretchr/testify/assert"
)

func Test_Reader_Count(t *testing.T) {
	buf := bytes.NewReader([]byte{1, 2, 3, 4, 5, 6})
	cr := counter.NewReader(buf)
	ioutil.ReadAll(cr)

	assert.Equal(t, cr.Count(), int64(6))
}

func Test_Reader_Nil(t *testing.T) {
	cr := counter.NewReader(nil)
	buf := make([]byte, 6)
	n, err := cr.Read(buf)
	assert.Nil(t, err)

	assert.Equal(t, n, 6)
	assert.Equal(t, cr.Count(), int64(6))
}

func Test_Reader_Callback(t *testing.T) {
	count := int64(-1)
	onRead := func(c int64) { count = c }

	cr := counter.NewReaderCallback(onRead, nil)

	buf := make([]byte, 6)

	cr.Read(buf)
	assert.Equal(t, count, int64(6))

	cr.Read(buf)
	assert.Equal(t, count, int64(12))

	cr.Read(buf)
	assert.Equal(t, count, int64(18))
}

func Test_Writer_Count(t *testing.T) {
	cw := counter.NewWriter(ioutil.Discard)
	buf := []byte{1, 2, 3, 4, 5, 6}
	for i := 0; i < 6; i++ {
		cw.Write(buf)
	}

	assert.Equal(t, cw.Count(), int64(36))
}

func Test_Writer_Nil(t *testing.T) {
	cw := counter.NewWriter(nil)
	buf := []byte{1, 2, 3, 4, 5, 6}
	for i := 0; i < 6; i++ {
		cw.Write(buf)
	}

	assert.Equal(t, cw.Count(), int64(36))
}

func Test_Writer_Callback(t *testing.T) {
	count := int64(-1)
	onWrite := func(c int64) { count = c }

	cw := counter.NewWriterCallback(onWrite, nil)
	buf := []byte{1, 2, 3, 4, 5, 6}

	cw.Write(buf)
	assert.Equal(t, count, int64(6))

	cw.Write(buf)
	assert.Equal(t, count, int64(12))

	cw.Write(buf)
	assert.Equal(t, count, int64(18))

	cw.Write(buf)
	assert.Equal(t, count, int64(24))
}
