package storage

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Storage[I any] interface {
	Get(ctx context.Context, id string) (*I, error)
	SearchBy(ctx context.Context, path string, value any) ([]I, error)
	List(ctx context.Context, limit uint8, skip uint8) ([]I, error)

	Set(ctx context.Context, data *I) (id string, err error)
	Remove(ctx context.Context, id string) error
}

type StorageType uint8

const (
	StorageType_Memory StorageType = iota
)

func NewStorage[I any](stype StorageType) (Storage[I], error) {
	switch stype {
	case StorageType_Memory:
		memstorage := new(MemoryStorage[I])
		memstorage.data = make(map[string]*I)
		return memstorage, nil
	default:
		return nil, errors.New("storage not supported")
	}
}

type MemoryStorage[I any] struct {
	mux  sync.RWMutex
	data map[string]*I
}

var ErrNotFound = errors.New("data not found")

func (ms *MemoryStorage[I]) Get(ctx context.Context, id string) (*I, error) {
	ms.mux.RLock()
	defer ms.mux.RUnlock()
	data, ok := ms.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	return data, nil
}

func (ms *MemoryStorage[I]) SearchBy(ctx context.Context, path string, value any) ([]I, error) {

	ms.mux.RLock()
	defer ms.mux.RUnlock()

	var results []I

	for _, d := range ms.data {
		fieldValue, err := getFieldByPath(d, path)
		if err != nil {
			continue
		}

		if compareValues(fieldValue, value) {
			results = append(results, *d)
		}

	}

	return results, nil
}

func getFieldByPath(obj any, path string) (any, error) {
	v := reflect.ValueOf(obj)

	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, errors.New("nil pointer")
		}

		v = v.Elem()
	}

	parts := strings.Split(path, ".")

	for _, part := range parts {
		if v.Kind() != reflect.Struct {
			return nil, errors.New("not a struct")
		}

		field := v.FieldByName(part)
		if !field.IsValid() {
			return nil, errors.New("field not found: " + part)
		}

		v = field
	}

	return v.Interface(), nil
}

func compareValues(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

func (ms *MemoryStorage[I]) List(ctx context.Context, _limit uint8, _skip uint8) ([]I, error) {
	ms.mux.RLock()
	defer ms.mux.RUnlock()

	data := make([]I, len(ms.data))

	for _, d := range ms.data {
		data = append(data, *d)
	}
	return data, nil
}

func (ms *MemoryStorage[I]) Set(ctx context.Context, data *I) (string, error) {
	ms.mux.Lock()
	defer ms.mux.Unlock()

	id := uuid.NewString()

	ms.data[id] = data

	return id, nil
}
func (ms *MemoryStorage[I]) Remove(ctx context.Context, id string) error {
	ms.mux.Lock()
	defer ms.mux.Unlock()

	delete(ms.data, id)

	return nil
}
