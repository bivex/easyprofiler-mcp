package parser

// ReadOptions configures how the profile is parsed
type ReadOptions struct {
	// MaxBlockDepth limits how deep we read nested blocks (0 = unlimited)
	MaxBlockDepth int

	// SampleBlocks if > 0, only reads every Nth block (for large files)
	SampleBlocks int

	// SkipContextSwitches skips reading context switch data
	SkipContextSwitches bool

	// SkipBookmarks skips reading bookmarks
	SkipBookmarks bool

	// MaxThreads limits how many threads to read (0 = all)
	MaxThreads int

	// ProgressCallback is called periodically during parsing
	ProgressCallback func(percent int)
}

// DefaultReadOptions returns sensible defaults
func DefaultReadOptions() ReadOptions {
	return ReadOptions{
		MaxBlockDepth:       0, // unlimited
		SampleBlocks:        0, // read all
		SkipContextSwitches: false,
		SkipBookmarks:       false,
		MaxThreads:          0, // all threads
	}
}

// FastReadOptions returns options optimized for large files (>100MB)
// This reduces memory usage at the cost of precision
func FastReadOptions() ReadOptions {
	return ReadOptions{
		MaxBlockDepth:       5, // limit nesting
		SampleBlocks:        10, // read every 10th block
		SkipContextSwitches: true,
		SkipBookmarks:       true,
		MaxThreads:          0,
	}
}
