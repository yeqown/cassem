package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"

	dbapi "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/log"
)

func TestParseEndpoints(t *testing.T) {
	assert.Equal(t, []string{"127.0.0.1:2021"}, parseEndpoints("127.0.0.1:2021"))
	assert.Equal(t, []string{"127.0.0.1:2021", "127.0.0.1:2022"}, parseEndpoints(" 127.0.0.1:2021, ,127.0.0.1:2022 "))
	assert.Equal(t, []string{"127.0.0.1:2021"}, parseEndpoints(""))
}

func TestDisplayValue(t *testing.T) {
	assert.Equal(t, "hello world", displayValue([]byte("hello world")))
	assert.Equal(t, "base64:AP8=", displayValue([]byte{0x00, 0xff}))
}

func TestResolveSetValue(t *testing.T) {
	literal, err := resolveSetValue(setValueInput{Value: "hello world", ValueSet: true})
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world"), literal)

	emptyLiteral, err := resolveSetValue(setValueInput{Value: "", ValueSet: true})
	require.NoError(t, err)
	assert.Equal(t, []byte(""), emptyLiteral)

	decoded, err := resolveSetValue(setValueInput{Base64: "aGVsbG8=", Base64Set: true})
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), decoded)

	emptyBase64, err := resolveSetValue(setValueInput{Base64: "", Base64Set: true})
	require.NoError(t, err)
	assert.Equal(t, []byte(""), emptyBase64)

	file := t.TempDir() + "/value.txt"
	require.NoError(t, os.WriteFile(file, []byte("from-file"), 0o600))
	fromFile, err := resolveSetValue(setValueInput{File: file, FileSet: true})
	require.NoError(t, err)
	assert.Equal(t, []byte("from-file"), fromFile)

	_, err = resolveSetValue(setValueInput{Value: "a", ValueSet: true, Base64: "Yg==", Base64Set: true})
	assert.ErrorContains(t, err, "exactly one of --value, --file, or --base64")

	asDir, err := resolveSetValue(setValueInput{IsDir: true})
	require.NoError(t, err)
	assert.Empty(t, asDir)
}

func TestResolveSetValueRejectsMissingForElement(t *testing.T) {
	_, err := resolveSetValue(setValueInput{})
	assert.ErrorContains(t, err, "exactly one of --value, --file, or --base64")
}

func TestResolveSetValueRejectsInvalidBase64(t *testing.T) {
	_, err := resolveSetValue(setValueInput{Base64: "not-base64", Base64Set: true})
	assert.Error(t, err)
}

func TestResolveSetValueRejectsDirWithExplicitEmptyLiteral(t *testing.T) {
	_, err := resolveSetValue(setValueInput{Value: "", ValueSet: true, IsDir: true})
	assert.ErrorContains(t, err, "--dir cannot be combined with --value, --file, or --base64")
}

func TestResolveSetValueRejectsDirWithExplicitEmptyBase64(t *testing.T) {
	_, err := resolveSetValue(setValueInput{Base64: "", Base64Set: true, IsDir: true})
	assert.ErrorContains(t, err, "--dir cannot be combined with --value, --file, or --base64")
}

func TestParseREPLLine(t *testing.T) {
	args, err := parseREPLLine(`set cassem/debug/key --value "hello world" --overwrite`)
	require.NoError(t, err)
	assert.Equal(t, []string{"set", "cassem/debug/key", "--value", "hello world", "--overwrite"}, args)

	args, err = parseREPLLine("   ")
	require.NoError(t, err)
	assert.Empty(t, args)

	_, err = parseREPLLine(`set "unterminated`)
	assert.Error(t, err)
}

func TestNormalizeREPLArgsMovesFlagsBeforePositionals(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeREPLArgs(app, []string{"set", "cassem/debug/key", "--value", "hello world", "--overwrite"})
	assert.Equal(t, []string{"set", "--value", "hello world", "--overwrite", "cassem/debug/key"}, args)
}

