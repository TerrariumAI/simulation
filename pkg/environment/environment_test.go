package environment

import (
	"context"
	b64 "encoding/base64"
	"reflect"
	"testing"

	api "github.com/terrariumai/simulation/pkg/api/environment"
	"google.golang.org/grpc/metadata"
)

const (
	redisAddr          = "localhost:6379"
	userinfoJSONString = "{\"id\":\"MOCK-UID\"}"
)

func TestCreateEntity(t *testing.T) {
	// ctxWithoutValidToken := context.Background()
	userinfoEnc := b64.StdEncoding.EncodeToString([]byte(userinfoJSONString))
	md := metadata.Pairs("x-endpoint-api-userinfo", userinfoEnc)
	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

	s := NewEnvironmentServer("testing", redisAddr)
	type args struct {
		ctx context.Context
		req *api.CreateEntityRequest
	}
	tests := []struct {
		name    string
		s       api.EnvironmentServer
		args    args
		want    *api.CreateEntityResponse
		wantErr bool
	}{
		{
			name: "Succesful Entity Creation",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.CreateEntityRequest{
					Entity: &api.Entity{
						Id:       "0",
						X:        1,
						Y:        1,
						Class:    1,
						OwnerUID: "MOCK-UID",
						ModelID:  "MOCK-MODEL-ID",
					},
				},
			},
			want: &api.CreateEntityResponse{
				Id: "0",
			},
		},
		{
			name: "Entity already in position",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.CreateEntityRequest{
					Entity: &api.Entity{
						Id:       "0",
						X:        1,
						Y:        1,
						Class:    1,
						OwnerUID: "MOCK-UID",
						ModelID:  "MOCK-MODEL-ID",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.CreateEntity(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("simulationService.CreateEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("environment.CreateEntity() = %v, want %v", got, tt.want)
			}
		})
	}

}

func TestGetEntity(t *testing.T) {
	// ctxWithoutValidToken := context.Background()
	userinfoEnc := b64.StdEncoding.EncodeToString([]byte(userinfoJSONString))
	md := metadata.Pairs("x-endpoint-api-userinfo", userinfoEnc)
	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

	s := NewEnvironmentServer("testing", redisAddr)

	type args struct {
		ctx context.Context
		req *api.GetEntityRequest
	}
	tests := []struct {
		name    string
		s       api.EnvironmentServer
		args    args
		want    *api.GetEntityResponse
		wantErr bool
	}{
		{
			name: "Succesful Get Entity",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.GetEntityRequest{
					Id: "0",
				},
			},
			want: &api.GetEntityResponse{
				Entity: &api.Entity{
					Id:       "0",
					ModelID:  "MOCK-MODEL-ID",
					OwnerUID: "MOCK-UID",
					Class:    1,
					X:        1,
					Y:        1,
				},
			},
		},
		{
			name: "Entity doesn't exist",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.GetEntityRequest{
					Id: "1",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetEntity(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("simulationService.GetEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("environment.GetEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEntitiesInRegion(t *testing.T) {
	// ctxWithoutValidToken := context.Background()
	userinfoEnc := b64.StdEncoding.EncodeToString([]byte(userinfoJSONString))
	md := metadata.Pairs("x-endpoint-api-userinfo", userinfoEnc)
	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

	s := NewEnvironmentServer("testing", redisAddr)

	type args struct {
		ctx context.Context
		req *api.GetEntitiesInRegionRequest
	}
	tests := []struct {
		name    string
		s       api.EnvironmentServer
		args    args
		want    *api.GetEntitiesInRegionResponse
		wantErr bool
	}{
		{
			name: "Succesful",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.GetEntitiesInRegionRequest{
					X: 0,
					Y: 0,
				},
			},
			want: &api.GetEntitiesInRegionResponse{
				Entities: []*api.Entity{
					&api.Entity{
						Id:      "0",
						ModelID: "MOCK-MODEL-ID",
						Class:   1,
						X:       1,
						Y:       1,
					}},
			},
		},
		{
			name: "Entity doesn't exist",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.GetEntitiesInRegionRequest{
					X: 1,
					Y: 1,
				},
			},
			want: &api.GetEntitiesInRegionResponse{
				Entities: []*api.Entity{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetEntitiesInRegion(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("simulationService.GetEntitiesInRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("environment.GetEntitiesInRegion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteAgentAction(t *testing.T) {
	// ctxWithoutValidToken := context.Background()
	userinfoEnc := b64.StdEncoding.EncodeToString([]byte(userinfoJSONString))
	md := metadata.Pairs("x-endpoint-api-userinfo", userinfoEnc)
	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

	s := NewEnvironmentServer("testing", redisAddr)

	type args struct {
		ctx context.Context
		req *api.ExecuteAgentActionRequest
	}
	tests := []struct {
		name    string
		s       api.EnvironmentServer
		args    args
		want    *api.ExecuteAgentActionResponse
		wantErr bool
	}{
		{
			name: "Succesful action execution",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.ExecuteAgentActionRequest{
					Id:        "0",
					Action:    1,
					Direction: 3,
				},
			},
			want: &api.ExecuteAgentActionResponse{
				WasSuccessful: true,
			},
		},
		{
			name: "Invalid action",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.ExecuteAgentActionRequest{
					Id:        "0",
					Action:    2,
					Direction: 0,
				},
			},
			want: &api.ExecuteAgentActionResponse{
				WasSuccessful: false,
			},
		},
		{
			name: "Entity doesn't exist",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.ExecuteAgentActionRequest{
					Id: "1",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.ExecuteAgentAction(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("environment.ExecuteAgentAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("environment.ExecuteAgentAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteEntity(t *testing.T) {
	// ctxWithoutValidToken := context.Background()
	userinfoEnc := b64.StdEncoding.EncodeToString([]byte(userinfoJSONString))
	md := metadata.Pairs("x-endpoint-api-userinfo", userinfoEnc)
	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

	s := NewEnvironmentServer("testing", redisAddr)

	type args struct {
		ctx context.Context
		req *api.DeleteEntityRequest
	}
	tests := []struct {
		name    string
		s       api.EnvironmentServer
		args    args
		want    *api.DeleteEntityResponse
		wantErr bool
	}{
		{
			name: "Succesful Get Entity",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.DeleteEntityRequest{
					Id: "0",
				},
			},
			want: &api.DeleteEntityResponse{
				Deleted: 1,
			},
		},
		{
			name: "Entity doesn't exist",
			s:    s,
			args: args{
				ctx: ctxWithValidSecret,
				req: &api.DeleteEntityRequest{
					Id: "1",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.DeleteEntity(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("simulationService.CreateEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("environment.CreateEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}
