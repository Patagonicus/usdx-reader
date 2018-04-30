package storage

import "github.com/Patagonicus/usdx-reader/pkg/usdx"

type Backend interface {
	Close() error
	InsertSong(song usdx.Song) error
	GetAll() Result
}

type Result interface {
	Next() bool
	Song() usdx.Song
	Err() error
	Close() error
}
