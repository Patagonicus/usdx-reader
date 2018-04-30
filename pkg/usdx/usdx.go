package usdx

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/Patagonicus/usdx-reader/pkg/encoding"
	"go.uber.org/zap"
)

// tags understood by USDX
var usdxTags = map[string]bool{
	"TITLE":           true,
	"ARTIST":          true,
	"MP3":             true,
	"BPM":             true,
	"GAP":             true,
	"COVER":           true,
	"BACKGROUND":      true,
	"VIDEO":           true,
	"VIDEOGAP":        true,
	"GENRE":           true,
	"EDITION":         true,
	"CREATOR":         true,
	"LANGUAGE":        true,
	"YEAR":            true,
	"START":           true,
	"END":             true,
	"RESOLUTION":      true,
	"NOTESGAP":        true,
	"RELATIVE":        true,
	"ENCODING":        true,
	"PREVIEWSTART":    true,
	"MEDLEYSTARTBEAT": true,
	"MEDLEYENDBEAT":   true,
	"CALCMEDLEY":      true,
	"DUETSINGERP1":    true,
	"DUETSINGERP2":    true,
	"P1":              true,
	"P2":              true,
}

type Reader struct {
	encodings       map[string]Encoding
	defaultEncoding Encoding
	l               *zap.Logger
}

func NewReader(l *zap.Logger) Reader {
	m := make(map[string]Encoding)
	for _, enc := range []Encoding{encoding.Auto, encoding.UTF8, encoding.CP1250, encoding.CP1252} {
		name := enc.Name()
		if _, ok := m[name]; ok {
			l.Warn("duplicate encoding",
				zap.String("name", name),
			)
		}
		m[name] = enc
	}
	return Reader{
		encodings:       m,
		defaultEncoding: encoding.Auto,
		l:               l,
	}
}

func (r Reader) Read(in io.ReadSeeker, dir, sourceFile string) (Song, []error, error) {
	l := r.l.With(
		zap.String("dir", dir),
		zap.String("sourceFile", sourceFile),
	)

	song := Song{
		Dir:        dir,
		SourceFile: sourceFile,
		Encoding:   r.defaultEncoding,
		Resolution: 4,
		CalcMedley: true,
	}

	hasBOM, err := checkBOM(in)
	if err != nil {
		l.Warn("error detecting byte order mark",
			zap.Error(err),
		)
		return song, nil, err
	}
	if hasBOM {
		l.Debug("detected BOM, setting encoding to UTF8")
		song.Encoding = encoding.UTF8
	}

	var warnings []error
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		if !isTag(line) {
			break
		}

		tag, value, err := getTagAndValue(line, song.Encoding)
		if err != nil {
			l.Warn("error decoding line",
				zap.Error(err),
			)
			return song, warnings, err
		}

		if seen[tag] && usdxTags[tag] {
			warnings = append(warnings, fmt.Errorf("duplicate tag '%v'", tag))
		}
		seen[tag] = true
		switch tag {
		case "DUETSINGERP1":
			seen["P1"] = true
		case "DUETSINGERP2":
			seen["P2"] = true
		case "P1":
			seen["DUETSINGERP1"] = true
		case "P2":
			seen["DUETSINGERP2"] = true
		}

		err = nil
		switch tag {
		case "TITLE":
			song.Title = value
		case "ARTIST":
			song.Artist = value
		case "MP3":
			song.SoundFile = value
		case "BPM":
			// I don't know why, but USDX replaces all commas before calling the parse method
			song.BPM, err = parseFloat32I18n(strings.Replace(value, ",", ".", -1))
		case "GAP":
			song.Gap, err = parseFloat32I18n(value)
		case "COVER":
			song.CoverPath = value
		case "BACKGROUND":
			song.BackgroundPath = value
		case "VIDEO":
			song.VideoPath = value
		case "VIDEOGAP":
			song.VideoGap, err = parseFloat32I18n(value)
		case "GENRE":
			song.Genre = value
		case "EDITION":
			song.Edition = value
		case "CREATOR":
			song.Creator = value
		case "LANGUAGE":
			song.Language = value
		case "YEAR":
			// for some reason some files have a year tag with no value. Suppress that error.
			if value != "" {
				song.Year, err = parseInt(value)
			}
		case "START":
			song.Start, err = parseFloat32I18n(value)
		case "END":
			song.End, err = parseInt(value)
		case "RESOLUTION":
			song.Resolution, err = parseInt(value)
		case "NOTESGAP":
			song.NotesGap, err = parseInt(value)
		case "RELATIVE":
			if strings.ToUpper(value) == "YES" {
				song.Relative = true
			}
		case "ENCODING":
			enc, ok := r.encodings[value]
			if ok {
				song.Encoding = enc
			} else {
				l.Warn("unknown encoding, keeping current on",
					zap.String("name", value),
					zap.String("current encoding", song.Encoding.Name()),
				)
			}
		case "PREVIEWSTART":
			song.PreviewStart, err = parseFloat32I18n(value)
		case "MEDLEYSTARTBEAT":
			if song.Relative {
				l.Warn("ignoring medley start beat because relative is set",
					zap.String("value", value),
				)
			} else {
				song.MedleyStartBeat, err = parseInt(value)
			}
		case "MEDLEYENDBEAT":
			if song.Relative {
				l.Warn("ignoring medley end beat because relative is set",
					zap.String("value", value),
				)
			} else {
				song.MedleyEndBeat, err = parseInt(value)
			}
		case "CALCMEDLEY":
			if strings.ToUpper(value) == "OFF" {
				song.CalcMedley = false
			}
		case "DUETSINGERP1", "P1":
			song.DuetSingerP1 = value
		case "DUETSINGERP2", "P2":
			song.DuetSingerP2 = value
		default:
			l.Warn("unknown tag",
				zap.String("tag", tag),
				zap.String("value", value),
			)
			song.CustomTags = append(song.CustomTags, Tag{
				Tag:     tag,
				Content: value,
			})
		}
		if err != nil {
			l.Warn("failed to parse tag",
				zap.String("tag", tag),
				zap.String("value", value),
				zap.Error(err),
			)
			warnings = append(warnings, fmt.Errorf("error adding tag '%v': %v", tag, err.Error()))
		}
	}
	if scanner.Err() != nil {
		return song, warnings, scanner.Err()
	}
	song.Notes = append(song.Notes, scanner.Text())
	for scanner.Scan() {
		song.Notes = append(song.Notes, scanner.Text())
	}
	return song, warnings, scanner.Err()
}

