package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hasura/gotel"
	"github.com/relychan/gohttpc"
	"github.com/relychan/gohttpc/httpconfig"
	"github.com/relychan/goutils"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("gorestly")

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		// Level: gohttpc.LogLevelTrace,
	}))
	slog.SetDefault(logger)

	_ = os.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "1000")

	otlpConfig := &gotel.OTLPConfig{
		ServiceName:         "gohttpc",
		OtlpTracesEndpoint:  "http://localhost:4317",
		OtlpMetricsEndpoint: "http://localhost:9090/api/v1/otlp/v1/metrics",
		OtlpMetricsProtocol: gotel.OTLPProtocolHTTPProtobuf,
		MetricsExporter:     gotel.OTELMetricsExporterOTLP,
	}

	exporters, err := gotel.SetupOTelExporters(context.Background(), otlpConfig, "v0.1.0", logger)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		_ = exporters.Shutdown(context.Background())
	}()

	metrics, err := gohttpc.NewHTTPClientMetrics(exporters.Meter, true)
	if err != nil {
		panic(err)
	}

	client, err := httpconfig.NewClientFromConfig(
		httpconfig.HTTPClientConfig{},
		gohttpc.WithTracer(exporters.Tracer),
		gohttpc.WithMetrics(metrics),
		gohttpc.EnableClientTrace(true),
	)
	if err != nil {
		panic(err)
	}

	for i := range 100 {
		getTodo(client, i+1)
		createPost(client, i+1)
		time.Sleep(time.Second)
	}

	goutils.CatchWarnErrorFunc(client.Close)
}

func getTodo(client *gohttpc.Client, id int) {
	ctx, span := tracer.Start(context.Background(), "getTodo")
	defer span.End()

	endpoint := "https://jsonplaceholder.typicode.com/todos/" + strconv.Itoa(id)

	resp, err := client.NewRequest(http.MethodGet, endpoint).Execute(ctx)
	if err != nil {
		panic(err)
	}

	_ = resp.Close()
}

func createPost(client *gohttpc.Client, id int) {
	ctx, span := tracer.Start(context.Background(), "createPost")
	defer span.End()

	endpoint := "https://jsonplaceholder.typicode.com/posts"

	req := client.NewRequest(http.MethodPost, endpoint)

	body, err := json.Marshal(map[string]any{
		"id":   id + 1,
		"name": "test",
	})
	if err != nil {
		panic(err)
	}

	req.SetBody(bytes.NewReader(body))

	resp, err := req.Execute(ctx)
	if err != nil {
		panic(err)
	}

	_ = resp.Close()
}
