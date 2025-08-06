package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"oil/config"
	"oil/infras/otel/mocks"
	todoMocks "oil/internal/domains/todo/mocks"
	"oil/internal/domains/todo/model"
	"oil/internal/domains/todo/model/dto"
	"oil/internal/domains/todo/service"
	cacheMocks "oil/shared/cache/mocks"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"oil/shared/timezone"
)

func TestTodoService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := todoMocks.NewMockTodo(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel)

	tests := []struct {
		name      string
		req       dto.CreateTodoRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful creation",
			req: dto.CreateTodoRequest{
				Title:       "Test Todo",
				Description: "Test Description",
			},
			setupMock: func() {
				mockRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "repository error",
			req: dto.CreateTodoRequest{
				Title:       "Test Todo",
				Description: "Test Description",
			},
			setupMock: func() {
				mockRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.WithValue(context.Background(), constant.ContextKeyUserID, "test-user-id")
			err := svc.Create(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTodoService_GetAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := todoMocks.NewMockTodo(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel)

	tests := []struct {
		name       string
		params     gDto.QueryParams
		filter     gDto.FilterGroup
		setupMock  func()
		wantErr    bool
		wantResult dto.GetTodosResponse
	}{
		{
			name: "successful get all",
			params: gDto.QueryParams{
				Limit: 10,
				Page:  1,
			},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockRepo.EXPECT().
					Count(gomock.Any(), gomock.Any()).
					Return(1, nil)

				todos := []model.Todo{
					{
						ID:          "test-id",
						Title:       "Test Todo",
						Description: "Test Description",
						Completed:   false,
						Metadata: gModel.Metadata{
							CreatedAt:  timezone.Now(),
							ModifiedAt: timezone.Now(),
							CreatedBy:  "test-user",
							ModifiedBy: "test-user",
						},
					},
				}

				mockRepo.EXPECT().
					GetAll(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(todos, nil)
			},
			wantErr: false,
			wantResult: dto.GetTodosResponse{
				TotalData: 1,
				TotalPage: 1,
			},
		},
		{
			name: "count error",
			params: gDto.QueryParams{
				Limit: 10,
				Page:  1,
			},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockRepo.EXPECT().
					Count(gomock.Any(), gomock.Any()).
					Return(0, errors.New("count error"))
			},
			wantErr: true,
		},
		{
			name: "get all error",
			params: gDto.QueryParams{
				Limit: 10,
				Page:  1,
			},
			filter: gDto.FilterGroup{},
			setupMock: func() {
				mockRepo.EXPECT().
					Count(gomock.Any(), gomock.Any()).
					Return(1, nil)

				mockRepo.EXPECT().
					GetAll(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("get all error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			result, err := svc.GetAll(ctx, tt.params, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult.TotalData, result.TotalData)
				assert.Equal(t, tt.wantResult.TotalPage, result.TotalPage)
			}
		})
	}
}

func TestTodoService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := todoMocks.NewMockTodo(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel)

	todo := model.Todo{
		ID:          "test-id",
		Title:       "Test Todo",
		Description: "Test Description",
		Completed:   false,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  "test-user",
			ModifiedBy: "test-user",
		},
	}

	tests := []struct {
		name      string
		id        string
		setupMock func()
		wantErr   bool
		wantID    string
	}{
		{
			name: "cache hit",
			id:   "test-id",
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
			wantID:  "",
		},
		{
			name: "cache miss, successful get from db",
			id:   "test-id",
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(todo, nil)

				mockCache.EXPECT().
					Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			wantErr: false,
			wantID:  "test-id",
		},
		{
			name: "todo not found",
			id:   "nonexistent-id",
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(model.Todo{}, nil) // Empty todo means not found
			},
			wantErr: true,
		},
		{
			name: "repository error",
			id:   "test-id",
			setupMock: func() {
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("cache miss"))

				mockRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(model.Todo{}, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			result, err := svc.Get(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantID != "" {
					assert.Equal(t, tt.wantID, result.ID)
				}
			}
		})
	}
}

func TestTodoService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := todoMocks.NewMockTodo(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel)

	tests := []struct {
		name      string
		req       dto.UpdateTodoRequest
		id        string
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful update",
			req: dto.UpdateTodoRequest{
				Title:       "Updated Title",
				Description: "Updated Description",
			},
			id: "test-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				mockCache.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "empty update request",
			req:  dto.UpdateTodoRequest{},
			id:   "test-id",
			setupMock: func() {
				// No mock expectations as validation should fail early
			},
			wantErr: true,
		},
		{
			name: "todo not found",
			req: dto.UpdateTodoRequest{
				Title: "Updated Title",
			},
			id: "nonexistent-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantErr: true,
		},
		{
			name: "exist check error",
			req: dto.UpdateTodoRequest{
				Title: "Updated Title",
			},
			id: "test-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "update error",
			req: dto.UpdateTodoRequest{
				Title: "Updated Title",
			},
			id: "test-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("update error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.WithValue(context.Background(), constant.ContextKeyUserID, "test-user-id")
			err := svc.Update(ctx, tt.req, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTodoService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := todoMocks.NewMockTodo(ctrl)
	mockCache := cacheMocks.NewMockRedisCache(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}
	cfg.Cache.TTL = 3600

	svc := service.New(mockRepo, cfg, mockCache, mockOtel)

	tests := []struct {
		name      string
		id        string
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful deletion",
			id:   "test-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockRepo.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil)

				mockCache.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "todo not found",
			id:   "nonexistent-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantErr: true,
		},
		{
			name: "exist check error",
			id:   "test-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "delete error",
			id:   "test-id",
			setupMock: func() {
				mockRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockRepo.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(errors.New("delete error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			err := svc.Delete(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
