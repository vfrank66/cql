// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interpreter

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/cql/model"
	"github.com/google/cql/result"
	"github.com/google/cql/types"
	"github.com/google/cql/ucum"
)

// ARITHMETIC OPERATORS - https://cql.hl7.org/09-b-cqlreference.html#arithmetic-operators-4

const (
	// Max/Min values for CQL decimals.
	// In the future we may switch to using math.MaxFloat64.
	maxDecimal = float64(99999999999999999999.99999999)
	minDecimal = float64(-99999999999999999999.99999999)
	// maxDecimalPrecision is the number of digits a CQL Decimal holds after the decimal point.
	maxDecimalPrecision = 8
)

// Abs(argument Decimal) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#abs
func evalAbsDecimal(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToFloat64(obj)
	if err != nil {
		return result.Value{}, err
	}
	return result.New(math.Abs(val))
}

// Abs(argument Integer) Integer
// https://cql.hl7.org/09-b-cqlreference.html#abs
func evalAbsInteger(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToInt32(obj)
	if err != nil {
		return result.Value{}, err
	}
	if val == math.MinInt32 {
		return result.New(nil)
	}
	if val < 0 {
		return result.New(-val)
	}
	return result.New(val)
}

// Abs(argument Long) Long
// https://cql.hl7.org/09-b-cqlreference.html#abs
func evalAbsLong(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToInt64(obj)
	if err != nil {
		return result.Value{}, err
	}
	if val == math.MinInt64 {
		return result.New(nil)
	}
	if val < 0 {
		return result.New(-val)
	}
	return result.New(val)
}

// Abs(argument Quantity) Quantity
// https://cql.hl7.org/09-b-cqlreference.html#abs
func evalAbsQuantity(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToQuantity(obj)
	if err != nil {
		return result.Value{}, err
	}
	val.Value = math.Abs(val.Value)
	return result.New(val)
}

// Ceiling(argument Decimal) Integer
// https://cql.hl7.org/09-b-cqlreference.html#ceiling
func evalCeiling(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToFloat64(obj)
	if err != nil {
		return result.Value{}, err
	}
	if val <= math.MinInt32-1 || val > math.MaxInt32 {
		return result.New(nil)
	}
	return result.New(int32(math.Ceil(val)))
}

// Exp(argument Decimal) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#exp
// Integer and long overloads are implicitly converted to decimal.
func evalExpDecimal(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToFloat64(obj)
	if err != nil {
		return result.Value{}, err
	}
	return result.New(math.Exp(val))
}

// Floor(argument Decimal) Integer
// https://cql.hl7.org/09-b-cqlreference.html#floor
func evalFloor(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToFloat64(obj)
	if err != nil {
		return result.Value{}, err
	}
	if val < math.MinInt32 || val >= math.MaxInt32+1 {
		return result.New(nil)
	}
	return result.New(int32(math.Floor(val)))
}

// Ln(argument Decimal) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#ln
func evalLn(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToFloat64(obj)
	if err != nil {
		return result.Value{}, err
	}
	if val <= 0 {
		return result.New(nil)
	}
	return result.New(math.Log(val))
}

// Log(argument Decimal) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#log
func evalLog(_ model.IBinaryExpression, lObj, rObj result.Value) (result.Value, error) {
	if result.IsNull(lObj) || result.IsNull(rObj) {
		return result.New(nil)
	}
	x, base, err := applyToValues(lObj, rObj, result.ToFloat64)
	if err != nil {
		return result.Value{}, err
	}
	val, err := log(x, base)
	if err != nil {
		return result.New(nil)
	}
	return result.New(val)
}

// log returns the logarithm of val with given base.
func log(val, base float64) (float64, error) {
	if val <= 0 || base <= 0 {
		return 0.0, fmt.Errorf("internal error - log base %v for val %v, all values	must be greater than 0", base, val)
	}
	if base == 1 {
		return 0.0, fmt.Errorf("internal error - log base %v is undefined", base)
	}
	return math.Log(val) / math.Log(base), nil
}

// op(left Integer, right Integer) Integer
// https://cql.hl7.org/09-b-cqlreference.html#add
// https://cql.hl7.org/09-b-cqlreference.html#subtract
// https://cql.hl7.org/09-b-cqlreference.html#multiply
// https://cql.hl7.org/09-b-cqlreference.html#truncated-divide
// https://cql.hl7.org/09-b-cqlreference.html#modulo
func evalArithmeticInteger(m model.IBinaryExpression, lObj, rObj result.Value) (result.Value, error) {
	if result.IsNull(lObj) || result.IsNull(rObj) {
		return result.New(nil)
	}
	l, r, err := applyToValues(lObj, rObj, result.ToInt32)
	if err != nil {
		return result.Value{}, err
	}
	return arithmetic(m, l, r)
}

// op(left Long, right Long) Long
// https://cql.hl7.org/09-b-cqlreference.html#add
// https://cql.hl7.org/09-b-cqlreference.html#subtract
// https://cql.hl7.org/09-b-cqlreference.html#multiply
// https://cql.hl7.org/09-b-cqlreference.html#truncated-divide
// https://cql.hl7.org/09-b-cqlreference.html#modulo
func evalArithmeticLong(m model.IBinaryExpression, lObj, rObj result.Value) (result.Value, error) {
	if result.IsNull(lObj) || result.IsNull(rObj) {
		return result.New(nil)
	}
	l, r, err := applyToValues(lObj, rObj, result.ToInt64)
	if err != nil {
		return result.Value{}, err
	}
	return arithmetic(m, l, r)
}

