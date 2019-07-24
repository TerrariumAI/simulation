package environment

import (
	"context"
	b64 "encoding/base64"
	"reflect"
	"testing"

	"github.com/alicebob/miniredis"
	api "github.com/terrariumai/simulation/pkg/api/environment"
	"google.golang.org/grpc/metadata"
)

func setup() (context.Context, *miniredis.Miniredis) {
	// Context setup
	userinfoJSONString := "{\"id\":\"MOCK-UID\"}"
	userinfoEnc := b64.StdEncoding.EncodeToString([]byte(userinfoJSONString))
	md := metadata.Pairs("x-endpoint-api-userinfo", userinfoEnc)
	ctxValidUserInfo := metadata.NewIncomingContext(context.Background(), md)
	// Redis Setup
	redisServer, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	return ctxValidUserInfo, redisServer
}

func teardown(redisServer *miniredis.Miniredis) {
	redisServer.Close()
}

func TestCreateEntity(t *testing.T) {
	ctx, redisServer := setup()
	defer teardown(redisServer)

	s := NewEnvironmentServer("testing", redisServer.Addr())
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
				ctx: ctx,
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
				ctx: ctx,
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
				t.Errorf("environment.CreateEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("environment.CreateEntity() = %v, want %v", got, tt.want)
			}
		})
	}

}

// func TestGetEntity(t *testing.T) {
// 	ctx, redisServer := setup()
// 	defer teardown(redisServer)

// 	s := NewEnvironmentServer("testing", redisServer.Addr())

// 	// Create entity to test on
// 	s.CreateEntity(ctx, &api.CreateEntityRequest{
// 		Entity: &api.Entity{
// 			Id:       "0",
// 			X:        1,
// 			Y:        1,
// 			Class:    1,
// 			OwnerUID: "MOCK-UID",
// 			ModelID:  "MOCK-MODEL-ID",
// 		},
// 	})

// 	type args struct {
// 		ctx context.Context
// 		req *api.GetEntityRequest
// 	}
// 	tests := []struct {
// 		name    string
// 		s       api.EnvironmentServer
// 		args    args
// 		want    *api.GetEntityResponse
// 		wantErr bool
// 	}{
// 		{
// 			name: "Succesful Get Entity",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.GetEntityRequest{
// 					Id: "0",
// 				},
// 			},
// 			want: &api.GetEntityResponse{
// 				Entity: &api.Entity{
// 					Id:       "0",
// 					ModelID:  "MOCK-MODEL-ID",
// 					OwnerUID: "MOCK-UID",
// 					Energy:   100,
// 					Health:   100,
// 					Class:    1,
// 					X:        1,
// 					Y:        1,
// 				},
// 			},
// 		},
// 		{
// 			name: "Entity doesn't exist",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.GetEntityRequest{
// 					Id: "1",
// 				},
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.s.GetEntity(tt.args.ctx, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("environment.GetEntity() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if err == nil && !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("environment.GetEntity() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestGetEntitiesInRegion(t *testing.T) {
// 	ctx, redisServer := setup()
// 	defer teardown(redisServer)

// 	s := NewEnvironmentServer("testing", redisServer.Addr())

// 	// Create entity to test on
// 	s.CreateEntity(ctx, &api.CreateEntityRequest{
// 		Entity: &api.Entity{
// 			Id:       "0",
// 			X:        1,
// 			Y:        1,
// 			Class:    1,
// 			OwnerUID: "MOCK-UID",
// 			ModelID:  "MOCK-MODEL-ID",
// 		},
// 	})

// 	type args struct {
// 		ctx context.Context
// 		req *api.GetEntitiesInRegionRequest
// 	}
// 	tests := []struct {
// 		name    string
// 		s       api.EnvironmentServer
// 		args    args
// 		want    *api.GetEntitiesInRegionResponse
// 		wantErr bool
// 	}{
// 		{
// 			name: "Succesful",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.GetEntitiesInRegionRequest{
// 					X: 0,
// 					Y: 0,
// 				},
// 			},
// 			want: &api.GetEntitiesInRegionResponse{
// 				Entities: []*api.Entity{
// 					&api.Entity{
// 						Id:       "0",
// 						ModelID:  "MOCK-MODEL-ID",
// 						OwnerUID: "MOCK-UID",
// 						Class:    1,
// 						Energy:   100,
// 						Health:   100,
// 						X:        1,
// 						Y:        1,
// 					}},
// 			},
// 		},
// 		{
// 			name: "Empty region",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.GetEntitiesInRegionRequest{
// 					X: 5,
// 					Y: 5,
// 				},
// 			},
// 			want: &api.GetEntitiesInRegionResponse{
// 				Entities: []*api.Entity{},
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.s.GetEntitiesInRegion(tt.args.ctx, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("environment.GetEntitiesInRegion() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if err == nil && !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("environment.GetEntitiesInRegion() got %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestExecuteAgentAction(t *testing.T) {
// 	ctx, redisServer := setup()
// 	defer teardown(redisServer)

// 	s := NewEnvironmentServer("testing", redisServer.Addr())

// 	type args struct {
// 		ctx context.Context
// 		req *api.ExecuteAgentActionRequest
// 	}
// 	tests := []struct {
// 		name    string
// 		s       api.EnvironmentServer
// 		args    args
// 		want    *api.ExecuteAgentActionResponse
// 		wantErr bool
// 	}{
// 		{
// 			name: "Succesful action execution",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.ExecuteAgentActionRequest{
// 					Id:        "0",
// 					Action:    1,
// 					Direction: 3,
// 				},
// 			},
// 			want: &api.ExecuteAgentActionResponse{
// 				WasSuccessful: true,
// 			},
// 		},
// 		{
// 			name: "Invalid action",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.ExecuteAgentActionRequest{
// 					Id:        "0",
// 					Action:    2,
// 					Direction: 0,
// 				},
// 			},
// 			want: &api.ExecuteAgentActionResponse{
// 				WasSuccessful: false,
// 			},
// 		},
// 		{
// 			name: "Entity doesn't exist",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.ExecuteAgentActionRequest{
// 					Id: "1",
// 				},
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.s.ExecuteAgentAction(tt.args.ctx, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("environment.ExecuteAgentAction() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if err == nil && !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("environment.ExecuteAgentAction() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestDeleteEntity(t *testing.T) {
// 	ctx, redisServer := setup()
// 	defer teardown(redisServer)

// 	s := NewEnvironmentServer("testing", redisServer.Addr())

// 	type args struct {
// 		ctx context.Context
// 		req *api.DeleteEntityRequest
// 	}
// 	tests := []struct {
// 		name    string
// 		s       api.EnvironmentServer
// 		args    args
// 		want    *api.DeleteEntityResponse
// 		wantErr bool
// 	}{
// 		{
// 			name: "Succesful Get Entity",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.DeleteEntityRequest{
// 					Id: "0",
// 				},
// 			},
// 			want: &api.DeleteEntityResponse{
// 				Deleted: 1,
// 			},
// 		},
// 		{
// 			name: "Entity doesn't exist",
// 			s:    s,
// 			args: args{
// 				ctx: ctx,
// 				req: &api.DeleteEntityRequest{
// 					Id: "1",
// 				},
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.s.DeleteEntity(tt.args.ctx, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("simulationService.CreateEntity() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if err == nil && !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("environment.CreateEntity() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
