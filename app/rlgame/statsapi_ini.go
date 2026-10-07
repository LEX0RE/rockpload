package rlgame

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	StatsAPIFileName     = "DefaultStatsAPI.ini"
	StatsAPIUserFileName = "TAStatsAPI.ini"
	StatsAPISection      = "TAGame.MatchStatsExporter_TA"

	KeyPort           = "Port"
	KeyWebPort        = "WebPort"
	KeyPacketSendRate = "PacketSendRate"

	DefaultStatsAPIPort    = 49123
	DefaultStatsAPIWebPort = 49124
	MaxPacketSendRate      = 120
	RecommendedSendRate    = 30
	maxPort                = 65535
	unrealArrayOperators   = "+-.!"
)

var (
	ErrInstallNotFound = errors.New("rocket league installation not found")
	ErrKeyNotFound     = errors.New("key not found in StatsAPI file")
	ErrFileChanged     = errors.New("the file was changed by another program, reload it before saving")
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type StatsAPIEntry struct {
	Key     string
	Value   string
	Comment string

	line int
}

type iniLine struct {
	text string
	eol  string
}

// StatsAPIFile edits DefaultStatsAPI.ini without touching what it does not understand.
// Only the value of edited keys is rewritten, comments, line endings and unknown lines are kept as is.
type StatsAPIFile struct {
	Path    string
	Entries []StatsAPIEntry

	// Problem explains why the file cannot be edited with the form (empty when it can)
	Problem string
	// RawEditable is false when the file content cannot be shown as text without corrupting it
	RawEditable bool

	original []byte
	bom      []byte
	lines    []iniLine
}

func LoadStatsAPIFile(path string) (*StatsAPIFile, error) {
	logger.FuncDebug()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	file := ParseStatsAPI(data)
	file.Path = path

	return file, nil
}

func ParseStatsAPI(data []byte) *StatsAPIFile {
	logger.FuncDebug()

	file := &StatsAPIFile{original: data, RawEditable: true}
	content := data

	if bytes.HasPrefix(content, utf8BOM) {
		file.bom = utf8BOM
		content = content[len(utf8BOM):]
	}

	if bytes.HasPrefix(content, []byte{0xFF, 0xFE}) || bytes.HasPrefix(content, []byte{0xFE, 0xFF}) {
		file.Problem = "The file is encoded in UTF-16, which is not supported by the editor."
		file.RawEditable = false
		return file
	}

	if !utf8.Valid(content) {
		file.Problem = "The file contains invalid characters, which is not supported by the editor."
		file.RawEditable = false
		return file
	}

	file.lines = splitLines(string(content))
	file.parse()

	return file
}

func splitLines(content string) []iniLine {
	lines := []iniLine{}

	for len(content) > 0 {
		index := strings.IndexByte(content, '\n')
		if index < 0 {
			lines = append(lines, iniLine{text: content})
			break
		}

		text, eol := content[:index], "\n"
		if strings.HasSuffix(text, "\r") {
			text, eol = text[:len(text)-1], "\r\n"
		}

		lines = append(lines, iniLine{text: text, eol: eol})
		content = content[index+1:]
	}

	return lines
}

func (f *StatsAPIFile) parse() {
	logger.FuncDebug()

	inSection := false
	sectionFound := false
	comments := []string{}

	for i, line := range f.lines {
		trimmed := strings.TrimSpace(line.text)

		switch {
		case trimmed == "":
			comments = comments[:0]

		case strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#"):
			comments = append(comments, strings.TrimSpace(trimmed[1:]))

		case strings.HasPrefix(trimmed, "["):
			name := strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
			inSection = strings.EqualFold(strings.TrimSpace(name), StatsAPISection)
			sectionFound = sectionFound || inSection
			comments = comments[:0]

		case !inSection:
			comments = comments[:0]

		default:
			rawKey, value, ok := strings.Cut(trimmed, "=")
			key := strings.TrimSpace(rawKey)

			if !ok || key == "" {
				f.setProblem(fmt.Sprintf("Line %d is not a \"Key=Value\" line.", i+1))
				continue
			}

			if strings.ContainsAny(key[:1], unrealArrayOperators) {
				f.setProblem(fmt.Sprintf("Line %d uses the Unreal array syntax (%q).", i+1, key))
				continue
			}

			if f.entryIndex(key) >= 0 {
				f.setProblem(fmt.Sprintf("The key %q is defined more than once.", key))
				continue
			}

			f.Entries = append(f.Entries, StatsAPIEntry{
				Key:     key,
				Value:   strings.TrimSpace(value),
				Comment: strings.Join(comments, " "),
				line:    i,
			})
			comments = comments[:0]
		}
	}

	if !sectionFound {
		f.setProblem(fmt.Sprintf("The section [%s] was not found.", StatsAPISection))
	} else if len(f.Entries) == 0 {
		f.setProblem(fmt.Sprintf("The section [%s] has no setting.", StatsAPISection))
	}
}

func (f *StatsAPIFile) setProblem(problem string) {
	if f.Problem == "" {
		f.Problem = problem
	}
}

// CanUseForm tells if the file is simple enough to be edited key by key.
func (f *StatsAPIFile) CanUseForm() bool {
	return f.Problem == ""
}

func (f *StatsAPIFile) entryIndex(key string) int {
	for i, entry := range f.Entries {
		if strings.EqualFold(entry.Key, key) {
			return i
		}
	}

	return -1
}

func (f *StatsAPIFile) Get(key string) (string, bool) {
	logger.FuncDebug()

	index := f.entryIndex(key)
	if index < 0 {
		return "", false
	}

	return f.Entries[index].Value, true
}

func (f *StatsAPIFile) GetInt(key string) (int, error) {
	logger.FuncDebug()

	value, ok := f.Get(key)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}

	return strconv.Atoi(value)
}

