package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kballard/go-shellquote"
	"github.com/urfave/cli/v2"
	dbapi "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/log"
)

const defaultEndpoint = "127.0.0.1:2021"
const defaultTimeout = 10 * time.Second
const maxTableValueWidth = 61

var defaultLocalClusterEndpoints = []string{"127.0.0.1:2021", "127.0.0.1:2022", "127.0.0.1:2023"}

func defaultCLILogLevel() log.Level {
	return log.LevelWarning
}

var dialKVClient = func(ctx context.Context, endpoints []string, mode dbapi.Mode) (dbapi.KVClient, func() error, error) {
	conn, err := dbapi.DialWithModeContext(ctx, endpoints, mode)
	if err != nil {
		return nil, nil, err
	}
	return dbapi.NewKVClient(conn), conn.Close, nil
}

func main() {
	log.SetLogLevel(defaultCLILogLevel())

	app := newApp(os.Stdout, os.Stderr)
	if err := app.Run(normalizeOneShotArgs(app, os.Args)); err != nil {
		os.Exit(exitCodeFromError(err))
	}
}

func newApp(out, errOut *os.File) *cli.App {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "cassemdb-viewer"
	app.Usage = "inspect and manage raw cassemdb KV data"
	app.Version = fmt.Sprintf(`{"version":%s,"buildTime":%s,"gitHash":%s}`,
		strconv.Quote(Version),
		strconv.Quote(BuildTime),
		strconv.Quote(GitHash),
	)
	app.Writer = out
	app.ErrWriter = errOut
	app.ExitErrHandler = func(cCtx *cli.Context, err error) {
		renderCLIError(cCtx, err)
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "endpoints", Aliases: []string{"e"}, Value: defaultEndpoint, Usage: "comma-separated cassemdb gRPC endpoints"},
		&cli.BoolFlag{Name: "json", Usage: "emit JSON output"},
		&cli.DurationFlag{Name: "timeout", Value: defaultTimeout, Usage: "timeout for unary requests"},
	}
	app.Commands = buildCommands()
	app.Action = runREPL
	return app
}

type valueView struct {
	Encoding string `json:"encoding"`
	Data     string `json:"data"`
}

func (v valueView) Display() string {
	if v.Encoding == "base64" {
		return "base64:" + v.Data
	}
	return v.Data
}

type entityView struct {
	Key          string    `json:"key"`
	Type         string    `json:"type"`
	Size         int32     `json:"size"`
	TTL          int32     `json:"ttl"`
	CreatedAt    int64     `json:"createdAt"`
	UpdatedAt    int64     `json:"updatedAt"`
	Value        valueView `json:"value"`
	DisplayValue string    `json:"-"`
	IsDir        bool      `json:"-"`
}

type changeView struct {
	Op      string     `json:"op"`
	Key     string     `json:"key"`
	Last    entityView `json:"last"`
	Current entityView `json:"current"`
}

func newValueView(value []byte) valueView {
	if isPrintableUTF8(value) {
		return valueView{Encoding: "text", Data: string(value)}
	}
	return valueView{Encoding: "base64", Data: base64.StdEncoding.EncodeToString(value)}
}

func newEntityView(entity *dbapi.Entity) entityView {
	if entity == nil {
		return entityView{}
	}

	if entity.GetTyp() == dbapi.EntityType_UNKNOWN && entity.GetSize() == 0 && entity.GetCreatedAt() == 0 && entity.GetUpdatedAt() == 0 && len(entity.GetVal()) == 0 {
		return entityView{
			Key:          entity.GetKey(),
			Type:         dbapi.EntityType_DIR.String(),
			DisplayValue: "<dir>",
			IsDir:        true,
		}
	}

	value := newValueView(entity.GetVal())
	size := entity.GetSize()
	if size == 0 && len(entity.GetVal()) > 0 {
		size = int32(len(entity.GetVal()))
	}
	return entityView{
		Key:          entity.GetKey(),
		Type:         entity.GetTyp().String(),
		Size:         size,
		TTL:          entity.GetTtl(),
		CreatedAt:    entity.GetCreatedAt(),
		UpdatedAt:    entity.GetUpdatedAt(),
		Value:        value,
		DisplayValue: value.Display(),
	}
}

