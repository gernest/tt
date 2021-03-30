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

package abi

import (
	"reflect"
	"testing"
)

func TestEncodeDecodeMap(t *testing.T) {
	m := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	b := EncodeMap(m)
	if b == nil {
		t.Fatal("expected non nil value")
	}

	mm := DecodeMap(b)
	if mm == nil {
		t.Fatal("expected non nil value")
	}

	if !reflect.DeepEqual(m, mm) {
		t.Fatalf("expected %#v to equal %#v", m, mm)
	}
	var emptyMap map[string]string
	if err := EncodeMap(emptyMap); err != nil {
		t.Fatal(err)
	}
}