// StatsAPIPorts are the sockets opened by the game, 0 meaning the socket is disabled.
type StatsAPIPorts struct {
	TCP int
	Web int
}

func DefaultStatsAPIPorts() StatsAPIPorts {
	return StatsAPIPorts{TCP: DefaultStatsAPIPort, Web: DefaultStatsAPIWebPort}
}

// Ports returns the TCP and WebSocket ports, using the game defaults for missing keys.
func (f *StatsAPIFile) Ports() (StatsAPIPorts, error) {
	logger.FuncDebug()

	ports := DefaultStatsAPIPorts()

	for key, target := range map[string]*int{KeyPort: &ports.TCP, KeyWebPort: &ports.Web} {
		if _, ok := f.Get(key); !ok {
			continue
		}

		port, err := f.GetInt(key)
		if err != nil {
			return StatsAPIPorts{}, fmt.Errorf("invalid %s: %w", key, err)
		}

		*target = port
	}

	return ports, nil
}

// IsSendRateDisabled tells if a PacketSendRate value disables the StatsAPI.
func IsSendRateDisabled(value string) bool {
	rate, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && rate == 0
}

// Set changes the value of an existing key. Unknown keys are never added.
func (f *StatsAPIFile) Set(key string, value string) error {
	logger.FuncDebug()

	if !f.CanUseForm() {
		return errors.New(f.Problem)
	}

	index := f.entryIndex(key)
	if index < 0 {
		return fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}

	value = strings.TrimSpace(value)
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("the value of %s cannot contain a new line", key)
	}

	entry := &f.Entries[index]
	line := &f.lines[entry.line]

	equalIndex := strings.IndexByte(line.text, '=')
	oldValue := line.text[equalIndex+1:]
	spacing := oldValue[:len(oldValue)-len(strings.TrimLeft(oldValue, " \t"))]
	line.text = line.text[:equalIndex+1] + spacing + value
	entry.Value = value

	return nil
}

// Validate checks the values of the known keys.
func (f *StatsAPIFile) Validate() error {
	logger.FuncDebug()

	values := map[string]string{}
	for _, entry := range f.Entries {
		if err := ValidateStatsAPIValue(entry.Key, entry.Value); err != nil {
			return err
		}
		values[strings.ToLower(entry.Key)] = entry.Value
	}

	port, hasPort := values[strings.ToLower(KeyPort)]
	webPort, hasWebPort := values[strings.ToLower(KeyWebPort)]
	if hasPort && hasWebPort && port != "0" && port == webPort {
		return fmt.Errorf("%s and %s must be different", KeyPort, KeyWebPort)
	}

	return nil
}

// ValidateStatsAPIValue checks a single value, unknown keys only need to fit on a line.
func ValidateStatsAPIValue(key string, value string) error {
	value = strings.TrimSpace(value)

	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("the value of %s cannot contain a new line", key)
	}

	switch {
	case strings.EqualFold(key, KeyPacketSendRate):
		return validateFloatRange(key, value, 0, MaxPacketSendRate)
	case strings.EqualFold(key, KeyPort), strings.EqualFold(key, KeyWebPort):
		return validateIntRange(key, value, 0, maxPort)
	}

	return nil
}

func validateFloatRange(key string, value string, minimum float64, maximum float64) error {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return fmt.Errorf("%s must be a number", key)
	}

	if number < minimum || number > maximum {
		return fmt.Errorf("%s must be between %g and %g", key, minimum, maximum)
	}

	return nil
}

func validateIntRange(key string, value string, minimum int, maximum int) error {
	number, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%s must be a whole number", key)
	}

	if number < minimum || number > maximum {
		return fmt.Errorf("%s must be between %d and %d", key, minimum, maximum)
	}

	return nil
}

func (f *StatsAPIFile) Bytes() []byte {
	logger.FuncDebug()

	if f.lines == nil {
		return f.original
	}

	var buffer bytes.Buffer
	buffer.Write(f.bom)

	for _, line := range f.lines {
		buffer.WriteString(line.text)
		buffer.WriteString(line.eol)
	}

	return buffer.Bytes()
}

// RawText returns the content shown in the raw editor.
func (f *StatsAPIFile) RawText() string {
	logger.FuncDebug()

	return strings.ReplaceAll(string(bytes.TrimPrefix(f.Bytes(), utf8BOM)), "\r\n", "\n")
}

// SetRawText replaces the whole content, keeping the line ending style of the file.
func (f *StatsAPIFile) SetRawText(text string) error {
	logger.FuncDebug()

	if !f.RawEditable {
		return errors.New(f.Problem)
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	if bytes.Contains(f.original, []byte("\r\n")) {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}

	data := append(append([]byte{}, f.bom...), text...)
	parsed := ParseStatsAPI(data)
	parsed.Path = f.Path
	parsed.original = f.original

	*f = *parsed

	return nil
}

// Save writes the file, refusing to overwrite it if it changed since it was loaded.
func (f *StatsAPIFile) Save() error {
	logger.FuncDebug()

	current, err := os.ReadFile(f.Path)
	if err != nil {
		return err
	}

	if !bytes.Equal(current, f.original) {
		return ErrFileChanged
	}

	data := f.Bytes()
	if bytes.Equal(data, f.original) {
		return nil
	}

	// Only rewrite the content so the file keeps its owner and permissions
	if err := os.WriteFile(f.Path, data, 0644); err != nil {
		return err
	}

	f.original = data

	return nil
}
