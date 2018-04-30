package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Patagonicus/usdx-reader/pkg/storage/mysql"
	"go.uber.org/zap"
)

func main() {
	//	l, err := zap.NewDevelopment()
	l, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer l.Sync()

	backend, err := mysql.OpenExisting("usdx:usdx@tcp(localhost)/usdx")
	if err != nil {
		l.Fatal("error conneting to database",
			zap.Error(err),
		)
	}
	defer backend.Close()

	f, err := os.Create("out.csv")
	if err != nil {
		l.Fatal("failed to open file",
			zap.Error(err),
		)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	err = w.Write([]string{
		"Artist",
		"Title",
		"Genre",
		"Edition",
		"Year",
		"File",
	})
	if err != nil {
		l.Fatal("failed to write header",
			zap.Error(err),
		)
	}

	result := backend.GetAll()
	for result.Next() {
		song := result.Song()
		l.Debug("writing song",
			zap.Any("song", song),
		)
		err = w.Write([]string{
			song.Artist,
			song.Title,
			song.Genre,
			song.Edition,
			strconv.Itoa(song.Year),
			filepath.Join(song.Dir, song.SourceFile),
		})
		if err != nil {
			l.Fatal("failed to write song",
				zap.Error(err),
				zap.Any("song", song),
			)
		}
	}
	err = result.Err()
	if err != nil {
		l.Fatal("error getting songs",
			zap.Error(err),
		)
	}
}
