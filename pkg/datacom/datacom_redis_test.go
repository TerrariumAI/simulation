package datacom_test

import (
	"reflect"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/mock"
	collectiveApi "github.com/terrariumai/simulation/pkg/api/collective"
	envApi "github.com/terrariumai/simulation/pkg/api/environment"
	"github.com/terrariumai/simulation/pkg/datacom"
	"github.com/terrariumai/simulation/pkg/datacom/mocks"
)

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
	}{
		{
			"Test succesful creation",
			args{
				entity: envApi.Entity{
					X:        123,
					Y:        456,
					OwnerUID: "MOCK-UID",
					ModelID:  "MOCK-MODEL-ID",
					Energy:   100,
					Health:   100,
					Id:       "0",
					Class:    1,
				},
				shouldPublish: true,
			},
			1,
			"142536:123:456:1:MOCK-UID:MOCK-MODEL-ID:100:100:0",
		},
		{
			"Test no publish",
			args{
				entity: envApi.Entity{
					X:        123,
					Y:        456,
					OwnerUID: "MOCK-UID",
					ModelID:  "MOCK-MODEL-ID",
					Energy:   100,
					Health:   100,
					Id:       "0",
					Class:    1,
				},
				shouldPublish: false,
			},
			0,
			"142536:123:456:1:MOCK-UID:MOCK-MODEL-ID:100:100:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup pubsub mock
			mockPAL := &mocks.PubsubAccessLayer{}
			dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)
			mockPAL.On("QueuePublishEvent", "createEntity", tt.args.entity).Return(nil)

			err := dc.CreateEntity(tt.args.entity, tt.args.shouldPublish)
			if err != nil {
				t.Errorf("got error: %v", err)
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

			keys, cursor, err := redisClient.ZScan("entities", 0, "*", 0).Result()
			if len(keys) != 2 {
				t.Errorf("expected keys to be larger than 0, got: %v", len(keys))
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
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("Entity")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)
	e := envApi.Entity{
		X: 0, Y: 0, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
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
			isOccupied, got, err := dc.IsCellOccupied(tt.args.x, tt.args.y)
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
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("Entity")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 0, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
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
					Class:    2,
					OwnerUID: "MOCK-UID-2",
					ModelID:  "MOCK-MODEL-ID-2",
					Energy:   90,
					Health:   90,
					Id:       "0",
				},
			},
			"000011:1:1:2:MOCK-UID-2:MOCK-MODEL-ID-2:90:90:0",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock for publish
			mockPAL.On("QueuePublishEvent", "updateEntity", tt.args.entity).Return(nil)

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
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("Entity")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 0, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
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
			"000000:0:0:1:MOCK-UID:MOCK-MODEL-ID:100:100:0",
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

			if *content != tt.expected {
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
		X: 0, Y: 0, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}

	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("Entity")).Return(nil)
	dc.CreateEntity(e, true)
	mockPAL.On("QueuePublishEvent", "deleteEntity", mock.AnythingOfType("Entity")).Return(nil)

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
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("Entity")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 0, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 1, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-2", Health: 100, Energy: 100, Id: "1",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 0, Y: 2, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-2", Health: 100, Energy: 100, Id: "2",
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
	mockPAL.On("QueuePublishEvent", "createEntity", mock.AnythingOfType("Entity")).Return(nil)
	dc, _ := datacom.NewDatacom("testing", redisServer.Addr(), mockPAL)

	dc.CreateEntity(envApi.Entity{
		X: 2, Y: 2, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "0",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 2, Y: 3, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID", Health: 100, Energy: 100, Id: "1",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 3, Y: 3, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-2", Health: 100, Energy: 100, Id: "2",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 4, Y: 4, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-3", Health: 100, Energy: 100, Id: "3",
	}, true)
	dc.CreateEntity(envApi.Entity{
		X: 66, Y: 66, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-4", Health: 100, Energy: 100, Id: "4",
	}, true)
	err := dc.CreateEntity(envApi.Entity{
		X: 1, Y: 1, Class: 1, OwnerUID: "MOCK-UID", ModelID: "MOCK-MODEL-ID-5", Health: 100, Energy: 100, Id: "5",
	}, true)
	println(err)

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
				Id: "2",
				Cells: []*collectiveApi.Entity{
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{
						Id:    "1",
						Class: 1,
					},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{
						Id:    "0",
						Class: 1,
					},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
				},
			},
			false,
		},
		{
			"Test 1 entity in vision",
			args{
				modelID: "MOCK-MODEL-ID-3",
			},
			&collectiveApi.Observation{
				Id: "3",
				Cells: []*collectiveApi.Entity{
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{
						Id:    "2",
						Class: 1,
					},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
				},
			},
			false,
		},
		{
			"Test 0 entities in vision",
			args{
				modelID: "MOCK-MODEL-ID-4",
			},
			&collectiveApi.Observation{
				Id: "4",
				Cells: []*collectiveApi.Entity{
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
				},
			},
			false,
		},
		{
			"Test rocks in invalid positions",
			args{
				modelID: "MOCK-MODEL-ID-5",
			},
			&collectiveApi.Observation{
				Id: "5",
				Cells: []*collectiveApi.Entity{
					&collectiveApi.Entity{Id: "", Class: 2},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 2},
					&collectiveApi.Entity{Id: "", Class: 0},
					&collectiveApi.Entity{Id: "", Class: 2},
					&collectiveApi.Entity{Id: "", Class: 2},
					&collectiveApi.Entity{Id: "", Class: 2},
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

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestPubnubPAL(t *testing.T) {
	p := datacom.NewPubnubPAL("prod", "sub-c-b4ba4e28-a647-11e9-ad2c-6ad2737329fc", "pub-c-83ed11c2-81e1-4d7f-8e94-0abff2b85825")
	p.QueuePublishEvent("test", envApi.Entity{Id: "test-id"})
	t.Log("Queued publish message, batching...")
	p.BatchPublish()
}
