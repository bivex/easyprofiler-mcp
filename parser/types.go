package parser

import "time"

const (
	EasyProfilerSignature = 0x45617379 // "Easy" in ASCII
	MinCompatibleVersion  = 0x00010000 // v0.1.0
	Version100            = 0x01000000 // v1.0.0
	Version130            = 0x01030000 // v1.3.0
	Version200            = 0x02000000 // v2.0.0
	Version210            = 0x02010000 // v2.1.0
)

// BlockType represents the type of profiler block
type BlockType uint8

const (
	BlockTypeEvent BlockType = 0
	BlockTypeBlock BlockType = 1
	BlockTypeValue BlockType = 2
)

// FileHeader represents the header of a .prof file
type FileHeader struct {
	Signature              uint32
	Version                uint32
	PID                    uint64
	CPUFrequency           int64
	BeginTime              uint64
	EndTime                uint64
	MemorySize             uint64
	DescriptorsMemorySize  uint64
	BlocksCount            uint32
	DescriptorsCount       uint32
	ThreadsCount           uint32
	BookmarksCount         uint16
	Padding                uint16
}

// BlockDescriptor describes a profiler block type
type BlockDescriptor struct {
	ID     uint32
	Line   int32
	Color  uint32
	Type   BlockType
	Status uint8
	Name   string
	File   string
}

// Block represents a profiler block (timing event)
type Block struct {
	Begin    uint64 // Timestamp in nanoseconds
	End      uint64 // Timestamp in nanoseconds
	ID       uint32 // Reference to BlockDescriptor
	Name     string // Runtime name (if any)
	Children []*Block
}

// Duration returns the duration of the block
func (b *Block) Duration() time.Duration {
	return time.Duration(b.End - b.Begin)
}

// ContextSwitch represents a context switch event
type ContextSwitch struct {
	Begin    uint64
	End      uint64
	ThreadID uint64
	Name     string
}

// Duration returns the duration of the context switch
func (cs *ContextSwitch) Duration() time.Duration {
	return time.Duration(cs.End - cs.Begin)
}

// ThreadData represents all profiling data for a single thread
type ThreadData struct {
	ThreadID        uint64
	ThreadName      string
	ContextSwitches []*ContextSwitch
	Blocks          []*Block
}

// Bookmark represents a user-defined bookmark
type Bookmark struct {
	Position uint64
	Color    uint32
	Text     string
}

// ProfileData represents the complete parsed profile
type ProfileData struct {
	Header      FileHeader
	Descriptors map[uint32]*BlockDescriptor
	Threads     map[uint64]*ThreadData
	Bookmarks   []*Bookmark

	// Memory statistics
	TotalBlocksCount int
	MemoryUsedBytes  int64
}

// NewProfileData creates a new empty ProfileData
func NewProfileData() *ProfileData {
	return &ProfileData{
		Descriptors: make(map[uint32]*BlockDescriptor),
		Threads:     make(map[uint64]*ThreadData),
		Bookmarks:   make([]*Bookmark, 0),
	}
}

// GetTotalDuration returns the total profiling duration
func (p *ProfileData) GetTotalDuration() time.Duration {
	return time.Duration(p.Header.EndTime - p.Header.BeginTime)
}

// GetThreadCount returns the number of threads
func (p *ProfileData) GetThreadCount() int {
	return len(p.Threads)
}

// GetBlocksCount returns the total number of blocks across all threads
func (p *ProfileData) GetBlocksCount() int {
	total := 0
	for _, thread := range p.Threads {
		total += countBlocks(thread.Blocks)
	}
	return total
}

func countBlocks(blocks []*Block) int {
	count := len(blocks)
	for _, block := range blocks {
		count += countBlocks(block.Children)
	}
	return count
}

// GetAllBlocks returns a flat list of all blocks from all threads
func (p *ProfileData) GetAllBlocks() []*Block {
	var result []*Block
	for _, thread := range p.Threads {
		result = append(result, flattenBlocks(thread.Blocks)...)
	}
	return result
}

func flattenBlocks(blocks []*Block) []*Block {
	var result []*Block
	for _, block := range blocks {
		result = append(result, block)
		result = append(result, flattenBlocks(block.Children)...)
	}
	return result
}
