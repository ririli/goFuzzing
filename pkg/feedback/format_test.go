package feedback

import (
	"testing"
)

func TestParseStdPairs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectCount int
		expectError bool
	}{
		{
			name:        "解析 COVERED 格式",
			input:       `[COVERED] 100,200|gopie/testdata/main.go:10,worker.go:20|1.00|observed;`,
			expectCount: 1,
			expectError: false,
		},
		{
			name:        "解析 SUSPECT 格式",
			input:       `[SUSPECT] 100,300|main.go:10,handler.go:30|0.60|inferred_parent1;`,
			expectCount: 1,
			expectError: false,
		},
		{
			name: "解析多行混合格式",
			input: `[COVERED] 100,200|main.go:10,worker.go:20|1.00|observed;
[SUSPECT] 100,300|main.go:10,handler.go:30|0.60|inferred_parent1;
[SUSPECT] 200,400|worker.go:20,service.go:40|0.50|inferred_child2;`,
			expectCount: 3,
			expectError: false,
		},
		{
			name:        "解析带 Windows 路径的格式",
			input:       `[COVERED] 100,200|D:\gopath\src\gopie\main.go:10,D:\gopath\src\gopie\worker.go:20|1.00|observed;`,
			expectCount: 1,
			expectError: false,
		},
		{
			name:        "空输入",
			input:       "",
			expectCount: 0,
			expectError: false,
		},
		{
			name: "包含无效行",
			input: `[COVERED] 100,200|main.go:10,worker.go:20|1.00|observed;
invalid line here
[SUSPECT] 300,400|test.go:5,prod.go:15|0.80|inferred_parent2;`,
			expectCount: 2, // 应该跳过无效行，解析出2个有效对
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseStdPairs(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("期望错误但没有收到错误")
			}

			if !tt.expectError && err != nil {
				t.Errorf("不期望错误但收到错误: %v", err)
			}

			if len(results) != tt.expectCount {
				t.Errorf("期望解析 %d 个结果，实际得到 %d 个", tt.expectCount, len(results))
			}

			// 打印解析结果用于调试
			for i, result := range results {
				t.Logf("[%d] FuncID1=%d, FuncID2=%d, CallLoc1=%s:%d, CallLoc2=%s:%d, Confidence=%.2f, SourceType=%s, IsObserved=%v",
					i,
					result.FuncID1,
					result.FuncID2,
					result.CallLoc1.File,
					result.CallLoc1.Line,
					result.CallLoc2.File,
					result.CallLoc2.Line,
					result.Confidence,
					result.SourceType,
					result.IsObserved,
				)
			}
		})
	}
}