// op(left Decimal, right Decimal) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#add
// https://cql.hl7.org/09-b-cqlreference.html#subtract
// https://cql.hl7.org/09-b-cqlreference.html#multiply
// https://cql.hl7.org/09-b-cqlreference.html#truncated-divide
// https://cql.hl7.org/09-b-cqlreference.html#modulo
func evalArithmeticDecimal(m model.IBinaryExpression, lObj result.Value, rObj result.Value) (result.Value, error) {
	if result.IsNull(lObj) || result.IsNull(rObj) {
		return result.New(nil)
	}
	l, r, err := applyToValues(lObj, rObj, result.ToFloat64)
	if err != nil {
		return result.Value{}, err
	}
	return arithmetic(m, l, r)
}

// op(left Quantity, right Quantity) Quantity
// https://cql.hl7.org/09-b-cqlreference.html#add
// https://cql.hl7.org/09-b-cqlreference.html#subtract
// While the docs for these functions are ambiguous on this topic, performing
// arithmetic on Quantities with a unit of Day or less to a Date/DateTime with
// month or year precision is undefined and should return null. This is because
// of the variability of what the definition of a month is. If a user needs this
// functionality they should use UCUM duration values.
// See: https://cql.hl7.org/09-b-cqlreference.html#equal
func evalArithmeticQuantity(m model.IBinaryExpression, lObj, rObj result.Value) (result.Value, error) {
	if result.IsNull(lObj) || result.IsNull(rObj) {
		return result.New(nil)
	}
	l, r, err := applyToValues(lObj, rObj, result.ToQuantity)
	if err != nil {
		return result.Value{}, err
	}
	return arithmeticQuantity(m, l, r)
}

// op(left Date, right Quantity) Date
// https://cql.hl7.org/09-b-cqlreference.html#add-1
// https://cql.hl7.org/09-b-cqlreference.html#subtract-1
func evalArithmeticDate(m model.IBinaryExpression, lObj, rObj result.Value) (result.Value, error) {
	if result.IsNull(lObj) || result.IsNull(rObj) {
		return result.New(nil)
	}
	r, err := result.ToQuantity(rObj)
	if err != nil {
		return result.Value{}, err
	}

	d, err := result.ToDateTime(lObj)
	if err != nil {
		return result.Value{}, err
	}
	allowUnsetPrec := false // Dates must have a precision
	err = validateDatePrecision(d.Precision, allowUnsetPrec)
	if err != nil {
		return result.Value{}, err
	}
	dtv, err := arithmeticDateTime(m, d, r)
	if err != nil {
		return result.Value{}, err
	}
	return result.New(result.Date{Date: dtv.Date, Precision: dtv.Precision})
}

// op(left DateTime, right Quantity) DateTime
// https://cql.hl7.org/09-b-cqlreference.html#add-1
// https://cql.hl7.org/09-b-cqlreference.html#subtract-1
func evalArithmeticDateTime(m model.IBinaryExpression, lObj, rObj result.Value) (result.Value, error) {
	if result.IsNull(lObj) || result.IsNull(rObj) {
		return result.New(nil)
	}
	r, err := result.ToQuantity(rObj)
	if err != nil {
		return result.Value{}, err
	}

	d, err := result.ToDateTime(lObj)
	if err != nil {
		return result.Value{}, err
	}
	allowUnsetPrec := false // DateTimes must have a precision.
	err = validateDateTimePrecision(d.Precision, allowUnsetPrec)
	if err != nil {
		return result.Value{}, err
	}
	dtv, err := arithmeticDateTime(m, d, r)
	if err != nil {
		return result.Value{}, err
	}
	return result.New(dtv)
}

func arithmetic[t float64 | int64 | int32](m model.IBinaryExpression, l, r t) (result.Value, error) {
	switch m.(type) {
	case *model.Add:
		return result.New(l + r)
	case *model.Subtract:
		return result.New(l - r)
	case *model.Multiply:
		return result.New(l * r)
	case *model.TruncatedDivide:
		if r == 0 {
			return result.New(nil)
		}
		// The first int64() truncates any decimal, then t() converts it back to the original type.
		return result.New(t(int64(l / r)))
	case *model.Divide:
		if r == 0 {
			return result.New(nil)
		}
		return result.New(l / r)
	case *model.Modulo:
		return mod(l, r)
	}
	return result.Value{}, fmt.Errorf("internal error - unsupported Binary Arithmetic Expression %v", m)
}

// Precision(arg Decimal) Integer
// https://cql.hl7.org/09-b-cqlreference.html#precision
func evalPrecisionDecimal(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	p, err := decimalPrecision(obj)
	if err != nil {
		return result.Value{}, err
	}
	return result.New(int32(p))
}

// decimalPrecision returns the number of digits after the decimal point of a Decimal.
// TODO: b/301606416 - Trailing zeros are only known for Decimal literals. Any other Decimal is a
// float64, so it is rounded to maxDecimalPrecision digits and its trailing zeros are dropped.
func decimalPrecision(obj result.Value) (int, error) {
	if l, ok := obj.SourceExpression().(*model.Literal); ok && l.GetResultType() == types.Decimal {
		_, fraction, _ := strings.Cut(l.Value, ".")
		return len(fraction), nil
	}
	d, err := result.ToFloat64(obj)
	if err != nil {
		return 0, err
	}
	_, fraction, _ := strings.Cut(strconv.FormatFloat(d, 'f', maxDecimalPrecision, 64), ".")
	return len(strings.TrimRight(fraction, "0")), nil
}

