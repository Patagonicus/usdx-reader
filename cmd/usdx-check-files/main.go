package main

import (
	"os"
	"path/filepath"

	"github.com/Patagonicus/usdx-reader/pkg/storage/mysql"
	"go.uber.org/zap"
)

func main() {
	l, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer l.Sync()

	backend, err := mysql.OpenExisting("usdx:usdx@tcp(localhost)/usdx")
	if err != nil {
		l.Fatal("error opening database",
			zap.Error(err),
		)
	}
	defer backend.Close()

	base := os.Args[1]
	l = l.With(zap.String("base", base))

	result := backend.GetAll()
	defer result.Close()
	for result.Next() {
		song := result.Song()
		l := l.With(
			zap.String("dir", song.Dir),
			zap.String("source", song.SourceFile),
		)

		check(l, "sound", base, song.Dir, song.SoundFile)
		check(l, "cover", base, song.Dir, song.CoverPath)
		check(l, "background", base, song.Dir, song.BackgroundPath)
		check(l, "video", base, song.Dir, song.VideoPath)
	}
	if result.Err() != nil {
		l.Error("error getting songs",
			zap.Error(result.Err()),
		)
	}
}

func check(l *zap.Logger, name, base, dir, file string) {
	if file == "" {
		return
	}

	path := filepath.Join(base, dir, file)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		l.Warn("file missing",
			zap.String("name", name),
			zap.String("file", file),
			zap.String("path", path),
		)
	}
}
