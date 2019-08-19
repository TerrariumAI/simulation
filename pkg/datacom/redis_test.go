package datacom_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/mock"
	collectiveApi "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	"github.com/terrariumai/simulation/pkg/datacom"
	"github.com/terrariumai/simulation/pkg/datacom/mocks"
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

func setup() *miniredis.Miniredis {
	// Redis Setup
	redisServer, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	return redisServer
}

func teardown(redisServer *miniredis.Miniredis) {
	redisServer.Close()
}

// -------------------------------------
// CREATE ENTITY
// -------------------------------------
func TestCreateEntity(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		entity        envApi.Entity
		shouldPublish bool
	}
	tests := []struct {
		name                 string
		args                 args
		expectedPublishCount int
		expected             string
		expectErr            error
	}{
		{
			name: "Test succesful creation",
			args: args{
				entity: envApi.Entity{
					X:        123,
					Y:        456,
					OwnerUID: "MOCK-UID",
					ModelID:  "MOCK-MODEL-ID",
					Energy:   100,
					Health:   100,
					Id:       "1",
					ClassID:  1,
				},
				shouldPublish: true,
			},
			expectedPublishCount: 1,
			expected:             "010111101011001010:123:456:1:MOCK-UID:MOCK-MODEL-ID:100:100:1",
		},
		{
			name: "Test invalid position error",
			args: args{
				entity: envApi.Entity{
					X:        512,
					Y:        456,
					OwnerUID: "MOCK-UID",
					ModelID:  "MOCK-MODEL-ID",
					Energy:   100,
					Health:   100,
					Id:       "0",
					ClassID:  1,
				},
				shouldPublish: true,
			},
			expectedPublishCount: 1,
			expectErr:            errors.New("invalid position"),
		},
		{
			name: "Test no publish",
			args: args{
				entity: envApi.Entity{
					X:        123,
					Y:        456,
					OwnerUID: "MOCK-UID",
					ModelID:  "MOCK-MODEL-ID",
					Energy:   100,
					Health:   100,
					Id:       "1",
					ClassID:  1,
				},
				shouldPublish: false,
			},
			expected: "010111101011001010:123:456:1:MOCK-UID:MOCK-MODEL-ID:100:100:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup pubsub mock
			redisServer.FlushDB()
			mockPAL := &mocks.PubsubAccessLayer{}
			dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)
			mockPAL.On("QueuePublishEvent", "createEntity", &tt.args.entity, tt.args.entity.X, tt.args.entity.Y).Return(nil)

			err := dc.CreateEntity(tt.args.entity, tt.args.shouldPublish)
			if err != nil && tt.expectErr != nil {
				if err.Error() != tt.expectErr.Error() {
					t.Errorf("expected error: %v, got %v", tt.expectErr, err)
					return
				}
				return
			} else if err != nil && tt.expectErr == nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Make sure the entity data is there
			redisClient := redis.NewClient(&redis.Options{
				Addr:     redisServer.Addr(),
				Password: "", // no password set
				DB:       0,  // use default DB
			})

			// Check publish calls
			mockPAL.AssertNumberOfCalls(t, "QueuePublishEvent", tt.expectedPublishCount)
			keys, cursor := redisClient.ZScan("entities", 0, "*", 0).Val()

			keys, cursor, err = redisClient.ZScan("entities", 0, "*", 0).Result()
			if err != nil {
				t.Errorf("error in scan: %v", err)
			}
			if len(keys) != 2 {
				t.Errorf("expected length of keys to be == 2, got: %v", len(keys))
				return
			}

			if keys[cursor] != tt.expected {
				t.Errorf("wanted %v, \n\t got: %v", tt.expected, keys[cursor])
			}
		})
	}
}