// Precision(arg Date) Integer
// Precision(arg DateTime) Integer
// https://cql.hl7.org/09-b-cqlreference.html#precision
func evalPrecisionDateTime(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	dt, err := result.ToDateTime(obj)
	if err != nil {
		return result.Value{}, err
	}
	switch dt.Precision {
	case model.YEAR:
		return result.New(4)
	case model.MONTH:
		return result.New(6)
	case model.DAY:
		return result.New(8)
	case model.HOUR:
		return result.New(10)
	case model.MINUTE:
		return result.New(12)
	case model.SECOND:
		return result.New(14)
	case model.MILLISECOND:
		return result.New(17)
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported DateTime precision %v", dt.Precision)
	}
}

// Precision(arg Time) Integer
// https://cql.hl7.org/09-b-cqlreference.html#precision
func evalPrecisionTime(_ model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	dt, err := result.ToDateTime(obj)
	if err != nil {
		return result.Value{}, err
	}
	switch dt.Precision {
	case model.HOUR:
		return result.New(2)
	case model.MINUTE:
		return result.New(4)
	case model.SECOND:
		return result.New(6)
	case model.MILLISECOND:
		return result.New(9)
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported Time precision %v", dt.Precision)
	}
}

// HighBoundary(input Decimal, precision Integer) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#highboundary
func evalHighBoundaryDecimal(_ model.IBinaryExpression, input, precision result.Value) (result.Value, error) {
	return decimalBoundary(input, precision, true)
}

// LowBoundary(input Decimal, precision Integer) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#lowboundary
func evalLowBoundaryDecimal(_ model.IBinaryExpression, input, precision result.Value) (result.Value, error) {
	return decimalBoundary(input, precision, false)
}

// decimalBoundary returns the greatest (high) or least value that input can stand for, with
// precision digits after the decimal point. A null precision means maxDecimalPrecision. A negative
// precision, or one above maxDecimalPrecision, returns null.
func decimalBoundary(input, precision result.Value, high bool) (result.Value, error) {
	if result.IsNull(input) {
		return result.New(nil)
	}
	p := maxDecimalPrecision
	if !result.IsNull(precision) {
		v, err := result.ToInt32(precision)
		if err != nil {
			return result.Value{}, err
		}
		p = int(v)
	}
	if p < 0 || p > maxDecimalPrecision {
		return result.New(nil)
	}
	d, err := result.ToFloat64(input)
	if err != nil {
		return result.Value{}, err
	}
	inputPrecision, err := decimalPrecision(input)
	if err != nil {
		return result.Value{}, err
	}

	// The digits are handled as a string so that float rounding does not change them.
	whole, fraction, _ := strings.Cut(strconv.FormatFloat(math.Abs(d), 'f', inputPrecision, 64), ".")
	// The high boundary of a positive value and the low boundary of a negative value are away from
	// zero, so the digits the input does not have are 9s instead of 0s.
	awayFromZero := high == (d >= 0)
	var boundary float64
	if p >= inputPrecision {
		fill := "0"
		if awayFromZero {
			fill = "9"
		}
		boundary, err = strconv.ParseFloat(whole+"."+fraction+strings.Repeat(fill, p-inputPrecision), 64)
	} else {
		boundary, err = strconv.ParseFloat(whole+"."+fraction[:p], 64)
		if err == nil && awayFromZero && strings.Trim(fraction[p:], "0") != "" {
			boundary, err = strconv.ParseFloat(strconv.FormatFloat(boundary+math.Pow10(-p), 'f', p, 64), 64)
		}
	}
	if err != nil {
		return result.Value{}, err
	}
	if d < 0 && boundary != 0 {
		boundary = -boundary
	}
	return result.New(boundary)
}

// precisionDigits is a precision of a Date, DateTime or Time, and its number of digits.
type precisionDigits struct {
	digits    int
	precision model.DateTimePrecision
}

// The precisions HighBoundary and LowBoundary support for each type, from coarsest to finest.
var (
	datePrecisionDigits = []precisionDigits{
		{4, model.YEAR}, {6, model.MONTH}, {8, model.DAY},
	}
	dateTimePrecisionDigits = []precisionDigits{
		{4, model.YEAR}, {6, model.MONTH}, {8, model.DAY}, {10, model.HOUR}, {12, model.MINUTE},
		{14, model.SECOND}, {17, model.MILLISECOND},
	}
	timePrecisionDigits = []precisionDigits{
		{2, model.HOUR}, {4, model.MINUTE}, {6, model.SECOND}, {9, model.MILLISECOND},
	}
)

// HighBoundary(input Date, precision Integer) Date
// HighBoundary(input DateTime, precision Integer) DateTime
// HighBoundary(input Time, precision Integer) Time
// https://cql.hl7.org/09-b-cqlreference.html#highboundary
func evalHighBoundaryDateTime(_ model.IBinaryExpression, input, precision result.Value) (result.Value, error) {
	return dateTimeBoundary(input, precision, true)
}

// LowBoundary(input Date, precision Integer) Date
// LowBoundary(input DateTime, precision Integer) DateTime
// LowBoundary(input Time, precision Integer) Time
// https://cql.hl7.org/09-b-cqlreference.html#lowboundary
func evalLowBoundaryDateTime(_ model.IBinaryExpression, input, precision result.Value) (result.Value, error) {
	return dateTimeBoundary(input, precision, false)
}

