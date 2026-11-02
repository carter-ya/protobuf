package proto

import (
	"github.com/gogo/protobuf/internal/storage"
	"reflect"
)

func generatedField(v reflect.Value, index int) reflect.Value { return storage.Field(v, index) }
