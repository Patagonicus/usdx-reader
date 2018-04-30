package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Patagonicus/usdx-reader/pkg/storage/mysql"
	"github.com/Patagonicus/usdx-reader/pkg/usdx"
	"go.uber.org/zap"
)

func loader(l *zap.Logger, base string, paths <-chan string, songs chan<- usdx.Song, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(songs)
	l = l.With(zap.String("base", base))
	for path := range paths {
		rel, err := filepath.Rel(base, path)
		if err != nil {
			l.Warn("failed to get relative path",
				zap.String("path", path),
				zap.Error(err),
			)
			continue
		}

		dir, source := filepath.Split(rel)

		song, warnings, err := tryLoad(l, base, dir, source)
		if err != nil {
			l.Warn("error loading file",
				zap.String("path", path),
				zap.Error(err),
			)
			continue
		}
		if len(warnings) > 0 {
			l.Warn("got warnings",
				zap.String("path", path),
				zap.Errors("warnings", warnings),
			)
		}

		songs <- song
	}
}

type inserter interface {
	InsertSong(song usdx.Song) error
}

func dispatch(l *zap.Logger, inserter inserter, songs <-chan usdx.Song, wg *sync.WaitGroup) {
	for song := range songs {
		err := inserter.InsertSong(song)
		if err != nil {
			l.Warn("error inserting song into database",
				zap.Error(err),
			)
		}
	}
	wg.Done()
}

func main() {
	//	l, err := zap.NewDevelopment()
	l, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer l.Sync()

	backend, err := mysql.OpenNew("usdx:usdx@tcp(localhost)/usdx")
	if err != nil {
		l.Fatal("error conneting to database",
			zap.Error(err),
		)
	}
	defer backend.Close()

	var dispatchers = runtime.NumCPU()
	wg := new(sync.WaitGroup)
	paths := make(chan string)
	songs := make(chan usdx.Song)
	wg.Add(1)
	go loader(l, os.Args[1], paths, songs, wg)
	wg.Add(dispatchers)
	for i := 0; i < dispatchers; i++ {
		go dispatch(l, backend, songs, wg)
	}

	err = filepath.Walk(os.Args[1], func(path string, info os.FileInfo, err error) error {
		l := l.With(
			zap.String("path", path),
			zap.Any("fileinfo", info),
		)
		if err != nil {
			l.Warn("error encountered while walking directory",
				zap.Error(err),
			)
			return nil
		}

		if info.IsDir() {
			l.Debug("path is a directory, nothing to do")
			return nil
		}

		if strings.ToLower(filepath.Ext(path)) != ".txt" {
			l.Debug("not a text file, skipping")
			return nil
		}

		paths <- path

		return nil
	})
	if err != nil {
		l.Error("error while walking directory",
			zap.Error(err),
		)
	}

	close(paths)
	wg.Wait()
}

func tryLoad(l *zap.Logger, base, dir, source string) (usdx.Song, []error, error) {
	file, err := os.Open(filepath.Join(base, dir, source))
	if err != nil {
		return usdx.Song{}, nil, err
	}
	defer file.Close()
	return usdx.NewReader(l).Read(file, dir, source)
}
