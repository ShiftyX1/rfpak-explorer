package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	PAK_MAGIC           = 0x4B504652 // 'R','F','P','K'
	PAK_VERSION         = 1
	PAK_HEADER_SIZE     = 24
	PAK_MAX_PATH_LENGTH = 4096
)

type FileEntry struct {
	Name     string
	Offset   uint64
	Size     uint64
	FullPath string
}

type PakReader struct {
	FilePath string
	Entries  []FileEntry
	file     *os.File
}

func NewPakReader() *PakReader {
	return &PakReader{
		Entries: make([]FileEntry, 0),
	}
}

func (r *PakReader) Open(path string) error {
	r.Close()

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	r.file = file
	r.FilePath = path

	if err := r.readHeader(); err != nil {
		r.Close()
		return err
	}

	return nil
}

func (r *PakReader) Close() {
	if r.file != nil {
		r.file.Close()
		r.file = nil
	}
	r.Entries = r.Entries[:0]
	r.FilePath = ""
}

func (r *PakReader) readHeader() error {
	var header struct {
		Magic      uint32
		Version    uint32
		EntryCount uint32
		Reserved   uint32
		TocOffset  uint64
	}

	if err := binary.Read(r.file, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	if header.Magic != PAK_MAGIC {
		return fmt.Errorf("invalid magic: 0x%08X (expected 0x%08X)", header.Magic, PAK_MAGIC)
	}

	if header.Version != PAK_VERSION {
		return fmt.Errorf("invalid version: %d (expected %d)", header.Version, PAK_VERSION)
	}

	if _, err := r.file.Seek(int64(header.TocOffset), io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to TOC: %w", err)
	}

	r.Entries = make([]FileEntry, 0, header.EntryCount)

	for i := uint32(0); i < header.EntryCount; i++ {
		var tocEntry struct {
			Offset     uint64
			Size       uint64
			PathLength uint32
		}

		if err := binary.Read(r.file, binary.LittleEndian, &tocEntry); err != nil {
			return fmt.Errorf("failed to read TOC entry %d: %w", i, err)
		}

		if tocEntry.PathLength > PAK_MAX_PATH_LENGTH {
			return fmt.Errorf("path length too large: %d", tocEntry.PathLength)
		}

		pathBytes := make([]byte, tocEntry.PathLength)
		if _, err := io.ReadFull(r.file, pathBytes); err != nil {
			return fmt.Errorf("failed to read path for entry %d: %w", i, err)
		}

		fullPath := string(pathBytes)
		name := filepath.Base(fullPath)

		entry := FileEntry{
			Name:     name,
			Offset:   tocEntry.Offset,
			Size:     tocEntry.Size,
			FullPath: fullPath,
		}

		r.Entries = append(r.Entries, entry)
	}

	return nil
}

func (r *PakReader) Extract(entry *FileEntry) ([]byte, error) {
	if r.file == nil {
		return nil, fmt.Errorf("archive not open")
	}

	if _, err := r.file.Seek(int64(entry.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to file: %w", err)
	}

	data := make([]byte, entry.Size)
	if _, err := io.ReadFull(r.file, data); err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	return data, nil
}
