package datacom_test

import (
	"reflect"
	"testing"
	"github.com/terrariumai/simulation/pkg/datacom"
	"github.com/alicebob/miniredis"
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

func TestPosToRedisIndex(t *testing.T) {
	type args struct {
		x int32
		y int32
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"Test 0,0",
			args{
				x: 0,
				y: 0,
			},
			"000000",
			false,
		},
		{
			"Test 1,1",
			args{
				x: 1,
				y: 1,
			},
			"000011",
			false,
		},
		{
			"Test 12,34",
			args{
				x: 12,
				y: 34,
			},
			"001324",
			false,
		},
		{
			"Test 123,456",
			args{
				x: 123,
				y: 456,
			},
			"142536",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := datacom.PosToRedisIndex(tt.args.x, tt.args.y)
			if (err != nil) != tt.wantErr {
				t.Errorf("got: %v, expected %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got: %v, want %v", got, tt.want)
			}
		})
	}
}

// func TestRedis(t *testing.T) {
// 	redisServer := setup()
// 	defer teardown(redisServer)
// 	dc, err := NewDatacom("testing", redisServer.Addr())
// 	if err != nil {
// 		t.Errorf("Couldn't create DataCom object: %v", err)
// 	}

// 	dc.
// }