func TestParseStdPairs_VerifyFields(t *testing.T) {
	input := `[COVERED] 123,456|pkg/test.go:42,pkg/handler.go:88|1.00|observed;
[SUSPECT] 789,101|pkg/service.go:15,pkg/controller.go:99|0.60|inferred_parent1;`

	results, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("期望2个结果，实际得到 %d 个", len(results))
	}

	// 验证第一个结果（COVERED）
	covered := results[0]
	if covered.FuncID1 != 123 {
		t.Errorf("第一个结果 FuncID1 期望 123，实际 %d", covered.FuncID1)
	}
	if covered.FuncID2 != 456 {
		t.Errorf("第一个结果 FuncID2 期望 456，实际 %d", covered.FuncID2)
	}
	if covered.CallLoc1.File != "pkg/test.go" {
		t.Errorf("第一个结果 CallLoc1.File 期望 'pkg/test.go'，实际 '%s'", covered.CallLoc1.File)
	}
	if covered.CallLoc1.Line != 42 {
		t.Errorf("第一个结果 CallLoc1.Line 期望 42，实际 %d", covered.CallLoc1.Line)
	}
	if covered.CallLoc2.File != "pkg/handler.go" {
		t.Errorf("第一个结果 CallLoc2.File 期望 'pkg/handler.go'，实际 '%s'", covered.CallLoc2.File)
	}
	if covered.CallLoc2.Line != 88 {
		t.Errorf("第一个结果 CallLoc2.Line 期望 88，实际 %d", covered.CallLoc2.Line)
	}
	if covered.Confidence != 1.00 {
		t.Errorf("第一个结果 Confidence 期望 1.00，实际 %.2f", covered.Confidence)
	}
	if covered.SourceType != "observed" {
		t.Errorf("第一个结果 SourceType 期望 'observed'，实际 '%s'", covered.SourceType)
	}
	if !covered.IsObserved {
		t.Errorf("第一个结果 IsObserved 期望 true，实际 %v", covered.IsObserved)
	}

	// 验证第二个结果（SUSPECT）
	suspect := results[1]
	if suspect.FuncID1 != 789 {
		t.Errorf("第二个结果 FuncID1 期望 789，实际 %d", suspect.FuncID1)
	}
	if suspect.FuncID2 != 101 {
		t.Errorf("第二个结果 FuncID2 期望 101，实际 %d", suspect.FuncID2)
	}
	if suspect.CallLoc1.File != "pkg/service.go" {
		t.Errorf("第二个结果 CallLoc1.File 期望 'pkg/service.go'，实际 '%s'", suspect.CallLoc1.File)
	}
	if suspect.CallLoc1.Line != 15 {
		t.Errorf("第二个结果 CallLoc1.Line 期望 15，实际 %d", suspect.CallLoc1.Line)
	}
	if suspect.CallLoc2.File != "pkg/controller.go" {
		t.Errorf("第二个结果 CallLoc2.File 期望 'pkg/controller.go'，实际 '%s'", suspect.CallLoc2.File)
	}
	if suspect.CallLoc2.Line != 99 {
		t.Errorf("第二个结果 CallLoc2.Line 期望 99，实际 %d", suspect.CallLoc2.Line)
	}
	if suspect.Confidence != 0.60 {
		t.Errorf("第二个结果 Confidence 期望 0.60，实际 %.2f", suspect.Confidence)
	}
	if suspect.SourceType != "inferred_parent1" {
		t.Errorf("第二个结果 SourceType 期望 'inferred_parent1'，实际 '%s'", suspect.SourceType)
	}
	if suspect.IsObserved {
		t.Errorf("第二个结果 IsObserved 期望 false，实际 %v", suspect.IsObserved)
	}
}

func TestParseStdPairs_WindowsPath(t *testing.T) {
	// 测试 Windows 路径解析（包含多个冒号）
	input := `[COVERED] 100,200|D:\gopath\src\gopie\pkg\test.go:42,C:\project\handler.go:88|1.00|observed;`

	results, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("期望1个结果，实际得到 %d 个", len(results))
	}

	result := results[0]

	// Windows 路径应该正确解析，最后一个冒号后面是行号
	expectedFile1 := "D:\\gopath\\src\\gopie\\pkg\\test.go"
	if result.CallLoc1.File != expectedFile1 {
		t.Errorf("CallLoc1.File 期望 '%s'，实际 '%s'", expectedFile1, result.CallLoc1.File)
	}
	if result.CallLoc1.Line != 42 {
		t.Errorf("CallLoc1.Line 期望 42，实际 %d", result.CallLoc1.Line)
	}

	expectedFile2 := "C:\\project\\handler.go"
	if result.CallLoc2.File != expectedFile2 {
		t.Errorf("CallLoc2.File 期望 '%s'，实际 '%s'", expectedFile2, result.CallLoc2.File)
	}
	if result.CallLoc2.Line != 88 {
		t.Errorf("CallLoc2.Line 期望 88，实际 %d", result.CallLoc2.Line)
	}
}

