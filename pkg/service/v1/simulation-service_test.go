package v1

import (
	"context"
	"reflect"
	"testing"

	v1 "github.com/olamai/simulation/pkg/api/v1"
)

func Test_simulationServiceServer_CreateAgent(t *testing.T) {
	ctx := context.Background()
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
			name: "OK",
			s:    s,
			args: args{
				ctx: ctx,
				req: &v1.CreateAgentRequest{
					Api: "v1",
					Agent: &v1.Agent{
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
				ctx: ctx,
				req: &v1.CreateAgentRequest{
					Api: "v1000",
					Agent: &v1.Agent{
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
				ctx: ctx,
				req: &v1.CreateAgentRequest{
					Api: "v1",
					Agent: &v1.Agent{
						X: 0,
						Y: 0,
					},
				},
			},
			wantErr: true,
		},
		// {
		// 	name: "INSERT failed",
		// 	s:    s,
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &v1.CreateRequest{
		// 			Api: "v1",
		// 			ToDo: &v1.ToDo{
		// 				Title:       "title",
		// 				Description: "description",
		// 				Reminder:    reminder,
		// 			},
		// 		},
		// 	},
		// 	mock: func() {
		// 		mock.ExpectExec("INSERT INTO ToDo").WithArgs("title", "description", tm).
		// 			WillReturnError(errors.New("INSERT failed"))
		// 	},
		// 	wantErr: true,
		// },
		// {
		// 	name: "LastInsertId failed",
		// 	s:    s,
		// 	args: args{
		// 		ctx: ctx,
		// 		req: &v1.CreateRequest{
		// 			Api: "v1",
		// 			ToDo: &v1.ToDo{
		// 				Title:       "title",
		// 				Description: "description",
		// 				Reminder:    reminder,
		// 			},
		// 		},
		// 	},
		// 	mock: func() {
		// 		mock.ExpectExec("INSERT INTO ToDo").WithArgs("title", "description", tm).
		// 			WillReturnResult(sqlmock.NewErrorResult(errors.New("LastInsertId failed")))
		// 	},
		// 	wantErr: true,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.CreateAgent(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("toDoServiceServer.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toDoServiceServer.Create() = %v, want %v", got, tt.want)
			}
		})
	}
}