func TestNormalizeREPLArgsSupportsInlineAssignments(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeREPLArgs(app, []string{"set", "cassem/debug/key", "--value=hello", "--base64=", "--ttl=3", "--overwrite"})
	assert.Equal(t, []string{"set", "--value", "hello", "--base64", "", "--ttl", "3", "--overwrite", "cassem/debug/key"}, args)
}

func TestNormalizeREPLArgsMovesGlobalFlagsBeforeCommand(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeREPLArgs(app, []string{"set", "cassem/debug/key", "--json", "--timeout=3s", "--value=hello", "-e", "127.0.0.1:2022"})
	assert.Equal(t, []string{"--json", "--timeout", "3s", "-e", "127.0.0.1:2022", "set", "--value", "hello", "cassem/debug/key"}, args)
}

func TestNormalizeOneShotArgsMovesPostKeyCommandFlags(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeOneShotArgs(app, []string{"cassemdb-viewer", "set", "cassem/debug/key", "--value", "hello world", "--overwrite"})
	assert.Equal(t, []string{"cassemdb-viewer", "set", "--value", "hello world", "--overwrite", "cassem/debug/key"}, args)
}

func TestNormalizeOneShotArgsPreservesGlobalFlagsBeforeCommand(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeOneShotArgs(app, []string{"cassemdb-viewer", "--json", "--timeout=3s", "set", "cassem/debug/key", "--value=hello", "--overwrite"})
	assert.Equal(t, []string{"cassemdb-viewer", "--json", "--timeout=3s", "set", "--value", "hello", "--overwrite", "cassem/debug/key"}, args)
}

func TestNormalizeOneShotArgsSkipsSeparatedLongGlobalFlagBeforeCommand(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeOneShotArgs(app, []string{"cassemdb-viewer", "--endpoints", "127.0.0.1:1", "set", "key", "--value", "hello", "--overwrite"})
	assert.Equal(t, []string{"cassemdb-viewer", "--endpoints", "127.0.0.1:1", "set", "--value", "hello", "--overwrite", "key"}, args)
}

func TestNormalizeOneShotArgsSkipsSeparatedShortGlobalFlagBeforeCommand(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeOneShotArgs(app, []string{"cassemdb-viewer", "-e", "127.0.0.1:1", "set", "key", "--value", "hello", "--overwrite"})
	assert.Equal(t, []string{"cassemdb-viewer", "-e", "127.0.0.1:1", "set", "--value", "hello", "--overwrite", "key"}, args)
}

func TestNormalizeOneShotArgsSkipsAssignedGlobalFlagBeforeCommand(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeOneShotArgs(app, []string{"cassemdb-viewer", "--endpoints=127.0.0.1:1", "set", "key", "--value", "hello", "--overwrite"})
	assert.Equal(t, []string{"cassemdb-viewer", "--endpoints=127.0.0.1:1", "set", "--value", "hello", "--overwrite", "key"}, args)
}

func TestNormalizeOneShotArgsSkipsMixedGlobalFlagsBeforeCommand(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeOneShotArgs(app, []string{"cassemdb-viewer", "--json", "--timeout", "1ms", "--endpoints", "127.0.0.1:1", "set", "key", "--value", "hello", "--overwrite"})
	assert.Equal(t, []string{"cassemdb-viewer", "--json", "--timeout", "1ms", "--endpoints", "127.0.0.1:1", "set", "--value", "hello", "--overwrite", "key"}, args)
}

func TestNormalizeOneShotArgsMovesCommandLocalGlobalFlagsBeforeCommand(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := normalizeOneShotArgs(app, []string{"cassemdb-viewer", "set", "cassem/debug/key", "--value=hello", "--json", "-e", "127.0.0.1:2022"})
	assert.Equal(t, []string{"cassemdb-viewer", "--json", "-e", "127.0.0.1:2022", "set", "--value", "hello", "cassem/debug/key"}, args)
}

func TestNormalizeOneShotArgsLeavesNoCommandInputUntouched(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	args := []string{"cassemdb-viewer", "--json"}
	assert.Equal(t, args, normalizeOneShotArgs(app, args))
}