// dateTimeBoundary returns the latest (high) or earliest value that a Date, DateTime or Time input
// can stand for at precision, given in digits. A null precision means the finest precision of the
// type. A precision between two precisions of the type is handled as the coarser one, and a
// precision outside the precisions of the type returns null.
func dateTimeBoundary(input, precision result.Value, high bool) (result.Value, error) {
	if result.IsNull(input) {
		return result.New(nil)
	}
	var supported []precisionDigits
	switch input.RuntimeType() {
	case types.Date:
		supported = datePrecisionDigits
	case types.DateTime:
		supported = dateTimePrecisionDigits
	case types.Time:
		supported = timePrecisionDigits
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported type %v for boundary", input.RuntimeType())
	}
	digits := supported[len(supported)-1].digits
	if !result.IsNull(precision) {
		v, err := result.ToInt32(precision)
		if err != nil {
			return result.Value{}, err
		}
		digits = int(v)
	}
	if digits < supported[0].digits || digits > supported[len(supported)-1].digits {
		return result.New(nil)
	}
	var target model.DateTimePrecision
	for _, s := range supported {
		if s.digits <= digits {
			target = s.precision
		}
	}

	dt, err := result.ToDateTime(input)
	if err != nil {
		return result.Value{}, err
	}
	t, err := boundaryTime(dt.Date, dt.Precision, target, high)
	if err != nil {
		return result.Value{}, err
	}
	switch input.RuntimeType() {
	case types.Date:
		return result.New(result.Date{Date: t, Precision: target})
	case types.DateTime:
		return result.New(result.DateTime{Date: t, Precision: target})
	default:
		return result.New(result.Time{Date: t, Precision: target})
	}
}

// boundaryTime returns t at precision target. The parts that t knows, up to its precision, are
// kept. The parts it does not know are set to their latest (high) or earliest value, and the parts
// finer than target are set to their earliest value.
func boundaryTime(t time.Time, precision, target model.DateTimePrecision, high bool) (time.Time, error) {
	order := []model.DateTimePrecision{model.YEAR, model.MONTH, model.DAY, model.HOUR, model.MINUTE, model.SECOND, model.MILLISECOND}
	known, last := slices.Index(order, precision), slices.Index(order, target)
	if known == -1 {
		return time.Time{}, fmt.Errorf("internal error - unsupported precision %v for boundary", precision)
	}
	parts := []int{t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond() / int(time.Millisecond)}
	earliest := []int{1, 1, 1, 0, 0, 0, 0}
	latest := []int{9999, 12, 31, 23, 59, 59, 999}
	for i := range parts {
		if i <= known && i <= last {
			continue
		}
		parts[i] = earliest[i]
		if high && i <= last {
			parts[i] = latest[i]
		}
	}
	if high && known < 2 && last >= 2 {
		// Day 0 of the next month is the last day of this month.
		parts[2] = time.Date(parts[0], time.Month(parts[1]+1), 0, 0, 0, 0, 0, time.UTC).Day()
	}
	return time.Date(parts[0], time.Month(parts[1]), parts[2], parts[3], parts[4], parts[5], parts[6]*int(time.Millisecond), t.Location()), nil
}

// ^(left Integer, right Integer) Integer
// ^(left Long, right Long) Long
// ^(left Decimal, right Decimal) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#power
// In this case because we need different logic for different types, we aren't using the
// evalArithmeticInteger function.
func evalPower(m model.IBinaryExpression, lObj, rObj result.Value) (result.Value, error) {
	if result.IsNull(lObj) || result.IsNull(rObj) {
		return result.New(nil)
	}
	switch m.GetResultType() {
	case types.Integer:
		l, r, err := applyToValues(lObj, rObj, result.ToInt32)
		if err != nil {
			return result.Value{}, err
		}
		bigPow := bigIntPow(int64(l), int64(r))
		if _, ok := bigPow.(float64); ok {
			return result.New(bigPow)
		}
		return result.New(int32(bigPow.(int64)))
	case types.Long:
		l, r, err := applyToValues(lObj, rObj, result.ToInt64)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(bigIntPow(l, r))
	case types.Decimal:
		l, r, err := applyToValues(lObj, rObj, result.ToFloat64)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(math.Pow(l, r))
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported type %v", m.GetResultType())
	}
}

// bigIntPow performs integer exponentiation on big ints using the big package.
// Returns a float64 if the right hand side is negative, otherwise returns an int64.
// We do this because Golang does not have native support for exponents on integers.
func bigIntPow(l, r int64) any {
	if r == 0 {
		return int64(1)
	}
	if r == 1 {
		return l
	}
	exponentNegative := r < 0
	if exponentNegative {
		r = -r
	}
	bigL := big.NewInt(l)
	bigR := big.NewInt(r)
	bigResult := new(big.Int)
	bigResult.Exp(bigL, bigR, nil)
	if exponentNegative {
		return 1.0 / float64(bigResult.Int64())
	}
	return bigResult.Int64()
}