func newChangeView(change *dbapi.Change) changeView {
	if change == nil {
		return changeView{}
	}

	return changeView{
		Op:      change.GetOp().String(),
		Key:     change.GetKey(),
		Last:    newEntityView(change.GetLast()),
		Current: newEntityView(change.GetCurrent()),
	}
}

type setValueInput struct {
	Value     string
	ValueSet  bool
	File      string
	FileSet   bool
	Base64    string
	Base64Set bool
	IsDir     bool
}

func parseEndpoints(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{defaultEndpoint}
	}

	parts := strings.Split(raw, ",")
	endpoints := make([]string, 0, len(parts))
	for _, part := range parts {
		endpoint := strings.TrimSpace(part)
		if endpoint == "" {
			continue
		}
		endpoints = append(endpoints, endpoint)
	}
	if len(endpoints) == 0 {
		return []string{defaultEndpoint}
	}
	return endpoints
}

func expandEndpointsForMode(endpoints []string, mode dbapi.Mode) []string {
	if mode != dbapi.Mode_X || len(endpoints) != 1 || endpoints[0] != defaultEndpoint {
		return endpoints
	}
	return append([]string(nil), defaultLocalClusterEndpoints...)
}

func displayValue(value []byte) string {
	if isPrintableUTF8(value) {
		return string(value)
	}
	return "base64:" + base64.StdEncoding.EncodeToString(value)
}

func isPrintableUTF8(value []byte) bool {
	if len(value) == 0 {
		return true
	}
	if !utf8.Valid(value) {
		return false
	}
	for len(value) > 0 {
		r, size := utf8.DecodeRune(value)
		if r == utf8.RuneError && size == 1 {
			return false
		}
		if r != '\n' && r != '\r' && r != '\t' && !unicode.IsPrint(r) {
			return false
		}
		value = value[size:]
	}
	return true
}

func resolveSetValue(input setValueInput) ([]byte, error) {
	provided := 0
	if input.ValueSet {
		provided++
	}
	if input.FileSet {
		provided++
	}
	if input.Base64Set {
		provided++
	}
	if provided != 1 {
		if input.IsDir && provided == 0 {
			return nil, nil
		}
		return nil, errors.New("exactly one of --value, --file, or --base64 must be provided")
	}
	if input.IsDir {
		return nil, errors.New("--dir cannot be combined with --value, --file, or --base64")
	}
	if input.ValueSet {
		return []byte(input.Value), nil
	}
	if input.FileSet {
		return os.ReadFile(input.File)
	}
	return base64.StdEncoding.DecodeString(input.Base64)
}

func parseREPLLine(line string) ([]string, error) {
	if strings.TrimSpace(line) == "" {
		return nil, nil
	}
	return shellquote.Split(line)
}

func normalizeREPLArgs(app *cli.App, args []string) []string {
	return normalizeCommandArgs(app, args)
}

func normalizeOneShotArgs(app *cli.App, args []string) []string {
	if len(args) <= 1 {
		return args
	}

	commandIndex := findCommandIndex(app.Commands, buildFlagKinds(app.Flags), args[1:])
	if commandIndex < 0 {
		return args
	}

	absoluteCommandIndex := commandIndex + 1
	commandArgs := normalizeCommandArgs(app, args[absoluteCommandIndex:])
	if len(commandArgs) == 0 {
		return args
	}

	normalized := make([]string, 0, len(args))
	normalized = append(normalized, args[:absoluteCommandIndex]...)
	normalized = append(normalized, commandArgs...)
	return normalized
}