func TestNewValueView(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		view := newValueView([]byte("hello"))
		assert.Equal(t, valueView{Encoding: "text", Data: "hello"}, view)
		assert.Equal(t, "hello", view.Display())
	})

	t.Run("binary", func(t *testing.T) {
		view := newValueView([]byte{0x00, 0xff})
		assert.Equal(t, valueView{Encoding: "base64", Data: "AP8="}, view)
		assert.Equal(t, "base64:AP8=", view.Display())
	})
}

func TestNewEntityView(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		view := newEntityView(&dbapi.Entity{
			Key:       "cassem/debug/key",
			Typ:       dbapi.EntityType_ELT,
			Size:      5,
			Ttl:       60,
			CreatedAt: 11,
			UpdatedAt: 22,
			Val:       []byte("hello"),
		})

		assert.Equal(t, "cassem/debug/key", view.Key)
		assert.Equal(t, "ELT", view.Type)
		assert.Equal(t, int32(5), view.Size)
		assert.Equal(t, int32(60), view.TTL)
		assert.Equal(t, int64(11), view.CreatedAt)
		assert.Equal(t, int64(22), view.UpdatedAt)
		assert.Equal(t, valueView{Encoding: "text", Data: "hello"}, view.Value)
		assert.Equal(t, "hello", view.DisplayValue)
	})

	t.Run("binary", func(t *testing.T) {
		view := newEntityView(&dbapi.Entity{
			Key: "cassem/debug/key",
			Typ: dbapi.EntityType_ELT,
			Val: []byte{0x00, 0xff},
		})

		assert.Equal(t, int32(2), view.Size)
		assert.Equal(t, valueView{Encoding: "base64", Data: "AP8="}, view.Value)
		assert.Equal(t, "base64:AP8=", view.DisplayValue)
	})
}

func TestNewChangeView(t *testing.T) {
	view := newChangeView(&dbapi.Change{
		Op:  dbapi.Change_Set,
		Key: "cassem/debug/key",
		Current: &dbapi.Entity{
			Key:  "cassem/debug/key",
			Typ:  dbapi.EntityType_ELT,
			Val:  []byte("next"),
			Size: 4,
		},
	})

	assert.Equal(t, "Set", view.Op)
	assert.Equal(t, "cassem/debug/key", view.Key)
	assert.Equal(t, valueView{Encoding: "text", Data: "next"}, view.Current.Value)
	assert.Equal(t, "next", view.Current.DisplayValue)
}

func TestWithKVClientUnarySharesTimeoutContext(t *testing.T) {
	app, ctx := newTestCLIContext(t, defaultTimeout)
	baseCtx := context.Background()
	ctx.Context = baseCtx

	originalDialer := dialKVClient
	defer func() { dialKVClient = originalDialer }()

	var dialCtx context.Context
	dialKVClient = func(ctx context.Context, endpoints []string, mode dbapi.Mode) (dbapi.KVClient, func() error, error) {
		dialCtx = ctx
		return nil, func() error { return nil }, nil
	}

	var rpcCtx context.Context
	err := withKVClient(ctx, dbapi.Mode_R, true, func(ctx context.Context, client dbapi.KVClient) error {
		rpcCtx = ctx
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, dialCtx)
	require.NotNil(t, rpcCtx)
	assert.True(t, dialCtx == rpcCtx)
	assert.False(t, baseCtx == rpcCtx)

	deadline, ok := rpcCtx.Deadline()
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(defaultTimeout), deadline, time.Second)
	_ = app
}

func TestWithKVClientExpandsSingleWriteEndpointToDefaultCluster(t *testing.T) {
	_, ctx := newTestCLIContext(t, defaultTimeout)
	require.NoError(t, ctx.Set("endpoints", "127.0.0.1:2021"))

	originalDialer := dialKVClient
	defer func() { dialKVClient = originalDialer }()

	var gotEndpoints []string
	dialKVClient = func(ctx context.Context, endpoints []string, mode dbapi.Mode) (dbapi.KVClient, func() error, error) {
		gotEndpoints = endpoints
		return nil, func() error { return nil }, nil
	}

	err := withKVClient(ctx, dbapi.Mode_X, true, func(ctx context.Context, client dbapi.KVClient) error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"127.0.0.1:2021", "127.0.0.1:2022", "127.0.0.1:2023"}, gotEndpoints)
}

