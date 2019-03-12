package v1

import (
	"context"
	"reflect"
	"testing"

	"google.golang.org/grpc/metadata"

	v1 "github.com/olamai/simulation/pkg/api/v1"
)

func Test_simulationServiceServer_CreateAgent(t *testing.T) {
	ctxWithoutValidToken := context.Background()
	md := metadata.Pairs("auth-token", "TEST-ID-TOKEN")
	ctxWithValidToken := metadata.NewIncomingContext(context.Background(), md)

	s := NewSimulationServiceServer("testing")

	type args struct {
		ctx context.Context
		req *v1.CreateAgentRequest
	}
	tests := []struct {
		name    string
		s       v1.SimulationServiceServer
		args    args
		want    *v1.CreateAgentResponse
		wantErr bool
	}{
		{
			name: "Succesful Agent Creation",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.CreateAgentRequest{
					Api: "v1",
					Agent: &v1.Entity{
						X: 0,
						Y: 0,
					},
				},
			},
			want: &v1.CreateAgentResponse{
				Api: "v1",
				Id:  0,
			},
		},
		{
			name: "Unsupported API",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.CreateAgentRequest{
					Api: "v1000",
					Agent: &v1.Entity{
						X: 0,
						Y: 0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Agent already exists error",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.CreateAgentRequest{
					Api: "v1",
					Agent: &v1.Entity{
						X: 0,
						Y: 0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid auth token",
			s:    s,
			args: args{
				ctx: ctxWithoutValidToken,
				req: &v1.CreateAgentRequest{
					Api: "v1",
					Agent: &v1.Entity{
						X: 0,
						Y: 0,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.CreateAgent(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("simulationService.CreateAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toDoServiceServer.CreateAgent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_simulationServiceServer_GetAgent(t *testing.T) {
	ctxWithoutValidToken := context.Background()
	md := metadata.Pairs("auth-token", "TEST-ID-TOKEN")
	ctxWithValidToken := metadata.NewIncomingContext(context.Background(), md)

	s := NewSimulationServiceServer("testing")

	// Create an agent to test on
	agent := &v1.Entity{
		X: 2,
		Y: -4,
	}
	resp, err := s.CreateAgent(ctxWithValidToken, &v1.CreateAgentRequest{
		Api:   "v1",
		Agent: agent,
	})
	if err != nil {
		t.Errorf("toDoServiceServer.Create() error creating test agent. Make sure CreateAgent functionality is working.")
		return
	}

	agentID := resp.Id

	type args struct {
		ctx context.Context
		req *v1.GetEntityRequest
	}
	tests := []struct {
		name    string
		s       v1.SimulationServiceServer
		args    args
		want    *v1.GetEntityResponse
		wantErr bool
	}{
		{
			name: "Succesful Agent Retreival",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.GetEntityRequest{
					Api: "v1",
					Id:  agentID,
				},
			},
			want: &v1.GetEntityResponse{
				Api:    "v1",
				Entity: agent,
			},
		},
		{
			name: "Unsupported API",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.GetEntityRequest{
					Api: "v1000",
					Id:  agentID,
				},
			},
			wantErr: true,
		},
		{
			name: "Agent not found by that ID",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.GetEntityRequest{
					Api: "v1",
					Id:  999,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid auth token",
			s:    s,
			args: args{
				ctx: ctxWithoutValidToken,
				req: &v1.GetEntityRequest{
					Api: "v1",
					Id:  0,
				},
			},
			want: &v1.GetEntityResponse{
				Api:    "v1",
				Entity: agent,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetEntity(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("simulationService.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simulationService.Create() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_simulationServiceServer_DeleteAgent(t *testing.T) {
	ctxWithoutValidToken := context.Background()
	md := metadata.Pairs("auth-token", "TEST-ID-TOKEN")
	ctxWithValidToken := metadata.NewIncomingContext(context.Background(), md)

	s := NewSimulationServiceServer("testing")

	// Create an agent to test on
	agent := &v1.Entity{
		X: 2,
		Y: -4,
	}
	resp, err := s.CreateAgent(ctxWithValidToken, &v1.CreateAgentRequest{
		Api:   "v1",
		Agent: agent,
	})
	if err != nil {
		t.Errorf("toDoServiceServer.Create() error creating test agent. Make sure CreateAgent functionality is working.")
		return
	}
	agentID := resp.Id

	// Run tests
	type args struct {
		ctx context.Context
		req *v1.DeleteAgentRequest
	}
	tests := []struct {
		name    string
		s       v1.SimulationServiceServer
		args    args
		want    *v1.DeleteAgentResponse
		wantErr bool
	}{
		{
			name: "Unsupported API",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.DeleteAgentRequest{
					Api: "v1000",
					Id:  agentID,
				},
			},
			wantErr: true,
		},
		{
			name: "Agent not found by that ID",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.DeleteAgentRequest{
					Api: "v1",
					Id:  999,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid auth token",
			s:    s,
			args: args{
				ctx: ctxWithoutValidToken,
				req: &v1.DeleteAgentRequest{
					Api: "v1",
					Id:  0,
				},
			},
			wantErr: true,
		},
		{
			name: "Successful Agent Delete",
			s:    s,
			args: args{
				ctx: ctxWithValidToken,
				req: &v1.DeleteAgentRequest{
					Api: "v1",
					Id:  agentID,
				},
			},
			want: &v1.DeleteAgentResponse{
				Api:     "v1",
				Deleted: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.DeleteAgent(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("simulationService.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simulationService.Create() = %v, want %v", got, tt.want)
			}
		})
	}
}
