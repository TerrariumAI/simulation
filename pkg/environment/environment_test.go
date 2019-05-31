package environment

// func Test_simulationServiceServer_CreateEntity(t *testing.T) {
// 	// ctxWithoutValidToken := context.Background()
// 	md := metadata.Pairs("auth-secret", "MOCK-SECRET")
// 	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

// 	s := NewSimulationServiceServer("testing")

// 	type args struct {
// 		ctx context.Context
// 		req *api.CreateEntityRequest
// 	}
// 	tests := []struct {
// 		name    string
// 		s       api.SimulationServiceServer
// 		args    args
// 		want    *api.CreateEntityResponse
// 		wantErr bool
// 	}{
// 		{
// 			name: "Succesful Agent Creation",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithValidSecret,
// 				req: &api.CreateEntityRequest{
// 					Api: "v1",
// 					Agent: &v1.Entity{
// 						X:         0,
// 						Y:         0,
// 						ModelName: "",
// 					},
// 				},
// 			},
// 			want: &v1.CreateEntityResponse{
// 				Api: "v1",
// 				Id:  0,
// 			},
// 		},
// 		// {
// 		// 	name: "Unsupported API",
// 		// 	s:    s,
// 		// 	args: args{
// 		// 		ctx: ctxWithValidSecret,
// 		// 		req: &v1.CreateEntityRequest{
// 		// 			Api:       "v1000",
// 		// 			Agent: &v1.Entity {
// 		// 				X:         0,
// 		// 				Y:         0,
// 		// 				ModelName: "",
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	wantErr: true,
// 		// },
// 		// {
// 		// 	name: "Agent already exists in that position error",
// 		// 	s:    s,
// 		// 	args: args{
// 		// 		ctx: ctxWithValidSecret,
// 		// 		req: &v1.CreateEntityRequest{
// 		// 			Api:       "v1",
// 		// 			Agent: &v1.Entity {
// 		// 				X:         0,
// 		// 				Y:         0,
// 		// 				ModelName: "",
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	wantErr: true,
// 		// },
// 		// {
// 		// 	name: "Invalid secret token",
// 		// 	s:    s,
// 		// 	args: args{
// 		// 		ctx: ctxWithoutValidToken,
// 		// 		req: &v1.CreateEntityRequest{
// 		// 			Api:       "v1",
// 		// 			Agent: &v1.Entity {
// 		// 				X:         0,
// 		// 				Y:         0,
// 		// 				ModelName: "",
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	wantErr: true,
// 		// },
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.s.CreateEntity(tt.args.ctx, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("simulationService.CreateEntity() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if err == nil && !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("toDoServiceServer.CreateEntity() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func Test_simulationServiceServer_GetEntity(t *testing.T) {
// 	ctxWithoutValidSecret := context.Background()
// 	md := metadata.Pairs("auth-secret", "MOCK-SECRET")
// 	ctxWithValidSecret := metadata.NewIncomingContext(context.Background(), md)

// 	s := NewSimulationServiceServer("testing")

// 	// Create an agent to test on
// 	resp, err := s.CreateAgent(ctxWithValidSecret, &v1.CreateAgentRequest{
// 		Api: "v1",
// 		X:   0,
// 		Y:   0,
// 	})
// 	if err != nil {
// 		t.Errorf("toDoServiceServer.Create() error creating test agent. Make sure CreateAgent functionality is working.")
// 		return
// 	}
// 	entityID := resp.Id
// 	// Entity we want to see in the response
// 	wantEntity := &v1.Entity{
// 		Id:    0,
// 		Class: "AGENT",
// 	}

// 	type args struct {
// 		ctx context.Context
// 		req *v1.GetEntityRequest
// 	}
// 	tests := []struct {
// 		name    string
// 		s       v1.SimulationServiceServer
// 		args    args
// 		want    *v1.GetEntityResponse
// 		wantErr bool
// 	}{
// 		{
// 			name: "Succesful Agent Retreival",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithValidSecret,
// 				req: &v1.GetEntityRequest{
// 					Api: "v1",
// 					Id:  entityID,
// 				},
// 			},
// 			want: &v1.GetEntityResponse{
// 				Api:    "v1",
// 				Entity: wantEntity,
// 			},
// 		},
// 		{
// 			name: "Unsupported API",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithValidSecret,
// 				req: &v1.GetEntityRequest{
// 					Api: "v1000",
// 					Id:  entityID,
// 				},
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			name: "Agent not found by that ID",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithValidSecret,
// 				req: &v1.GetEntityRequest{
// 					Api: "v1",
// 					Id:  999,
// 				},
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			name: "Invalid secret token",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithoutValidSecret,
// 				req: &v1.GetEntityRequest{
// 					Api: "v1",
// 					Id:  0,
// 				},
// 			},
// 			want: &v1.GetEntityResponse{
// 				Api:    "v1",
// 				Entity: wantEntity,
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.s.GetEntity(tt.args.ctx, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("simulationService.Create() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if err == nil && !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("simulationService.Create() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func Test_simulationServiceServer_DeleteAgent(t *testing.T) {
// 	ctxWithoutValidToken := context.Background()
// 	md := metadata.Pairs("auth-token", "TEST-ID-TOKEN")
// 	ctxWithValidToken := metadata.NewIncomingContext(context.Background(), md)

// 	s := NewSimulationServiceServer("testing")

// 	// Create an agent to test on
// 	agent := &v1.Entity{
// 		X: 2,
// 		Y: -4,
// 	}
// 	resp, err := s.CreateAgent(ctxWithValidToken, &v1.CreateAgentRequest{
// 		Api:   "v1",
// 		Agent: agent,
// 	})
// 	if err != nil {
// 		t.Errorf("toDoServiceServer.Create() error creating test agent. Make sure CreateAgent functionality is working.")
// 		return
// 	}
// 	agentID := resp.Id

// 	// Run tests
// 	type args struct {
// 		ctx context.Context
// 		req *v1.DeleteAgentRequest
// 	}
// 	tests := []struct {
// 		name    string
// 		s       v1.SimulationServiceServer
// 		args    args
// 		want    *v1.DeleteAgentResponse
// 		wantErr bool
// 	}{
// 		{
// 			name: "Unsupported API",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithValidToken,
// 				req: &v1.DeleteAgentRequest{
// 					Api: "v1000",
// 					Id:  agentID,
// 				},
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			name: "Agent not found by that ID",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithValidToken,
// 				req: &v1.DeleteAgentRequest{
// 					Api: "v1",
// 					Id:  999,
// 				},
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			name: "Invalid auth token",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithoutValidToken,
// 				req: &v1.DeleteAgentRequest{
// 					Api: "v1",
// 					Id:  0,
// 				},
// 			},
// 			wantErr: true,
// 		},
// 		{
// 			name: "Successful Agent Delete",
// 			s:    s,
// 			args: args{
// 				ctx: ctxWithValidToken,
// 				req: &v1.DeleteAgentRequest{
// 					Api: "v1",
// 					Id:  agentID,
// 				},
// 			},
// 			want: &v1.DeleteAgentResponse{
// 				Api:     "v1",
// 				Deleted: 1,
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.s.DeleteAgent(tt.args.ctx, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("simulationService.Create() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if err == nil && !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("simulationService.Create() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
