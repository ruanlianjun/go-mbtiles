package mbtiles

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Mbtiles struct {
	conn      *sql.DB
	Format    TileFormat
	timestamp time.Time
	filename  string
}

type metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func New(path string) (*Mbtiles, error) {
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("file:%s not exists\n", path)
	}
	if err != nil {
		return nil, err
	}

	//未完成的文件不允许打开
	if _, err := os.Stat(path + "-journal"); err == nil {
		return nil, fmt.Errorf("refusing to open mbtiles file with associated -journal file (incomplete tileset")
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err = sqlDb.Ping(); err != nil {
		sqlDb.Close()
	}

	m := &Mbtiles{
		conn:      sqlDb,
		Format:    0,
		timestamp: stat.ModTime().Round(time.Second),
		filename:  path,
	}
	err = m.validateRequiredTables()
	if err != nil {
		return nil, err
	}

	_, err = m.GetTileFormat()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m Mbtiles) ReadTile(z int64, x int64, y int64, data *[]byte) error {
	db := m.conn
	var tileData []byte
	row := db.QueryRow("select tile_data from tiles where zoom_level = ? and tile_column = ? and tile_row = ?", z, x, y)
	err := row.Scan(&tileData)
	if err != nil {
		return err
	}
	*data = tileData
	return nil
}

func (m Mbtiles) ReadMetadata() (map[string]interface{}, error) {
	db := m.conn
	stmt, err := db.Prepare("select name, value from metadata where value is not ''")
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	metadataTmp := make(map[string]interface{})

	for rows.Next() {
		tmp := &metadata{}
		err := rows.Scan(&tmp.Name, &tmp.Value)
		if err != nil {
			return nil, err
		}
		if tmp == nil {
			return nil, errors.New("get metadata is empty")
		}
		key := tmp.Name
		value := tmp.Value
		switch key {
		case "maxzoom", "minzoom":
			metadataTmp[key], err = strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("cannot read metadata item %s: %v", key, err)
			}
		case "bounds", "center":
			metadataTmp[key], err = m.parseFloats(value)
			if err != nil {
				return nil, fmt.Errorf("cannot read metadata item %s: %v", key, err)
			}
		case "json":
			err = json.Unmarshal([]byte(value), &metadataTmp)
			if err != nil {
				return nil, fmt.Errorf("unable to parse JSON metadata item: %v", err)
			}
		default:
			metadataTmp[key] = value
		}
	}
	_, hasMinZoom := metadataTmp["minzoom"]
	_, hasMaxZoom := metadataTmp["maxzoom"]

	if !(hasMinZoom && hasMaxZoom) {
		minAndMax := struct {
			Minzoom int `json:"minzoom"`
			Maxzoom int `json:"maxzoom"`
		}{}

		row := db.QueryRow("select min(zoom_level), max(zoom_level) from tiles")

		err := row.Scan(&minAndMax)
		if err != nil {
			return nil, err
		}

		metadataTmp["minzoom"] = minAndMax.Minzoom
		metadataTmp["maxzoom"] = minAndMax.Maxzoom
	}

	return metadataTmp, nil
}

func (m *Mbtiles) validateRequiredTables() error {
	db := m.conn
	stmt := db.QueryRow("SELECT count(*) as c FROM sqlite_master WHERE name in ('tiles', 'metadata')")
	err := stmt.Err()
	if err != nil {
		return err
	}
	var num int
	err = stmt.Scan(&num)
	if err != nil {
		return err
	}
	if num < 2 {
		return errors.New("missing one or more required tables: tiles, metadata")
	}
	return nil
}

func (m *Mbtiles) GetTileFormat() (TileFormat, error) {
	db := m.conn
	stmt, err := db.Prepare("select tile_data from tiles limit 1")
	if err != nil {
		return 0, err
	}

	row := stmt.QueryRow()
	magicWord := make([]byte, 8)
	err = row.Scan(&magicWord)
	if err != nil {
		return 0, err
	}

	tileFormat, err := detectTileFormat(&magicWord)
	if err != nil {
		return 0, err
	}
	m.Format = tileFormat
	return tileFormat, nil
}

func (m *Mbtiles) parseFloats(str string) ([]float64, error) {
	split := strings.Split(str, ",")
	var out []float64
	for _, v := range split {
		value, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return out, fmt.Errorf("could not parse %q to floats: %v", str, err)
		}
		out = append(out, value)
	}
	return out, nil
}