// -------------------------------------
// Is Cell Occupied
// -------------------------------------
func TestIsCellOccupied(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)
	// Setup pubsub mock
	mockPAL := &mocks.PubsubAccessLayer{}
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("*endpoints_terrariumai_environment.Entity"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)
	e := envApi.Entity{
		X: 0, Y: 0, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}
	dc.CreateEntity(e, true)

	type args struct {
		x uint32
		y uint32
	}
	tests := []struct {
		name           string
		args           args
		expected       bool
		expectedEntity *envApi.Entity
		expectErr      bool
	}{
		{
			"Test cell is occupied",
			args{
				x: 0,
				y: 0,
			},
			true,
			&e,
			false,
		},
		{
			name: "Test cell unoccupied",
			args: args{
				x: 1,
				y: 0,
			},
			expected:       false,
			expectedEntity: nil,
			expectErr:      false,
		},
		{
			"Test error on invalid position",
			args{
				x: 3333,
				y: 0,
			},
			false,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isOccupied, got, _, err := dc.IsCellOccupied(tt.args.x, tt.args.y)
			if err != nil && tt.expectErr {
				return
			} else if err != nil && !tt.expectErr {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if isOccupied != tt.expected {
				t.Errorf("expected %v, \n\t got: %v", tt.expected, isOccupied)
			}

			if !reflect.DeepEqual(got, tt.expectedEntity) {
				t.Errorf("got %v, expected %v", got, tt.expectedEntity)
			}
		})
	}
}

// -------------------------------------
// Update Entity
// -------------------------------------
func TestUpdateEntity(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)
	// Setup pubsub mock
	mockPAL := &mocks.PubsubAccessLayer{}
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("*endpoints_terrariumai_environment.Entity"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 0, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}, true)

	type args struct {
		origionalContent string
		entity           envApi.Entity
	}
	tests := []struct {
		name      string
		args      args
		expected  string
		expectErr bool
	}{
		{
			"Test update every field",
			args{
				entity: envApi.Entity{
					X:        1,
					Y:        1,
					ClassID:  2,
					OwnerUID: "MOCK-UID-2",
					ModelID:  "MOCK-MODEL-ID-2",
					Energy:   90,
					Health:   90,
					Id:       "0",
				},
			},
			"000000000000000011:1:1:2:MOCK-UID-2:MOCK-MODEL-ID-2:90:90:0",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock for publish
			mockPAL.On("QueuePublishEvent", "updateEntity", &tt.args.entity, mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil)

			// Make sure the entity data is there
			redisClient := redis.NewClient(&redis.Options{
				Addr:     redisServer.Addr(),
				Password: "", // no password set
				DB:       0,  // use default DB
			})

			// Get the origional content
			keys, cursor, err := redisClient.ZScan("entities", 0, "*", 0).Result()

			err = dc.UpdateEntity(keys[cursor], tt.args.entity)
			if err != nil && tt.expectErr {
				return
			} else if err != nil && !tt.expectErr {
				t.Errorf("unexpected error: %v", err)
				return
			}

			keys, cursor, err = redisClient.ZScan("entities", 0, "*", 0).Result()

			if keys[cursor] != tt.expected {
				t.Errorf("expected %v, \n\t got: %v", tt.expected, keys[cursor])
			}
		})
	}
}

// -------------------------------------
// Get Entity
// -------------------------------------
func TestGetEntity(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)
	// Setup pubsub mock
	mockPAL := &mocks.PubsubAccessLayer{}
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("*endpoints_terrariumai_environment.Entity"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 0, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}, true)

	type args struct {
		id string
	}
	tests := []struct {
		name      string
		args      args
		expected  string
		expectErr bool
	}{
		{
			"Test success",
			args{
				id: "0",
			},
			"000000000000000000:0:0:1:MOCK-UID:MOCK-MODEL-ID:100:100:0",
			false,
		},
		{
			"Test fail",
			args{
				id: "1",
			},
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make sure the entity data is there
			redisClient := redis.NewClient(&redis.Options{
				Addr:     redisServer.Addr(),
				Password: "", // no password set
				DB:       0,  // use default DB
			})

			// Get the origional content
			keys, cursor, err := redisClient.ZScan("entities", 0, "*", 0).Result()

			_, content, err := dc.GetEntity(tt.args.id)
			if err != nil && tt.expectErr {
				return
			} else if err != nil && !tt.expectErr {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if content != tt.expected {
				t.Errorf("expected %v, \n\t got: %v", tt.expected, keys[cursor])
			}
		})
	}
}