func TestWithKVClientWatchSeparatesDialAndOperationContext(t *testing.T) {
	_, ctx := newTestCLIContext(t, defaultTimeout)
	baseCtx := context.Background()
	ctx.Context = baseCtx

	originalDialer := dialKVClient
	defer func() { dialKVClient = originalDialer }()

	var dialCtx context.Context
	dialKVClient = func(ctx context.Context, endpoints []string, mode dbapi.Mode) (dbapi.KVClient, func() error, error) {
		dialCtx = ctx
		return nil, func() error { return nil }, nil
	}

	var rpcCtx context.Context
	err := withKVClient(ctx, dbapi.Mode_R, false, func(ctx context.Context, client dbapi.KVClient) error {
		rpcCtx = ctx
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, dialCtx)
	require.NotNil(t, rpcCtx)
	assert.True(t, baseCtx == rpcCtx)
	assert.False(t, dialCtx == rpcCtx)

	_, hasDeadline := rpcCtx.Deadline()
	assert.False(t, hasDeadline)

	dialDeadline, ok := dialCtx.Deadline()
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(defaultTimeout), dialDeadline, time.Second)
}

func TestPrintErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	err := printError(&buf, errors.New("boom"))
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload))
	assert.Equal(t, false, payload["ok"])
	assert.Equal(t, "boom", payload["error"])
}

func TestPrintErrorJSONWithNilError(t *testing.T) {
	var buf bytes.Buffer
	err := printError(&buf, nil)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload))
	assert.Equal(t, false, payload["ok"])
	assert.Equal(t, "", payload["error"])
}

func TestRenderCLIErrorJSON(t *testing.T) {
	var out bytes.Buffer
	app, ctx := newTestCLIContextWithIO(t, defaultTimeout, &out, &out)
	_ = app

	renderCLIError(ctx, errors.New("usage: set <key>"))

	var payload map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	assert.Equal(t, false, payload["ok"])
	assert.Equal(t, "usage: set <key>", payload["error"])
}

func TestEntityViewJSONUsesStructuredValue(t *testing.T) {
	view := newEntityView(&dbapi.Entity{
		Key: "cassem/debug/key",
		Typ: dbapi.EntityType_ELT,
		Val: []byte{0x00, 0xff},
	})

	data, err := json.Marshal(view)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"key":"cassem/debug/key",
		"type":"ELT",
		"size":2,
		"ttl":0,
		"createdAt":0,
		"updatedAt":0,
		"value":{"encoding":"base64","data":"AP8="}
	}`, string(data))
}

func TestChangeViewJSONUsesStructuredValue(t *testing.T) {
	view := newChangeView(&dbapi.Change{
		Op:  dbapi.Change_Set,
		Key: "cassem/debug/key",
		Last: &dbapi.Entity{
			Key: "cassem/debug/key",
			Typ: dbapi.EntityType_ELT,
			Val: []byte("prev"),
		},
		Current: &dbapi.Entity{
			Key: "cassem/debug/key",
			Typ: dbapi.EntityType_ELT,
			Val: []byte{0x00, 0xff},
		},
	})

	data, err := json.Marshal(view)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"op":"Set",
		"key":"cassem/debug/key",
		"last":{
			"key":"cassem/debug/key",
			"type":"ELT",
			"size":4,
			"ttl":0,
			"createdAt":0,
			"updatedAt":0,
			"value":{"encoding":"text","data":"prev"}
		},
		"current":{
			"key":"cassem/debug/key",
			"type":"ELT",
			"size":2,
			"ttl":0,
			"createdAt":0,
			"updatedAt":0,
			"value":{"encoding":"base64","data":"AP8="}
		}
	}`, string(data))
}

