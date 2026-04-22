package graphv1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/Keyhole-Koro/SynthifyShared/gen/synthify/graph/v1"
	http "net/http"
	strings "strings"
)

const _ = connect.IsAtLeastVersion1_13_0

const (
	WorkerServiceName                         = "synthify.graph.v1.WorkerService"
	WorkerServiceExecuteApprovedPlanProcedure = "/synthify.graph.v1.WorkerService/ExecuteApprovedPlan"
)

type WorkerServiceClient interface {
	ExecuteApprovedPlan(context.Context, *connect.Request[v1.ExecuteApprovedPlanRequest]) (*connect.Response[v1.ExecuteApprovedPlanResponse], error)
}

func NewWorkerServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) WorkerServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &workerServiceClient{
		executeApprovedPlan: connect.NewClient[v1.ExecuteApprovedPlanRequest, v1.ExecuteApprovedPlanResponse](
			httpClient,
			baseURL+WorkerServiceExecuteApprovedPlanProcedure,
			connect.WithClientOptions(opts...),
		),
	}
}

type workerServiceClient struct {
	executeApprovedPlan *connect.Client[v1.ExecuteApprovedPlanRequest, v1.ExecuteApprovedPlanResponse]
}

func (c *workerServiceClient) ExecuteApprovedPlan(ctx context.Context, req *connect.Request[v1.ExecuteApprovedPlanRequest]) (*connect.Response[v1.ExecuteApprovedPlanResponse], error) {
	return c.executeApprovedPlan.CallUnary(ctx, req)
}

type WorkerServiceHandler interface {
	ExecuteApprovedPlan(context.Context, *connect.Request[v1.ExecuteApprovedPlanRequest]) (*connect.Response[v1.ExecuteApprovedPlanResponse], error)
}

func NewWorkerServiceHandler(svc WorkerServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	executeApprovedPlanHandler := connect.NewUnaryHandler(
		WorkerServiceExecuteApprovedPlanProcedure,
		svc.ExecuteApprovedPlan,
		connect.WithHandlerOptions(opts...),
	)
	return "/synthify.graph.v1.WorkerService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case WorkerServiceExecuteApprovedPlanProcedure:
			executeApprovedPlanHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

type UnimplementedWorkerServiceHandler struct{}

func (UnimplementedWorkerServiceHandler) ExecuteApprovedPlan(context.Context, *connect.Request[v1.ExecuteApprovedPlanRequest]) (*connect.Response[v1.ExecuteApprovedPlanResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("synthify.graph.v1.WorkerService.ExecuteApprovedPlan is not implemented"))
}