func TestDeleteEntity(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)
	// Setup pubsub mock
	mockPAL := &mocks.PubsubAccessLayer{}
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

	e := envApi.Entity{
		X: 0, Y: 0, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}

	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("*endpoints_terrariumai_environment.Entity"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil)
	dc.CreateEntity(e, true)
	mockPAL.On("QueuePublishEvent", "deleteEntity", mock.AnythingOfType("*endpoints_terrariumai_environment.Entity"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil)

	got, _, err := dc.GetEntity(e.Id)
	if !reflect.DeepEqual(*got, e) {
		t.Errorf("got %v, expected %v", got, e)
	}
	if err != nil {
		t.Errorf("unexpected err: %v", e)
	}

	gotCount, err := dc.DeleteEntity(e.Id)
	var expectedCount int64 = 1
	expectErr := false

	if err != nil && expectErr {
		return
	} else if err != nil && !expectErr {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if gotCount != expectedCount {
		t.Errorf("got: %v , expected: %v", got, expectedCount)
		return
	}

	got, _, err = dc.GetEntity(e.Id)
	if err == nil {
		t.Errorf("expected error, got %v", got)
	}
}

// -------------------------------------
// Get Entities For model
// -------------------------------------
func TestGetEntitiesForModel(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)
	// Setup pubsub mock
	mockPAL := &mocks.PubsubAccessLayer{}
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("*endpoints_terrariumai_environment.Entity"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 0, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 1, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-2", Health: 100, Energy: 100, Id: "1",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 2, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-2", Health: 100, Energy: 100, Id: "2",
	}, true)

	type args struct {
		id string
	}
	tests := []struct {
		name      string
		args      args
		expected  int
		expectErr bool
	}{
		{
			"Test 1",
			args{
				id: "MOCK-MODEL-ID",
			},
			1,
			false,
		},
		{
			"Test 2",
			args{
				id: "MOCK-MODEL-ID-2",
			},
			2,
			false,
		},
		{
			"Test none",
			args{
				id: "",
			},
			0,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entitiesContent, err := dc.GetEntitiesForModel(tt.args.id)
			if err != nil && tt.expectErr {
				return
			} else if err != nil && !tt.expectErr {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(entitiesContent) != tt.expected {
				t.Errorf("expected %v, \n\t got: %v", tt.expected, len(entitiesContent))
			}
		})
	}
}

// -------------------------------------
// Get Observations For Entity
// -------------------------------------
func TestGetObservationsForEntity(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)
	// Setup pubsub mock
	mockPAL := &mocks.PubsubAccessLayer{}
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("*endpoints_terrariumai_environment.Entity"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)
	dc.EntityVisionDist = 2

	dc.CreateEntity(envApi.Entity{
		X: 1, Y: 1, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 0, ClassID: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-2", Health: 100, Energy: 100, Id: "2",
	}, true)

	type args struct {
		modelID string
	}
	tests := []struct {
		name      string
		args      args
		expected  *collectiveApi.Observation
		expectErr bool
	}{
		{
			"Test full vision",
			args{
				modelID: "MOCK-MODEL-ID-2",
			},
			&collectiveApi.Observation{
				Id:      "2",
				IsAlive: true,
				Energy:  100,
				Health:  100,
				Sight: []*collectiveApi.Entity{
					&collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 0}, &collectiveApi.Entity{ClassID: 0}, &collectiveApi.Entity{ClassID: 0},
					&collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 0}, &collectiveApi.Entity{Id: "0", ClassID: 1}, &collectiveApi.Entity{ClassID: 0},
					&collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 0}, &collectiveApi.Entity{ClassID: 0},
					&collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2},
					&collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2}, &collectiveApi.Entity{ClassID: 2},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := dc.GetEntitiesForModel(tt.args.modelID)
			if err != nil && tt.expectErr {
				return
			} else if err != nil && !tt.expectErr {
				t.Errorf("unexpected error: %v", err)
				return
			}

			entity := entities[0]

			got, err := dc.GetObservationForEntity(entity)
			if err != nil && tt.expectErr {
				return
			} else if err != nil && !tt.expectErr {
				t.Errorf("unexpected error: %v", err)
				return
			}

			assert.Equal(t, 24, len(got.Sight), "Number of cells should be dist*dist-1")

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCreateEffect(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		effect envApi.Effect
	}

	tests := []struct {
		name             string
		args             args
		PALMockFuncCalls []mockFuncCall
		want             int
		wantErr          error
	}{
		{
			name: "Succesful add effect",
			args: args{
				effect: envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
			PALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"createEffect", &envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1}, uint32(1), uint32(1)},
					resp: []interface{}{nil},
				},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup pubsub mock
			mockPAL := &mocks.PubsubAccessLayer{}
			dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

			// Setup mock
			for _, mockFuncCall := range tt.PALMockFuncCalls {
				mockPAL.On(mockFuncCall.name, mockFuncCall.args...).Return(mockFuncCall.resp...)
			}
			// Call function
			err := dc.CreateEffect(tt.args.effect)
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
			// Query to check that it exists in DB
			redisClient := redis.NewClient(&redis.Options{
				Addr:     redisServer.Addr(),
				Password: "", // no password set
				DB:       0,  // use default DB
			})
			keys, cursor := redisClient.ZScan("effects", 0, "*", 0).Val()
			if len(keys) != tt.want {
				t.Errorf("got %v, want %v", len(keys), tt.want)
			}

			effect := envApi.Effect{}
			values := strings.Split(keys[cursor], "-")
			content := strings.ReplaceAll(values[1], "%n", "\n")
			proto.UnmarshalText(content, &effect)
			fmt.Println(content)

			if !reflect.DeepEqual(effect, tt.args.effect) {
				t.Errorf("got %v, expected %v", effect, tt.args.effect)
			}
		})
	}
}

