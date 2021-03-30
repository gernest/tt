/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package host

import (
	"reflect"

	"github.com/wasmerio/wasmer-go/wasmer"
)

func convertFromGoType(t reflect.Type) *wasmer.ValueType {
	switch t.Kind() {
	case reflect.Int32:
		return wasmer.NewValueType(wasmer.I32)
	case reflect.Int64:
		return wasmer.NewValueType(wasmer.I64)
	case reflect.Float32:
		return wasmer.NewValueType(wasmer.F32)
	case reflect.Float64:
		return wasmer.NewValueType(wasmer.F64)
	}

	return nil
}

func convertToGoTypes(in wasmer.Value) reflect.Value {
	switch in.Kind() {
	case wasmer.I32:
		return reflect.ValueOf(in.I32())
	case wasmer.I64:
		return reflect.ValueOf(in.I64())
	case wasmer.F32:
		return reflect.ValueOf(in.F32())
	case wasmer.F64:
		return reflect.ValueOf(in.F64())
	}

	return reflect.Value{}
}

func convertFromGoValue(val reflect.Value) wasmer.Value {
	switch val.Kind() {
	case reflect.Int32:
		return wasmer.NewI32(int32(val.Int()))
	case reflect.Int64:
		return wasmer.NewI64(int64(val.Int()))
	case reflect.Float32:
		return wasmer.NewF32(float32(val.Float()))
	case reflect.Float64:
		return wasmer.NewF64(float64(val.Float()))
	}

	return wasmer.Value{}
}
