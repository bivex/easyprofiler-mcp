package parser

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Reader parses EasyProfiler .prof files
type Reader struct {
	reader  io.ReadSeeker
	data    *ProfileData
	options ReadOptions
}

// NewReader creates a new Reader from a file path with default options
func NewReader(filePath string) (*Reader, error) {
	return NewReaderWithOptions(filePath, DefaultReadOptions())
}

// NewReaderWithOptions creates a new Reader with custom read options
func NewReaderWithOptions(filePath string, options ReadOptions) (*Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Check file size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// For large files (>100MB), suggest fast options if using defaults
	fileSize := stat.Size()
	if fileSize > 100*1024*1024 && options.SampleBlocks == 0 {
		// User passed default options for a large file - could be slow
		// But we respect their choice
	}

	return &Reader{
		reader:  file,
		data:    NewProfileData(),
		options: options,
	}, nil
}

// Parse reads and parses the entire .prof file
func (r *Reader) Parse() (*ProfileData, error) {
	// Read header
	if err := r.readHeader(); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Validate signature
	if r.data.Header.Signature != EasyProfilerSignature {
		return nil, fmt.Errorf("invalid file signature: 0x%X", r.data.Header.Signature)
	}

	// Validate version
	if r.data.Header.Version < MinCompatibleVersion {
		return nil, fmt.Errorf("unsupported version: 0x%X", r.data.Header.Version)
	}

	// Read descriptors
	if err := r.readDescriptors(); err != nil {
		return nil, fmt.Errorf("failed to read descriptors: %w", err)
	}

	// Read threads
	if err := r.readThreads(); err != nil {
		return nil, fmt.Errorf("failed to read threads: %w", err)
	}

	// Read bookmarks (if present and not skipped)
	if !r.options.SkipBookmarks && r.data.Header.Version >= Version210 && r.data.Header.BookmarksCount > 0 {
		if err := r.readBookmarks(); err != nil {
			return nil, fmt.Errorf("failed to read bookmarks: %w", err)
		}
	}

	// Calculate memory usage
	r.data.TotalBlocksCount = r.data.GetBlocksCount()
	r.data.MemoryUsedBytes = int64(r.data.Header.MemorySize)

	return r.data, nil
}

// Close closes the underlying file
func (r *Reader) Close() error {
	if closer, ok := r.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (r *Reader) readHeader() error {
	header := &r.data.Header

	// Read signature and version
	if err := binary.Read(r.reader, binary.LittleEndian, &header.Signature); err != nil {
		return err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &header.Version); err != nil {
		return err
	}

	// Version-specific header reading
	if header.Version < Version130 {
		// Version < 1.3.0: PID is uint32
		var pid32 uint32
		if err := binary.Read(r.reader, binary.LittleEndian, &pid32); err != nil {
			return err
		}
		header.PID = uint64(pid32)
	} else {
		// Version >= 1.3.0: PID is uint64
		if err := binary.Read(r.reader, binary.LittleEndian, &header.PID); err != nil {
			return err
		}
	}

	// Common fields
	if err := binary.Read(r.reader, binary.LittleEndian, &header.CPUFrequency); err != nil {
		return err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &header.BeginTime); err != nil {
		return err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &header.EndTime); err != nil {
		return err
	}

	if header.Version < Version200 {
		// Version 1.x format
		if err := binary.Read(r.reader, binary.LittleEndian, &header.BlocksCount); err != nil {
			return err
		}
		if err := binary.Read(r.reader, binary.LittleEndian, &header.MemorySize); err != nil {
			return err
		}
		if err := binary.Read(r.reader, binary.LittleEndian, &header.DescriptorsCount); err != nil {
			return err
		}
		if err := binary.Read(r.reader, binary.LittleEndian, &header.DescriptorsMemorySize); err != nil {
			return err
		}
	} else {
		// Version 2.x format
		if err := binary.Read(r.reader, binary.LittleEndian, &header.MemorySize); err != nil {
			return err
		}
		if err := binary.Read(r.reader, binary.LittleEndian, &header.DescriptorsMemorySize); err != nil {
			return err
		}
		if err := binary.Read(r.reader, binary.LittleEndian, &header.BlocksCount); err != nil {
			return err
		}
		if err := binary.Read(r.reader, binary.LittleEndian, &header.DescriptorsCount); err != nil {
			return err
		}

		if header.Version >= Version210 {
			// Version 2.1.0+ has additional fields
			if err := binary.Read(r.reader, binary.LittleEndian, &header.ThreadsCount); err != nil {
				return err
			}
			if err := binary.Read(r.reader, binary.LittleEndian, &header.BookmarksCount); err != nil {
				return err
			}
			if err := binary.Read(r.reader, binary.LittleEndian, &header.Padding); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Reader) readDescriptors() error {
	for i := uint32(0); i < r.data.Header.DescriptorsCount; i++ {
		descriptor, err := r.readDescriptor()
		if err != nil {
			return fmt.Errorf("failed to read descriptor %d: %w", i, err)
		}
		r.data.Descriptors[descriptor.ID] = descriptor
	}
	return nil
}

func (r *Reader) readDescriptor() (*BlockDescriptor, error) {
	var size uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	descriptor := &BlockDescriptor{}

	// Read base descriptor data
	if err := binary.Read(r.reader, binary.LittleEndian, &descriptor.ID); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &descriptor.Line); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &descriptor.Color); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &descriptor.Type); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &descriptor.Status); err != nil {
		return nil, err
	}

	// Read name length
	var nameLength uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &nameLength); err != nil {
		return nil, err
	}

	// Read name
	nameBytes := make([]byte, nameLength)
	if _, err := io.ReadFull(r.reader, nameBytes); err != nil {
		return nil, err
	}
	descriptor.Name = string(nameBytes[:len(nameBytes)-1]) // Remove null terminator

	// Read file name (remaining bytes)
	remainingSize := size - (4 + 4 + 4 + 1 + 1 + 2 + nameLength)
	if remainingSize > 0 {
		fileBytes := make([]byte, remainingSize)
		if _, err := io.ReadFull(r.reader, fileBytes); err != nil {
			return nil, err
		}
		descriptor.File = string(fileBytes[:len(fileBytes)-1]) // Remove null terminator
	}

	return descriptor, nil
}

