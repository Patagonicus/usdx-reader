package main

import (
	"os"

	"github.com/Patagonicus/usdx-reader/pkg/usdx"
	"go.uber.org/zap"
)

func main() {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer l.Sync()

	f, err := os.Open(os.Args[1])
	if err != nil {
		l.Fatal("failed to open file",
			zap.String("path", os.Args[1]),
			zap.Error(err),
		)
	}
	defer f.Close()

	song, warnings, err := usdx.NewReader(l).Read(f, "", "")
	if err != nil {
		l.Fatal("failed to read file",
			zap.String("path", os.Args[1]),
			zap.Error(err),
		)
	}

	l.Info("file info",
		zap.Any("info", song),
		zap.Any("warnings", warnings),
	)
}