// Arithmetic operations for Quantity values.
// Has support for conversion between units.
func arithmeticQuantity(m model.IBinaryExpression, l, r result.Quantity) (result.Value, error) {
	switch m.(type) {
	case *model.Add, *model.Subtract:
		// For addition and subtraction, convert the right operand to the left's unit if needed.
		if l.Unit != r.Unit {
			convertedVal, err := ucum.ConvertUnit(r.Value, string(r.Unit), string(l.Unit))
			if err != nil {
				return result.Value{}, fmt.Errorf("cannot convert between units: %s and %s: %v", r.Unit, l.Unit, err)
			}
			r.Value = convertedVal
			r.Unit = l.Unit
		}

		// Now perform the operation with matching units
		if m.GetName() == "Add" {
			return result.New(result.Quantity{Value: l.Value + r.Value, Unit: l.Unit})
		}
		return result.New(result.Quantity{Value: l.Value - r.Value, Unit: l.Unit})
	case *model.Multiply:
		resultUnit := ucum.GetProductOfUnits(string(l.Unit), string(r.Unit))
		return result.New(result.Quantity{Value: l.Value * r.Value, Unit: model.Unit(resultUnit)})
	case *model.TruncatedDivide:
		if r.Value == 0 {
			return result.New(nil)
		}
		// Otherwise, the result unit is the quotient of the units
		resultUnit := ucum.GetQuotientOfUnits(string(l.Unit), string(r.Unit))
		return result.New(result.Quantity{Value: float64(int64(l.Value / r.Value)), Unit: model.Unit(resultUnit)})

	case *model.Divide:
		if r.Value == 0 {
			return result.New(nil)
		}
		// For division, the result unit is dimensionless "1" if units are the same.
		if l.Unit == r.Unit {
			return result.New(result.Quantity{Value: l.Value / r.Value, Unit: model.ONEUNIT})
		}
		// Otherwise, the result unit is the quotient of the units
		resultUnit := ucum.GetQuotientOfUnits(string(l.Unit), string(r.Unit))
		return result.New(result.Quantity{Value: l.Value / r.Value, Unit: model.Unit(resultUnit)})
	case *model.Modulo:
		if r.Value == 0 {
			return result.New(nil)
		}
		// For modulo, units must match or be convertible
		if l.Unit != r.Unit {
			// Try to convert r to l's unit
			convertedVal, err := ucum.ConvertUnit(r.Value, string(r.Unit), string(l.Unit))
			if err != nil {
				return result.New(nil)
			}
			r.Value = convertedVal
		}
		return result.New(result.Quantity{Value: math.Mod(l.Value, r.Value), Unit: l.Unit})
	}
	return result.Value{}, fmt.Errorf("internal error - unsupported Binary Arithmetic Expression %v", m)
}

// arithmeticDateTime performs arithmetic operations for Date, Quantity values.
// When performing arithmetic over differing precisions, only whole values up to
// the given Date or DateTime's precision should be added.
func arithmeticDateTime(m model.IBinaryExpression, l result.DateTime, r result.Quantity) (result.DateTime, error) {
	var sign int64
	switch m.(type) {
	case *model.Add:
		sign = 1
	case *model.Subtract:
		sign = -1
	default:
		return result.DateTime{}, fmt.Errorf("internal error - unsupported Binary Arithmetic Expression %v", m)
	}
	cq, err := convertQuantityUpToPrecision(r, l.Precision)
	if err != nil {
		return result.DateTime{}, err
	}

	switch cq.Unit {
	case model.YEARUNIT:
		return result.DateTime{Date: l.Date.AddDate(int(sign)*int(cq.Value), 0, 0), Precision: l.Precision}, nil
	case model.MONTHUNIT:
		return result.DateTime{Date: l.Date.AddDate(0, int(sign)*int(cq.Value), 0), Precision: l.Precision}, nil
	case model.WEEKUNIT:
		// Weeks need to be converted to days before they can be operated on.
		return result.DateTime{Date: l.Date.AddDate(0, 0, int(sign)*int(cq.Value*7)), Precision: l.Precision}, nil
	case model.DAYUNIT:
		return result.DateTime{Date: l.Date.AddDate(0, 0, int(sign)*int(cq.Value)), Precision: l.Precision}, nil
	case model.HOURUNIT:
		d := time.Hour * time.Duration(sign*int64(cq.Value))
		return result.DateTime{Date: l.Date.Add(d), Precision: l.Precision}, nil
	case model.MINUTEUNIT:
		d := time.Minute * time.Duration(sign*int64(cq.Value))
		return result.DateTime{Date: l.Date.Add(d), Precision: l.Precision}, nil
	// Seconds and Milliseconds shouldn't be truncated so we have to convert them to
	// nanoseconds manually.
	case model.SECONDUNIT:
		d := time.Duration(sign * int64(cq.Value*1000*1000*1000))
		return result.DateTime{Date: l.Date.Add(d), Precision: l.Precision}, nil
	case model.MILLISECONDUNIT:
		d := time.Duration(sign * int64(cq.Value*1000*1000))
		return result.DateTime{Date: l.Date.Add(d), Precision: l.Precision}, nil
	}
	return result.DateTime{}, fmt.Errorf("internal error - unsupported quantity unit %v in arithmetic operation", cq.Unit)
}

// Truncate(arg Decimal) Integer
// https://cql.hl7.org/09-b-cqlreference.html#truncate
func evalTruncate(_ model.IUnaryExpression, decimalVal result.Value) (result.Value, error) {
	if result.IsNull(decimalVal) {
		return result.New(nil)
	}
	d, err := result.ToFloat64(decimalVal)
	if err != nil {
		return result.Value{}, err
	}
	return result.New(int32(d))
}

// -(argument Integer) Integer
// https://cql.hl7.org/09-b-cqlreference.html#negate
func evalNegateInteger(m model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToInt32(obj)
	if err != nil {
		return result.Value{}, err
	}
	min, err := minValue(types.Integer, nil)
	if err != nil {
		return result.Value{}, err
	}
	if obj.Equal(min) {
		return result.New(nil)
	}
	return result.New(-val)
}

