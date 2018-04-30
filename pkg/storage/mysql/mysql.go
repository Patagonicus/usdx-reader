package mysql

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/Patagonicus/usdx-reader/pkg/storage"
	"github.com/Patagonicus/usdx-reader/pkg/usdx"
	_ "github.com/go-sql-driver/mysql"
)

type MySQL struct {
	db         *sql.DB
	insertSong *sql.Stmt
	insertTag  *sql.Stmt
	selectAll  *sql.Stmt
}

func OpenExisting(dataSourceName string) (*MySQL, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	insertSong, insertTag, selectAll, err := prepareStatements(db)

	if err != nil {
		db.Close()
		return nil, err
	}
	return &MySQL{
		db:         db,
		insertSong: insertSong,
		insertTag:  insertTag,
		selectAll:  selectAll,
	}, nil
}

func prepareStatements(db *sql.DB) (insertSong *sql.Stmt, insertTag *sql.Stmt, selectAll *sql.Stmt, err error) {
	if err == nil {
		insertSong, err = db.Prepare("INSERT INTO songs (directory, source, title, artist, sound, bpm, gap, cover, background, video, video_gap, genre, edition, creator, language, year, start, end, resolution, notes_gap, relative, preview_start, medley_start_beat, medley_end_beat, calc_medley, player1, player2, notes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	}

	if err == nil {
		insertTag, err = db.Prepare("INSERT INTO custom (song, tag, content) VALUES (?, ?, ?)")
	}

	if err == nil {
		selectAll, err = db.Prepare("SELECT directory, source, title, artist, sound, bpm, gap, cover, background, video, video_gap, genre, edition, creator, language, year, start, end, resolution, notes_gap, relative, preview_start, medley_start_beat, medley_end_beat, calc_medley, player1, player2, notes FROM songs ORDER BY ID ASC")
	}

	return
}

func OpenNew(dataSourceName string) (*MySQL, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	err = execute(db,
		"DROP TABLE IF EXISTS custom",
		"DROP TABLE IF EXISTS songs",
		`CREATE TABLE songs (
			ID int NOT NULL AUTO_INCREMENT PRIMARY KEY,
			directory VARCHAR(4096),
			source VARCHAR(1024),
			title VARCHAR(1024),
			artist VARCHAR(1024),
			sound VARCHAR(1024),
			bpm FLOAT,
			gap FLOAT,
			cover VARCHAR(1024),
			background VARCHAR(1024),
			video VARCHAR(1024),
			video_gap FLOAT,
			genre VARCHAR(1024),
			edition VARCHAR(1024),
			creator VARCHAR(1024),
			language VARCHAR(1024),
			year INT,
			start FLOAT,
			end INT,
			resolution INT,
			notes_gap INT,
			relative BOOL,
			preview_start FLOAT,
			medley_start_beat INT,
			medley_end_beat INT,
			calc_medley BOOL,
			player1 VARCHAR(1024),
			player2 VARCHAR(1024),
			notes LONGTEXT
		)`,
		`CREATE TABLE custom (
			ID int NOT NULL AUTO_INCREMENT PRIMARY KEY,
			song INT,
			tag VARCHAR(1024),
			content VARCHAR(1024),
			FOREIGN KEY (song) REFERENCES songs(ID) ON DELETE CASCADE
		)`,
	)
	if err != nil {
		db.Close()
		return nil, err
	}

	insertSong, insertTag, selectAll, err := prepareStatements(db)

	if err != nil {
		db.Close()
		return nil, err
	}
	return &MySQL{
		db:         db,
		insertSong: insertSong,
		insertTag:  insertTag,
		selectAll:  selectAll,
	}, nil
}

func (m *MySQL) Close() error {
	return m.db.Close()
}

func (m *MySQL) InsertSong(song usdx.Song) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	result, err := tx.Stmt(m.insertSong).Exec(
		song.Dir,
		song.SourceFile,
		song.Title,
		song.Artist,
		song.SoundFile,
		song.BPM,
		song.Gap,
		song.CoverPath,
		song.BackgroundPath,
		song.VideoPath,
		song.VideoGap,
		song.Genre,
		song.Edition,
		song.Creator,
		song.Language,
		song.Year,
		song.Start,
		song.End,
		song.Resolution,
		song.NotesGap,
		song.Relative,
		song.PreviewStart,
		song.MedleyStartBeat,
		song.MedleyEndBeat,
		song.CalcMedley,
		song.DuetSingerP1,
		song.DuetSingerP2,
		strings.Join(song.Notes, "\r\n"),
	)
	if err != nil {
		rollbackErr := tx.Rollback()
		log.Printf("error rolling back transaction: %v", rollbackErr)
		return err
	}

	songID, err := result.LastInsertId()
	if err != nil {
		rollbackErr := tx.Rollback()
		log.Printf("error rolling back transaction: %v", rollbackErr)
		return errors.New("failed to get last insert ID")
	}
	insertTag := tx.Stmt(m.insertTag)
	for _, tag := range song.CustomTags {
		_, err = insertTag.Exec(songID, tag.Tag, tag.Content)
		if err != nil {
			rollbackErr := tx.Rollback()
			log.Printf("error rolling back transaction: %v", rollbackErr)
			return err
		}
	}

	return tx.Commit()
}

func (m *MySQL) GetAll() storage.Result {
	rows, err := m.selectAll.Query()
	return &result{
		rows: rows,
		err:  err,
	}
}

type result struct {
	rows *sql.Rows
	song usdx.Song
	err  error
}

func (r *result) Next() bool {
	if r.err != nil {
		log.Printf("got error")
		return false
	}

	ok := r.rows.Next()
	if !ok {
		r.err = r.rows.Err()
		return false
	}

	var notes string
	err := r.rows.Scan(
		&r.song.Dir,
		&r.song.SourceFile,
		&r.song.Title,
		&r.song.Artist,
		&r.song.SoundFile,
		&r.song.BPM,
		&r.song.Gap,
		&r.song.CoverPath,
		&r.song.BackgroundPath,
		&r.song.VideoPath,
		&r.song.VideoGap,
		&r.song.Genre,
		&r.song.Edition,
		&r.song.Creator,
		&r.song.Language,
		&r.song.Year,
		&r.song.Start,
		&r.song.End,
		&r.song.Resolution,
		&r.song.NotesGap,
		&r.song.Relative,
		&r.song.PreviewStart,
		&r.song.MedleyStartBeat,
		&r.song.MedleyEndBeat,
		&r.song.CalcMedley,
		&r.song.DuetSingerP1,
		&r.song.DuetSingerP2,
		&notes,
	)
	if err != nil {
		r.err = err
		return false
	}

	r.song.Notes = strings.Split(notes, "\r\n")
	return true
}

func (r *result) Song() usdx.Song {
	return r.song
}

func (r *result) Err() error {
	return r.err
}

func (r *result) Close() error {
	return r.rows.Close()
}

func execute(db *sql.DB, statements ...string) error {
	for _, statement := range statements {
		_, err := db.Exec(statement)
		if err != nil {
			return err
		}
	}
	return nil
}
