package environment

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"reflect"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/golang/protobuf/ptypes/empty"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	datacom "github.com/terrariumai/simulation/pkg/datacom"
	"github.com/terrariumai/simulation/pkg/environment/mocks"
	"google.golang.org/grpc/metadata"
)

type mockFuncCall struct {
	name string
	args []interface{}
	resp []interface{}
}

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

	type args struct {
		ctx context.Context
		req *envApi.CreateEntityRequest
	}
	tests := []struct {
		name                  string
		args                  args
		mockRMMetadata        *datacom.RemoteModel
		mockRMMetadataError   error
		mockIsCellOccupied    bool
		mockIsCellOccupiedErr error
		want                  *envApi.CreateEntityResponse
		wantErr               bool
		wantErrMessage        string
	}{
		{
			name: "Entity not in request error",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{},
			},
			wantErr:        true,
			wantErrMessage: "entity not in request",
		},
		{
			name: "Invalid class",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						Class: 4,
					},
				},
			},
			wantErr:        true,
			wantErrMessage: "invalid class",
		},
		{
			name: "Missing model id",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{},
				},
			},
			wantErr:        true,
			wantErrMessage: "missing model id",
		},
		{
			name: "Missing model id",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID: "asdf",
					},
				},
			},
			mockRMMetadataError: errors.New("remote model does not exist"),
			wantErr:             true,
			wantErrMessage:      "remote model does not exist",
		},
		{
			name: "No access",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID: "MOCK-ID",
					},
				},
			},
			mockRMMetadata: &datacom.RemoteModel{
				OwnerUID: "incorrect-uid",
			},
			wantErr:        true,
			wantErrMessage: "you do not own that remote model",
		},
		{
			name: "No access",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
					},
				},
			},
			mockRMMetadata: &datacom.RemoteModel{
				OwnerUID:     "MOCK-UID",
				ConnectCount: 0,
			},
			wantErr:        true,
			wantErrMessage: "rm is offline",
		},
		{
			name: "Invalid position (over max position)",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
						X:        1000,
						Y:        50,
					},
				},
			},
			mockRMMetadata: &datacom.RemoteModel{
				OwnerUID:     "MOCK-UID",
				ConnectCount: 1,
			},
			wantErr:        true,
			wantErrMessage: "invalid position",
		},
		{
			name: "Invalid position (under min position)",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
						X:        5,
						Y:        0,
					},
				},
			},
			mockRMMetadata: &datacom.RemoteModel{
				OwnerUID:     "MOCK-UID",
				ConnectCount: 1,
			},
			wantErr:        true,
			wantErrMessage: "invalid position",
		},
		{
			name: "Invalid position (cell occupied)",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
						X:        200,
						Y:        50,
					},
				},
			},
			mockIsCellOccupiedErr: errors.New("cell is already occupied"),
			mockRMMetadata: &datacom.RemoteModel{
				OwnerUID:     "MOCK-UID",
				ConnectCount: 1,
			},
			wantErr:        true,
			wantErrMessage: "cell is already occupied",
		},
		{
			name: "Success",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
						X:        2,
						Y:        2,
						Id:       "0",
					},
				},
			},
			mockRMMetadata: &datacom.RemoteModel{
				OwnerUID:     "MOCK-UID",
				ConnectCount: 1,
			},
			want: &envApi.CreateEntityResponse{
				Id: "0",
			},
		},
		{
			name: "Success on edge",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
						X:        1,
						Y:        1,
						Id:       "0",
					},
				},
			},
			mockRMMetadata: &datacom.RemoteModel{
				OwnerUID:     "MOCK-UID",
				ConnectCount: 1,
			},
			want: &envApi.CreateEntityResponse{
				Id: "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockDAL := &mocks.DataAccessLayer{}
			s := NewEnvironmentServer("testing", mockDAL)

			if tt.args.req.Entity != nil {
				e := tt.args.req.Entity
				e.Energy = 100
				e.Health = 100
				mockDAL.On("CreateEntity", *tt.args.req.Entity, true).Return(nil)
				mockDAL.On("GetRemoteModelMetadataByID", tt.args.req.Entity.ModelID).Return(tt.mockRMMetadata, tt.mockRMMetadataError)
				mockDAL.On("IsCellOccupied", tt.args.req.Entity.X, tt.args.req.Entity.Y).Return(tt.mockIsCellOccupied, nil, tt.mockIsCellOccupiedErr)
			}

			got, err := s.CreateEntity(tt.args.ctx, tt.args.req)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("error: %v, wantErr: %v", err, tt.wantErr)
					return
				}
				if err.Error() != tt.wantErrMessage {
					t.Errorf("error message: '%v', want error message: '%v'", err, tt.wantErrMessage)
					return
				}
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("environment.CreateEntity() = %v, want %v", got, tt.want)
			}
		})
	}

}