func normalizeCommandArgs(app *cli.App, args []string) []string {
	if len(args) <= 1 {
		return args
	}

	command := findCommand(app.Commands, args[0])
	if command == nil {
		return args
	}

	globalFlagKinds := buildFlagKinds(app.Flags)
	commandFlagKinds := buildFlagKinds(command.Flags)

	globalArgs := make([]string, 0, len(args)-1)
	commandArgs := make([]string, 0, len(args)-1)
	positionalArgs := make([]string, 0, len(args)-1)
	for i := 1; i < len(args); i++ {
		current := args[i]
		base, inlineValue, hasInlineValue := splitFlagAssignment(current)

		appendFlag := func(dst []string, kind flagKind) ([]string, bool) {
			switch kind {
			case flagBool:
				return append(dst, current), true
			case flagValue:
				dst = append(dst, base)
				if hasInlineValue {
					return append(dst, inlineValue), true
				}
				if i+1 < len(args) {
					i++
					dst = append(dst, args[i])
				}
				return dst, true
			default:
				return dst, false
			}
		}

		if next, ok := appendFlag(commandArgs, commandFlagKinds[base]); ok {
			commandArgs = next
			continue
		}
		if next, ok := appendFlag(globalArgs, globalFlagKinds[base]); ok {
			globalArgs = next
			continue
		}
		positionalArgs = append(positionalArgs, current)
	}

	normalized := make([]string, 0, len(args))
	normalized = append(normalized, globalArgs...)
	normalized = append(normalized, args[0])
	normalized = append(normalized, commandArgs...)
	normalized = append(normalized, positionalArgs...)
	return normalized
}

type flagKind uint8

const (
	flagUnknown flagKind = iota
	flagBool
	flagValue
)

func buildFlagKinds(flags []cli.Flag) map[string]flagKind {
	kinds := make(map[string]flagKind)
	for _, flag := range flags {
		kind := flagValue
		if _, ok := flag.(*cli.BoolFlag); ok {
			kind = flagBool
		}
		for _, name := range flag.Names() {
			kinds["--"+name] = kind
			if len(name) == 1 {
				kinds["-"+name] = kind
			}
		}
	}
	return kinds
}

func splitFlagAssignment(arg string) (string, string, bool) {
	if !strings.HasPrefix(arg, "-") {
		return arg, "", false
	}
	base, value, ok := strings.Cut(arg, "=")
	if !ok {
		return arg, "", false
	}
	return base, value, true
}

func findCommand(commands []*cli.Command, name string) *cli.Command {
	for _, command := range commands {
		if command.HasName(name) {
			return command
		}
	}
	return nil
}

func findCommandIndex(commands []*cli.Command, globalFlagKinds map[string]flagKind, args []string) int {
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		base, _, hasInlineValue := splitFlagAssignment(arg)
		if kind := globalFlagKinds[base]; kind != flagUnknown {
			if kind == flagValue && !hasInlineValue && idx+1 < len(args) {
				idx++
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if findCommand(commands, arg) != nil {
			return idx
		}
		return -1
	}
	return -1
}

func buildCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:      "get",
			Usage:     "get one key",
			ArgsUsage: "<key>",
			Action:    getCommand,
		},
		{
			Name:      "mget",
			Usage:     "get multiple keys",
			ArgsUsage: "<key...>",
			Action:    mgetCommand,
		},
		{
			Name:  "list",
			Usage: "list keys by prefix/range",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "prefix", Value: "cassem", Usage: "prefix key to list"},
				&cli.StringFlag{Name: "seek", Usage: "seek key for pagination"},
				&cli.IntFlag{Name: "limit", Value: 100, Usage: "page size, 1-100"},
			},
			Action: listCommand,
		},
		{
			Name:      "ttl",
			Usage:     "read key ttl",
			ArgsUsage: "<key>",
			Action:    ttlCommand,
		},
		{
			Name:      "set",
			Usage:     "set one key",
			ArgsUsage: "<key>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "value", Usage: "literal UTF-8 value"},
				&cli.StringFlag{Name: "file", Usage: "read value bytes from file"},
				&cli.StringFlag{Name: "base64", Usage: "base64-encoded value"},
				&cli.IntFlag{Name: "ttl", Usage: "ttl in seconds"},
				&cli.BoolFlag{Name: "dir", Usage: "write as directory"},
				&cli.BoolFlag{Name: "overwrite", Usage: "overwrite existing key"},
			},
			Action: setCommand,
		},
		{
			Name:      "unset",
			Usage:     "delete one key",
			ArgsUsage: "<key>",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "dir", Usage: "delete as directory"},
			},
			Action: unsetCommand,
		},
		{
			Name:      "expire",
			Usage:     "expire one key",
			ArgsUsage: "<key>",
			Action:    expireCommand,
		},
		{
			Name:      "watch",
			Usage:     "watch key changes",
			UsageText: "cassemdb-viewer watch <key...>",
			ArgsUsage: "<key...>",
			Action:    watchCommand,
		},
	}
}