func isTag(line string) bool {
	return strings.HasPrefix(line, "#")
}

func getTagAndValue(line string, decoder Encoding) (string, string, error) {
	line = strings.TrimLeft(line, "#")

	var tag, value string

	sep := strings.IndexRune(line, ':')
	if sep < 0 {
		value = line
	} else {
		tag, value = line[:sep], line[sep+1:]
	}

	value, err := decoder.Decode(value)
	return tag, value, err
}

func parseFloat32I18n(s string) (float32, error) {
	// this is how USDX does it
	val, err := strconv.ParseFloat(strings.Replace(trim(s), ",", ".", 1), 32)
	return float32(val), err
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(trim(s))
}

func trim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r)
	})
}

func checkBOM(r io.ReadSeeker) (bool, error) {
	var bom [3]byte
	_, err := io.ReadFull(r, bom[:])
	if err != nil {
		return false, err
	}
	if bom[0] == 0xef && bom[1] == 0xbb && bom[2] == 0xbf {
		return true, nil
	}
	_, err = r.Seek(0, io.SeekStart)
	return false, err
}

type Song struct {
	Dir             string
	SourceFile      string
	Title           string
	Artist          string
	SoundFile       string
	BPM             float32
	Gap             float32
	CoverPath       string
	BackgroundPath  string
	VideoPath       string
	VideoGap        float32
	Genre           string
	Edition         string
	Creator         string
	Language        string
	Year            int
	Start           float32
	End             int
	Resolution      int
	NotesGap        int
	Relative        bool
	Encoding        Encoding
	PreviewStart    float32
	MedleyStartBeat int
	MedleyEndBeat   int
	CalcMedley      bool
	DuetSingerP1    string
	DuetSingerP2    string
	CustomTags      []Tag
	Notes           []string
}

type Tag struct {
	Tag     string
	Content string
}

type Encoding interface {
	Name() string
	Decode(s string) (string, error)
}