// -(argument Long) Long
// https://cql.hl7.org/09-b-cqlreference.html#negate
func evalNegateLong(m model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToInt64(obj)
	if err != nil {
		return result.Value{}, err
	}
	min, err := minValue(types.Long, nil)
	if err != nil {
		return result.Value{}, err
	}
	if obj.Equal(min) {
		return result.New(nil)
	}
	return result.New(-val)
}

// -(argument Decimal) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#negate
func evalNegateDecimal(m model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToFloat64(obj)
	if err != nil {
		return result.Value{}, err
	}
	min, err := minValue(types.Decimal, nil)
	if err != nil {
		return result.Value{}, err
	}
	if obj.Equal(min) {
		return result.New(nil)
	}
	return result.New(-val)
}

// -(argument Quantity) Quantity
// https://cql.hl7.org/09-b-cqlreference.html#negate
func evalNegateQuantity(m model.IUnaryExpression, obj result.Value) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	val, err := result.ToQuantity(obj)
	if err != nil {
		return result.Value{}, err
	}
	min, err := minValue(types.Quantity, nil)
	if err != nil {
		return result.Value{}, err
	}
	if obj.Equal(min) {
		return result.New(nil)
	}
	val.Value = -val.Value
	return result.New(val)
}

// Round(argument Decimal) Decimal
// Round(argument Decimal, argument Integer) Decimal
// https://cql.hl7.org/09-b-cqlreference.html#round
// If a precision is specified but is null then the default precision is 0.
// If a precision is specified but is negative then an error is returned. This is technically
// undefined behavior in the CQL spec, but we choose to throw an error here.
func evalRound(_ model.INaryExpression, operands []result.Value) (result.Value, error) {
	// if len(operands) == 1 {
	// 	return roundValue(operands[0])
	// }
	decimalVal := operands[0]
	var precisionVal result.Value
	var err error
	// retrieve the precision if it exists, otherwise default to 0.
	if len(operands) == 2 {
		precisionVal = operands[1]
	} else {
		precisionVal, err = result.New(0)
		if err != nil {
			return result.Value{}, err
		}
	}
	if result.IsNull(decimalVal) {
		return result.New(nil)
	}

	p, err := result.ToInt32(precisionVal)
	if err != nil {
		p = 0
	}
	if p < 0 {
		return result.Value{}, fmt.Errorf("internal error - precision must be non-negative, got %v", p)
	}
	d, err := result.ToFloat64(decimalVal)
	if err != nil {
		return result.Value{}, err
	}
	ratio := math.Pow10(int(p))
	// CQL currently implements its own special version of rounding for now (which will be changed in
	// the future). For now if the value is negative we round towards zero.
	ratioedDecimal := d * ratio
	_, frac := math.Modf(ratioedDecimal)
	// Use a small epsilon (1e-12) to detect halfway values, handling float precision errors from
	// multiplication.
	if math.Abs(frac+0.5) < 1e-12 {
		// force go to round towards zero
		ratioedDecimal += 0.1
	}
	return result.New(math.Round(ratioedDecimal) / ratio)
}

// predecessor of<T>(obj T) T
// https://cql.hl7.org/09-b-cqlreference.html#predecessor
func (i *interpreter) evalPredecessor(m model.IUnaryExpression, obj result.Value) (result.Value, error) {
	return predecessor(obj, &i.evaluationTimestamp)
}

// successor of<T>(obj T) T
// https://cql.hl7.org/09-b-cqlreference.html#successor
func (i *interpreter) evalSuccessor(m model.IUnaryExpression, obj result.Value) (result.Value, error) {
	return successor(obj, &i.evaluationTimestamp)
}

func predecessor(obj result.Value, evaluationTimestamp *time.Time) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	minVal, err := minValue(obj.RuntimeType(), evaluationTimestamp)
	if err != nil {
		return result.Value{}, err
	}
	if obj.Equal(minVal) {
		return result.Value{}, fmt.Errorf("tried to compute predecessor for value that is already a min value, %v", obj.GolangValue())
	}

	switch t := obj.RuntimeType(); t {
	case types.Integer:
		i, err := result.ToInt32(obj)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(i - 1)
	case types.Long:
		l, err := result.ToInt64(obj)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(l - 1)
	case types.Decimal:
		d, err := result.ToFloat64(obj)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(d - 0.00000001)
	case types.Quantity:
		// TODO: b/329707836 -  Determine under what cases quantities should be incremented by whole numbers in stead of decimals.
		q, err := result.ToQuantity(obj)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(result.Quantity{Value: q.Value - 0.00000001, Unit: q.Unit})
	case types.Date, types.DateTime, types.Time:
		return dateTimePredecessor(obj, evaluationTimestamp)
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported type %v", t)
	}
}

func successor(obj result.Value, evaluationTimestamp *time.Time) (result.Value, error) {
	if result.IsNull(obj) {
		return result.New(nil)
	}
	maxVal, err := maxValue(obj.RuntimeType(), evaluationTimestamp)
	if err != nil {
		return result.Value{}, err
	}
	if obj.Equal(maxVal) {
		return result.Value{}, fmt.Errorf("tried to compute successor for value that is already a max value, %v", obj.GolangValue())
	}

	switch t := obj.RuntimeType(); t {
	case types.Integer:
		i, err := result.ToInt32(obj)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(i + 1)
	case types.Long:
		l, err := result.ToInt64(obj)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(l + 1)
	case types.Decimal:
		d, err := result.ToFloat64(obj)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(d + 0.00000001)
	case types.Quantity:
		// TODO: b/329707836 -  Determine under what cases quantities should be incremented by whole numbers in stead of decimals.
		q, err := result.ToQuantity(obj)
		if err != nil {
			return result.Value{}, err
		}
		return result.New(result.Quantity{Value: q.Value + 0.00000001, Unit: q.Unit})
	case types.Date, types.DateTime, types.Time:
		return dateTimeSuccessor(obj, evaluationTimestamp)
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported type %v", t)
	}
}

