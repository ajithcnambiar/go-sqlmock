package sqlmock

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestQueryExpectationArgComparison(t *testing.T) {
	e := &queryBasedExpectation{converter: driver.DefaultParameterConverter}
	against := []namedValue{{Value: int64(5), Ordinal: 1}}
	if err := e.argsMatches(against); err != nil {
		t.Errorf("arguments should match, since the no expectation was set, but got err: %s", err)
	}

	e.args = []driver.Value{5, "str"}

	against = []namedValue{{Value: int64(5), Ordinal: 1}}
	if err := e.argsMatches(against); err == nil {
		t.Error("arguments should not match, since the size is not the same")
	}

	against = []namedValue{
		{Value: int64(3), Ordinal: 1},
		{Value: "str", Ordinal: 2},
	}
	if err := e.argsMatches(against); err == nil {
		t.Error("arguments should not match, since the first argument (int value) is different")
	}

	against = []namedValue{
		{Value: int64(5), Ordinal: 1},
		{Value: "st", Ordinal: 2},
	}
	if err := e.argsMatches(against); err == nil {
		t.Error("arguments should not match, since the second argument (string value) is different")
	}

	against = []namedValue{
		{Value: int64(5), Ordinal: 1},
		{Value: "str", Ordinal: 2},
	}
	if err := e.argsMatches(against); err != nil {
		t.Errorf("arguments should match, but it did not: %s", err)
	}

	const longForm = "Jan 2, 2006 at 3:04pm (MST)"
	tm, _ := time.Parse(longForm, "Feb 3, 2013 at 7:54pm (PST)")
	e.args = []driver.Value{5, tm}

	against = []namedValue{
		{Value: int64(5), Ordinal: 1},
		{Value: tm, Ordinal: 2},
	}
	if err := e.argsMatches(against); err != nil {
		t.Error("arguments should match, but it did not")
	}

	e.args = []driver.Value{5, AnyArg()}
	if err := e.argsMatches(against); err != nil {
		t.Errorf("arguments should match, but it did not: %s", err)
	}
}

func TestQueryExpectationArgComparisonBool(t *testing.T) {
	var e *queryBasedExpectation

	e = &queryBasedExpectation{args: []driver.Value{true}, converter: driver.DefaultParameterConverter}
	against := []namedValue{
		{Value: true, Ordinal: 1},
	}
	if err := e.argsMatches(against); err != nil {
		t.Error("arguments should match, since arguments are the same")
	}

	e = &queryBasedExpectation{args: []driver.Value{false}, converter: driver.DefaultParameterConverter}
	against = []namedValue{
		{Value: false, Ordinal: 1},
	}
	if err := e.argsMatches(against); err != nil {
		t.Error("arguments should match, since argument are the same")
	}

	e = &queryBasedExpectation{args: []driver.Value{true}, converter: driver.DefaultParameterConverter}
	against = []namedValue{
		{Value: false, Ordinal: 1},
	}
	if err := e.argsMatches(against); err == nil {
		t.Error("arguments should not match, since argument is different")
	}

	e = &queryBasedExpectation{args: []driver.Value{false}, converter: driver.DefaultParameterConverter}
	against = []namedValue{
		{Value: true, Ordinal: 1},
	}
	if err := e.argsMatches(against); err == nil {
		t.Error("arguments should not match, since argument is different")
	}
}

func ExampleExpectedExec() {
	db, mock, _ := New()
	result := NewErrorResult(fmt.Errorf("some error"))
	mock.ExpectExec("^INSERT (.+)").WillReturnResult(result)
	res, _ := db.Exec("INSERT something")
	_, err := res.LastInsertId()
	fmt.Println(err)
	// Output: some error
}

func TestBuildQuery(t *testing.T) {
	db, mock, _ := New()
	query := `
		SELECT
			name,
			email,
			address,
			anotherfield
		FROM user
		where
			name    = 'John'
			and
			address = 'Jakarta'

	`

	mock.ExpectQuery(query)
	mock.ExpectExec(query)
	mock.ExpectPrepare(query)

	db.QueryRow(query)
	db.Exec(query)
	db.Prepare(query)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

type CustomConverter struct{}

func (s CustomConverter) ConvertValue(v interface{}) (driver.Value, error) {
	switch v.(type) {
	case string:
		return v.(string), nil
	case []string:
		return v.([]string), nil
	case int:
		return v.(int), nil
	default:
		return nil, errors.New(fmt.Sprintf("cannot convert %T with value %v", v, v))
	}
}
func TestCustomValueConverterQueryScan(t *testing.T) {
	db, mock, _ := New(ValueConverterOption(CustomConverter{}))
	query := `
		SELECT
			name,
			email,
			address,
			anotherfield
		FROM user
		where
			name    = 'John'
			and
			address = 'Jakarta'

	`
	expectedStringValue := "ValueOne"
	expectedIntValue := 2
	expectedArrayValue := []string{"Three", "Four"}
	mock.ExpectQuery(query).WillReturnRows(mock.NewRows([]string{"One", "Two", "Three"}).AddRow(expectedStringValue, expectedIntValue, []string{"Three", "Four"}))
	row := db.QueryRow(query)
	var stringValue string
	var intValue int
	var arrayValue []string
	if e := row.Scan(&stringValue, &intValue, &arrayValue); e != nil {
		t.Error(e)
	}
	if stringValue != expectedStringValue {
		t.Errorf("Expectation %s does not met: %s", expectedStringValue, stringValue)
	}
	if intValue != expectedIntValue {
		t.Errorf("Expectation %d does not met: %d", expectedIntValue, intValue)
	}
	if !reflect.DeepEqual(expectedArrayValue, arrayValue) {
		t.Errorf("Expectation %v does not met: %v", expectedArrayValue, arrayValue)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
