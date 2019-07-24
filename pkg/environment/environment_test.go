package environment

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"reflect"
	"testing"

	"github.com/alicebob/miniredis"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	datacom "github.com/terrariumai/simulation/pkg/datacom"
	"github.com/terrariumai/simulation/pkg/environment/mocks"
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
			name: "Invalid position",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
						X:        1000,
						Y:        0,
					},
				},
			},
			mockIsCellOccupiedErr: errors.New("invalid position"),
			mockRMMetadata: &datacom.RemoteModel{
				OwnerUID:     "MOCK-UID",
				ConnectCount: 1,
			},
			wantErr:        true,
			wantErrMessage: "invalid position",
		},
		{
			name: "Invalid position",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
						X:        1000,
						Y:        0,
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
			name: "Invalid position",
			args: args{
				ctx: ctx,
				req: &envApi.CreateEntityRequest{
					Entity: &envApi.Entity{
						ModelID:  "MOCK-ID",
						OwnerUID: "MOCK-UID",
						X:        1000,
						Y:        0,
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
				mockDAL.On("CreateEntity", *tt.args.req.Entity).Return(nil)
				mockDAL.On("GetRemoteModelMetadataByID", tt.args.req.Entity.ModelID).Return(tt.mockRMMetadata, tt.mockRMMetadataError)
				mockDAL.On("IsCellOccupied", tt.args.req.Entity.X, tt.args.req.Entity.Y).Return(tt.mockIsCellOccupied, tt.mockIsCellOccupiedErr)
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

	tests := []struct {
		name              string
		args              args
		mockGetEntityResp getEntityResp
		want              *envApi.ExecuteAgentActionResponse
		wantErr           bool
		wantErrMessage    string
	}{
		{
			name: "Does not exist",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{},
			},
			wantErr: true,
			mockGetEntityResp: getEntityResp{
				entity:  nil,
				content: "",
				err:     errors.New("entity does not exist"),
			},
			wantErrMessage: "entity does not exist",
		},
		{
			name: "Does not exist",
			args: args{
				ctx: ctx,
				req: &envApi.ExecuteAgentActionRequest{},
			},
			wantErr: true,
			mockGetEntityResp: getEntityResp{
				entity:  nil,
				content: "",
				err:     errors.New("entity does not exist"),
			},
			wantErrMessage: "entity does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockDAL := &mocks.DataAccessLayer{}
			s := NewEnvironmentServer("testing", mockDAL)

			mockDAL.On("GetEntity", tt.args.req.Id).Return(tt.mockGetEntityResp.entity, &tt.mockGetEntityResp.content, tt.mockGetEntityResp.err)

			got, err := s.ExecuteAgentAction(tt.args.ctx, tt.args.req)
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