// dateTimePredecessor computes the predecessor for a date time value.
// Returns error for unsupported types, precisions or values out of range.
func dateTimePredecessor(dt result.Value, evaluationTimestamp *time.Time) (result.Value, error) {
	t := dt.RuntimeType()
	switch t {
	case types.Date, types.DateTime, types.Time:
		// Valid types
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported type %v for date time predecessor", t)
	}

	minVal, err := minValue(dt.RuntimeType(), evaluationTimestamp)
	if err != nil {
		return result.Value{}, err
	}
	d, err := result.ToDateTime(dt)
	if err != nil {
		return result.Value{}, err
	}
	minDt, err := result.ToDateTime(minVal)
	if err != nil {
		return result.Value{}, err
	}

	// Check if the Value is the minimum at its precision.
	minDt.Precision = d.Precision
	if cmpResult, err := compareDateTime(d, minDt); err != nil {
		return result.Value{}, err
	} else if cmpResult == leftEqualRight || cmpResult == leftBeforeRight {
		return result.Value{}, fmt.Errorf("tried to compute predecessor for %s that is already a min value for it's precision, %v", t, dt.GolangValue())
	}

	var predecessorVal result.DateTime
	switch d.Precision {
	case model.YEAR:
		predecessorVal = result.DateTime{Date: d.Date.AddDate(-1, 0, 0), Precision: d.Precision}
	case model.MONTH:
		predecessorVal = result.DateTime{Date: d.Date.AddDate(0, -1, 0), Precision: d.Precision}
	case model.DAY:
		predecessorVal = result.DateTime{Date: d.Date.AddDate(0, 0, -1), Precision: d.Precision}
	case model.HOUR:
		predecessorVal = result.DateTime{Date: d.Date.Add(-time.Hour), Precision: d.Precision}
	case model.MINUTE:
		predecessorVal = result.DateTime{Date: d.Date.Add(-time.Minute), Precision: d.Precision}
	case model.SECOND:
		predecessorVal = result.DateTime{Date: d.Date.Add(-time.Second), Precision: d.Precision}
	case model.MILLISECOND:
		predecessorVal = result.DateTime{Date: d.Date.Add(-time.Millisecond), Precision: d.Precision}
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported precision %v in %s predecessor", d.Precision, t)
	}

	switch dt.RuntimeType() {
	case types.Date:
		return result.New(result.Date{Date: predecessorVal.Date, Precision: predecessorVal.Precision})
	case types.DateTime:
		return result.New(result.DateTime{Date: predecessorVal.Date, Precision: predecessorVal.Precision})
	case types.Time:
		return result.New(result.Time{Date: predecessorVal.Date, Precision: predecessorVal.Precision})
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported type %v for date time predecessor", t)
	}
}

// dateTimeSuccessor computes the successor for a date time value.
// Returns error for unsupported types, precisions or values out of range.
func dateTimeSuccessor(dt result.Value, evaluationTimestamp *time.Time) (result.Value, error) {
	t := dt.RuntimeType()
	switch t {
	case types.Date, types.DateTime, types.Time:
		// Valid types
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported type %v for date time successor", t)
	}

	maxVal, err := maxValue(dt.RuntimeType(), evaluationTimestamp)
	if err != nil {
		return result.Value{}, err
	}
	d, err := result.ToDateTime(dt)
	if err != nil {
		return result.Value{}, err
	}
	maxDt, err := result.ToDateTime(maxVal)
	if err != nil {
		return result.Value{}, err
	}

	// Check if the Value is the maximum at its precision.
	maxDt.Precision = d.Precision
	if cmpResult, err := compareDateTime(d, maxDt); err != nil {
		return result.Value{}, err
	} else if cmpResult == leftEqualRight || cmpResult == leftAfterRight {
		return result.Value{}, fmt.Errorf("tried to compute successor for %s that is already a max value for it's precision, %v", t, dt.GolangValue())
	}

	var successorVal result.DateTime
	switch d.Precision {
	case model.YEAR:
		successorVal = result.DateTime{Date: d.Date.AddDate(1, 0, 0), Precision: d.Precision}
	case model.MONTH:
		successorVal = result.DateTime{Date: d.Date.AddDate(0, 1, 0), Precision: d.Precision}
	case model.DAY:
		successorVal = result.DateTime{Date: d.Date.AddDate(0, 0, 1), Precision: d.Precision}
	case model.HOUR:
		successorVal = result.DateTime{Date: d.Date.Add(time.Hour), Precision: d.Precision}
	case model.MINUTE:
		successorVal = result.DateTime{Date: d.Date.Add(time.Minute), Precision: d.Precision}
	case model.SECOND:
		successorVal = result.DateTime{Date: d.Date.Add(time.Second), Precision: d.Precision}
	case model.MILLISECOND:
		successorVal = result.DateTime{Date: d.Date.Add(time.Millisecond), Precision: d.Precision}
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported precision %v in date time successor", d.Precision)
	}

	switch dt.RuntimeType() {
	case types.Date:
		return result.New(result.Date{Date: successorVal.Date, Precision: successorVal.Precision})
	case types.DateTime:
		return result.New(result.DateTime{Date: successorVal.Date, Precision: successorVal.Precision})
	case types.Time:
		return result.New(result.Time{Date: successorVal.Date, Precision: successorVal.Precision})
	default:
		return result.Value{}, fmt.Errorf("internal error - unsupported type %v for date time successor", t)
	}
}

