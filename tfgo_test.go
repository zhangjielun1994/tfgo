/*
Copyright 2017 Paolo Galeone. All right reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tfgo_test

import (
	tg "github.com/galeone/tfgo"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"math"
	"reflect"
	"testing"
)

func TestNewScope(t *testing.T) {
	root := tg.NewRoot()
	scope := tg.NewScope(root)
	if scope == nil {
		t.Error("NewScope shouldn't return nil")
	}
}

func TestTensor(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code panic, but it shouldn't: %v", r)
		}
	}()

	root := tg.NewRoot()
	tensorA := tg.NewTensor(root, tg.Const(root, [3]int32{1, 2, 3}))
	if tensorA == nil {
		t.Fatal("NewTensor shouldn't return nil")
	}
	// shouldn't panic
	tensorA.Check()
	tensorB := tg.NewTensor(root, tg.Const(root, [3]int32{1, 2, 3}))

	// For not changing the content of A
	// Create a new tensor with the same content of A
	// on every invocation.
	// Change the content on the fly is useful when used chaning the operations

	add := tensorA.Clone().Add(tensorB.Output).Output
	mul := tensorA.Clone().Mul(tensorB.Output).Output
	// types must be always well defined
	// never use a number, e.i. 2, but force a type e.i. int32(2)
	pow := tensorA.Clone().Pow(tg.Const(root, int32(2))).Output
	sqrt := tensorA.Clone().Sqrt().Output
	square := tensorA.Clone().Square().Output
	shape32 := tensorA.Clone().Shape32(true)
	shape64 := tensorA.Clone().Shape64(true)
	if len(shape32) != len(shape64) {
		t.Errorf("Expected len(shape32) = len(shape64), but got: %v != %v", len(shape32), len(shape64))
	}
	// remove first dim
	shape32 = tensorA.Clone().Shape32(false)
	shape64 = tensorA.Clone().Shape64(false)
	if len(shape32) != len(shape64) {
		t.Errorf("Expected len(shape32) = len(shape64), but got: %v != %v", len(shape32), len(shape64))
	}

	matA := tg.NewTensor(root, tg.Const(root, [2][2]int32{{1, 2}, {-1, -2}}))
	matB := tg.NewTensor(root, tg.Const(root, [2][1]int32{{10}, {100}}))
	// chain op without clone, matA now is matmul result
	matA = matA.MatMul(matB.Output)

	result := tg.Exec(root, []tf.Output{add, mul, pow, sqrt, square, matA.Output}, nil, nil)
	if result[0].Value().([]int32)[0] != 2 {
		t.Errorf("Expected 2 as first value in sum, but got: %v", result[0].Value().([]int32)[0])
	}

	if result[1].Value().([]int32)[0] != 1 {
		t.Errorf("Expected 1 as first value in mul, but got: %v", result[1].Value().([]int32))
	}
	if result[2].Value().([]int32)[0] != 1 {
		t.Errorf("Expected 1 as first value in pow, but got: %v", result[2].Value().([]int32)[0])
	}

	if result[3].Value().([]int32)[0] != 1 {
		t.Errorf("Expected 1 as first value in sqrt, but got: %v", result[3].Value().([]int32)[0])
	}
	if result[4].Value().([]int32)[0] != result[2].Value().([]int32)[0] {
		t.Errorf("Expected output of square being equal to tensor² but got: %v vs %v", result[4].Value().([]int32), result[2].Value().([]int32))
	}

	if result[5].Value().([][]int32)[0][0] != 210 {
		t.Errorf("Expected output of matmul in pos 0,0 should be 210, but got: %v", result[5].Value().([][]int32))
	}
}

func TestTensorPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	root := tg.NewRoot()
	tensorA := tg.NewTensor(root, tg.Const(root, [3]int32{1, 2, 3}))
	if tensorA == nil {
		t.Fatal("NewTensor shouldn't return nil")
	}
	// shouldn't panic
	tensorA.Check()
	tensorB := tg.NewTensor(root, tg.Const(root, [3]int32{1, 2, 3}))

	add := tensorA.Add(tensorB.Output)
	result := tg.Exec(root, []tf.Output{add.Output}, nil, nil)
	if result[0].Value().([]int32)[0] != 2 {
		t.Errorf("Expected 2 as first value in sum, but got: %v", result[0].Value().([]int32)[0])
	}
	// After the Exec operation, everything should panic because the graph has been finalized
	// and the graph, thus, has been built and it's unmodifiable
	tensorA = tensorA.Cast(tf.Float)
	if tensorA == nil {
		t.Error("Cast operation shouldn't return nil")
	}

}

func TestBatchify(t *testing.T) {
	root := tg.NewRoot()
	var tensors []tf.Output
	for i := 0; i < 10; i++ {
		tensors = append(tensors, tg.Const(root, [3]int32{1, 2, 3}))
	}
	batch := tg.Batchify(root, tensors)

	if batch.Shape().NumDimensions() != 2 {
		t.Errorf("Expected 2D tensor, but got: %dD tensor", batch.Shape().NumDimensions())
	}

	shape, _ := batch.Shape().ToSlice()
	if shape[0] != 10 || shape[1] != 3 {
		t.Errorf("Expected shape (10,3), got  (%d,%d)", shape[0], shape[1])
	}

	result := tg.Exec(root, []tf.Output{batch}, nil, nil)
	// Note the cast to [][] and not to [10][3]
	matrixResult := result[0].Value().([][]int32)
	var expectedMatrix [][]int32
	row := []int32{1, 2, 3}
	for i := 0; i < 10; i++ {
		expectedMatrix = append(expectedMatrix, row)
	}
	if !reflect.DeepEqual(matrixResult, expectedMatrix) {
		t.Errorf("Expected matrix %v\n Got matrix %v", expectedMatrix, matrixResult)
	}
}

func TestIsClose(t *testing.T) {
	root := tg.NewRoot()
	A := tg.Const(root, []float32{0.1, 0.2, 0.3, 1e-1, 1e-2, 1e-3, 1e-4, 1e-6, 5e-5})
	B := tg.Const(root, []float32{0.11, 0.2, 0.299, 0, 2e-2, 2e-3, 2e-4, 0, 10})
	relTol := tg.Const(root, float32(1e-3))
	absTol := tg.Const(root, float32(1e-6))
	isClose := tg.IsClose(root, A, B, relTol, absTol)

	expected := []bool{false, true, false, false, false, false, false, true, false}
	results := tg.Exec(root, []tf.Output{isClose}, nil, nil)
	result := results[0].Value().([]bool)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected  %v\n Got  %v", expected, result)
	}
}

func TestPanicModelRestore(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	// Panics because the tag does not exist
	tg.LoadModel("test_models/export", []string{"tagwat"}, nil)
}

func TestPanicModel(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	model := tg.LoadModel("test_models/export", []string{"tag"}, nil)

	// Should panic
	model.Op("does not exists", 0)
	// Esists but wroing output number (1 instead of 0)
	model.Op("LeNetDropout/softmax_linear/Identity", 1)
}

func TestLoadModel(t *testing.T) {
	model := tg.LoadModel("test_models/export", []string{"tag"}, nil)

	fakeInput, _ := tf.NewTensor([1][28][28][1]float32{})
	results := model.Exec([]tf.Output{
		model.Op("LeNetDropout/softmax_linear/Identity", 0),
	}, map[tf.Output]*tf.Tensor{
		model.Op("input_", 0): fakeInput,
	})

	if results[0].Shape()[0] != 1 || results[0].Shape()[1] != 10 {
		t.Errorf("Expected output shape of [1,10], got %v", results[0].Shape())
	}
}

func TestIsIntegerFloat(t *testing.T) {
	root := tg.NewRoot()
	B := tg.Const(root, []float32{0.11})

	if tg.IsInteger(B.DataType()) {
		t.Error("Expected a float type, but integer found")
	}
	if !tg.IsFloat(B.DataType()) {
		t.Error("Expected a float type, but float32 has been considered not float")
	}
}

func TestMaxMinValue(t *testing.T) {
	root := tg.NewRoot()
	A := tg.Const(root, int64(0))
	B := tg.Const(root, float64(0))

	if tg.MaxValue(A.DataType()) != math.MaxInt64 {
		t.Errorf("expected MaxValue of dype int64 to be equal to math.MaxInt64, but got %v", tg.MaxValue(A.DataType()))
	}

	if tg.MaxValue(B.DataType()) != math.MaxFloat64 {
		t.Errorf("expected MaxValue of dype float64 to be equal to math.MaxFloat64 but got %v", tg.MaxValue(B.DataType()))
	}

	if tg.MinValue(A.DataType()) != math.MinInt64 {
		t.Errorf("expected MinValue of dype int64 to be equal to math.MinInt64, but got %v", tg.MinValue(A.DataType()))
	}

	if tg.MinValue(B.DataType()) != math.SmallestNonzeroFloat64 {
		t.Errorf("expected MinValue of dype float64 to be equal to math.SmallestNonzeroFloat64 but got %v", tg.MinValue(B.DataType()))
	}
}