func TestPrintChangeViewUsesDisplayValue(t *testing.T) {
	var out bytes.Buffer
	app := cli.NewApp()
	app.Writer = &out
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.Bool("json", false, "")
	ctx := cli.NewContext(app, set, nil)

	require.NoError(t, printChangeView(ctx, changeView{
		Op:  "Set",
		Key: "cassem/debug/key",
		Last: entityView{
			DisplayValue: "prev",
			Value:        valueView{Encoding: "text", Data: "prev"},
		},
		Current: entityView{
			DisplayValue: "base64:AP8=",
			Value:        valueView{Encoding: "base64", Data: "AP8="},
		},
	}))

	assert.Equal(t, "Set\tcassem/debug/key\tlast=prev\tcurrent=base64:AP8=\n", out.String())
}

func TestPrintEntityViewsShowsDirectoryRowsCleanly(t *testing.T) {
	var out bytes.Buffer
	app := cli.NewApp()
	app.Writer = &out
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.Bool("json", false, "")
	ctx := cli.NewContext(app, set, nil)

	require.NoError(t, printEntityViews(ctx, []entityView{newEntityView(&dbapi.Entity{Key: "normalized"})}))

	got := out.String()
	assert.Contains(t, got, "normalized")
	assert.Contains(t, got, "DIR")
	assert.Contains(t, got, "<dir>")
	assert.NotContains(t, got, "UNKNOWN")
}