func withKVClient(c *cli.Context, mode dbapi.Mode, useTimeout bool, fn func(context.Context, dbapi.KVClient) error) error {
	operationCtx := c.Context
	dialCtx := operationCtx
	cancel := func() {}
	if useTimeout {
		operationCtx, cancel = context.WithTimeout(operationCtx, c.Duration("timeout"))
		dialCtx = operationCtx
	} else {
		dialCtx, cancel = context.WithTimeout(dialCtx, c.Duration("timeout"))
	}
	defer cancel()

	endpoints := expandEndpointsForMode(parseEndpoints(c.String("endpoints")), mode)
	client, closeClient, err := dialKVClient(dialCtx, endpoints, mode)
	if err != nil {
		return fmt.Errorf("dial cassemdb: %w", err)
	}
	defer func() {
		if closeClient != nil {
			_ = closeClient()
		}
	}()

	return fn(operationCtx, client)
}

func printJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func printError(w io.Writer, err error) error {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return printJSON(w, map[string]any{
		"ok":    false,
		"error": message,
	})
}

func renderCLIError(c *cli.Context, err error) {
	if err == nil {
		return
	}
	if c != nil && c.Bool("json") {
		_ = printError(c.App.ErrWriter, err)
		return
	}
	_, _ = fmt.Fprintf(c.App.ErrWriter, "error: %v\n", err)
}

func exitCodeFromError(err error) int {
	var exitErr cli.ExitCoder
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func printOK(c *cli.Context, payload map[string]any) error {
	if c.Bool("json") {
		if payload == nil {
			payload = map[string]any{}
		}
		payload["ok"] = true
		return printJSON(c.App.Writer, payload)
	}

	_, _ = fmt.Fprintln(c.App.Writer, "OK")
	return nil
}

func printEntityViews(c *cli.Context, views []entityView) error {
	if c.Bool("json") {
		return printJSON(c.App.Writer, views)
	}

	tw := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "KEY\tTYPE\tSIZE\tTTL\tCREATED_AT\tUPDATED_AT\tVALUE")
	for _, view := range views {
		size := strconv.FormatInt(int64(view.Size), 10)
		ttl := strconv.FormatInt(int64(view.TTL), 10)
		createdAt := strconv.FormatInt(view.CreatedAt, 10)
		updatedAt := strconv.FormatInt(view.UpdatedAt, 10)
		if view.IsDir {
			size = "-"
			ttl = "-"
			createdAt = "-"
			updatedAt = "-"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", view.Key, view.Type, size, ttl, createdAt, updatedAt, truncateTableValue(view.DisplayValue))
	}
	return tw.Flush()
}

func truncateTableValue(value string) string {
	if utf8.RuneCountInString(value) <= maxTableValueWidth {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxTableValueWidth-3]) + "..."
}

func printPaginationHint(c *cli.Context, nextSeekKey string) {
	_, _ = fmt.Fprintf(c.App.Writer, "has more: use --seek %q for next page\n", nextSeekKey)
}

