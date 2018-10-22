package csv

import (
	"encoding/csv"
	"io"

	"github.com/Patagonicus/usdx-reader/pkg/usdx"
)

type Backend struct {
	songs []usdx.Song
	out   string
}

func New(path string) (Backend, error) {
}

func NewFromReader(r io.Reader, out string) {
	c := csv.NewReader(r)
}
