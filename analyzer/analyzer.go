package analyzer

import (
	"fmt"
	"sort"
	"time"

	"github.com/yourusername/easyprofiler-mcp/parser"
)

// Analyzer provides performance analysis tools
type Analyzer struct {
	profile *parser.ProfileData
}

// NewAnalyzer creates a new analyzer for the given profile
func NewAnalyzer(profile *parser.ProfileData) *Analyzer {
	return &Analyzer{profile: profile}
}

// BlockInfo contains analyzed block information
type BlockInfo struct {
	Name        string
	File        string
	Line        int32
	Duration    time.Duration
	CallCount   int
	ThreadID    uint64
	ThreadName  string
	AvgDuration time.Duration
}

// ThreadStats contains thread statistics
type ThreadStats struct {
	ThreadID          uint64
	ThreadName        string
	TotalDuration     time.Duration
	BlockCount        int
	ContextSwitches   int
	AvgBlockDuration  time.Duration
	PercentOfTotal    float64
}

// PerformanceIssue represents a detected performance problem
type PerformanceIssue struct {
	Type        string
	Severity    string // "high", "medium", "low"
	Description string
	Location    string
	Duration    time.Duration
	ThreadID    uint64
	ThreadName  string
}

// GetSlowestBlocks returns the N slowest blocks
func (a *Analyzer) GetSlowestBlocks(limit int) []*BlockInfo {
	var allBlocks []*BlockInfo

	for threadID, thread := range a.profile.Threads {
		blocks := a.analyzeBlocksRecursive(thread.Blocks, threadID, thread.ThreadName)
		allBlocks = append(allBlocks, blocks...)
	}

	// Sort by duration
	sort.Slice(allBlocks, func(i, j int) bool {
		return allBlocks[i].Duration > allBlocks[j].Duration
	})

	if limit > len(allBlocks) {
		limit = len(allBlocks)
	}

	return allBlocks[:limit]
}

func (a *Analyzer) analyzeBlocksRecursive(blocks []*parser.Block, threadID uint64, threadName string) []*BlockInfo {
	var result []*BlockInfo

	for _, block := range blocks {
		descriptor := a.profile.Descriptors[block.ID]

		name := block.Name
		if name == "" && descriptor != nil {
			name = descriptor.Name
		}

		file := ""
		line := int32(0)
		if descriptor != nil {
			file = descriptor.File
			line = descriptor.Line
		}

		result = append(result, &BlockInfo{
			Name:       name,
			File:       file,
			Line:       line,
			Duration:   block.Duration(),
			CallCount:  1,
			ThreadID:   threadID,
			ThreadName: threadName,
		})

		// Recursively process children
		result = append(result, a.analyzeBlocksRecursive(block.Children, threadID, threadName)...)
	}

	return result
}

// GetThreadStatistics returns statistics for all threads
func (a *Analyzer) GetThreadStatistics() []*ThreadStats {
	totalDuration := a.profile.GetTotalDuration()
	var stats []*ThreadStats

	for threadID, thread := range a.profile.Threads {
		threadDuration := a.calculateThreadDuration(thread.Blocks)
		blockCount := a.countBlocks(thread.Blocks)

		avgBlockDuration := time.Duration(0)
		if blockCount > 0 {
			avgBlockDuration = threadDuration / time.Duration(blockCount)
		}

		percentOfTotal := 0.0
		if totalDuration > 0 {
			percentOfTotal = float64(threadDuration) / float64(totalDuration) * 100
		}

		stats = append(stats, &ThreadStats{
			ThreadID:         threadID,
			ThreadName:       thread.ThreadName,
			TotalDuration:    threadDuration,
			BlockCount:       blockCount,
			ContextSwitches:  len(thread.ContextSwitches),
			AvgBlockDuration: avgBlockDuration,
			PercentOfTotal:   percentOfTotal,
		})
	}

	// Sort by total duration
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalDuration > stats[j].TotalDuration
	})

	return stats
}

func (a *Analyzer) calculateThreadDuration(blocks []*parser.Block) time.Duration {
	total := time.Duration(0)
	for _, block := range blocks {
		total += block.Duration()
	}
	return total
}

func (a *Analyzer) countBlocks(blocks []*parser.Block) int {
	count := len(blocks)
	for _, block := range blocks {
		count += a.countBlocks(block.Children)
	}
	return count
}

// GetHotspots returns functions with the highest cumulative time
func (a *Analyzer) GetHotspots(limit int) []*BlockInfo {
	// Group blocks by name and aggregate
	blockMap := make(map[string]*BlockInfo)

	for threadID, thread := range a.profile.Threads {
		a.aggregateBlocks(thread.Blocks, threadID, thread.ThreadName, blockMap)
	}

	// Convert map to slice
	var hotspots []*BlockInfo
	for _, info := range blockMap {
		if info.CallCount > 0 {
			info.AvgDuration = info.Duration / time.Duration(info.CallCount)
		}
		hotspots = append(hotspots, info)
	}

	// Sort by total duration
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Duration > hotspots[j].Duration
	})

	if limit > len(hotspots) {
		limit = len(hotspots)
	}

	return hotspots[:limit]
}

