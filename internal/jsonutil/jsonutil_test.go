package jsonutil

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Name  string   `json:"name"`
	Value int      `json:"value"`
	Tags  []string `json:"tags"`
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "simple struct",
			input: testStruct{Name: "test", Value: 42, Tags: []string{"a", "b"}},
			want:  `{"name":"test","value":42,"tags":["a","b"]}`,
		},
		{
			name:  "string slice",
			input: []string{"one", "two", "three"},
			want:  `["one","two","three"]`,
		},
		{
			name:  "map",
			input: map[string]int{"a": 1, "b": 2},
			want:  `{"a":1,"b":2}`,
		},
		{
			name:  "nil value",
			input: nil,
			want:  "null",
		},
		{
			name:  "empty struct",
			input: testStruct{},
			want:  `{"name":"","value":0,"tags":null}`,
		},
		{
			name: "complex nested structure",
			input: map[string]interface{}{
				"struct": testStruct{Name: "nested", Value: 100},
				"array":  []int{1, 2, 3},
				"bool":   true,
			},
			want: `{"array":[1,2,3],"bool":true,"struct":{"name":"nested","value":100,"tags":null}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalJSON(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// For maps, we need to handle potential key ordering differences
			if strings.Contains(tt.want, `"a":`) && strings.Contains(tt.want, `"b":`) {
				var gotMap, wantMap map[string]interface{}
				require.NoError(t, json.Unmarshal(got, &gotMap))
				require.NoError(t, json.Unmarshal([]byte(tt.want), &wantMap))
				assert.Equal(t, wantMap, gotMap)
			} else {
				assert.JSONEq(t, tt.want, string(got))
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		target  interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name:   "unmarshal to struct",
			input:  `{"name":"test","value":42,"tags":["a","b"]}`,
			target: testStruct{},
			want:   testStruct{Name: "test", Value: 42, Tags: []string{"a", "b"}},
		},
		{
			name:   "unmarshal to slice",
			input:  `["one","two","three"]`,
			target: []string{},
			want:   []string{"one", "two", "three"},
		},
		{
			name:   "unmarshal to map",
			input:  `{"a":1,"b":2}`,
			target: map[string]int{},
			want:   map[string]int{"a": 1, "b": 2},
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json`,
			target:  testStruct{},
			wantErr: true,
		},
		{
			name:    "type mismatch",
			input:   `{"name":"test"}`,
			target:  []string{},
			wantErr: true,
		},
		{
			name:   "null value",
			input:  `null`,
			target: &testStruct{},
			want:   (*testStruct)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch target := tt.target.(type) {
			case testStruct:
				got, err := UnmarshalJSON[testStruct]([]byte(tt.input))
				if tt.wantErr {
					require.Error(t, err)
					assert.Contains(t, err.Error(), "unmarshal JSON")
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			case []string:
				got, err := UnmarshalJSON[[]string]([]byte(tt.input))
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			case map[string]int:
				got, err := UnmarshalJSON[map[string]int]([]byte(tt.input))
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			case *testStruct:
				got, err := UnmarshalJSON[*testStruct]([]byte(tt.input))
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			default:
				t.Fatalf("unexpected target type: %T", target)
			}
		})
	}
}

func TestGenerateTestJSON(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		template interface{}
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "generate struct array",
			count:    3,
			template: testStruct{Name: "template", Value: 10},
			wantLen:  3,
		},
		{
			name:     "generate string array",
			count:    5,
			template: "test",
			wantLen:  5,
		},
		{
			name:     "generate empty array",
			count:    0,
			template: "test",
			wantLen:  0,
		},
		{
			name:     "negative count",
			count:    -1,
			template: "test",
			wantErr:  true,
		},
		{
			name:     "generate map array",
			count:    2,
			template: map[string]int{"key": 123},
			wantLen:  2,
		},
		{
			name:     "large count",
			count:    1000,
			template: "item",
			wantLen:  1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateTestJSON(tt.count, tt.template)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Unmarshal to verify the content
			var result []interface{}
			require.NoError(t, json.Unmarshal(got, &result))
			assert.Len(t, result, tt.wantLen)

			// Verify each item matches the template
			if tt.wantLen > 0 {
				// Check first and last items
				switch tmpl := tt.template.(type) {
				case string:
					assert.Equal(t, tmpl, result[0])
					assert.Equal(t, tmpl, result[tt.wantLen-1])
				case testStruct:
					// For structs, JSON unmarshals to map[string]interface{}
					firstItem := result[0].(map[string]interface{})
					assert.Equal(t, tmpl.Name, firstItem["name"])
					assert.InEpsilon(t, float64(tmpl.Value), firstItem["value"], 0.0001)
				}
			}
		})
	}
}

