package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/olamai/simulation/pkg/api/v1"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion = "v1"
)

// toDoServiceServer is implementation of v1.ToDoServiceServer proto interface
type toDoServiceServer struct {
	todos []v1.ToDo
}

func (s *toDoServiceServer) NewTodo(title string, description string) int64 {
	id := int64(len(s.todos))
	s.todos = append(s.todos, v1.ToDo{
		Id:          id,
		Title:       title,
		Description: description,
	})
	return id
}

// NewToDoServiceServer creates ToDo service
func NewToDoServiceServer() v1.ToDoServiceServer {
	return &toDoServiceServer{
		todos: []v1.ToDo{
			v1.ToDo{},
		},
	}
}

// checkAPI checks if the API version requested by client is supported by server
func (s *toDoServiceServer) checkAPI(api string) error {
	// API version is "" means use current version of the service
	if len(api) > 0 {
		if apiVersion != api {
			return status.Errorf(codes.Unimplemented,
				"unsupported API version: service implements API version '%s', but asked for '%s'", apiVersion, api)
		}
	}
	return nil
}

// Create new todo task
func (s *toDoServiceServer) Create(ctx context.Context, req *v1.CreateRequest) (*v1.CreateResponse, error) {
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	// Create todo
	id := s.NewTodo(req.ToDo.Title, req.ToDo.Description)

	return &v1.CreateResponse{
		Api: apiVersion,
		Id:  id,
	}, nil
}

// Read todo task
func (s *toDoServiceServer) Read(ctx context.Context, req *v1.ReadRequest) (*v1.ReadResponse, error) {
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	todo := s.todos[req.Id]

	return &v1.ReadResponse{
		Api:  apiVersion,
		ToDo: &todo,
	}, nil

}

// Update todo task
func (s *toDoServiceServer) Update(ctx context.Context, req *v1.UpdateRequest) (*v1.UpdateResponse, error) {
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	s.todos[req.ToDo.Id].Title = req.ToDo.Title
	s.todos[req.ToDo.Id].Description = req.ToDo.Description

	return &v1.UpdateResponse{
		Api:     apiVersion,
		Updated: 1,
	}, nil
}

// Delete todo task
func (s *toDoServiceServer) Delete(ctx context.Context, req *v1.DeleteRequest) (*v1.DeleteResponse, error) {
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}

	return &v1.DeleteResponse{
		Api:     apiVersion,
		Deleted: 0,
	}, nil
}

// Read all todo tasks
func (s *toDoServiceServer) ReadAll(ctx context.Context, req *v1.ReadAllRequest) (*v1.ReadAllResponse, error) {
	// check if the API version requested by client is supported by server
	if err := s.checkAPI(req.Api); err != nil {
		return nil, err
	}
	list := []*v1.ToDo{}
	for _, todo := range s.todos {
		list = append(list, &todo)
	}

	return &v1.ReadAllResponse{
		Api:   apiVersion,
		ToDos: list,
	}, nil
}