func TestPrintEntityViewsTruncatesLongValues(t *testing.T) {
	var out bytes.Buffer
	app := cli.NewApp()
	app.Writer = &out
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.Bool("json", false, "")
	ctx := cli.NewContext(app, set, nil)

	view := newEntityView(&dbapi.Entity{Key: "cassem/debug/key", Typ: dbapi.EntityType_ELT, Val: []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")})
	require.NoError(t, printEntityViews(ctx, []entityView{view}))

	got := out.String()
	assert.Contains(t, got, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUV...")
	assert.NotContains(t, got, "WXYZ")
}

func TestPrintPaginationHintShowsNextSeekFlag(t *testing.T) {
	var out bytes.Buffer
	_, ctx := newTestCLIContextWithIO(t, defaultTimeout, &out, nil)

	printPaginationHint(ctx, "6")

	assert.Equal(t, "has more: use --seek \"6\" for next page\n", out.String())
}

func TestListCommandDefaultsPrefixToCassem(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := newApp(os.Stdout, os.Stderr)
	app.Writer = stdout
	app.ErrWriter = stderr

	originalDialer := dialKVClient
	defer func() { dialKVClient = originalDialer }()

	var gotReq *dbapi.RangeReq
	dialKVClient = func(ctx context.Context, endpoints []string, mode dbapi.Mode) (dbapi.KVClient, func() error, error) {
		return fakeKVClient{
			rangeFunc: func(ctx context.Context, req *dbapi.RangeReq) (*dbapi.RangeResp, error) {
				gotReq = req
				return &dbapi.RangeResp{}, nil
			},
		}, func() error { return nil }, nil
	}

	err := app.Run([]string{"cassemdb-viewer", "list"})
	require.NoError(t, err)
	require.NotNil(t, gotReq)
	assert.Equal(t, "cassem", gotReq.GetKey())
	assert.Empty(t, stderr.String())
}

func TestListCommandUsesRangeEntityKeys(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := newApp(os.Stdout, os.Stderr)
	app.Writer = stdout
	app.ErrWriter = stderr

	originalDialer := dialKVClient
	defer func() { dialKVClient = originalDialer }()

	dialKVClient = func(ctx context.Context, endpoints []string, mode dbapi.Mode) (dbapi.KVClient, func() error, error) {
		return fakeKVClient{
			rangeFunc: func(ctx context.Context, req *dbapi.RangeReq) (*dbapi.RangeResp, error) {
				return &dbapi.RangeResp{Entities: []*dbapi.Entity{{Key: "cassem/acl"}}}, nil
			},
		}, func() error { return nil }, nil
	}

	err := app.Run([]string{"cassemdb-viewer", "list"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "cassem/acl")
	assert.Empty(t, stderr.String())
}

type fakeKVClient struct {
	rangeFunc func(context.Context, *dbapi.RangeReq) (*dbapi.RangeResp, error)
}

func (f fakeKVClient) GetKV(context.Context, *dbapi.GetKVReq, ...grpc.CallOption) (*dbapi.GetKVResp, error) {
	return nil, errors.New("not implemented")
}

func (f fakeKVClient) GetKVs(context.Context, *dbapi.GetKVsReq, ...grpc.CallOption) (*dbapi.GetKVsResp, error) {
	return nil, errors.New("not implemented")
}

func (f fakeKVClient) SetKV(context.Context, *dbapi.SetKVReq, ...grpc.CallOption) (*dbapi.Empty, error) {
	return nil, errors.New("not implemented")
}

func (f fakeKVClient) UnsetKV(context.Context, *dbapi.UnsetKVReq, ...grpc.CallOption) (*dbapi.Empty, error) {
	return nil, errors.New("not implemented")
}

func (f fakeKVClient) Watch(context.Context, *dbapi.WatchReq, ...grpc.CallOption) (dbapi.KV_WatchClient, error) {
	return nil, errors.New("not implemented")
}

func (f fakeKVClient) TTL(context.Context, *dbapi.TtlReq, ...grpc.CallOption) (*dbapi.TtlResp, error) {
	return nil, errors.New("not implemented")
}

func (f fakeKVClient) Expire(context.Context, *dbapi.ExpireReq, ...grpc.CallOption) (*dbapi.Empty, error) {
	return nil, errors.New("not implemented")
}

func (f fakeKVClient) Range(ctx context.Context, req *dbapi.RangeReq, opts ...grpc.CallOption) (*dbapi.RangeResp, error) {
	if f.rangeFunc == nil {
		return nil, errors.New("not implemented")
	}
	return f.rangeFunc(ctx, req)
}

func (f fakeKVClient) CompactElementHistory(context.Context, *dbapi.CompactElementHistoryReq, ...grpc.CallOption) (*dbapi.CompactElementHistoryResp, error) {
	return nil, errors.New("not implemented")
}

func TestVersionStringUsesQuotedJSONValues(t *testing.T) {
	app := newApp(os.Stdout, os.Stderr)
	assert.JSONEq(t, `{"version":"undefined","buildTime":"undefined","gitHash":"undefined"}`, app.Version)
}

func TestAppRunJSONExitError(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := newApp(os.Stdout, os.Stderr)
	app.Writer = stdout
	app.ErrWriter = stderr

	err := app.Run([]string{"cassemdb-viewer", "--json", "set"})
	require.Error(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &payload))
	assert.Equal(t, false, payload["ok"])
	assert.Equal(t, "usage: set <key>", payload["error"])
}

func TestDefaultCLILogLevelSuppressesBelowWarning(t *testing.T) {
	assert.Equal(t, log.LevelWarning, defaultCLILogLevel())
}

func TestAppRunCommandErrorUsesErrorWriter(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := newApp(os.Stdout, os.Stderr)
	app.Writer = stdout
	app.ErrWriter = stderr

	err := app.Run([]string{"cassemdb-viewer", "set"})
	require.Error(t, err)

	assert.Empty(t, stdout.String())
	assert.Equal(t, "error: usage: set <key>\n", stderr.String())
}

func newTestCLIContext(t *testing.T, timeout time.Duration) (*cli.App, *cli.Context) {
	t.Helper()
	return newTestCLIContextWithIO(t, timeout, nil, nil)
}

func newTestCLIContextWithIO(t *testing.T, timeout time.Duration, out, errOut io.Writer) (*cli.App, *cli.Context) {
	t.Helper()

	app := cli.NewApp()
	if out != nil {
		app.Writer = out
	}
	if errOut != nil {
		app.ErrWriter = errOut
	}
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.Duration("timeout", defaultTimeout, "")
	set.String("endpoints", defaultEndpoint, "")
	set.Bool("json", false, "")
	require.NoError(t, set.Set("timeout", timeout.String()))
	require.NoError(t, set.Set("endpoints", defaultEndpoint))
	require.NoError(t, set.Set("json", "true"))

	ctx := cli.NewContext(app, set, nil)
	ctx.Context = context.Background()
	return app, ctx
}
