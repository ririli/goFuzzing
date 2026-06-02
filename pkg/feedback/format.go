package feedback

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseStdPairs 从字符串中解析可疑并发对信息和 [FB] 操作日志
// 支持的格式：
//
//	[COVERED] FuncID1,FuncID2|File1:Line1,File2:Line2|Confidence|SourceType;
//	[SUSPECT] FuncID1,FuncID2|File1:Line1,File2:Line2|Confidence|SourceType;
//	[FB]chan: obj=ADDR; opId=ID; funcId=ID; op=TYPE;
//	[FB]wg: obj=ADDR; opId=ID; funcId=ID; op=TYPE;
func ParseStdPairs(s string) ([]*SuspiciousPairInfo, []*OpInfo, error) {
	var results []*SuspiciousPairInfo
	var ops []*OpInfo

	// 按行分割
	lines := strings.Split(s, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 先尝试解析 [FB] 操作日志
		if strings.HasPrefix(line, "[FB]") {
			if op, err := parseFBOp(line); err == nil && op != nil {
				ops = append(ops, op)
			}
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
	fmt.Println("\nParseStdOps\n", ops)
	return results, ops, nil
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

// parseFBOp 解析 [FB] 格式的操作日志
// 格式: [FB]chan: obj=ADDR; opId=ID; funcId=ID; op=TYPE;
//
//	[FB]wg: obj=ADDR; opId=ID; funcId=ID; op=TYPE;
//
// select 中的操作额外带 select=1:
//
//	[FB]chan: obj=ADDR; opId=ID; funcId=ID; op=TYPE; select=1;
func parseFBOp(line string) (*OpInfo, error) {
	// 去除 [FB] 前缀
	content := strings.TrimPrefix(line, "[FB]")

	// 提取 objKind: "chan:" 或 "wg:"
	var objKind OpKind
	if strings.HasPrefix(content, "chan:") {
		objKind = OpKindChannel
		content = strings.TrimPrefix(content, "chan:")
	} else if strings.HasPrefix(content, "wg:") {
		objKind = OpKindWaitGroup
		content = strings.TrimPrefix(content, "wg:")
	} else {
		return nil, fmt.Errorf("unknown FB object kind: %s", content)
	}

	// 去除末尾分号
	content = strings.TrimSpace(content)
	content = strings.TrimSuffix(content, ";")

	// 按 ";" 分割 key=value 对
	pairs := strings.Split(content, ";")
	op := &OpInfo{ObjKind: objKind}

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		switch key {
		case "obj":
			v, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid obj: %v", err)
			}
			op.ObjAddr = v
		case "opId":
			v, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid opId: %v", err)
			}
			op.OpId = v
		case "funcId":
			v, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid funcId: %v", err)
			}
			op.FuncId = v
		case "op":
			op.OpType = OpType(val)
		case "select":
			op.IsSelect = val == "1"
		}
	}

	return op, nil
}