func TestPrettyPrint(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name: "simple struct",
			input: testStruct{
				Name:  "pretty",
				Value: 42,
				Tags:  []string{"tag1", "tag2"},
			},
			want: `{
  "name": "pretty",
  "value": 42,
  "tags": [
    "tag1",
    "tag2"
  ]
}`,
		},
		{
			name: "nested structure",
			input: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "value",
					"array":  []int{1, 2, 3},
				},
			},
			want: `{
  "level1": {
    "array": [
      1,
      2,
      3
    ],
    "level2": "value"
  }
}`,
		},
		{
			name:  "simple array",
			input: []string{"a", "b", "c"},
			want: `[
  "a",
  "b",
  "c"
]`,
		},
		{
			name:  "nil value",
			input: nil,
			want:  "null",
		},
		{
			name:  "empty object",
			input: map[string]interface{}{},
			want:  "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PrettyPrint(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Normalize the comparison by parsing both as JSON
			if strings.Contains(tt.want, `"array"`) && strings.Contains(tt.want, `"level2"`) {
				// For maps with potential key ordering issues
				var gotObj, wantObj interface{}
				require.NoError(t, json.Unmarshal([]byte(got), &gotObj))
				require.NoError(t, json.Unmarshal([]byte(tt.want), &wantObj))
				assert.Equal(t, wantObj, gotObj)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCompactJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name: "compact pretty printed JSON",
			input: `{
  "name": "test",
  "value": 42,
  "tags": [
    "a",
    "b"
  ]
}`,
			want: `{"name":"test","tags":["a","b"],"value":42}`,
		},
		{
			name:  "already compact",
			input: `{"a":1,"b":2}`,
			want:  `{"a":1,"b":2}`,
		},
		{
			name:  "remove extra whitespace",
			input: `{  "key"  :  "value"  ,  "num"  :  123  }`,
			want:  `{"key":"value","num":123}`,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}`,
			wantErr: true,
		},
		{
			name:  "array with whitespace",
			input: `[ 1 , 2 , 3 ]`,
			want:  `[1,2,3]`,
		},
		{
			name:  "empty object",
			input: `{ }`,
			want:  `{}`,
		},
		{
			name:  "null value",
			input: ` null `,
			want:  `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompactJSON([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "parse JSON for compaction")
				return
			}
			require.NoError(t, err)

			// For objects, parse and compare to handle key ordering
			if strings.HasPrefix(tt.want, "{") && strings.HasSuffix(tt.want, "}") {
				assert.JSONEq(t, tt.want, string(got))
			} else {
				assert.Equal(t, tt.want, string(got))
			}
		})
	}
}

