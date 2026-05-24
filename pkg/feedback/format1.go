package feedback

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseStdPairs 从字符串中解析可疑并发对信息
// 支持的格式：
//
//	[COVERED] FuncID1,FuncID2|File1:Line1,File2:Line2|Confidence|SourceType;
//	[SUSPECT] FuncID1,FuncID2|File1:Line1,File2:Line2|Confidence|SourceType;
func ParseStdPairs(s string) ([]*SuspiciousPairInfo, error) {
	var results []*SuspiciousPairInfo

	// 按行分割
	lines := strings.Split(s, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		pair, err := parseSinglePair(line)
		if err != nil {
			// 跳过无法解析的行
			continue
		}
		if pair != nil {
			results = append(results, pair)
		}
	}
	fmt.Println("ParseStdPairs\n", results)
	return results, nil
}

// parseSinglePair 解析单行可疑对信息
func parseSinglePair(line string) (*SuspiciousPairInfo, error) {
	// 检查前缀
	var isObserved bool
	if strings.HasPrefix(line, "[COVERED]") {
		isObserved = true
		line = strings.TrimPrefix(line, "[COVERED]")
	} else if strings.HasPrefix(line, "[SUSPECT]") {
		isObserved = false
		line = strings.TrimPrefix(line, "[SUSPECT]")
	} else {
		return nil, fmt.Errorf("invalid prefix: %s", line)
	}

	// 去除前后空格和分号
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ";")

	// 按 | 分割成三部分
	parts := strings.Split(line, "|")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid format: expected 4 parts separated by '|', got %d", len(parts))
	}

	// 第一部分: FuncID1,FuncID2
	funcIDs := strings.Split(parts[0], ",")
	if len(funcIDs) != 2 {
		return nil, fmt.Errorf("invalid function IDs format: %s", parts[0])
	}

	funcID1, err := strconv.ParseUint(strings.TrimSpace(funcIDs[0]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid FuncID1: %v", err)
	}

	funcID2, err := strconv.ParseUint(strings.TrimSpace(funcIDs[1]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid FuncID2: %v", err)
	}

	// 第二部分: File1:Line1,File2:Line2
	locations := strings.Split(parts[1], ",")
	if len(locations) != 2 {
		return nil, fmt.Errorf("invalid locations format: %s", parts[1])
	}

	callLoc1, err := parseLocation(locations[0])
	if err != nil {
		return nil, fmt.Errorf("invalid CallLoc1: %v", err)
	}

	callLoc2, err := parseLocation(locations[1])
	if err != nil {
		return nil, fmt.Errorf("invalid CallLoc2: %v", err)
	}

	// 第三部分: Confidence
	confidence, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid confidence: %v", err)
	}

	// 第四部分: SourceType
	sourceType := strings.TrimSpace(parts[3])

	return &SuspiciousPairInfo{
		FuncID1:    funcID1,
		FuncID2:    funcID2,
		CallLoc1:   callLoc1,
		CallLoc2:   callLoc2,
		Confidence: confidence,
		SourceType: sourceType,
		IsObserved: isObserved,
	}, nil
}

// parseLocation 解析位置信息 "File:Line"
func parseLocation(locStr string) (CallLocationInfo, error) {
	locStr = strings.TrimSpace(locStr)

	// 找到最后一个冒号的位置（因为文件路径中可能包含盘符如 D:\...）
	lastColonIdx := strings.LastIndex(locStr, ":")
	if lastColonIdx == -1 {
		return CallLocationInfo{}, fmt.Errorf("invalid location format: %s", locStr)
	}

	file := locStr[:lastColonIdx]
	lineStr := locStr[lastColonIdx+1:]

	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return CallLocationInfo{}, fmt.Errorf("invalid line number: %v", err)
	}

	return CallLocationInfo{
		File: file,
		Line: line,
	}, nil
}
