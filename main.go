package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

var version string

var (
	borderColor  = lipgloss.Color("#303030")
	subtleColor  = lipgloss.Color("#b0b0b0")
	successColor = lipgloss.Color("#00aa00")
	errorColor   = lipgloss.Color("#ca0000")
	getColor     = successColor
	postColor    = lipgloss.Color("#0000bb")
	putColor     = lipgloss.Color("#cc7700")
	keyColor     = lipgloss.Color("#f92672")
)

func ParseHTTPHeaders(headers string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(headers))

	for scanner.Scan() {
		line := scanner.Text()
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}

func Timestamp(date string) string {
	style := lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(subtleColor)
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return style.Render(date)
	}
	return style.Render(t.Local().Format("2006-01-02 15:04:05"))
}

func HighlightSyntax(source string, lang string) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}
	style := styles.Get("monokai")
	var result strings.Builder
	err = formatter.Format(&result, style, iterator)
	if err != nil {
		return source
	}
	return result.String()
}

func Frame(s ...string) string {
	return lipgloss.
		NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(s...)
}

func Iteration(s string) string {
	return lipgloss.
		NewStyle().
		PaddingRight(1).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(borderColor).
		Render(s)
}

func Colored(s string, color lipgloss.Color) string {
	return lipgloss.
		NewStyle().
		Foreground(color).
		Render(s)
}

func Bold(s string) string {
	return lipgloss.
		NewStyle().
		Bold(true).
		Render(s)
}

func Trim(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", ""))
}

func Status(status string) string {
	style := lipgloss.
		NewStyle().
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(borderColor)

	var s string
	switch status[0] {
	case '2':
		s = Colored(status, successColor)
	case '4':
		s = Colored(status, errorColor)
	default:
		s = status

	}
	return style.Render(s)
}

func Method(method string) string {
	style := lipgloss.
		NewStyle().
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(borderColor)

	var m string

	switch method {
	case "GET":
		m = Colored(method, getColor)
	case "POST":
		m = Colored(method, postColor)
	case "PUT":
		m = Colored(method, putColor)
	case "DELETE":
		m = Colored(method, errorColor)
	default:
		m = method
	}

	return style.Render(m)
}

func DecomposeUrl(u string) (string, url.Values) {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return u, nil
	}
	return parsedUrl.Path, parsedUrl.Query()
}

func Url(u string) string {
	style := lipgloss.
		NewStyle().
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(borderColor)

	return style.Render(u)
}

func LogEntryHeader(entry search.Log, i int) string {
	style := lipgloss.
		NewStyle().
		MaxWidth(120).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(borderColor)

	parsedUrl, _ := DecomposeUrl(entry.Url)

	return style.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			Iteration(fmt.Sprintf("#%d", i+1)),
			Method(entry.Method),
			Status(entry.AnswerCode),
			Url(parsedUrl),
			Timestamp(entry.Timestamp),
		),
	)
}

func HeaderSection(entry search.Log) string {
	style := lipgloss.
		NewStyle().
		MaxWidth(120)

	headers := ParseHTTPHeaders(entry.QueryHeaders)
	// Sort the headers alphabetically (case-insentive)
	var keys []string
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	// Style the keys in a different color
	var styledHeaders []string
	for _, key := range keys {
		styledHeaders = append(
			styledHeaders,
			Colored(key, keyColor)+": "+headers[key],
		)
	}

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			Bold("Headers"),
			strings.Join(styledHeaders, "\n"),
		),
	)
}

func QueryParamFromUrlSection(entry search.Log) string {
	style := lipgloss.
		NewStyle().
		MaxWidth(120).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(borderColor)

	_, queryParams := DecomposeUrl(entry.Url)

	if queryParams == nil || len(queryParams) == 0 {
		return ""
	}
	var styledQueryParams []string
	for key, values := range queryParams {
		styledQueryParams = append(
			styledQueryParams,
			Colored(key, keyColor)+": "+strings.Join(values, ", "),
		)
	}

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			Bold("Query params (from URL)"),
			strings.Join(styledQueryParams, "\n"),
		),
	)
}