func (r *Reader) readThreads() error {
	threadsRead := uint32(0)
	expectedThreads := r.data.Header.ThreadsCount

	// If version < 2.1.0, we don't know thread count in advance
	if r.data.Header.Version < Version210 {
		expectedThreads = 0xFFFFFFFF // Read until we hit signature
	}

	for threadsRead < expectedThreads {
		// Try to read thread ID
		var threadID uint64

		if r.data.Header.Version < Version130 {
			// Version < 1.3.0: thread_id is uint32
			var threadID32 uint32
			err := binary.Read(r.reader, binary.LittleEndian, &threadID32)
			if err == io.EOF {
				break // End of file
			}
			if err != nil {
				return fmt.Errorf("failed to read thread ID: %w", err)
			}
			threadID = uint64(threadID32)

			// Check if this is the end signature (for versions without thread count)
			if threadID32 == EasyProfilerSignature && expectedThreads == 0xFFFFFFFF {
				return nil // End of threads section
			}
		} else {
			// Version >= 1.3.0: thread_id is uint64
			err := binary.Read(r.reader, binary.LittleEndian, &threadID)
			if err == io.EOF {
				break // End of file
			}
			if err != nil {
				return fmt.Errorf("failed to read thread ID: %w", err)
			}

			// Check if this is the end signature (for versions without thread count)
			if uint32(threadID&0xFFFFFFFF) == EasyProfilerSignature && expectedThreads == 0xFFFFFFFF {
				// Read remaining 4 bytes to complete the uint64
				return nil // End of threads section
			}
		}

		thread, err := r.readThread(threadID)
		if err != nil {
			return fmt.Errorf("failed to read thread %d: %w", threadID, err)
		}
		r.data.Threads[threadID] = thread
		threadsRead++
	}

	// Read end signature
	var signature uint32
	if err := binary.Read(r.reader, binary.LittleEndian, &signature); err != nil {
		// EOF is acceptable here if we've read all expected threads
		if err == io.EOF && threadsRead == expectedThreads {
			return nil
		}
		return fmt.Errorf("failed to read end signature: %w", err)
	}

	if signature != EasyProfilerSignature {
		return fmt.Errorf("invalid end signature: 0x%X, expected 0x%X", signature, EasyProfilerSignature)
	}

	return nil
}