func printChangeView(c *cli.Context, view changeView) error {
	if c.Bool("json") {
		return printJSON(c.App.Writer, view)
	}

	_, _ = fmt.Fprintf(c.App.Writer, "%s\t%s\tlast=%s\tcurrent=%s\n", view.Op, view.Key, view.Last.DisplayValue, view.Current.DisplayValue)
	return nil
}

func getCommand(c *cli.Context) error {
	if c.NArg() != 1 {
		return cli.Exit("usage: get <key>", 2)
	}

	key := c.Args().Get(0)
	return withKVClient(c, dbapi.Mode_R, true, func(ctx context.Context, client dbapi.KVClient) error {
		resp, err := client.GetKV(ctx, &dbapi.GetKVReq{Key: key})
		if err != nil {
			return fmt.Errorf("get key %q: %w", key, err)
		}
		return printEntityViews(c, []entityView{newEntityView(resp.GetEntity())})
	})
}

func mgetCommand(c *cli.Context) error {
	if c.NArg() < 1 || c.NArg() > 100 {
		return cli.Exit("usage: mget <key...> with 1-100 keys", 2)
	}

	keys := c.Args().Slice()
	return withKVClient(c, dbapi.Mode_R, true, func(ctx context.Context, client dbapi.KVClient) error {
		resp, err := client.GetKVs(ctx, &dbapi.GetKVsReq{Keys: keys})
		if err != nil {
			return fmt.Errorf("mget keys: %w", err)
		}

		views := make([]entityView, 0, len(resp.GetEntities()))
		for _, entity := range resp.GetEntities() {
			views = append(views, newEntityView(entity))
		}
		return printEntityViews(c, views)
	})
}

func listCommand(c *cli.Context) error {
	limit := c.Int("limit")
	if limit < 1 || limit > 100 {
		return cli.Exit("--limit must be between 1 and 100", 2)
	}

	prefix := c.String("prefix")
	seek := c.String("seek")
	return withKVClient(c, dbapi.Mode_R, true, func(ctx context.Context, client dbapi.KVClient) error {
		resp, err := client.Range(ctx, &dbapi.RangeReq{Key: prefix, Seek: seek, Limit: int32(limit)})
		if err != nil {
			return fmt.Errorf("list prefix %q: %w", prefix, err)
		}

		views := make([]entityView, 0, len(resp.GetEntities()))
		for _, entity := range resp.GetEntities() {
			views = append(views, newEntityView(entity))
		}

		if c.Bool("json") {
			return printJSON(c.App.Writer, struct {
				Entities    []entityView `json:"entities"`
				HasMore     bool         `json:"hasMore"`
				NextSeekKey string       `json:"nextSeekKey"`
			}{
				Entities:    views,
				HasMore:     resp.GetHasMore(),
				NextSeekKey: resp.GetNextSeekKey(),
			})
		}

		if err := printEntityViews(c, views); err != nil {
			return err
		}
		if resp.GetHasMore() {
			printPaginationHint(c, resp.GetNextSeekKey())
		}
		return nil
	})
}

func ttlCommand(c *cli.Context) error {
	if c.NArg() != 1 {
		return cli.Exit("usage: ttl <key>", 2)
	}

	key := c.Args().Get(0)
	return withKVClient(c, dbapi.Mode_R, true, func(ctx context.Context, client dbapi.KVClient) error {
		resp, err := client.TTL(ctx, &dbapi.TtlReq{Key: key})
		if err != nil {
			return fmt.Errorf("ttl key %q: %w", key, err)
		}

		if c.Bool("json") {
			return printJSON(c.App.Writer, map[string]any{"key": key, "ttl": resp.GetTtl()})
		}

		_, _ = fmt.Fprintf(c.App.Writer, "%d\n", resp.GetTtl())
		return nil
	})
}

