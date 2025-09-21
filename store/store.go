package store

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/klauspost/reedsolomon"
)

type Store struct {
	encoder reedsolomon.Encoder
	disks   []string

	k, n int
}

func New(k, n int, baseDir string) (*Store, error) {
	enc, err := reedsolomon.New(k, n)
	if err != nil {
		return nil, err
	}

	disks, err := findDisks(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find disks: %w", err)
	}

	if len(disks) < k+n {
		return nil, errors.New("not enough directories for the given k and n")
	}

	return &Store{
		encoder: enc,
		disks:   disks,
		k:       k,
		n:       n,
	}, nil
}

func (s *Store) Save(name string, r io.Reader) error {
	// TODO: chunk the input reader if it's too large
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	encoded, err := s.encoder.Split(b)
	if err != nil {
		return err
	}
	if err := s.encoder.Encode(encoded); err != nil {
		return err
	}

	writers := make([]io.Writer, len(s.disks))
	for i, disk := range s.disks {
		dir := filepath.Join(disk, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		f, err := os.OpenFile(filepath.Join(dir, "shard"), os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writers[i] = f
	}

	return parallelWrite(writers, encoded)
}

func (s *Store) Get(name string, w io.Writer) error {
	readers := make([]io.Reader, len(s.disks))

	unread := 0
	for i, disk := range s.disks {
		f, err := os.OpenFile(filepath.Join(disk, name, "shard"), os.O_RDONLY, 0644)
		if err != nil {
			readers[i] = nil
			unread++
			slog.Warn("failed to open shard", "err", err)
			continue
		}
		readers[i] = f
	}
	shards, err := parallelRead(readers)
	if err != nil {
		return err
	}
	if err = s.encoder.ReconstructData(shards); err != nil {
		return fmt.Errorf("failed to reconstruct data from disks(%d/%d): %w", unread, len(s.disks), err)
	}

	err = s.encoder.Join(w, shards, len(shards[0])*s.k)
	return err
}

func parallelWrite(writers []io.Writer, data [][]byte) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(writers))
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			writer := writers[i]
			if writer == nil {
				return
			}
			_, err := writer.Write(data[i])
			if err != nil {
				errCh <- err
				return
			}
			if r, ok := writer.(io.Closer); ok {
				r.Close()
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

func parallelRead(readers []io.Reader) ([][]byte, error) {
	var wg sync.WaitGroup
	errCh := make(chan error, len(readers))
	data := make([][]byte, len(readers))
	for i := range readers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			reader := readers[i]
			if reader == nil {
				return
			}
			b, err := io.ReadAll(reader)
			if err != nil {
				errCh <- err
				return
			}
			data[i] = b
			if r, ok := reader.(io.Closer); ok {
				r.Close()
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	return data, errors.Join(errs...)
}

func findDisks(baseDir string) ([]string, error) {
	if _, err := os.Stat(baseDir); errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	disks := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			disks = append(disks, filepath.Join(baseDir, entry.Name()))
		}
	}
	if len(disks) == 0 {
		return nil, errors.New("no disks found")
	}
	return disks, nil
}