func TestGetEntity(t *testing.T) {
	ctx, redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		ctx context.Context
		req *envApi.GetEntityRequest
	}
	tests := []struct {
		name                   string
		args                   args
		mockGetEntityResponse  *envApi.Entity
		mockGetEntityResponse2 *string
		mockGetEntityErr       error
		want                   *envApi.GetEntityResponse
		wantErr                bool
		wantErrMessage         string
	}{
		{
			name: "Does not exist",
			args: args{
				ctx: ctx,
				req: &envApi.GetEntityRequest{},
			},
			wantErr:          true,
			mockGetEntityErr: errors.New("entity does not exist"),
			wantErrMessage:   "entity does not exist",
		},
		{
			name: "Success",
			args: args{
				ctx: ctx,
				req: &envApi.GetEntityRequest{},
			},
			mockGetEntityResponse: &envApi.Entity{
				Id: "test",
			},
			want: &envApi.GetEntityResponse{
				Entity: &envApi.Entity{
					Id: "test",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockDAL := &mocks.DataAccessLayer{}
			s := NewEnvironmentServer("testing", mockDAL)

			mockDAL.On("GetEntity", tt.args.req.Id).Return(tt.mockGetEntityResponse, tt.mockGetEntityResponse2, tt.mockGetEntityErr)

			got, err := s.GetEntity(tt.args.ctx, tt.args.req)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("error: %v, wantErr: %v", err, tt.wantErr)
					return
				}
				if err.Error() != tt.wantErrMessage {
					t.Errorf("error message: '%v', want error message: '%v'", err, tt.wantErrMessage)
					return
				}
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

}

func TestDeleteEntity(t *testing.T) {
	ctx, redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		ctx context.Context
		req *envApi.DeleteEntityRequest
	}
	tests := []struct {
		name                     string
		args                     args
		mockDeleteEntityResponse int64
		mockDeleteEntityErr      error
		want                     *envApi.DeleteEntityResponse
		wantErr                  bool
		wantErrMessage           string
	}{
		{
			name: "Does not exist",
			args: args{
				ctx: ctx,
				req: &envApi.DeleteEntityRequest{},
			},
			wantErr:             true,
			mockDeleteEntityErr: errors.New("entity does not exist"),
			wantErrMessage:      "entity does not exist",
		},
		{
			name: "Success",
			args: args{
				ctx: ctx,
				req: &envApi.DeleteEntityRequest{},
			},
			mockDeleteEntityResponse: 1,
			want: &envApi.DeleteEntityResponse{
				Deleted: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockDAL := &mocks.DataAccessLayer{}
			s := NewEnvironmentServer("testing", mockDAL)

			mockDAL.On("DeleteEntity", tt.args.req.Id).Return(tt.mockDeleteEntityResponse, tt.mockDeleteEntityErr)

			got, err := s.DeleteEntity(tt.args.ctx, tt.args.req)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("error: %v, wantErr: %v", err, tt.wantErr)
					return
				}
				if err.Error() != tt.wantErrMessage {
					t.Errorf("error message: '%v', want error message: '%v'", err, tt.wantErrMessage)
					return
				}
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

}

func TestExecuteAgentAction(t *testing.T) {
	ctx, redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		ctx context.Context
		req *envApi.ExecuteAgentActionRequest
	}
	type getEntityResp struct {
		entity  *envApi.Entity
		content string
		err     error
	}
	type updateEntityArgs struct {
		entity           envApi.Entity
		origionalContent string
	}
	type isCellOccupiedArgs struct {
		x uint32
		y uint32
	}
	type isCellOccupiedResp struct {
		isOccupied bool
		e          *envApi.Entity
		err        error
	}

	tests := []struct {
		name             string
		args             args
		DALMockFuncCalls []mockFuncCall
		want             *envApi.ExecuteAgentActionResponse
		wantErr          error
	}{
		{
			name: "entity does not exist fails",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{},
			},
			DALMockFuncCalls: []mockFuncCall{
				{
					name: "GetEntity",
					args: []interface{}{""},
					resp: []interface{}{nil, "", errors.New("entity does not exist")},
				},
			},
			wantErr: errors.New("entity does not exist"),
		},
		{
			name: "rest reduces energy by 1",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{
					Id:     "mock-entity-id",
					Action: 0,
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{
					name: "GetEntity",
					args: []interface{}{"mock-entity-id"},
					resp: []interface{}{&envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Energy: 100, Health: 100}, "mock-original-content", nil},
				},
				{
					name: "UpdateEntity",
					args: []interface{}{"mock-original-content", envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Energy: 99, Health: 100}},
					resp: []interface{}{nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: true,
				IsAlive:       true,
			},
		},
		{
			name: "cannot move to an occupied cell (no error)",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{
					Id:        "mock-entity-id",
					Action:    1,
					Direction: 3,
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{
					name: "GetEntity",
					args: []interface{}{"mock-entity-id"},
					resp: []interface{}{&envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Energy: 100, Health: 100}, "mock-original-content", nil},
				},
				{
					name: "IsCellOccupied",
					args: []interface{}{uint32(2), uint32(1)},
					resp: []interface{}{true, nil, nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			},
		},
		// {
		// 	name: "can move to an empty cell",
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &envApi.ExecuteAgentActionRequest{
		// 			Id:        "mock-entity-id",
		// 			Action:    1,
		// 			Direction: 3,
		// 		},
		// 	},
		// 	mockGetEntityResp: getEntityResp{
		// 		entity:  &envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Energy: 100, Health: 100},
		// 		content: "mock-original-content",
		// 		err:     nil,
		// 	},
		// 	wantIsCellOccupiedArgs: isCellOccupiedArgs{x: 2, y: 1},
		// 	mockIsCellOccupiedResp: isCellOccupiedResp{false, nil, nil},
		// 	wantUpdateEntityArgs: updateEntityArgs{
		// 		origionalContent: "mock-original-content",
		// 		entity:           envApi.Entity{Id: "mock-entity-id", X: 2, Y: 1, Energy: 97, Health: 100},
		// 	},
		// 	want: &envApi.ExecuteAgentActionResponse{
		// 		WasSuccessful: true,
		// 		IsAlive:       true,
		// 	},
		// },
		// {
		// 	name: "moving with too little energy kills",
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &envApi.ExecuteAgentActionRequest{
		// 			Id:        "mock-entity-id",
		// 			Action:    1,
		// 			Direction: 3,
		// 		},
		// 	},
		// 	mockGetEntityResp: getEntityResp{
		// 		entity:  &envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Energy: 1, Health: 0},
		// 		content: "mock-original-content",
		// 		err:     nil,
		// 	},
		// 	wantIsCellOccupiedArgs: isCellOccupiedArgs{x: 2, y: 1},
		// 	mockIsCellOccupiedResp: isCellOccupiedResp{false, nil, nil},
		// 	want: &envApi.ExecuteAgentActionResponse{
		// 		WasSuccessful: false,
		// 		IsAlive:       false,
		// 	},
		// },
		// {
		// 	name: "Succesful eat",
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &envApi.ExecuteAgentActionRequest{
		// 			Id:        "mock-entity-id",
		// 			Action:    2,
		// 			Direction: 3,
		// 		},
		// 	},
		// 	mockGetEntityResp: getEntityResp{
		// 		entity:  &envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Health: 100, Energy: 80, Class: 1},
		// 		content: "mock-original-content",
		// 		err:     nil,
		// 	},
		// 	mockCreateEntityResp:   nil,
		// 	wantIsCellOccupiedArgs: isCellOccupiedArgs{x: 2, y: 1},
		// 	mockIsCellOccupiedResp: isCellOccupiedResp{true, &envApi.Entity{Id: "mock-entity-id", X: 2, Y: 1, Class: 3}, nil},
		// 	wantUpdateEntityArgs: updateEntityArgs{
		// 		origionalContent: "mock-original-content",
		// 		entity:           envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Energy: 89, Health: 100, Class: 1},
		// 	},
		// 	want: &envApi.ExecuteAgentActionResponse{
		// 		WasSuccessful: true,
		// 		IsAlive:       true,
		// 	},
		// },
		// {
		// 	name: "Cannot eat non-food entity",
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &envApi.ExecuteAgentActionRequest{
		// 			Id:        "mock-entity-id",
		// 			Action:    2,
		// 			Direction: 3,
		// 		},
		// 	},
		// 	mockGetEntityResp: getEntityResp{
		// 		entity:  &envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Health: 100, Energy: 80, Class: 1},
		// 		content: "mock-original-content",
		// 		err:     nil,
		// 	},
		// 	wantIsCellOccupiedArgs: isCellOccupiedArgs{x: 2, y: 1},
		// 	mockIsCellOccupiedResp: isCellOccupiedResp{true, &envApi.Entity{Id: "mock-entity-id", X: 2, Y: 1, Class: 1}, nil},
		// 	want: &envApi.ExecuteAgentActionResponse{
		// 		WasSuccessful: false,
		// 		IsAlive:       true,
		// 	},
		// },
		// // ATTACK
		// {
		// 	name: "Cannot attack empty cell",
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &envApi.ExecuteAgentActionRequest{
		// 			Id:        "mock-entity-id",
		// 			Action:    3,
		// 			Direction: 3,
		// 		},
		// 	},
		// 	mockGetEntityResp: getEntityResp{
		// 		entity:  &envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Health: 100, Energy: 80, Class: 1},
		// 		content: "mock-original-content",
		// 		err:     nil,
		// 	},
		// 	wantIsCellOccupiedArgs: isCellOccupiedArgs{x: 2, y: 1},
		// 	mockIsCellOccupiedResp: isCellOccupiedResp{false, nil, nil},
		// 	want: &envApi.ExecuteAgentActionResponse{
		// 		WasSuccessful: false,
		// 		IsAlive:       true,
		// 	},
		// },
		// {
		// 	name: "Cannot attack non-agent entity",
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &envApi.ExecuteAgentActionRequest{
		// 			Id:        "mock-entity-id",
		// 			Action:    3,
		// 			Direction: 3,
		// 		},
		// 	},
		// 	mockGetEntityResp: getEntityResp{
		// 		entity:  &envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Health: 100, Energy: 80, Class: 1},
		// 		content: "mock-original-content",
		// 		err:     nil,
		// 	},
		// 	wantIsCellOccupiedArgs: isCellOccupiedArgs{x: 2, y: 1},
		// 	mockIsCellOccupiedResp: isCellOccupiedResp{true, &envApi.Entity{Id: "mock-entity-id", X: 2, Y: 1, Class: 3}, nil},
		// 	want: &envApi.ExecuteAgentActionResponse{
		// 		WasSuccessful: false,
		// 		IsAlive:       true,
		// 	},
		// },
		// {
		// 	name: "Succesful attack",
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &envApi.ExecuteAgentActionRequest{
		// 			Id:        "mock-entity-id",
		// 			Action:    3,
		// 			Direction: 3,
		// 		},
		// 	},
		// 	mockGetEntityResp: getEntityResp{
		// 		entity:  &envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Health: 100, Energy: 80, Class: 1},
		// 		content: "mock-original-content",
		// 		err:     nil,
		// 	},
		// 	wantIsCellOccupiedArgs: isCellOccupiedArgs{x: 2, y: 1},
		// 	mockIsCellOccupiedResp: isCellOccupiedResp{true, &envApi.Entity{Id: "mock-entity-id-2", X: 2, Y: 1, Class: 3}, nil},
		// 	want: &envApi.ExecuteAgentActionResponse{
		// 		WasSuccessful: false,
		// 		IsAlive:       true,
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockDAL := &mocks.DataAccessLayer{}
			s := NewEnvironmentServer("testing", mockDAL)
			for _, mockFuncCall := range tt.DALMockFuncCalls {
				mockDAL.On(mockFuncCall.name, mockFuncCall.req...).Return(mockFuncCall.resp...)
			}
			// for _, mock := range tt.DALGetEntityMocks {
			// 	mockDAL.On("GetEntity", mock.req...).Return(mock.resp...)
			// }
			// mockDAL.On("CreateEntity", mock.AnythingOfType("Entity"), mock.AnythingOfType("bool")).Return(tt.mockCreateEntityResp)
			// mockDAL.On("UpdateEntity", tt.wantUpdateEntityArgs.origionalContent, tt.wantUpdateEntityArgs.entity).Return(nil)
			// mockDAL.On("IsCellOccupied", tt.wantIsCellOccupiedArgs.x, tt.wantIsCellOccupiedArgs.y).Return(tt.mockIsCellOccupiedResp.isOccupied, tt.mockIsCellOccupiedResp.e, tt.mockIsCellOccupiedResp.err)
			// mockDAL.On("DeleteEntity", tt.args.req.Id).Return(int64(1), nil)

			got, err := s.ExecuteAgentAction(tt.args.ctx, tt.args.req)
			if err != nil {
				if tt.wantErr == nil {
					t.Errorf("error: %v, wantErr: %v", err, tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("error message: '%v', want error message: '%v'", err, tt.wantErr.Error())
					return
				}
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", *got, *tt.want)
			}
		})
	}

}

func TestResetWorld(t *testing.T) {
	ctx, redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		ctx context.Context
		req *empty.Empty
	}
	tests := []struct {
		name           string
		args           args
		want           *empty.Empty
		wantErr        bool
		wantErrMessage string
	}{
		{
			name: "Success",
			args: args{
				ctx: ctx,
				req: &empty.Empty{},
			},
			want: &empty.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockDAL := &mocks.DataAccessLayer{}
			s := NewEnvironmentServer("testing", mockDAL)

			got, err := s.ResetWorld(tt.args.ctx, tt.args.req)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("error: %v, wantErr: %v", err, tt.wantErr)
					return
				}
				if err.Error() != tt.wantErrMessage {
					t.Errorf("error message: '%v', want error message: '%v'", err, tt.wantErrMessage)
					return
				}
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

}

func TestGetEntitiesInRegion(t *testing.T) {
	ctx, redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		ctx context.Context
		req *envApi.GetEntitiesInRegionRequest
	}
	type getEntitiesInRegionReq struct {
		x uint32
		y uint32
	}
	type getEntitiesInRegionResp struct {
		entities []*envApi.Entity
		err      error
	}
	tests := []struct {
		name                        string
		args                        args
		want                        *envApi.GetEntitiesInRegionResponse
		wantGetEntitiesInRegionReq  getEntitiesInRegionReq
		mockGetEntitiesInRegionResp getEntitiesInRegionResp
		wantErr                     bool
		wantErrMessage              string
	}{
		{
			name: "Error",
			args: args{
				ctx: ctx,
				req: &envApi.GetEntitiesInRegionRequest{
					X: 1,
					Y: 2,
				},
			},
			wantGetEntitiesInRegionReq: getEntitiesInRegionReq{x: 1, y: 2},
			mockGetEntitiesInRegionResp: getEntitiesInRegionResp{
				entities: nil,
				err:      errors.New("invalid region"),
			},
			wantErr:        true,
			wantErrMessage: "invalid region",
		},
		{
			name: "Success",
			args: args{
				ctx: ctx,
				req: &envApi.GetEntitiesInRegionRequest{
					X: 1,
					Y: 2,
				},
			},
			wantGetEntitiesInRegionReq: getEntitiesInRegionReq{x: 1, y: 2},
			mockGetEntitiesInRegionResp: getEntitiesInRegionResp{
				entities: []*envApi.Entity{},
				err:      nil,
			},
			want: &envApi.GetEntitiesInRegionResponse{
				Entities: []*envApi.Entity{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockDAL := &mocks.DataAccessLayer{}
			mockDAL.On("GetEntitiesInRegion", tt.wantGetEntitiesInRegionReq.x, tt.wantGetEntitiesInRegionReq.y).Return(tt.mockGetEntitiesInRegionResp.entities, tt.mockGetEntitiesInRegionResp.err)

			s := NewEnvironmentServer("testing", mockDAL)

			got, err := s.GetEntitiesInRegion(tt.args.ctx, tt.args.req)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("error: %v, wantErr: %v", err, tt.wantErr)
					return
				}
				if err.Error() != tt.wantErrMessage {
					t.Errorf("error message: '%v', want error message: '%v'", err, tt.wantErrMessage)
					return
				}
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

}
