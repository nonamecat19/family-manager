package rpc

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"google.golang.org/protobuf/types/known/emptypb"
)

func serveProcedure(t *testing.T, procedure string, fail bool) *connect.Client[emptypb.Empty, emptypb.Empty] {
	t.Helper()
	log, _ := debugLogger()
	handler := connect.NewUnaryHandler(procedure,
		func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			if fail {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("no"))
			}
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(Observe(log)),
	)
	mux := http.NewServeMux()
	mux.Handle(procedure, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return connect.NewClient[emptypb.Empty, emptypb.Empty](srv.Client(), srv.URL+procedure)
}

func TestObserveCountsCallsByProcedureAndCode(t *testing.T) {
	const procedure = "/metrics.test.v1.Svc/Count"
	ok := serveProcedure(t, procedure, false)
	denied := serveProcedure(t, procedure, true)

	okBefore := testutil.ToFloat64(requestsTotal.WithLabelValues(procedure, "ok"))
	deniedBefore := testutil.ToFloat64(requestsTotal.WithLabelValues(procedure, "permission_denied"))

	for range 2 {
		if _, err := ok.CallUnary(context.Background(), connect.NewRequest(&emptypb.Empty{})); err != nil {
			t.Fatalf("ok call: %v", err)
		}
	}
	if _, err := denied.CallUnary(context.Background(), connect.NewRequest(&emptypb.Empty{})); err == nil {
		t.Fatal("denied call succeeded")
	}

	if got := testutil.ToFloat64(requestsTotal.WithLabelValues(procedure, "ok")) - okBefore; got != 2 {
		t.Errorf("ok count delta = %v, want 2", got)
	}
	if got := testutil.ToFloat64(requestsTotal.WithLabelValues(procedure, "permission_denied")) - deniedBefore; got != 1 {
		t.Errorf("permission_denied count delta = %v, want 1", got)
	}
}

func TestMetricsHandlerExposesRPCAndRuntimeMetrics(t *testing.T) {
	const procedure = "/metrics.test.v1.Svc/Scrape"
	client := serveProcedure(t, procedure, false)
	if _, err := client.CallUnary(context.Background(), connect.NewRequest(&emptypb.Empty{})); err != nil {
		t.Fatalf("call: %v", err)
	}

	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, MetricsPath, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	for _, want := range []string{
		`rpc_server_requests_total{code="ok",procedure="` + procedure + `"}`,
		`rpc_server_request_duration_seconds_bucket{procedure="` + procedure + `"`,
		"go_goroutines",
		"process_cpu_seconds_total",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("scrape is missing %q", want)
		}
	}
}
