package bug

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const raceMarker = "WARNING: DATA RACE"

var (
	accessHeaderRE = regexp.MustCompile(`(?i)^(?:Previous )?(Read|Write) at .+ by goroutine [0-9]+:$`)
	fileLineRE     = regexp.MustCompile(`^(.+\.go):([0-9]+)(?:\s+\+0x[0-9a-fA-F]+)?$`)
	hexValueRE     = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	goroutineIDRE  = regexp.MustCompile(`(?i)goroutine\s+[0-9]+`)
	createdByGIDRE = regexp.MustCompile(`(?i)\s+in\s+goroutine\s+[0-9]+$`)
	recoveredRE    = regexp.MustCompile(`(?i)\s*\[\s*recovered(?:\s*,\s*repanicked)?\s*\]\s*$`)
	spaceRE        = regexp.MustCompile(`\s+`)
	boundsExprRE   = regexp.MustCompile(`\[[^]]*\]`)
	boundsSizeRE   = regexp.MustCompile(`(?i)(\b(?:length|capacity)\s+)-?[0-9]+`)
	// nilReceiverRE 匹配栈帧中方法接收者为 0x0 的形态，如 (*BeeMap).Count(0x0)
	nilReceiverRE = regexp.MustCompile(`\)\.\w+\(0x0[,)]`)
)

type stackFrame struct {
	function string
	file     string
	line     string
}

