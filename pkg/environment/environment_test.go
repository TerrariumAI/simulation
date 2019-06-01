package environment

import (
	"context"
	"reflect"
	"testing"

	api "github.com/terrariumai/simulation/pkg/api/environment"
	"google.golang.org/grpc/metadata"
)

func TestCreateEntity(t *testing.T) {
	// ctxWithoutValidToken := context.Background()
	md := metadata.Pairs("auth-secret", "MOCK-SECRET")
	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

	s := NewEnvironmentServer("testing")

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
						Id:        "0",
						X:         1,
						Y:         1,
						OwnerUID:  "MOCK-ID",
						ModelName: "",
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
						Id:        "0",
						X:         1,
						Y:         1,
						OwnerUID:  "MOCK-ID",
						ModelName: "",
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
	md := metadata.Pairs("auth-secret", "MOCK-SECRET")
	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

	s := NewEnvironmentServer("testing")

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
					OwnerUID: "MOCK_USER_ID",
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
				t.Errorf("simulationService.CreateEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("environment.CreateEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}