func QueryParamSection(entry search.Log) string {
	queryParams := entry.QueryParams
	if queryParams == nil {
		return ""
	}

	style := lipgloss.
		NewStyle().
		MaxWidth(120).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(borderColor)

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			Bold("Query params (from response)"),
			*queryParams,
		),
	)
}

func RequestBodySection(entry search.Log) string {
	requestBody := entry.QueryBody
	if len(requestBody) == 0 {
		return ""
	}
	if len(Trim(requestBody)) == 2 {
		requestBody = "{}"
	}

	style := lipgloss.
		NewStyle().
		MaxWidth(120).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(borderColor)

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			Bold("Request body"),
			HighlightSyntax(strings.Trim(requestBody, "\n"), "json"),
		),
	)
}

func ResponseSection(entry search.Log, WithResponse bool) string {
	if !WithResponse {
		return ""
	}
	style := lipgloss.
		NewStyle().
		MaxWidth(120).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(borderColor)

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			Bold("Response (first 1,000 characters)"),
			HighlightSyntax(strings.Trim(entry.Answer, "\n"), "json"),
		),
	)
}

func PrintLogEntry(entry search.Log, i int, WithResponse bool) {
	s := Frame(
		lipgloss.JoinVertical(
			lipgloss.Left,
			LogEntryHeader(entry, i),
			HeaderSection(entry),
			QueryParamFromUrlSection(entry),
			QueryParamSection(entry),
			RequestBodySection(entry),
			ResponseSection(entry, WithResponse),
		),
	)

	fmt.Println(s)
}

type Options struct {
	WithResponse bool
	LastN        int
	Offset       int
	QueryType    string
	Version      bool
}

func cli() Options {
	opts := Options{}
	flag.BoolVar(
		&opts.WithResponse,
		"response",
		false,
		"Whether to print the API response.",
	)
	flag.IntVar(
		&opts.LastN,
		"last",
		10,
		"How many log entries to print.",
	)
	flag.IntVar(
		&opts.Offset,
		"offset",
		0,
		"The number of the first entry to retrieve (starts with 0).",
	)
	flag.StringVar(
		&opts.QueryType,
		"type",
		"all",
		"Type of log entry: all, build, error, query.",
	)
	flag.BoolVar(
		&opts.Version,
		"version",
		false,
		"Print the version information.",
	)
	flag.Parse()

	validQueryTypes := map[string]bool{
		"all":   true,
		"build": true,
		"error": true,
		"query": true,
	}

	if !validQueryTypes[opts.QueryType] {
		log.Fatalf(
			"Invalid type for -request: %s. Allowed are 'all', 'build', 'error', 'query'\n",
			opts.QueryType,
		)
	}

	if opts.Offset < 0 {
		log.Fatalln("-offset must be a positive integer")
	}
	return opts
}

// LoadConfig loads the configuration file with the Algolia credentials
func LoadConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	envPath := filepath.Join(home, ".config", "search-logs.env")
	err = godotenv.Load(envPath)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	opts := cli()

	if opts.Version {
		fmt.Println(version)
		return
	}

	err := LoadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	appID := os.Getenv("ALGOLIA_APPLICATION_ID")
	apiKey := os.Getenv("ALGOLIA_API_KEY")

	client, err := search.NewClient(appID, apiKey)
	if err != nil {
		log.Fatalln(err)
	}

	response, err := client.GetLogs(
		client.
			NewApiGetLogsRequest().
			WithType(search.LogType(opts.QueryType)).
			WithLength(int32(opts.LastN)).
			WithOffset(int32(opts.Offset)),
	)
	if err != nil {
		log.Fatalln(err)
	}

	for i, entry := range response.Logs {
		PrintLogEntry(entry, opts.Offset+i, opts.WithResponse)
	}
}
