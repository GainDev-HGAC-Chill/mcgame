package valuecopy

import "testing"

func TestCopyPreservesTypesAndDetachesNestedContainers(t *testing.T) {
	source := map[string]interface{}{"n": int64(4130220000285), "nested": []interface{}{map[int]int{1: 7}}, "ints": []int{1, 2}}
	copy := Copy(source)
	copy["nested"].([]interface{})[0].(map[int]int)[1] = 99
	copy["ints"].([]int)[0] = 99
	if source["nested"].([]interface{})[0].(map[int]int)[1] != 7 || source["ints"].([]int)[0] != 1 || copy["n"] != int64(4130220000285) {
		t.Fatal("copy leaked containers or coerced numbers")
	}
	if Copy[any](nil) != nil || Copy[map[string]any](nil) != nil || Copy[[]int](nil) != nil {
		t.Fatal("nil values must stay nil")
	}
}