func TestMergeJSON(t *testing.T) {
	tests := []struct {
		name    string
		jsons   []string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "merge two objects",
			jsons: []string{
				`{"a":1,"b":2}`,
				`{"c":3,"d":4}`,
			},
			want: `{"a":1,"b":2,"c":3,"d":4}`,
		},
		{
			name: "override values",
			jsons: []string{
				`{"a":1,"b":2}`,
				`{"b":3,"c":4}`,
			},
			want: `{"a":1,"b":3,"c":4}`,
		},
		{
			name: "merge multiple objects",
			jsons: []string{
				`{"a":1}`,
				`{"b":2}`,
				`{"c":3}`,
				`{"a":10}`,
			},
			want: `{"a":10,"b":2,"c":3}`,
		},
		{
			name:  "empty merge",
			jsons: []string{},
			want:  `{}`,
		},
		{
			name: "merge with empty object",
			jsons: []string{
				`{"a":1}`,
				`{}`,
				`{"b":2}`,
			},
			want: `{"a":1,"b":2}`,
		},
		{
			name: "invalid JSON",
			jsons: []string{
				`{"a":1}`,
				`{invalid}`,
			},
			wantErr: true,
			errMsg:  "failed to unmarshal JSON at index 1",
		},
		{
			name: "non-object JSON",
			jsons: []string{
				`{"a":1}`,
				`[1,2,3]`,
			},
			wantErr: true,
			errMsg:  "failed to unmarshal JSON at index 1",
		},
		{
			name: "merge with nested objects",
			jsons: []string{
				`{"user":{"name":"John","age":30}}`,
				`{"user":{"city":"NYC"},"active":true}`,
			},
			want: `{"active":true,"user":{"city":"NYC"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert string slices to []byte slices
			jsonBytes := make([][]byte, len(tt.jsons))
			for i, j := range tt.jsons {
				jsonBytes[i] = []byte(j)
			}

			got, err := MergeJSON(jsonBytes...)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}

// Benchmark tests
func BenchmarkMarshalJSON(b *testing.B) {
	data := testStruct{
		Name:  "benchmark",
		Value: 12345,
		Tags:  []string{"tag1", "tag2", "tag3", "tag4", "tag5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalJSON(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	data := []byte(`{"name":"benchmark","value":12345,"tags":["tag1","tag2","tag3","tag4","tag5"]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := UnmarshalJSON[testStruct](data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateTestJSON(b *testing.B) {
	template := map[string]interface{}{
		"id":     123,
		"name":   "test",
		"active": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateTestJSON(100, template)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestMarshalJSONErrorCases tests additional error scenarios for MarshalJSON
func TestMarshalJSONErrorCases(t *testing.T) {
	// Create a type that fails to marshal
	type unmarshalableType struct {
		Ch chan int `json:"ch"`
	}

	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "unmarshalable type with channel",
			input:   unmarshalableType{Ch: make(chan int)},
			wantErr: true,
		},
		{
			name:    "function pointer",
			input:   func() {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MarshalJSON(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "marshal to JSON")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestPrettyPrintErrorCases tests error scenarios for PrettyPrint
func TestPrettyPrintErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "unmarshalable type with function",
			input:   func() {},
			wantErr: true,
		},
		{
			name:    "unmarshalable type with channel",
			input:   struct{ Ch chan int }{Ch: make(chan int)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PrettyPrint(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "pretty print JSON")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCompactJSONErrorCases tests additional error scenarios for CompactJSON
func TestCompactJSONErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "completely malformed JSON",
			input:   `this is not json at all`,
			wantErr: true,
			errMsg:  "parse JSON for compaction",
		},
		{
			name:    "truncated JSON",
			input:   `{"key":"val`,
			wantErr: true,
			errMsg:  "parse JSON for compaction",
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
			errMsg:  "parse JSON for compaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CompactJSON([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGenerateTestJSONMarshalError tests marshal error in GenerateTestJSON
func TestGenerateTestJSONMarshalError(t *testing.T) {
	// Create a template that will fail to marshal
	template := make(chan int)

	_, err := GenerateTestJSON(1, template)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generate test JSON")
}

// TestMergeJSONMarshalError tests marshal error in MergeJSON
func TestMergeJSONMarshalError(t *testing.T) {
	// We can't easily trigger the marshal error in MergeJSON since it marshals
	// a simple map[string]interface{}, but we can test with valid input to ensure
	// no marshal errors occur
	jsons := [][]byte{
		[]byte(`{"a":1}`),
		[]byte(`{"b":2}`),
	}

	result, err := MergeJSON(jsons...)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1,"b":2}`, string(result))
}