func setCommand(c *cli.Context) error {
	if c.NArg() != 1 {
		return cli.Exit("usage: set <key>", 2)
	}

	key := c.Args().Get(0)
	value, err := resolveSetValue(setValueInput{
		Value:     c.String("value"),
		ValueSet:  c.IsSet("value"),
		File:      c.String("file"),
		FileSet:   c.IsSet("file"),
		Base64:    c.String("base64"),
		Base64Set: c.IsSet("base64"),
		IsDir:     c.Bool("dir"),
	})
	if err != nil {
		return err
	}

	return withKVClient(c, dbapi.Mode_X, true, func(ctx context.Context, client dbapi.KVClient) error {
		_, err := client.SetKV(ctx, &dbapi.SetKVReq{
			Key:       key,
			IsDir:     c.Bool("dir"),
			Ttl:       int32(c.Int("ttl")),
			Val:       value,
			Overwrite: c.Bool("overwrite"),
		})
		if err != nil {
			return fmt.Errorf("set key %q: %w", key, err)
		}
		return printOK(c, map[string]any{"key": key})
	})
}

func unsetCommand(c *cli.Context) error {
	if c.NArg() != 1 {
		return cli.Exit("usage: unset <key>", 2)
	}

	key := c.Args().Get(0)
	return withKVClient(c, dbapi.Mode_X, true, func(ctx context.Context, client dbapi.KVClient) error {
		_, err := client.UnsetKV(ctx, &dbapi.UnsetKVReq{Key: key, IsDir: c.Bool("dir")})
		if err != nil {
			return fmt.Errorf("unset key %q: %w", key, err)
		}
		return printOK(c, map[string]any{"key": key})
	})
}

func expireCommand(c *cli.Context) error {
	if c.NArg() != 1 {
		return cli.Exit("usage: expire <key>", 2)
	}

	key := c.Args().Get(0)
	return withKVClient(c, dbapi.Mode_X, true, func(ctx context.Context, client dbapi.KVClient) error {
		_, err := client.Expire(ctx, &dbapi.ExpireReq{Key: key})
		if err != nil {
			return fmt.Errorf("expire key %q: %w", key, err)
		}
		return printOK(c, map[string]any{"key": key})
	})
}

func watchCommand(c *cli.Context) error {
	if c.NArg() < 1 || c.NArg() > 20 {
		return cli.Exit("usage: watch <key...> with 1-20 keys", 2)
	}

	keys := c.Args().Slice()
	return withKVClient(c, dbapi.Mode_R, false, func(ctx context.Context, client dbapi.KVClient) error {
		stream, err := client.Watch(ctx, &dbapi.WatchReq{Keys: keys})
		if err != nil {
			return fmt.Errorf("watch keys: %w", err)
		}
		for {
			change, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fmt.Errorf("watch stream: %w", err)
			}
			if err := printChangeView(c, newChangeView(change)); err != nil {
				return err
			}
		}
	})
}

func runREPL(c *cli.Context) error {
	reader := bufio.NewScanner(os.Stdin)
	for {
		_, _ = fmt.Fprint(c.App.Writer, "cassemdb-viewer> ")
		if !reader.Scan() {
			return reader.Err()
		}

		args, err := parseREPLLine(reader.Text())
		if err != nil {
			renderCLIError(c, fmt.Errorf("parse error: %w", err))
			continue
		}
		if len(args) == 0 {
			continue
		}
		args = normalizeREPLArgs(c.App, args)

		switch args[0] {
		case "exit", "quit":
			return nil
		case "help":
			_ = cli.ShowAppHelp(c)
			continue
		}

		runArgs := []string{c.App.Name, "--endpoints", strings.Join(parseEndpoints(c.String("endpoints")), ","), "--timeout", c.Duration("timeout").String()}
		if c.Bool("json") {
			runArgs = append(runArgs, "--json")
		}
		runArgs = append(runArgs, args...)
		_ = c.App.RunContext(c.Context, runArgs)
	}
}
