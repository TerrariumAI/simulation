package environment

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"

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

type numOfCallsAssertion struct {
	name string
	num  int
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
		name             string
		args             args
		DALMockFuncCalls []mockFuncCall
		want             *envApi.CreateEntityResponse
		wantErr          error
	}{
		{
			name: "Fails if entity is not in the request",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{},
			},
			wantErr: errors.New("entity not in request"),
		},
		{
			name: "Class must be valid",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ClassID: 4,
					},
				},
			},
			wantErr: errors.New("invalid class"),
		},
		{
			name: "Must specify model id",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{},
				},
			},
			wantErr: errors.New("missing model id"),
		},
		{
			name: "Model must exist",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID: "mock-model-id",
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{nil, errors.New("remote model does not exist")},
				},
			},
			wantErr: errors.New("remote model does not exist"),
		},
		{
			name: "User must have access to the model",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID: "mock-model-id",
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{&datacom.RemoteModel{ID: "mock-model-id", OwnerUID: "incorrect-model-id"}, nil},
				},
			},
			wantErr: errors.New("you do not own that remote model"),
		},
		{
			name: "Model must be online",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID: "mock-model-id",
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{&datacom.RemoteModel{ID: "mock-model-id", OwnerUID: "MOCK-UID", ConnectCount: 0}, nil},
				},
			},
			wantErr: errors.New("rm is offline"),
		},
		{
			name: "Cannot create more than 5 entities manually",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID: "mock-model-id",
						X:       1,
						Y:       1,
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{&datacom.RemoteModel{ID: "mock-model-id", OwnerUID: "MOCK-UID", ConnectCount: 1}, nil},
				},
				{ // Get entities for the RM
					name: "GetEntitiesForModel",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{[]envApi.Entity{envApi.Entity{}, envApi.Entity{}, envApi.Entity{}, envApi.Entity{}, envApi.Entity{}}, nil},
				},
			},
			wantErr: errors.New("you can only manually create 5 entities at a time"),
		},
		{
			name: "Invalid position (over max position)",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "mock-model-id",
						OwnerUID: "MOCK-UID",
						X:        1000,
						Y:        50,
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{&datacom.RemoteModel{ID: "mock-model-id", OwnerUID: "MOCK-UID", ConnectCount: 1}, nil},
				},
				{ // Get entities for the RM
					name: "GetEntitiesForModel",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{[]envApi.Entity{}, nil},
				},
			},
			wantErr: errors.New("invalid position"),
		},
		{
			name: "Invalid position (under min position)",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "mock-model-id",
						OwnerUID: "MOCK-UID",
						X:        0,
						Y:        5,
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{&datacom.RemoteModel{ID: "mock-model-id", OwnerUID: "MOCK-UID", ConnectCount: 1}, nil},
				},
				{ // Get entities for the RM
					name: "GetEntitiesForModel",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{[]envApi.Entity{}, nil},
				},
			},
			wantErr: errors.New("invalid position"),
		},
		{
			name: "Cannot create entity in occupied cell",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "mock-model-id",
						OwnerUID: "MOCK-UID",
						X:        1,
						Y:        1,
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{&datacom.RemoteModel{ID: "mock-model-id", OwnerUID: "MOCK-UID", ConnectCount: 1}, nil},
				},
				{ // Get entities for the RM
					name: "GetEntitiesForModel",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{[]envApi.Entity{}, nil},
				},
				{ // Check if the cell is occupied in the target position
					name: "IsCellOccupied",
					args: []interface{}{uint32(1), uint32(1)},
					resp: []interface{}{true, nil, "", nil},
				},
			},
			wantErr: errors.New("cell is already occupied"),
		},
		{
			name: "Succesful in middle position",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "mock-model-id",
						OwnerUID: "MOCK-UID",
						X:        25,
						Y:        25,
						Id:       "0",
						ClassID:  1,
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{&datacom.RemoteModel{ID: "mock-model-id", OwnerUID: "MOCK-UID", ConnectCount: 1}, nil},
				},
				{ // Get entities for the RM
					name: "GetEntitiesForModel",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{[]envApi.Entity{}, nil},
				},
				{ // Check if the cell is occupied in the target position
					name: "IsCellOccupied",
					args: []interface{}{uint32(25), uint32(25)},
					resp: []interface{}{false, nil, "", nil},
				},
				{ // Create the entity
					name: "CreateEntity",
					args: []interface{}{envApi.Entity{Id: "0", ClassID: 1, X: uint32(25), Y: uint32(25), Energy: uint32(100), Health: uint32(100), OwnerUID: "MOCK-UID", ModelID: "mock-model-id"}, true},
					resp: []interface{}{nil},
				},
			},
			want: &envApi.CreateEntityResponse{
				Id: "0",
			},
		},
		{
			name: "Succesful on edge",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "mock-model-id",
						OwnerUID: "MOCK-UID",
						X:        1,
						Y:        1,
						Id:       "0",
						ClassID:  1,
					},
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "GetRemoteModelMetadataByID",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{&datacom.RemoteModel{ID: "mock-model-id", OwnerUID: "MOCK-UID", ConnectCount: 1}, nil},
				},
				{ // Get entities for the RM
					name: "GetEntitiesForModel",
					args: []interface{}{"mock-model-id"},
					resp: []interface{}{[]envApi.Entity{}, nil},
				},
				{ // Check if the cell is occupied in the target position
					name: "IsCellOccupied",
					args: []interface{}{uint32(1), uint32(1)},
					resp: []interface{}{false, nil, "", nil},
				},
				{ // Create the entity
					name: "CreateEntity",
					args: []interface{}{envApi.Entity{Id: "0", ClassID: 1, X: uint32(1), Y: uint32(1), Energy: uint32(100), Health: uint32(100), OwnerUID: "MOCK-UID", ModelID: "mock-model-id"}, true},
					resp: []interface{}{nil},
				},
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
			for _, mockFuncCall := range tt.DALMockFuncCalls {
				mockDAL.On(mockFuncCall.name, mockFuncCall.args...).Return(mockFuncCall.resp...)
			}
			// Call function
			got, err := s.CreateEntity(tt.args.ctx, tt.args.req)
			// Check results
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
		mockGetEntityResponse2 string
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

	tests := []struct {
		name                 string
		args                 args
		DALMockFuncCalls     []mockFuncCall
		numOfCallsAssertions []numOfCallsAssertion
		want                 *envApi.ExecuteAgentActionResponse
		wantErr              error
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
					resp: []interface{}{true, nil, "", nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			},
		},
		{
			name: "can move to an empty cell",
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
					resp: []interface{}{false, nil, "", nil},
				},
				{
					name: "UpdateEntity",
					args: []interface{}{"mock-original-content", envApi.Entity{Id: "mock-entity-id", X: 2, Y: 1, Energy: 97, Health: 100}},
					resp: []interface{}{nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: true,
				IsAlive:       true,
			},
		},
		{
			name: "moving with too little energy kills",
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
					resp: []interface{}{&envApi.Entity{Id: "mock-entity-id", X: 1, Y: 1, Energy: 1, Health: 1}, "mock-original-content", nil},
				},
				{
					name: "IsCellOccupied",
					args: []interface{}{uint32(2), uint32(1)},
					resp: []interface{}{false, nil, "", nil},
				},
				{
					name: "DeleteEntity",
					args: []interface{}{"mock-entity-id"},
					resp: []interface{}{int64(1), nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       false,
			},
		},
		{
			name: "Succesful eat",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{
					Id:        "mock-agent-id",
					Action:    2,
					Direction: 3,
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the agent
					name: "GetEntity",
					args: []interface{}{"mock-agent-id"},
					resp: []interface{}{&envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Energy: 80, Health: 100, ClassID: 1}, "mock-original-content", nil},
				},
				{ // Target cell is occupied by food
					name: "IsCellOccupied",
					args: []interface{}{uint32(2), uint32(1)},
					resp: []interface{}{true, &envApi.Entity{Id: "mock-food-id", X: 2, Y: 1, ClassID: 3}, "mock:food:content", nil},
				},
				{ // Update the agent after eating
					name: "UpdateEntity",
					args: []interface{}{"mock-original-content", envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Energy: 89, Health: 100, ClassID: 1}},
					resp: []interface{}{nil},
				},
				{ // Delete the food
					name: "DeleteEntity",
					args: []interface{}{"mock-food-id"},
					resp: []interface{}{int64(1), nil},
				},
				{ // Create a new food entity somewhere
					name: "CreateEntity",
					args: []interface{}{mock.AnythingOfType("Entity"), true},
					resp: []interface{}{nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: true,
				IsAlive:       true,
			},
		},
		{
			name: "Cannot eat non-food entity",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{
					Id:        "mock-agent-id",
					Action:    2,
					Direction: 3,
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the agent
					name: "GetEntity",
					args: []interface{}{"mock-agent-id"},
					resp: []interface{}{&envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Health: 100, Energy: 80, ClassID: 1}, "mock-original-content", nil},
				},
				{ // Target cell is occupied by an agent
					name: "IsCellOccupied",
					args: []interface{}{uint32(2), uint32(1)},
					resp: []interface{}{true, &envApi.Entity{Id: "mock-agent-id-2", X: 2, Y: 1, ClassID: 1}, "mock:agent:content", nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			},
		},
		// // ATTACK
		{
			name: "Cannot attack empty cell",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{
					Id:        "mock-agent-id",
					Action:    3,
					Direction: 3,
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the agent
					name: "GetEntity",
					args: []interface{}{"mock-agent-id"},
					resp: []interface{}{&envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Health: 100, Energy: 80, ClassID: 1}, "mock-original-content", nil},
				},
				{ // Target cell is empty
					name: "IsCellOccupied",
					args: []interface{}{uint32(2), uint32(1)},
					resp: []interface{}{false, nil, "", nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			},
		},
		{
			name: "Cannot attack non-agent entity",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{
					Id:        "mock-agent-id",
					Action:    3,
					Direction: 3,
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the agent
					name: "GetEntity",
					args: []interface{}{"mock-agent-id"},
					resp: []interface{}{&envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Health: 100, Energy: 80, ClassID: 1}, "mock-original-content", nil},
				},
				{ // Target cell is occupied by food
					name: "IsCellOccupied",
					args: []interface{}{uint32(2), uint32(1)},
					resp: []interface{}{true, &envApi.Entity{Id: "mock-food-id", X: 2, Y: 1, ClassID: 3}, "mock:food:content", nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: false,
				IsAlive:       true,
			},
		},
		{
			name: "Succesful attack decreases health of other, energy of this",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{
					Id:        "mock-agent-id",
					Action:    3,
					Direction: 3,
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the agent
					name: "GetEntity",
					args: []interface{}{"mock-agent-id"},
					resp: []interface{}{&envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Health: 100, Energy: 100, ClassID: 1}, "mock:origional:agent:content", nil},
				},
				{ // Target cell is occupied by another agent
					name: "IsCellOccupied",
					args: []interface{}{uint32(2), uint32(1)},
					resp: []interface{}{true, &envApi.Entity{Id: "mock-agent-id-2", X: 2, Y: 1, Health: 100, Energy: 100, ClassID: 1}, "mock:origional:agent2:content", nil},
				},
				{ // Update the other agent's health
					name: "UpdateEntity",
					args: []interface{}{
						mock.MatchedBy(func(content string) bool {
							return content == "mock:origional:agent:content"
						}),
						envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Health: 100, Energy: 94, ClassID: 1},
					},
					resp: []interface{}{nil},
				},
				{ // Update the other agent's health
					name: "UpdateEntity",
					args: []interface{}{
						mock.MatchedBy(func(content string) bool {
							return content == "mock:origional:agent2:content"
						}),
						envApi.Entity{Id: "mock-agent-id-2", X: 2, Y: 1, Health: 90, Energy: 100, ClassID: 1},
					},
					resp: []interface{}{nil},
				},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: true,
				IsAlive:       true,
			},
		},
		{
			name: "Succesful attack kills other, energy of this",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{
					Id:        "mock-agent-id",
					Action:    3,
					Direction: 3,
				},
			},
			DALMockFuncCalls: []mockFuncCall{
				{ // Get the agent
					name: "GetEntity",
					args: []interface{}{"mock-agent-id"},
					resp: []interface{}{&envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Health: 100, Energy: 100, ClassID: 1}, "mock:origional:agent:content", nil},
				},
				{ // Target cell is occupied by another agent
					name: "IsCellOccupied",
					args: []interface{}{uint32(2), uint32(1)},
					resp: []interface{}{true, &envApi.Entity{Id: "mock-agent-id-2", X: 2, Y: 1, Health: 5, Energy: 100, ClassID: 1}, "mock:origional:agent2:content", nil},
				},
				{ // Kill other agent
					name: "DeleteEntity",
					args: []interface{}{"mock-agent-id-2"},
					resp: []interface{}{int64(1), nil},
				},
				{ // Update this agent
					name: "UpdateEntity",
					args: []interface{}{
						mock.MatchedBy(func(content string) bool {
							return content == "mock:origional:agent:content"
						}),
						envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Health: 100, Energy: 94, ClassID: 1},
					},
					resp: []interface{}{nil},
				},
			},
			numOfCallsAssertions: []numOfCallsAssertion{
				{"DeleteEntity", 1},
				{"UpdateEntity", 1},
			},
			want: &envApi.ExecuteAgentActionResponse{
				WasSuccessful: true,
				IsAlive:       true,
			},
		},
		// {
		// 	name: "Succesful attack kills other if health is low, decreases energy of this",
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &envApi.ExecuteAgentActionRequest{
		// 			Id:        "mock-agent-id",
		// 			Action:    3,
		// 			Direction: 3,
		// 		},
		// 	},
		// 	DALMockFuncCalls: []mockFuncCall{
		// 		{ // Get the agent
		// 			name: "GetEntity",
		// 			args: []interface{}{"mock-agent-id"},
		// 			resp: []interface{}{&envApi.Entity{Id: "mock-agent-id", X: 1, Y: 1, Health: 100, Energy: 100, ClassID: 1}, "mock:origional:agent:content", nil},
		// 		},
		// 		{ // Target cell is occupied by another agent
		// 			name: "IsCellOccupied",
		// 			args: []interface{}{uint32(2), uint32(1)},
		// 			resp: []interface{}{true, &envApi.Entity{Id: "mock-agent-id-2", X: 2, Y: 1, Health: 100, Energy: 100, ClassID: 1}, "mock:origional:agent2:content", nil},
		// 		},
		// 		{ // Update the other agent's health
		// 			name: "UpdateEntity",
		// 			args: []interface{}{
		// 				mock.MatchedBy(func(content string) bool {
		// 					return content == "mock:origional:agent:content"
		// 				}),
		// 				mock.AnythingOfType("Entity"),
		// 			},
		// 			resp: []interface{}{nil},
		// 		},
		// 		{ // Update the other agent's health
		// 			name: "UpdateEntity",
		// 			args: []interface{}{
		// 				mock.MatchedBy(func(content string) bool {
		// 					return content == "mock:origional:agent2:content"
		// 				}),
		// 				mock.AnythingOfType("Entity"),
		// 			},
		// 			resp: []interface{}{nil},
		// 		},
		// 	},
		// 	want: &envApi.ExecuteAgentActionResponse{
		// 		WasSuccessful: true,
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
				mockDAL.On(mockFuncCall.name, mockFuncCall.args...).Return(mockFuncCall.resp...)
			}

			got, err := s.ExecuteAgentAction(tt.args.ctx, tt.args.req)

			for _, assertion := range tt.numOfCallsAssertions {
				mockDAL.AssertNumberOfCalls(t, assertion.name, assertion.num)
			}

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