func (r *Reader) readThread(threadID uint64) (*ThreadData, error) {
	thread := &ThreadData{
		ThreadID:        threadID,
		ContextSwitches: make([]*ContextSwitch, 0),
		Blocks:          make([]*Block, 0),
	}

	// Read thread name length
	var nameSize uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &nameSize); err != nil {
		return nil, err
	}

	// Read thread name
	if nameSize > 0 {
		nameBytes := make([]byte, nameSize)
		if _, err := io.ReadFull(r.reader, nameBytes); err != nil {
			return nil, err
		}
		thread.ThreadName = string(nameBytes)
	}

	// Read context switches count
	var csCount uint32
	if err := binary.Read(r.reader, binary.LittleEndian, &csCount); err != nil {
		return nil, err
	}

	// Read context switches (or skip them if option is set)
	if r.options.SkipContextSwitches {
		// Skip context switches by reading and discarding
		for i := uint32(0); i < csCount; i++ {
			var size uint16
			if err := binary.Read(r.reader, binary.LittleEndian, &size); err != nil {
				return nil, err
			}
			// Skip the data
			if _, err := r.reader.Seek(int64(size), io.SeekCurrent); err != nil {
				return nil, err
			}
		}
	} else {
		// Read context switches normally
		for i := uint32(0); i < csCount; i++ {
			cs, err := r.readContextSwitch()
			if err != nil {
				return nil, fmt.Errorf("failed to read context switch %d: %w", i, err)
			}
			thread.ContextSwitches = append(thread.ContextSwitches, cs)
		}
	}

	// Read blocks count
	var blocksCount uint32
	if err := binary.Read(r.reader, binary.LittleEndian, &blocksCount); err != nil {
		return nil, err
	}

	// Read blocks
	for i := uint32(0); i < blocksCount; i++ {
		block, err := r.readBlock()
		if err != nil {
			return nil, fmt.Errorf("failed to read block %d: %w", i, err)
		}
		thread.Blocks = append(thread.Blocks, block)
	}

	return thread, nil
}

func (r *Reader) readContextSwitch() (*ContextSwitch, error) {
	var size uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	cs := &ContextSwitch{}

	if err := binary.Read(r.reader, binary.LittleEndian, &cs.ThreadID); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &cs.Begin); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &cs.End); err != nil {
		return nil, err
	}

	// Read name (remaining bytes)
	remainingSize := size - 24 // 8 + 8 + 8
	if remainingSize > 0 {
		nameBytes := make([]byte, remainingSize)
		if _, err := io.ReadFull(r.reader, nameBytes); err != nil {
			return nil, err
		}
		cs.Name = string(nameBytes[:len(nameBytes)-1]) // Remove null terminator
	}

	return cs, nil
}

func (r *Reader) readBlock() (*Block, error) {
	var size uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	block := &Block{
		Children: make([]*Block, 0),
	}

	if err := binary.Read(r.reader, binary.LittleEndian, &block.Begin); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &block.End); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &block.ID); err != nil {
		return nil, err
	}

	// Read name (remaining bytes)
	remainingSize := size - 20 // 8 + 8 + 4
	if remainingSize > 0 {
		nameBytes := make([]byte, remainingSize)
		if _, err := io.ReadFull(r.reader, nameBytes); err != nil {
			return nil, err
		}
		if len(nameBytes) > 0 && nameBytes[len(nameBytes)-1] == 0 {
			block.Name = string(nameBytes[:len(nameBytes)-1])
		} else {
			block.Name = string(nameBytes)
		}
	}

	return block, nil
}

func (r *Reader) readBookmarks() error {
	for i := uint16(0); i < r.data.Header.BookmarksCount; i++ {
		bookmark, err := r.readBookmark()
		if err != nil {
			return fmt.Errorf("failed to read bookmark %d: %w", i, err)
		}
		r.data.Bookmarks = append(r.data.Bookmarks, bookmark)
	}

	// Read end signature
	var signature uint32
	if err := binary.Read(r.reader, binary.LittleEndian, &signature); err != nil {
		return err
	}
	if signature != EasyProfilerSignature {
		return fmt.Errorf("invalid bookmarks end signature: 0x%X", signature)
	}

	return nil
}

func (r *Reader) readBookmark() (*Bookmark, error) {
	var size uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	bookmark := &Bookmark{}

	if err := binary.Read(r.reader, binary.LittleEndian, &bookmark.Position); err != nil {
		return nil, err
	}
	if err := binary.Read(r.reader, binary.LittleEndian, &bookmark.Color); err != nil {
		return nil, err
	}

	// Read text (remaining bytes)
	remainingSize := size - 12 // 8 + 4
	if remainingSize > 0 {
		textBytes := make([]byte, remainingSize)
		if _, err := io.ReadFull(r.reader, textBytes); err != nil {
			return nil, err
		}
		bookmark.Text = string(textBytes[:len(textBytes)-1]) // Remove null terminator
	}

	return bookmark, nil
}
