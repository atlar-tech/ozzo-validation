// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Struct1 struct {
	Field1 int
	Field2 *int
	Field3 []int
	Field4 [4]int
	field5 int
	Struct2
	S1               *Struct2
	S2               Struct2
	JSONField        int `json:"some_json_field"`
	JSONIgnoredField int `json:"-"`
}

type Struct2 struct {
	Field21 string
	Field22 string
}

type Struct3 struct {
	*Struct2
	S1 string
}

func foundStructField(v reflect.Value, field reflect.Value) bool {
	_, ok := findStructField(v, field)
	return ok
}

func TestFindStructField(t *testing.T) {
	var s1 Struct1
	v1 := reflect.ValueOf(&s1).Elem()
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.Field1)))
	assert.False(t, foundStructField(v1, reflect.ValueOf(s1.Field2)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.Field2)))
	assert.False(t, foundStructField(v1, reflect.ValueOf(s1.Field3)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.Field3)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.Field4)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.field5)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.Struct2)))
	assert.False(t, foundStructField(v1, reflect.ValueOf(s1.S1)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.S1)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.Field21)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.Field22)))
	assert.True(t, foundStructField(v1, reflect.ValueOf(&s1.Struct2.Field22)))
	s2 := reflect.ValueOf(&s1.Struct2).Elem()
	assert.True(t, foundStructField(s2, reflect.ValueOf(&s1.Field21)))
	assert.True(t, foundStructField(s2, reflect.ValueOf(&s1.Struct2.Field21)))
	assert.True(t, foundStructField(s2, reflect.ValueOf(&s1.Struct2.Field22)))
	s3 := Struct3{
		Struct2: &Struct2{},
	}
	v3 := reflect.ValueOf(&s3).Elem()
	assert.True(t, foundStructField(v3, reflect.ValueOf(&s3.Struct2)))
	assert.True(t, foundStructField(v3, reflect.ValueOf(&s3.Field21)))
}

func TestValidateStruct(t *testing.T) {
	var m0 *Model1
	m1 := Model1{A: "abc", B: "xyz", c: "abc", G: "xyz", H: []string{"abc", "abc"}, I: map[string]string{"foo": "abc"}}
	m2 := Model1{E: String123("xyz")}
	m3 := Model2{}
	m4 := Model2{M3: Model3{A: "abc"}, Model3: Model3{A: "abc"}}
	m5 := Model2{Model3: Model3{A: "internal"}}
	tests := []struct {
		tag   string
		model interface{}
		rules []*FieldRules
		err   string
	}{
		// empty rules
		{"t1.1", &m1, []*FieldRules{}, ""},
		{"t1.2", &m1, []*FieldRules{Field(&m1.A), Field(&m1.B)}, ""},
		// normal rules
		{"t2.1", &m1, []*FieldRules{Field(&m1.A, &validateAbc{}), Field(&m1.B, &validateXyz{})}, ""},
		{"t2.2", &m1, []*FieldRules{Field(&m1.A, &validateXyz{}), Field(&m1.B, &validateAbc{})}, "A: error xyz; B: error abc."},
		{"t2.3", &m1, []*FieldRules{Field(&m1.A, &validateXyz{}), Field(&m1.c, &validateXyz{})}, "A: error xyz; c: error xyz."},
		{"t2.4", &m1, []*FieldRules{Field(&m1.D, Length(0, 5))}, ""},
		{"t2.5", &m1, []*FieldRules{Field(&m1.F, Length(0, 5))}, ""},
		{"t2.6", &m1, []*FieldRules{Field(&m1.H, Each(&validateAbc{})), Field(&m1.I, Each(&validateAbc{}))}, ""},
		{"t2.7", &m1, []*FieldRules{Field(&m1.H, Each(&validateXyz{})), Field(&m1.I, Each(&validateXyz{}))}, "H: (0: error xyz; 1: error xyz.); I: (foo: error xyz.)."},
		// non-struct pointer
		{"t3.1", m1, []*FieldRules{}, ErrStructPointer.Error()},
		{"t3.2", nil, []*FieldRules{}, ErrStructPointer.Error()},
		{"t3.3", m0, []*FieldRules{}, ""},
		{"t3.4", &m0, []*FieldRules{}, ErrStructPointer.Error()},
		// invalid field spec
		{"t4.1", &m1, []*FieldRules{Field(m1)}, ErrFieldPointer(0).Error()},
		{"t4.2", &m1, []*FieldRules{Field(&m1)}, ErrFieldNotFound(0).Error()},
		// struct tag
		{"t5.1", &m1, []*FieldRules{Field(&m1.G, &validateAbc{})}, "g: error abc."},
		// validatable field
		{"t6.1", &m2, []*FieldRules{Field(&m2.E)}, "E: error 123."},
		{"t6.2", &m2, []*FieldRules{Field(&m2.E, Skip)}, ""},
		{"t6.3", &m2, []*FieldRules{Field(&m2.E, Skip.When(true))}, ""},
		{"t6.4", &m2, []*FieldRules{Field(&m2.E, Skip.When(false))}, "E: error 123."},
		// Required, NotNil
		{"t7.1", &m2, []*FieldRules{Field(&m2.F, Required)}, "F: cannot be blank."},
		{"t7.2", &m2, []*FieldRules{Field(&m2.F, NotNil)}, "F: is required."},
		{"t7.3", &m2, []*FieldRules{Field(&m2.F, Skip, Required)}, ""},
		{"t7.4", &m2, []*FieldRules{Field(&m2.F, Skip, NotNil)}, ""},
		{"t7.5", &m2, []*FieldRules{Field(&m2.F, Skip.When(true), Required)}, ""},
		{"t7.6", &m2, []*FieldRules{Field(&m2.F, Skip.When(true), NotNil)}, ""},
		{"t7.7", &m2, []*FieldRules{Field(&m2.F, Skip.When(false), Required)}, "F: cannot be blank."},
		{"t7.8", &m2, []*FieldRules{Field(&m2.F, Skip.When(false), NotNil)}, "F: is required."},
		// embedded structs
		{"t8.1", &m3, []*FieldRules{Field(&m3.M3, Skip)}, ""},
		{"t8.2", &m3, []*FieldRules{Field(&m3.M3)}, "M3: (A: error abc.)."},
		{"t8.3", &m3, []*FieldRules{Field(&m3.Model3, Skip)}, ""},
		{"t8.4", &m3, []*FieldRules{Field(&m3.Model3)}, "A: error abc."},
		{"t8.5", &m4, []*FieldRules{Field(&m4.M3)}, ""},
		{"t8.6", &m4, []*FieldRules{Field(&m4.Model3)}, ""},
		{"t8.7", &m3, []*FieldRules{Field(&m3.A, Required), Field(&m3.B, Required)}, "A: cannot be blank; B: cannot be blank."},
		{"t8.8", &m3, []*FieldRules{Field(&m4.A, Required)}, "field #0 cannot be found in the struct"},
		// internal error
		{"t9.1", &m5, []*FieldRules{Field(&m5.A, &validateAbc{}), Field(&m5.B, Required), Field(&m5.A, &validateInternalError{})}, "error internal"},
	}
	for _, test := range tests {
		err1 := ValidateStruct(test.model, test.rules...)
		err2 := ValidateStructWithContext(context.Background(), test.model, test.rules...)
		assertError(t, test.err, err1, test.tag)
		assertError(t, test.err, err2, test.tag)
	}

	// embedded struct
	err := Validate(&m3)
	assert.EqualError(t, err, "A: error abc.")

	a := struct {
		Name  string
		Value string
	}{"name", "demo"}
	err = ValidateStruct(&a,
		Field(&a.Name, Required),
		Field(&a.Value, Required, Length(5, 10)),
	)
	assert.EqualError(t, err, "Value: the length must be between 5 and 10.")
}