func (a *Analyzer) aggregateBlocks(blocks []*parser.Block, threadID uint64, threadName string, blockMap map[string]*BlockInfo) {
	for _, block := range blocks {
		descriptor := a.profile.Descriptors[block.ID]

		name := block.Name
		if name == "" && descriptor != nil {
			name = descriptor.Name
		}

		key := name
		if descriptor != nil {
			key = fmt.Sprintf("%s:%s:%d", name, descriptor.File, descriptor.Line)
		}

		if existing, ok := blockMap[key]; ok {
			existing.Duration += block.Duration()
			existing.CallCount++
		} else {
			file := ""
			line := int32(0)
			if descriptor != nil {
				file = descriptor.File
				line = descriptor.Line
			}

			blockMap[key] = &BlockInfo{
				Name:       name,
				File:       file,
				Line:       line,
				Duration:   block.Duration(),
				CallCount:  1,
				ThreadID:   threadID,
				ThreadName: threadName,
			}
		}

		// Recursively process children
		a.aggregateBlocks(block.Children, threadID, threadName, blockMap)
	}
}

// AnalyzePerformanceIssues detects common performance problems
func (a *Analyzer) AnalyzePerformanceIssues() []*PerformanceIssue {
	var issues []*PerformanceIssue

	// Detect long blocking operations (>100ms)
	issues = append(issues, a.detectLongBlocks()...)

	// Detect thread imbalance
	issues = append(issues, a.detectThreadImbalance()...)

	// Detect excessive context switches
	issues = append(issues, a.detectExcessiveContextSwitches()...)

	// Detect hot functions (>10% of total time)
	issues = append(issues, a.detectHotFunctions()...)

	// Sort by severity
	sort.Slice(issues, func(i, j int) bool {
		severityOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
		return severityOrder[issues[i].Severity] < severityOrder[issues[j].Severity]
	})

	return issues
}

func (a *Analyzer) detectLongBlocks() []*PerformanceIssue {
	var issues []*PerformanceIssue
	threshold := 100 * time.Millisecond

	for threadID, thread := range a.profile.Threads {
		blocks := a.findLongBlocks(thread.Blocks, threshold)
		for _, block := range blocks {
			descriptor := a.profile.Descriptors[block.ID]
			name := block.Name
			location := "unknown"

			if descriptor != nil {
				if name == "" {
					name = descriptor.Name
				}
				location = fmt.Sprintf("%s:%d", descriptor.File, descriptor.Line)
			}

			severity := "medium"
			if block.Duration() > 500*time.Millisecond {
				severity = "high"
			}

			issues = append(issues, &PerformanceIssue{
				Type:        "Long Blocking Operation",
				Severity:    severity,
				Description: fmt.Sprintf("Block '%s' took %v", name, block.Duration()),
				Location:    location,
				Duration:    block.Duration(),
				ThreadID:    threadID,
				ThreadName:  thread.ThreadName,
			})
		}
	}

	return issues
}

func (a *Analyzer) findLongBlocks(blocks []*parser.Block, threshold time.Duration) []*parser.Block {
	var result []*parser.Block

	for _, block := range blocks {
		if block.Duration() > threshold {
			result = append(result, block)
		}
		result = append(result, a.findLongBlocks(block.Children, threshold)...)
	}

	return result
}

func (a *Analyzer) detectThreadImbalance() []*PerformanceIssue {
	var issues []*PerformanceIssue

	stats := a.GetThreadStatistics()
	if len(stats) < 2 {
		return issues
	}

	// Find max and min thread durations
	maxDuration := stats[0].TotalDuration
	minDuration := stats[len(stats)-1].TotalDuration

	// If imbalance is more than 2x, report it
	if minDuration > 0 && float64(maxDuration)/float64(minDuration) > 2.0 {
		issues = append(issues, &PerformanceIssue{
			Type:        "Thread Imbalance",
			Severity:    "medium",
			Description: fmt.Sprintf("Thread workload imbalance detected: max=%v, min=%v (ratio=%.2fx)",
				maxDuration, minDuration, float64(maxDuration)/float64(minDuration)),
			Location:    "across all threads",
			Duration:    maxDuration - minDuration,
		})
	}

	return issues
}

func (a *Analyzer) detectExcessiveContextSwitches() []*PerformanceIssue {
	var issues []*PerformanceIssue
	threshold := 1000

	for threadID, thread := range a.profile.Threads {
		if len(thread.ContextSwitches) > threshold {
			issues = append(issues, &PerformanceIssue{
				Type:        "Excessive Context Switches",
				Severity:    "medium",
				Description: fmt.Sprintf("Thread has %d context switches (threshold: %d)",
					len(thread.ContextSwitches), threshold),
				Location:    thread.ThreadName,
				ThreadID:    threadID,
				ThreadName:  thread.ThreadName,
			})
		}
	}

	return issues
}

func (a *Analyzer) detectHotFunctions() []*PerformanceIssue {
	var issues []*PerformanceIssue
	totalDuration := a.profile.GetTotalDuration()
	threshold := 0.10 // 10%

	hotspots := a.GetHotspots(10)
	for _, hotspot := range hotspots {
		percent := float64(hotspot.Duration) / float64(totalDuration)
		if percent > threshold {
			severity := "low"
			if percent > 0.3 {
				severity = "high"
			} else if percent > 0.2 {
				severity = "medium"
			}

			location := hotspot.Name
			if hotspot.File != "" {
				location = fmt.Sprintf("%s (%s:%d)", hotspot.Name, hotspot.File, hotspot.Line)
			}

			issues = append(issues, &PerformanceIssue{
				Type:        "Hot Function",
				Severity:    severity,
				Description: fmt.Sprintf("Function '%s' consumes %.1f%% of total time (%v total, %d calls, avg %v)",
					hotspot.Name, percent*100, hotspot.Duration, hotspot.CallCount, hotspot.AvgDuration),
				Location:    location,
				Duration:    hotspot.Duration,
			})
		}
	}

	return issues
}
