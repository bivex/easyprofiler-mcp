package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yourusername/easyprofiler-mcp/analyzer"
	"github.com/yourusername/easyprofiler-mcp/parser"
)

var (
	currentProfile  *parser.ProfileData
	currentAnalyzer *analyzer.Analyzer
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"EasyProfiler Analysis Server",
		"1.0.0",
		server.WithLogging(),
	)

	// Register tools
	registerTools(s)

	// Start server using stdio
	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}

func registerTools(s *server.MCPServer) {
	// Tool 1: Load profile
	loadProfileTool := mcp.NewTool("load_profile",
		mcp.WithDescription("Load an EasyProfiler .prof file for analysis. For large files (>100MB), use fast_mode=true"),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the .prof file to load"),
		),
		mcp.WithBoolean("fast_mode",
			mcp.Description("Use fast mode for large files - skips context switches and bookmarks (default: false)"),
		),
	)

	s.AddTool(loadProfileTool, loadProfileHandler)

	// Tool 2: Get slowest blocks
	slowestBlocksTool := mcp.NewTool("get_slowest_blocks",
		mcp.WithDescription("Get the slowest profiling blocks"),
		mcp.WithNumber("limit",
			mcp.Description("Number of blocks to return (default: 10)"),
		),
	)

	s.AddTool(slowestBlocksTool, getSlowestBlocksHandler)

	// Tool 3: Get thread statistics
	threadStatsTool := mcp.NewTool("get_thread_statistics",
		mcp.WithDescription("Get statistics for all threads in the profile"),
	)

	s.AddTool(threadStatsTool, getThreadStatisticsHandler)

	// Tool 4: Get hotspots
	hotspotsTool := mcp.NewTool("get_hotspots",
		mcp.WithDescription("Get functions with the highest cumulative execution time"),
		mcp.WithNumber("limit",
			mcp.Description("Number of hotspots to return (default: 10)"),
		),
	)

	s.AddTool(hotspotsTool, getHotspotsHandler)

	// Tool 5: Analyze performance issues
	analyzeIssuesTool := mcp.NewTool("analyze_performance_issues",
		mcp.WithDescription("Perform comprehensive performance analysis and detect common issues"),
	)

	s.AddTool(analyzeIssuesTool, analyzePerformanceIssuesHandler)
}

func loadProfileHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, ok := request.Params.Arguments["file_path"].(string)
	if !ok {
		return mcp.NewToolResultError("file_path parameter is required"), nil
	}

	// Check if fast mode is requested
	fastMode := false
	if fast, ok := request.Params.Arguments["fast_mode"].(bool); ok {
		fastMode = fast
	}

	// Choose read options
	options := parser.DefaultReadOptions()
	if fastMode {
		options = parser.FastReadOptions()
	}

	// Parse the profile
	reader, err := parser.NewReaderWithOptions(filePath, options)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to open file: %v", err)), nil
	}
	defer reader.Close()

	profile, err := reader.Parse()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse profile: %v", err)), nil
	}

	// Store globally
	currentProfile = profile
	currentAnalyzer = analyzer.NewAnalyzer(profile)

	// Prepare summary
	summary := map[string]interface{}{
		"status":            "success",
		"file":              filePath,
		"fast_mode":         fastMode,
		"version":           fmt.Sprintf("0x%X", profile.Header.Version),
		"pid":               profile.Header.PID,
		"total_duration":    profile.GetTotalDuration().String(),
		"threads_count":     profile.GetThreadCount(),
		"blocks_count":      profile.GetBlocksCount(),
		"descriptors_count": len(profile.Descriptors),
		"bookmarks_count":   len(profile.Bookmarks),
		"memory_mb":         fmt.Sprintf("%.2f", float64(profile.Header.MemorySize)/1024/1024),
	}

	data, _ := json.MarshalIndent(summary, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func getSlowestBlocksHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if currentAnalyzer == nil {
		return mcp.NewToolResultError("No profile loaded. Use load_profile first."), nil
	}

	limit := 10
	if l, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(l)
	}

	blocks := currentAnalyzer.GetSlowestBlocks(limit)

	// Format results
	results := make([]map[string]interface{}, len(blocks))
	for i, block := range blocks {
		results[i] = map[string]interface{}{
			"rank":        i + 1,
			"name":        block.Name,
			"file":        block.File,
			"line":        block.Line,
			"duration":    block.Duration.String(),
			"duration_ns": block.Duration.Nanoseconds(),
			"thread_id":   block.ThreadID,
			"thread_name": block.ThreadName,
		}
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func getThreadStatisticsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if currentAnalyzer == nil {
		return mcp.NewToolResultError("No profile loaded. Use load_profile first."), nil
	}

	stats := currentAnalyzer.GetThreadStatistics()

	// Format results
	results := make([]map[string]interface{}, len(stats))
	for i, stat := range stats {
		results[i] = map[string]interface{}{
			"thread_id":          stat.ThreadID,
			"thread_name":        stat.ThreadName,
			"total_duration":     stat.TotalDuration.String(),
			"block_count":        stat.BlockCount,
			"context_switches":   stat.ContextSwitches,
			"avg_block_duration": stat.AvgBlockDuration.String(),
			"percent_of_total":   fmt.Sprintf("%.2f%%", stat.PercentOfTotal),
		}
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func getHotspotsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if currentAnalyzer == nil {
		return mcp.NewToolResultError("No profile loaded. Use load_profile first."), nil
	}

	limit := 10
	if l, ok := request.Params.Arguments["limit"].(float64); ok {
		limit = int(l)
	}

	hotspots := currentAnalyzer.GetHotspots(limit)
	totalDuration := currentProfile.GetTotalDuration()

	// Format results
	results := make([]map[string]interface{}, len(hotspots))
	for i, hotspot := range hotspots {
		percent := float64(hotspot.Duration) / float64(totalDuration) * 100

		results[i] = map[string]interface{}{
			"rank":             i + 1,
			"name":             hotspot.Name,
			"file":             hotspot.File,
			"line":             hotspot.Line,
			"total_duration":   hotspot.Duration.String(),
			"call_count":       hotspot.CallCount,
			"avg_duration":     hotspot.AvgDuration.String(),
			"percent_of_total": fmt.Sprintf("%.2f%%", percent),
		}
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func analyzePerformanceIssuesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if currentAnalyzer == nil {
		return mcp.NewToolResultError("No profile loaded. Use load_profile first."), nil
	}

	issues := currentAnalyzer.AnalyzePerformanceIssues()

	// Group by severity
	grouped := map[string][]map[string]interface{}{
		"high":   make([]map[string]interface{}, 0),
		"medium": make([]map[string]interface{}, 0),
		"low":    make([]map[string]interface{}, 0),
	}

	for _, issue := range issues {
		issueData := map[string]interface{}{
			"type":        issue.Type,
			"description": issue.Description,
			"location":    issue.Location,
		}

		if issue.Duration > 0 {
			issueData["duration"] = issue.Duration.String()
		}
		if issue.ThreadName != "" {
			issueData["thread_name"] = issue.ThreadName
		}

		grouped[issue.Severity] = append(grouped[issue.Severity], issueData)
	}

	result := map[string]interface{}{
		"total_issues": len(issues),
		"by_severity":  grouped,
		"summary": fmt.Sprintf("Found %d performance issues (%d high, %d medium, %d low)",
			len(issues),
			len(grouped["high"]),
			len(grouped["medium"]),
			len(grouped["low"])),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