func TestSuspiciousPairInfo_String(t *testing.T) {
	tests := []struct {
		name     string
		info     *SuspiciousPairInfo
		expected string
	}{
		{
			name: "COVERED 格式输出",
			info: &SuspiciousPairInfo{
				FuncID1:    100,
				FuncID2:    200,
				CallLoc1:   CallLocationInfo{File: "main.go", Line: 10},
				CallLoc2:   CallLocationInfo{File: "worker.go", Line: 20},
				Confidence: 1.00,
				SourceType: "observed",
				IsObserved: true,
			},
			expected: "[COVERED] 100,200|main.go:10,worker.go:20|1.00|observed;\n",
		},
		{
			name: "SUSPECT 格式输出",
			info: &SuspiciousPairInfo{
				FuncID1:    100,
				FuncID2:    300,
				CallLoc1:   CallLocationInfo{File: "main.go", Line: 10},
				CallLoc2:   CallLocationInfo{File: "handler.go", Line: 30},
				Confidence: 0.60,
				SourceType: "inferred_parent1",
				IsObserved: false,
			},
			expected: "[SUSPECT] 100,300|main.go:10,handler.go:30|0.60|inferred_parent1;\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.String()
			if result != tt.expected {
				t.Errorf("期望 '%s'，实际 '%s'", tt.expected, result)
			}
		})
	}
}

func TestParseStdPairs_RoundTrip(t *testing.T) {
	// 测试序列化和反序列化的一致性
	original := &SuspiciousPairInfo{
		FuncID1:    123,
		FuncID2:    456,
		CallLoc1:   CallLocationInfo{File: "pkg/test.go", Line: 42},
		CallLoc2:   CallLocationInfo{File: "pkg/handler.go", Line: 88},
		Confidence: 0.75,
		SourceType: "inferred_child2",
		IsObserved: false,
	}

	// 序列化为字符串
	str := original.String()
	t.Logf("序列化结果: %s", str)

	// 反序列化
	results, err := ParseStdPairs(str)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("期望1个结果，实际得到 %d 个", len(results))
	}

	parsed := results[0]

	// 验证所有字段是否一致
	if parsed.FuncID1 != original.FuncID1 {
		t.Errorf("FuncID1 不一致: 原始=%d, 解析后=%d", original.FuncID1, parsed.FuncID1)
	}
	if parsed.FuncID2 != original.FuncID2 {
		t.Errorf("FuncID2 不一致: 原始=%d, 解析后=%d", original.FuncID2, parsed.FuncID2)
	}
	if parsed.CallLoc1.File != original.CallLoc1.File {
		t.Errorf("CallLoc1.File 不一致: 原始=%s, 解析后=%s", original.CallLoc1.File, parsed.CallLoc1.File)
	}
	if parsed.CallLoc1.Line != original.CallLoc1.Line {
		t.Errorf("CallLoc1.Line 不一致: 原始=%d, 解析后=%d", original.CallLoc1.Line, parsed.CallLoc1.Line)
	}
	if parsed.CallLoc2.File != original.CallLoc2.File {
		t.Errorf("CallLoc2.File 不一致: 原始=%s, 解析后=%s", original.CallLoc2.File, parsed.CallLoc2.File)
	}
	if parsed.CallLoc2.Line != original.CallLoc2.Line {
		t.Errorf("CallLoc2.Line 不一致: 原始=%d, 解析后=%d", original.CallLoc2.Line, parsed.CallLoc2.Line)
	}
	if parsed.Confidence != original.Confidence {
		t.Errorf("Confidence 不一致: 原始=%.2f, 解析后=%.2f", original.Confidence, parsed.Confidence)
	}
	if parsed.SourceType != original.SourceType {
		t.Errorf("SourceType 不一致: 原始=%s, 解析后=%s", original.SourceType, parsed.SourceType)
	}
	if parsed.IsObserved != original.IsObserved {
		t.Errorf("IsObserved 不一致: 原始=%v, 解析后=%v", original.IsObserved, parsed.IsObserved)
	}
}
