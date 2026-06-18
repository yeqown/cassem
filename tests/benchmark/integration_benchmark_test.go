//go:build integration

package benchmark_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/tests/testutil"
)

func BenchmarkCassemDBSetKV32B(b *testing.B) {
	cluster := testutil.UseDBCluster(b)
	cc := testutil.DialCassemDB(b, cluster.DBEndpoints, apikv.Mode_X)
	b.Cleanup(func() { _ = cc.Close() })
	client := apikv.NewKVClient(cc)
	payload := []byte("12312312312312312312312312312312")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := client.SetKV(ctx, &apikv.SetKVReq{
			Key:       fmt.Sprintf("benchmark/cassemdb/set/%d", i),
			Val:       payload,
			Ttl:       30,
			Overwrite: true,
		})
		cancel()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCassemDBGetKV(b *testing.B) {
	cluster := testutil.UseDBCluster(b)
	cc := testutil.DialCassemDB(b, cluster.DBEndpoints, apikv.Mode_X)
	b.Cleanup(func() { _ = cc.Close() })
	client := apikv.NewKVClient(cc)
	key := "benchmark/cassemdb/get/key"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, err := client.SetKV(ctx, &apikv.SetKVReq{Key: key, Val: []byte("value"), Ttl: 30, Overwrite: true})
	cancel()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := client.GetKV(ctx, &apikv.GetKVReq{Key: key})
			cancel()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCassemadmCreateElement(b *testing.B) {
	cluster := testutil.UseAdmCluster(b)
	adm := testutil.NewHTTPClient(cluster.AdmBaseURL)
	adm.Session = testutil.LoginSuperadmin(b, cluster.AdmBaseURL)
	app := fmt.Sprintf("benchapp%d", time.Now().UnixNano())
	env := "bench"
	adm.DoJSON(b, http.MethodPost, "/api/apps/"+app, map[string]any{"name": app, "description": "benchmark app"}, nil)
	adm.DoJSON(b, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s", app, env), nil, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adm.DoJSON(b, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s/elements/bench-%d", app, env, i), map[string]any{
			"raw":         fmt.Sprintf("benchmark value %d", i),
			"contentType": concept.ContentType_PLAINTEXT,
		}, nil)
	}
}