// maximum<T>() T
// https://cql.hl7.org/09-b-cqlreference.html#maximum
func (i *interpreter) evalMaxValue(m *model.MaxValue) (result.Value, error) {
	return maxValue(m.ValueType, &i.evaluationTimestamp)
}

// minimum<T>() T
// https://cql.hl7.org/09-b-cqlreference.html#minimum
func (i *interpreter) evalMinValue(m *model.MinValue) (result.Value, error) {
	return minValue(m.ValueType, &i.evaluationTimestamp)
}

// Note: For Date/Time based values, the spec states that an engine can choose to set the timezone
// to UTC for min/max values, we use the evaluation timestamp's timezone.
// We do this because when creating a literal it's also in the evaluation timestamp's timezone, and
// some external tests will fail when these are different.
func maxValue(t types.IType, evaluationTimestamp *time.Time) (result.Value, error) {
	switch t {
	case types.Integer:
		return result.New(int32(math.MaxInt32))
	case types.Long:
		return result.New(int64(math.MaxInt64))
	case types.Decimal:
		return result.New(maxDecimal)
	case types.Quantity:
		return result.New(result.Quantity{Value: maxDecimal, Unit: "1"})
	case types.Date:
		if evaluationTimestamp == nil {
			return result.Value{}, fmt.Errorf("internal error - evaluation timestamp cannot be nil for Date max value")
		}
		return result.New(result.Date{Date: time.Date(9999, 12, 31, 0, 0, 0, 0, evaluationTimestamp.Location()), Precision: model.DAY})
	case types.DateTime:
		if evaluationTimestamp == nil {
			return result.Value{}, fmt.Errorf("internal error - evaluation timestamp cannot be nil for DateTime max value")
		}
		return result.New(result.DateTime{Date: time.Date(9999, 12, 31, 23, 59, 59, 999, evaluationTimestamp.Location()), Precision: model.MILLISECOND})
	case types.Time:
		if evaluationTimestamp == nil {
			return result.Value{}, fmt.Errorf("internal error - evaluation timestamp cannot be nil for Time max value")
		}
		return result.New(result.Time{Date: time.Date(0, time.January, 1, 23, 59, 59, 999000000, evaluationTimestamp.Location()), Precision: model.MILLISECOND})
	default:
		return result.Value{}, fmt.Errorf("unsupported type, cannot compute max value for: %v", t)
	}
}

// Note: For Date/Time based values, the spec states that an engine can choose to set the timezone
// to UTC for min/max values, we use the evaluation timestamp's timezone.
// We do this because when creating a literal it's also in the evaluation timestamp's timezone, and
// some external tests will fail when these are different.
func minValue(t types.IType, evaluationTimestamp *time.Time) (result.Value, error) {
	switch t {
	case types.Integer:
		return result.New(int32(math.MinInt32))
	case types.Long:
		return result.New(int64(math.MinInt64))
	case types.Decimal:
		return result.New(minDecimal)
	case types.Quantity:
		return result.New(result.Quantity{Value: minDecimal, Unit: "1"})
	case types.Date:
		if evaluationTimestamp == nil {
			return result.Value{}, fmt.Errorf("internal error - evaluation timestamp cannot be nil for Date min value")
		}
		return result.New(result.Date{Date: time.Date(1, 1, 1, 0, 0, 0, 0, evaluationTimestamp.Location()), Precision: model.DAY})
	case types.DateTime:
		if evaluationTimestamp == nil {
			return result.Value{}, fmt.Errorf("internal error - evaluation timestamp cannot be nil for DateTime min value")
		}
		return result.New(result.DateTime{Date: time.Date(1, 1, 1, 0, 0, 0, 0, evaluationTimestamp.Location()), Precision: model.MILLISECOND})
	case types.Time:
		if evaluationTimestamp == nil {
			return result.Value{}, fmt.Errorf("internal error - evaluation timestamp cannot be nil for Time min value")
		}
		return result.New(result.Time{Date: time.Date(0, time.January, 1, 0, 0, 0, 0, evaluationTimestamp.Location()), Precision: model.MILLISECOND})
	default:
		return result.Value{}, fmt.Errorf("unsupported type, cannot compute min value for: %v", t)
	}
}

// https://cql.hl7.org/09-b-cqlreference.html#modulo
// According to the spec, "If the result of the modulo cannot be represented, or the right argument
// is 0, the result is null.".
func mod(l, r any) (result.Value, error) {
	// The modulo operator doesn't support floats so we need to call math.Mod.
	switch l.(type) {
	case int32, int:
		rVal := r.(int32)
		if rVal == 0 {
			return result.New(nil)
		}
		return result.New(l.(int32) % r.(int32))
	case int64:
		rVal := r.(int64)
		if rVal == 0 {
			return result.New(nil)
		}
		return result.New(l.(int64) % r.(int64))
	case float64:
		rVal := r.(float64)
		if rVal == 0 {
			return result.New(nil)
		}
		return result.New(math.Mod(l.(float64), r.(float64)))
	}
	return result.Value{}, fmt.Errorf("internal error - mod does not support %v", reflect.TypeOf(l))
}