func TestValidateStructWithContext(t *testing.T) {
	m1 := Model1{A: "abc", B: "xyz", c: "abc", G: "xyz"}
	m2 := Model2{Model3: Model3{A: "internal"}}
	m3 := Model5{}
	tests := []struct {
		tag   string
		model interface{}
		rules []*FieldRules
		err   string
	}{
		// normal rules
		{"t1.1", &m1, []*FieldRules{Field(&m1.A, &validateContextAbc{}), Field(&m1.B, &validateContextXyz{})}, ""},
		{"t1.2", &m1, []*FieldRules{Field(&m1.A, &validateContextXyz{}), Field(&m1.B, &validateContextAbc{})}, "A: error xyz; B: error abc."},
		{"t1.3", &m1, []*FieldRules{Field(&m1.A, &validateContextXyz{}), Field(&m1.c, &validateContextXyz{})}, "A: error xyz; c: error xyz."},
		{"t1.4", &m1, []*FieldRules{Field(&m1.G, &validateContextAbc{})}, "g: error abc."},
		// skip rule
		{"t2.1", &m1, []*FieldRules{Field(&m1.G, Skip, &validateContextAbc{})}, ""},
		{"t2.2", &m1, []*FieldRules{Field(&m1.G, &validateContextAbc{}, Skip)}, "g: error abc."},
		// internal error
		{"t3.1", &m2, []*FieldRules{Field(&m2.A, &validateContextAbc{}), Field(&m2.B, Required), Field(&m2.A, &validateInternalError{})}, "error internal"},
	}
	for _, test := range tests {
		err := ValidateStructWithContext(context.Background(), test.model, test.rules...)
		assertError(t, test.err, err, test.tag)
	}

	//embedded struct
	err := ValidateWithContext(context.Background(), &m3)
	if assert.NotNil(t, err) {
		assert.Equal(t, "A: error abc.", err.Error())
	}

	a := struct {
		Name  string
		Value string
	}{"name", "demo"}
	err = ValidateStructWithContext(context.Background(), &a,
		Field(&a.Name, Required),
		Field(&a.Value, Required, Length(5, 10)),
	)
	if assert.NotNil(t, err) {
		assert.Equal(t, "Value: the length must be between 5 and 10.", err.Error())
	}
}

func Test_getErrorFieldName(t *testing.T) {
	var s1 Struct1
	v1 := reflect.ValueOf(&s1).Elem()

	sf1, ok := findStructField(v1, reflect.ValueOf(&s1.Field1))
	assert.True(t, ok)
	assert.Equal(t, "Field1", getErrorFieldName(sf1))

	jsonField, ok := findStructField(v1, reflect.ValueOf(&s1.JSONField))
	assert.True(t, ok)
	assert.Equal(t, "some_json_field", getErrorFieldName(jsonField))

	jsonIgnoredField, ok := findStructField(v1, reflect.ValueOf(&s1.JSONIgnoredField))
	assert.True(t, ok)
	assert.Equal(t, "JSONIgnoredField", getErrorFieldName(jsonIgnoredField))
}
