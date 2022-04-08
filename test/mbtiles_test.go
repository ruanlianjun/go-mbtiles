package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ruanlianjun/go-mbtiles"
)

func TestOpenMbtiles(t *testing.T) {
	path := "/Users/ruanlianjun/Desktop/geography-class-jpg.mbtiles"

	mbtilesRead, err := mbtiles.New(path)
	assert.NoError(t, err)
	t.Logf("get mbtiles info:%+v\n", mbtilesRead)

	metadata, err := mbtilesRead.ReadMetadata()
	assert.NoError(t, err)

	t.Logf("get mbtile metadata:%+v\n", metadata)
	var tmp []byte
	err = mbtilesRead.ReadTile(0, 0, 0, &tmp)
	assert.NoError(t, err)
	t.Logf("read mbtiles data:%#v\n", tmp)
}