func (f stackFrame) String() string {
	location := filepath.Base(strings.ReplaceAll(f.file, `\`, "/")) + ":" + f.line
	if f.function == "" {
		return location
	}
	return f.function + "@" + location
}

type raceAccess struct {
	kind  string
	frame stackFrame
}

func (a raceAccess) String() string {
	return strings.ToLower(a.kind) + " " + a.frame.String()
}

// Parse 从一次执行中提取并去重运行时预言机事件。
// 两个输出流都会被检查，因为 Go 可能根据二进制文件的启动方式，
// 将测试输出和诊断信息写入不同的描述符（stderr/stdout）。
func Parse(stdout, stderr string) []Event {
	seen := make(map[string]struct{})
	var events []Event
	for _, stream := range []string{stdout, stderr} {
		for _, event := range parseStream(stream) {
			key := string(event.Kind) + "\x00" + event.Signature
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			events = append(events, event)
		}
	}
	return events
}

func parseStream(stream string) []Event {
	stream = normalizeNewlines(stream)
	lines := strings.Split(stream, "\n")
	var events []Event
	events = append(events, parseRaceEvents(lines)...)
	events = append(events, parseCrashEvents(lines)...)
	return events
}

func parseRaceEvents(lines []string) []Event {
	var events []Event
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != raceMarker {
			continue
		}

		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == raceMarker {
				end = j
				break
			}
			if trimmed == "==================" {
				end = j + 1
				break
			}
		}
		block := lines[i:end]
		events = append(events, newRaceEvent(block))
		i = end - 1
	}
	return events
}

func newRaceEvent(block []string) Event {
	accesses := parseRaceAccesses(block)
	report := strings.TrimSpace(strings.Join(block, "\n"))
	if len(accesses) >= 2 {
		items := []string{accesses[0].String(), accesses[1].String()}
		sort.Strings(items)
		message := strings.Join(items, " <-> ")
		return Event{
			Kind:      KindDataRace,
			Signature: signature(string(KindDataRace) + "\n" + message),
			Message:   message,
			Report:    report,
		}
	}

	normalized := normalizeDynamicText(report)
	return Event{
		Kind:      KindDataRace,
		Signature: signature(string(KindDataRace) + "\n" + normalized),
		Message:   "data race (access frames unavailable)",
		Report:    report,
	}
}

func parseRaceAccesses(block []string) []raceAccess {
	var accesses []raceAccess
	for i := 0; i < len(block); i++ {
		match := accessHeaderRE.FindStringSubmatch(strings.TrimSpace(block[i]))
		if match == nil {
			continue
		}

		end := len(block)
		for j := i + 1; j < len(block); j++ {
			trimmed := strings.TrimSpace(block[j])
			if accessHeaderRE.MatchString(trimmed) || strings.HasPrefix(trimmed, "Goroutine ") || trimmed == "==================" {
				end = j
				break
			}
		}
		if frame, ok := firstApplicationFrame(block, i+1, end); ok {
			accesses = append(accesses, raceAccess{kind: match[1], frame: frame})
		}
		i = end - 1
		if len(accesses) == 2 {
			break
		}
	}
	return accesses
}

func parseCrashEvents(lines []string) []Event {
	type candidate struct {
		event    Event
		identity string
		hasFrame bool
	}

	var candidates []candidate
	framedIdentities := make(map[string]struct{})
	for i, line := range lines {
		headerKind, message, ok := parseCrashHeader(line)
		if !ok {
			continue
		}
		kind := classifyCrash(headerKind, message)
		reportEnd := crashReportEnd(lines, i+1, message)
		if kind == KindPanic && isNilReceiverCrash(message, lines[i+1:reportEnd]) {
			// 疑似测试顺序依赖（见 KindTestOrderPanic），降级为非触发种类
			kind = KindTestOrderPanic
		}
		frame, hasFrame := firstCrashApplicationFrame(lines, i+1, message)
		key := string(kind) + "\n" + message
		if hasFrame {
			key += "\n" + frame.String()
		}
		sig := signature(key)
		identity := string(kind) + "\x00" + message
		if hasFrame {
			framedIdentities[identity] = struct{}{}
		}

		candidates = append(candidates, candidate{
			event: Event{
				Kind:      kind,
				Signature: sig,
				Message:   message,
				Report:    strings.TrimSpace(strings.Join(lines[i:reportEnd], "\n")),
			},
			identity: identity,
			hasFrame: hasFrame,
		})
	}

	seen := make(map[string]struct{})
	events := make([]Event, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.hasFrame {
			if _, ok := framedIdentities[candidate.identity]; ok {
				continue
			}
		}
		key := string(candidate.event.Kind) + "\x00" + candidate.event.Signature
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		events = append(events, candidate.event)
	}
	return events
}

func parseCrashHeader(line string) (Kind, string, bool) {
	line = strings.TrimSpace(line)
	lower := strings.ToLower(line)
	var kind Kind
	var message string
	switch {
	case strings.HasPrefix(lower, "panic:"):
		kind = KindPanic
		message = strings.TrimSpace(line[len("panic:"):])
	case strings.HasPrefix(lower, "fatal error:"):
		kind = KindFatal
		message = strings.TrimSpace(line[len("fatal error:"):])
	default:
		return "", "", false
	}
	message = normalizeCrashMessage(message)
	if message == "" {
		message = "<empty>"
	}
	return kind, message, true
}

func normalizeCrashMessage(message string) string {
	message = recoveredRE.ReplaceAllString(strings.TrimSpace(message), "")
	message = hexValueRE.ReplaceAllString(message, "0x?")
	message = goroutineIDRE.ReplaceAllString(message, "goroutine ?")
	message = strings.ToLower(spaceRE.ReplaceAllString(message, " "))
	if strings.HasPrefix(message, "test timed out after ") {
		return "test timed out"
	}
	if strings.HasPrefix(message, "runtime error: index out of range") ||
		strings.HasPrefix(message, "runtime error: slice bounds out of range") {
		message = boundsExprRE.ReplaceAllString(message, "[?]")
		return boundsSizeRE.ReplaceAllString(message, "${1}?")
	}
	return message
}

func classifyCrash(headerKind Kind, message string) Kind {
	if strings.HasPrefix(message, "test timed out") || message == "all goroutines are asleep - deadlock!" {
		return KindHangCandidate
	}
	return headerKind
}

// isNilReceiverCrash 判断 panic 是否为 nil 接收者解引用：nil 指针消息 +
// 栈帧中方法接收者显示为 0x0（如 (*BeeMap).Count(0x0)）。结合 GoPie 单独
// 运行 _1 测试的调度方式，该模式通常是测试间顺序依赖（包级共享变量未初始化）
// 而非被测代码缺陷。
func isNilReceiverCrash(message string, report []string) bool {
	if !strings.Contains(message, "invalid memory address or nil pointer dereference") {
		return false
	}
	for _, line := range report {
		if nilReceiverRE.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func firstCrashApplicationFrame(lines []string, start int, message string) (stackFrame, bool) {
	end := crashReportEnd(lines, start, message)
	return firstApplicationFrame(lines, start, end)
}

func crashReportEnd(lines []string, start int, message string) int {
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == raceMarker || trimmed == "==================" || strings.HasPrefix(trimmed, "FAIL\t") || strings.HasPrefix(trimmed, "--- FAIL:") || strings.HasPrefix(trimmed, "----------------------------------------") {
			return i
		}
		_, nextMessage, ok := parseCrashHeader(lines[i])
		if ok && nextMessage != message {
			return i
		}
	}
	return len(lines)
}

func firstApplicationFrame(lines []string, start, end int) (stackFrame, bool) {
	var previous string
	for i := start; i < end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		match := fileLineRE.FindStringSubmatch(trimmed)
		if match == nil {
			if trimmed != "" {
				previous = trimmed
			}
			continue
		}

		frame := stackFrame{
			function: normalizeFunction(previous),
			file:     strings.ReplaceAll(match[1], `\`, "/"),
			line:     match[2],
		}
		if isApplicationFrame(frame) {
			return frame, true
		}
		previous = ""
	}
	return stackFrame{}, false
}

func normalizeFunction(function string) string {
	function = strings.TrimSpace(strings.TrimPrefix(function, "created by "))
	function = createdByGIDRE.ReplaceAllString(function, "")
	function = hexValueRE.ReplaceAllString(function, "0x?")
	if idx := strings.LastIndex(function, "("); idx > strings.LastIndex(function, ".") {
		function = function[:idx]
	}
	return strings.TrimSpace(function)
}

func isApplicationFrame(frame stackFrame) bool {
	function := frame.function
	file := strings.ToLower(strings.ReplaceAll(frame.file, `\`, "/"))
	excludedFunctions := []string{
		"internal/runtime/",
		"runtime.",
		"runtime/",
		"testing.",
		"testing/",
		"reflect.",
		"reflect/",
		"sync.",
		"sync/",
		"toolkit/pkg/goroutine.",
		"toolkit/pkg/operation.",
		"toolkit/pkg/function.",
		"toolkit/pkg/inst.",
	}
	for _, prefix := range excludedFunctions {
		if strings.HasPrefix(function, prefix) {
			return false
		}
	}
	excludedPaths := []string{
		"/src/internal/runtime/",
		"/src/runtime/",
		"/src/testing/",
		"/src/reflect/",
		"/src/sync/",
		"/toolkit/pkg/goroutine/",
		"/toolkit/pkg/operation/",
		"/toolkit/pkg/function/",
		"/toolkit/pkg/inst/",
	}
	for _, fragment := range excludedPaths {
		if strings.Contains(file, fragment) {
			return false
		}
	}
	return true
}

func normalizeDynamicText(text string) string {
	text = normalizeNewlines(text)
	text = hexValueRE.ReplaceAllString(text, "0x?")
	text = goroutineIDRE.ReplaceAllString(text, "goroutine ?")
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(spaceRE.ReplaceAllString(lines[i], " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizeNewlines(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}

func signature(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
