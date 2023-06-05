package utils

import (
	"reflect"
	"testing"
)

func TestRemoveDuplicates(t *testing.T) {
	arr := []string{"cat", "dog", "bird", "cat", "bird"}
	expected := []string{"cat", "dog", "bird"}
	result := RemoveDuplicates(arr)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "orange"}
	if !Contains("banana", slice) {
		t.Errorf("Expected to find banana in slice, but didn't")
	}

	if Contains("grape", slice) {
		t.Errorf("Did not expect to find grape in slice, but did")
	}

	m := map[string]int{"a": 1, "b": 2, "c": 3}
	if !Contains("b", m) {
		t.Errorf("Expected to find key b in map, but didn't")
	}

	if Contains("d", m) {
		t.Errorf("Did not expect to find key d in map, but did")
	}
}
