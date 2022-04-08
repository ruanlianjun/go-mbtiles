### GO读取瓦片的`mbtiles`文件

```go
path := "/Users/ruanlianjun/Desktop/geography-class-jpg.mbtiles"

mbtilesRead, err := mbtiles.New(path)
if err != nil {
  log.Fatalf("read mbtiles file err:%v\n",err)
}
t.Logf("get mbtiles info:%+v\n", mbtilesRead)
// read metadata
metadata, err := mbtilesRead.ReadMetadata()

if err !=nil {
  log.Fatalf("read mbtiles metadata err:%v\n",err)
}

// read tile
var tmp []byte
err = mbtilesRead.ReadTile(0, 0, 0, &tmp)

if err !=nil {
  log.Fatalf("read mbtiles metadata err:%v\n",err)
}

```

