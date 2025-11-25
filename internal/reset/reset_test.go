package reset

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResetParam_GenResetAction_Map(t *testing.T) {
	param := &ResetParam{
		FieldName: "testMap",
		FieldType: "map[string]int",
		MapFlag:   true,
	}

	param.GenResetAction()

	expected := "clear(rs.testMap)\n"
	assert.Equal(t, expected, param.ResetAction)
}

func TestResetParam_GenResetAction_Array(t *testing.T) {
	param := &ResetParam{
		FieldName: "testSlice",
		FieldType: "[]int",
		ArrayFlag: true,
	}

	param.GenResetAction()

	expected := "rs.testSlice = rs.testSlice[:0]\n"
	assert.Equal(t, expected, param.ResetAction)
}

func TestResetParam_GenResetAction_ChildStruct(t *testing.T) {
	param := &ResetParam{
		FieldName:       "child",
		FieldType:       "*ResetableStruct",
		PointerFlag:     true,
		ChildStructFlag: true,
	}

	param.GenResetAction()

	expected := `var rsInterface interface{} = ResetableStruct{}
	if resetter, ok := rsInterface.(interface{ Reset() }); ok && rs.child != nil {
		resetter.Reset()
	}`
	assert.Equal(t, expected, param.ResetAction)
}

func TestResetParam_GenResetAction_Bool(t *testing.T) {
	param := &ResetParam{
		FieldName: "testBool",
		FieldType: "bool",
	}

	param.GenResetAction()

	expected := "rs.testBool = false\n"
	assert.Equal(t, expected, param.ResetAction)
}

func TestResetParam_GenResetAction_PointerBool(t *testing.T) {
	param := &ResetParam{
		FieldName:   "testBoolPtr",
		FieldType:   "*bool",
		PointerFlag: true,
	}

	param.GenResetAction()

	assert.True(t, strings.Contains(param.ResetAction, "*rs.testBoolPtr = false"))
	assert.True(t, strings.Contains(param.ResetAction, "if rs.testBoolPtr != nil"))
}

func TestResetParam_GenResetAction_String(t *testing.T) {
	param := &ResetParam{
		FieldName: "testString",
		FieldType: "string",
	}

	param.GenResetAction()

	expected := "rs.testString = \"\"\n"
	assert.Equal(t, expected, param.ResetAction)
}

func TestResetParam_GenResetAction_PointerString(t *testing.T) {
	param := &ResetParam{
		FieldName:   "testStringPtr",
		FieldType:   "*string",
		PointerFlag: true,
	}

	param.GenResetAction()

	assert.True(t, strings.Contains(param.ResetAction, "*rs.testStringPtr = \"\""))
	assert.True(t, strings.Contains(param.ResetAction, "if rs.testStringPtr != nil"))
}

func TestResetParam_GenResetAction_BasicTypeNum(t *testing.T) {
	param := &ResetParam{
		FieldName: "testInt",
		FieldType: "int",
	}

	param.GenResetAction()

	expected := "rs.testInt = 0\n"
	assert.Equal(t, expected, param.ResetAction)
}

func TestResetParam_GenResetAction_PointerBasicTypeNum(t *testing.T) {
	param := &ResetParam{
		FieldName:   "testIntPtr",
		FieldType:   "*int",
		PointerFlag: true,
	}

	param.GenResetAction()

	assert.True(t, strings.Contains(param.ResetAction, "*rs.testIntPtr = 0"))
	assert.True(t, strings.Contains(param.ResetAction, "if rs.testIntPtr != nil"))
}

func TestResetParam_GenResetAction_UnsupportedType(t *testing.T) {
	param := &ResetParam{
		FieldName: "testChan",
		FieldType: "chan int",
	}

	param.GenResetAction()

	// Для неподдерживаемых типов ResetAction должен остаться пустым
	assert.Equal(t, "", param.ResetAction)
}