func TestGetEffectsInRegion(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		x0 uint32
		y0 uint32
		x1 uint32
		y1 uint32
	}

	tests := []struct {
		name             string
		preCallEffects   []envApi.Effect
		args             args
		PALMockFuncCalls []mockFuncCall
		want             []*envApi.Effect
		wantErr          error
	}{
		{
			name: "Get single effect in region 0.0",
			preCallEffects: []envApi.Effect{
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
			args: args{
				x0: 0,
				y0: 0,
				x1: 9,
				y1: 9,
			},
			PALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"createEffect", mock.AnythingOfType("*endpoints_terrariumai_environment.Effect"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")},
					resp: []interface{}{nil},
				},
			},
			want: []*envApi.Effect{
				&envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
		},
		{
			name: "Get single effect in region 0.0 with effects just outside region",
			preCallEffects: []envApi.Effect{
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
				envApi.Effect{X: 10, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
				envApi.Effect{X: 1, Y: 10, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
			args: args{
				x0: 0,
				y0: 0,
				x1: 9,
				y1: 9,
			},
			PALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"createEffect", mock.AnythingOfType("*endpoints_terrariumai_environment.Effect"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")},
					resp: []interface{}{nil},
				},
			},
			want: []*envApi.Effect{
				&envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
		},
		{
			name: "Get multiple effects, diff pos, same region",
			preCallEffects: []envApi.Effect{
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
				envApi.Effect{X: 1, Y: 2, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
			args: args{
				x0: 0,
				y0: 0,
				x1: 9,
				y1: 9,
			},
			PALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"createEffect", mock.AnythingOfType("*endpoints_terrariumai_environment.Effect"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")},
					resp: []interface{}{nil},
				},
			},
			want: []*envApi.Effect{
				&envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
				&envApi.Effect{X: 1, Y: 2, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
		},
		{
			name: "Get multiple effects, same pos, same region",
			preCallEffects: []envApi.Effect{
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(1), Value: 2},
			},
			args: args{
				x0: 0,
				y0: 0,
				x1: 9,
				y1: 9,
			},
			PALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"createEffect", mock.AnythingOfType("*endpoints_terrariumai_environment.Effect"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")},
					resp: []interface{}{nil},
				},
			},
			want: []*envApi.Effect{
				&envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(1), Value: 2},
				&envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup pubsub mock
			mockPAL := &mocks.PubsubAccessLayer{}
			dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

			// Setup mock
			for _, mockFuncCall := range tt.PALMockFuncCalls {
				mockPAL.On(mockFuncCall.name, mockFuncCall.args...).Return(mockFuncCall.resp...)
			}
			// Setup effects
			for _, effect := range tt.preCallEffects {
				dc.CreateEffect(effect)
			}

			// Call function
			effects, err := dc.GetEffectsInSpace(tt.args.x0, tt.args.y0, tt.args.x1, tt.args.y1)
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

			if !reflect.DeepEqual(effects, tt.want) {
				t.Errorf("got %v, expected %v", effects, tt.want)
			}

			// Clear db
			redisClient := redis.NewClient(&redis.Options{
				Addr:     redisServer.Addr(),
				Password: "", // no password set
				DB:       0,  // use default DB
			})
			redisClient.FlushDB()
		})
	}
}

func TestDeleteEffect(t *testing.T) {
	redisServer := setup()
	defer teardown(redisServer)

	type args struct {
		effect envApi.Effect
	}

	tests := []struct {
		name             string
		preCallEffects   []envApi.Effect
		args             args
		PALMockFuncCalls []mockFuncCall
		want             []*envApi.Effect
		wantErr          error
	}{
		{
			name: "Delete single effect",
			preCallEffects: []envApi.Effect{
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
			args: args{
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
			PALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"createEffect", mock.AnythingOfType("*endpoints_terrariumai_environment.Effect"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")},
					resp: []interface{}{nil},
				},
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"deleteEffect", mock.AnythingOfType("*endpoints_terrariumai_environment.Effect"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")},
					resp: []interface{}{nil},
				},
			},
			want: []*envApi.Effect{},
		},
		{
			name: "Multiple effects in same position (same index), delete only removes 1",
			preCallEffects: []envApi.Effect{
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 2},
			},
			args: args{
				envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 1},
			},
			PALMockFuncCalls: []mockFuncCall{
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"createEffect", mock.AnythingOfType("*endpoints_terrariumai_environment.Effect"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")},
					resp: []interface{}{nil},
				},
				{ // Get the metadata for the RM
					name: "QueuePublishEvent",
					args: []interface{}{"deleteEffect", mock.AnythingOfType("*endpoints_terrariumai_environment.Effect"), mock.AnythingOfType("uint32"), mock.AnythingOfType("uint32")},
					resp: []interface{}{nil},
				},
			},
			want: []*envApi.Effect{
				&envApi.Effect{X: 1, Y: 1, Timestamp: time.Now().Unix(), ClassID: envApi.Effect_Class(0), Value: 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup pubsub mock
			mockPAL := &mocks.PubsubAccessLayer{}
			dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

			// Setup mock
			for _, mockFuncCall := range tt.PALMockFuncCalls {
				mockPAL.On(mockFuncCall.name, mockFuncCall.args...).Return(mockFuncCall.resp...)
			}
			// Setup effects
			for _, effect := range tt.preCallEffects {
				dc.CreateEffect(effect)
			}

			// Call function
			_, err := dc.DeleteEffect(tt.args.effect)
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

			// Check
			effects, _ := dc.GetEffectsInSpace(0, 0, 9, 9)

			if !reflect.DeepEqual(effects, tt.want) {
				t.Errorf("got %v, expected %v", effects, tt.want)
			}

			// Clear db
			redisClient := redis.NewClient(&redis.Options{
				Addr:     redisServer.Addr(),
				Password: "", // no password set
				DB:       0,  // use default DB
			})
			redisClient.FlushDB()
		})
	}
}
func TestPubnubPAL(t *testing.T) {
	p := datacom.NewPubnubPAL("testing", "sub-c-b4ba4e28-a647-11e9-ad2c-6ad2737329fc", "pub-c-83ed11c2-81e1-4d7f-8e94-0abff2b85825")
	p.QueuePublishEvent("updateEntity", &envApi.Entity{Id: "test-id", Y: 0}, 0, 0)
	p.QueuePublishEvent("updateEntity", &envApi.Entity{Id: "test-id-2", X: 5, Y: 0}, 5, 0)
	t.Log("Queued publish message, batching...")
	p.BatchPublish()
}
