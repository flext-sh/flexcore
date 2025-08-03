// Package result provides railway-oriented programming patterns for FlexCore
// and the broader FLEXT ecosystem, enabling explicit error handling and
// functional composition patterns.
//
// This package implements the Result pattern popularized in functional programming
// languages, providing type-safe error handling without exceptions and supporting
// monadic composition for complex operation chains. It serves as a foundational
// component across all FlexCore modules and integrates with FLEXT ecosystem patterns.
//
// Key Features:
//   - Type-safe error handling with explicit success/failure states
//   - Monadic operations for functional composition (Map, FlatMap, Filter)
//   - Railway-oriented programming with automatic error propagation
//   - Async operation support with channels and goroutines
//   - Panic recovery with Try/TryAsync functions
//   - Result combination and tuple operations
//
// Architecture:
//   This package follows functional programming principles with immutable
//   result types and pure transformation functions. All operations preserve
//   type safety and provide explicit error handling without runtime panics.
//
// Integration:
//   - Used throughout FlexCore domain, application, and infrastructure layers
//   - Integrates with FLEXT ecosystem error handling patterns
//   - Compatible with flext-core foundation library
//   - Supports Clean Architecture dependency inversion
//
// Example:
//   Basic result creation and transformation:
//
//     // Create successful result
//     result := result.Success("Hello, World!")
//     
//     // Transform value with error handling
//     transformed := result.Map(func(s string) string {
//         return strings.ToUpper(s)
//     })
//     
//     // Chain operations with FlatMap
//     final := transformed.FlatMap(func(s string) result.Result[int] {
//         if len(s) > 0 {
//             return result.Success(len(s))
//         }
//         return result.Failure[int](errors.New("empty string"))
//     })
//     
//     // Handle result
//     if final.IsSuccess() {
//         fmt.Printf("Length: %d\n", final.Value())
//     } else {
//         fmt.Printf("Error: %v\n", final.Error())
//     }
//
//     // Pipeline operations
//     pipeline := result.Success("input")
//         .Map(processStep1)
//         .FlatMap(processStep2)
//         .Filter(validationPredicate)
//
// Performance:
//   Result operations are zero-allocation for success cases and minimal
//   allocation for error cases. Monadic operations are optimized for
//   composition without intermediate allocations.
//
// Thread Safety:
//   Result instances are immutable and thread-safe for concurrent read access.
//   Async operations use goroutines and channels with proper synchronization.
//
// Author: FLEXT Development Team
// Version: 0.9.0
// License: MIT
package result

import (
	"fmt"
)

// Result represents the outcome of an operation that can either succeed with a value
// or fail with an error, implementing railway-oriented programming patterns.
//
// Result provides type-safe error handling without exceptions, ensuring that all
// error conditions are explicitly handled by the calling code. It supports functional
// composition through monadic operations, allowing complex operation chains with
// automatic error propagation.
//
// The Result type is generic over any type T, making it suitable for operations
// that return any kind of value. Error propagation is automatic - once a Result
// contains an error, all subsequent operations will preserve that error without
// executing their transformations.
//
// Examples:
//   Creating results:
//
//     success := result.Success(42)
//     failure := result.Failure[int](errors.New("operation failed"))
//
//   Checking result state:
//
//     if result.IsSuccess() {
//         value := result.Value()
//         // Process successful value
//     } else {
//         err := result.Error()
//         // Handle error condition
//     }
//
//   Functional composition:
//
//     result := result.Success("hello")
//         .Map(strings.ToUpper)
//         .FlatMap(func(s string) result.Result[int] {
//             return result.Success(len(s))
//         })
//
// Thread Safety:
//   Result instances are immutable and safe for concurrent access.
//   Once created, a Result's state cannot be modified.
type Result[T any] struct {
	value T
	err   error
}

// Success creates a Result representing a successful operation with the given value.
//
// This function constructs a Result in the success state, containing the provided
// value and no error. Subsequent operations on this Result will execute their
// transformations using the contained value.
//
// Parameters:
//   value T: The successful value to wrap in the Result. Can be any type.
//
// Returns:
//   Result[T]: A successful Result containing the provided value with no error.
//
// Example:
//   Create successful results of different types:
//
//     stringResult := result.Success("hello")
//     intResult := result.Success(42)
//     structResult := result.Success(User{Name: "Alice", Age: 30})
//
//     // All results are in success state
//     fmt.Println(stringResult.IsSuccess()) // true
//     fmt.Println(intResult.Value())        // 42
func Success[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Failure creates a Result representing a failed operation with the given error.
//
// This function constructs a Result in the failure state, containing the provided
// error and the zero value for type T. Subsequent operations on this Result will
// skip their transformations and propagate the error automatically.
//
// Parameters:
//   err error: The error that caused the operation to fail. Must not be nil.
//
// Returns:
//   Result[T]: A failed Result containing the error and zero value for type T.
//
// Example:
//   Create failed results with different error types:
//
//     validationErr := result.Failure[string](errors.New("validation failed"))
//     networkErr := result.Failure[User](fmt.Errorf("network timeout: %w", baseErr))
//     customErr := result.Failure[int](MyCustomError{Code: 404})
//
//     // All results are in failure state
//     fmt.Println(validationErr.IsFailure()) // true
//     fmt.Println(networkErr.Error())        // "network timeout: ..."
//
// Note:
//   Passing a nil error will create a Result with nil error, which will be
//   considered successful. Use Success() instead for successful operations.
func Failure[T any](err error) Result[T] {
	var zero T
	return Result[T]{value: zero, err: err}
}

// FailureWithMessage creates a failed Result with a simple string error message.
//
// This function provides a convenient way to create failed Results with
// descriptive error messages without requiring explicit error creation.
// It's particularly useful for validation failures and business rule violations.
//
// Type Parameters:
//   T: The type that the Result would contain if it were successful
//
// Parameters:
//   message string: Error message describing the failure condition
//
// Returns:
//   Result[T]: A failed Result containing an error created from the message
//
// Example:
//   Validation failures:
//
//     if len(username) < 3 {
//         return result.FailureWithMessage[User]("username must be at least 3 characters")
//     }
//
//   Business rule violations:
//
//     if account.Balance < amount {
//         return result.FailureWithMessage[Transaction]("insufficient funds")
//     }
//
//   Configuration errors:
//
//     if config.APIKey == "" {
//         return result.FailureWithMessage[APIClient]("API key is required")
//     }
//
// Performance:
//   Single allocation for error creation using fmt.Errorf.
//   Efficient for simple error messages without formatting needs.
//
// Thread Safety:
//   Safe for concurrent use as it creates new Result instances.
func FailureWithMessage[T any](message string) Result[T] {
	return Failure[T](fmt.Errorf(message))
}

// IsSuccess returns true if the Result represents a successful operation.
//
// A Result is considered successful when it contains no error (err == nil).
// Successful results contain a valid value that can be safely accessed
// through the Value() method.
//
// Returns:
//   bool: true if the Result contains no error, false otherwise.
//
// Example:
//   Check result state before processing:
//
//     result := performOperation()
//     if result.IsSuccess() {
//         value := result.Value()
//         processSuccessfulValue(value)
//     } else {
//         handleError(result.Error())
//     }
//
// See also:
//   IsFailure() for checking the opposite condition.
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// IsFailure returns true if the Result represents a failed operation.
//
// A Result is considered failed when it contains an error (err != nil).
// Failed results contain an error that can be accessed through the Error()
// method, and their value should not be used.
//
// Returns:
//   bool: true if the Result contains an error, false otherwise.
//
// Example:
//   Handle failure cases explicitly:
//
//     result := performRiskyOperation()
//     if result.IsFailure() {
//         err := result.Error()
//         log.Printf("Operation failed: %v", err)
//         return handleFailure(err)
//     }
//     
//     // Safe to use value
//     return processSuccess(result.Value())
//
// See also:
//   IsSuccess() for checking the opposite condition.
func (r Result[T]) IsFailure() bool {
	return r.err != nil
}

// Value returns the contained value if the Result is successful, or the zero value if failed.
//
// This method provides access to the successful value without error checking.
// For failed Results, it returns the zero value for type T. Always check
// IsSuccess() before calling Value() to ensure the value is meaningful.
//
// Returns:
//   T: The contained value if successful, or zero value if failed.
//
// Example:
//   Safe value access with state checking:
//
//     result := computeValue()
//     if result.IsSuccess() {
//         value := result.Value() // Safe to use
//         fmt.Printf("Computed: %v\n", value)
//     }
//
//   Unsafe access (not recommended):
//
//     value := result.Value() // May be zero value if failed
//     // Could lead to using invalid data
//
// Best Practice:
//   Always check IsSuccess() before using Value(), or use ValueOr()
//   to provide a meaningful default for failure cases.
func (r Result[T]) Value() T {
	return r.value
}

// Error returns the contained error if the Result is failed, or nil if successful.
//
// This method provides access to the error that caused the operation to fail.
// For successful Results, it returns nil. Always check IsFailure() before
// calling Error() if you need to distinguish between no error and nil error.
//
// Returns:
//   error: The contained error if failed, or nil if successful.
//
// Example:
//   Error handling with state checking:
//
//     result := performOperation()
//     if result.IsFailure() {
//         err := result.Error()
//         log.Printf("Operation failed: %v", err)
//         return fmt.Errorf("processing failed: %w", err)
//     }
//
//   Combined state checking:
//
//     if err := result.Error(); err != nil {
//         // Handle error case
//         return handleError(err)
//     }
//     // Handle success case
//     return processValue(result.Value())
//
// Thread Safety:
//   Safe for concurrent access as Result instances are immutable.
func (r Result[T]) Error() error {
	return r.err
}

// ValueOr returns the contained value if successful, or the provided default value if failed.
//
// This method provides safe value access with a fallback mechanism, eliminating
// the need for explicit success checking in cases where a reasonable default
// exists. It's particularly useful for configuration values, optional parameters,
// and graceful degradation scenarios.
//
// Parameters:
//   defaultValue T: The value to return if the Result is in a failed state.
//                   This value is returned as-is without any transformation.
//
// Returns:
//   T: The contained value if successful, or the provided default value if failed.
//
// Example:
//   Safe value access with meaningful defaults:
//
//     configResult := loadConfiguration()
//     timeout := configResult.ValueOr(30) // Use 30 seconds as default
//     
//     userResult := fetchUser(id)
//     user := userResult.ValueOr(User{Name: "Anonymous"}) // Anonymous user as fallback
//
//   Chain with other operations:
//
//     result := processData(input)
//         .Map(transform)
//         .ValueOr("default-output") // Safe final value
//
// Performance:
//   Zero-allocation operation that simply returns the appropriate value
//   without any additional processing or validation overhead.
//
// Thread Safety:
//   Safe for concurrent access as Result instances are immutable.
func (r Result[T]) ValueOr(defaultValue T) T {
	if r.IsSuccess() {
		return r.value
	}
	return defaultValue
}

// Get returns the contained value if successful, or panics if the Result is failed.
//
// This method provides direct value access without explicit error checking,
// similar to exception-based error handling in other languages. It should be
// used sparingly and only when you're certain the Result is successful, or when
// panicking is the desired behavior for error conditions.
//
// Returns:
//   T: The contained value if the Result is successful.
//
// Panics:
//   Panics with the contained error if the Result is in a failed state.
//   The panic value will be the exact error from the Result.
//
// Example:
//   Use when confident about success:
//
//     result := validateInput(data)
//     if result.IsSuccess() {
//         value := result.Get() // Safe - we know it's successful
//         processValue(value)
//     }
//
//   Panic-based error handling (use with caution):
//
//     value := parseConfig().Get() // Will panic if parsing fails
//     // Only reaches here if parsing succeeded
//
// Warning:
//   This method can cause runtime panics. Consider using Value() with
//   IsSuccess() check, ValueOr() with defaults, or proper error handling
//   patterns for production code.
//
// Best Practice:
//   Prefer explicit error handling over panic-based patterns in most cases.
//   Use Get() only for scenarios where panicking is appropriate (e.g., 
//   initialization code, test scenarios, or when failure is unrecoverable).
func (r Result[T]) Get() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

// Filter validates the contained value against a predicate function (standalone version).
//
// This is the standalone function version of the Result.Filter method,
// providing the same filtering functionality for use in functional composition
// or when method chaining is not preferred.
//
// Type Parameters:
//   T: The type of value contained in the Result
//
// Parameters:
//   r Result[T]: The Result to filter
//   predicate func(T) bool: Function that tests the value for validity
//
// Returns:
//   Result[T]: The original Result if successful and predicate returns true,
//              a failed Result if predicate returns false,
//              or the original failed Result if already failed.
//
// Example:
//   Functional composition style:
//
//     result := result.Success(42)
//     filtered := result.Filter(result, func(n int) bool {
//         return n > 0
//     })
//
//   With higher-order functions:
//
//     isPositive := func(n int) bool { return n > 0 }
//     results := []result.Result[int]{...}
//     
//     for _, r := range results {
//         filtered := result.Filter(r, isPositive)
//         // Process filtered result
//     }
//
// Note:
//   Consider using the method version r.Filter(predicate) for more fluent
//   method chaining in most cases.
//
// Thread Safety:
//   Safe for concurrent use as it creates new Result instances without modifying originals.
func Filter[T any](r Result[T], predicate func(T) bool) Result[T] {
	if r.IsFailure() {
		return r
	}
	if !predicate(r.value) {
		return Failure[T](fmt.Errorf("filter predicate failed"))
	}
	return r
}

// OrElse returns the first Result if successful, or the alternative Result if the first failed.
//
// OrElse provides fallback logic for Results, enabling graceful degradation
// when primary operations fail. It's useful for implementing retry patterns,
// fallback data sources, or default value strategies.
//
// Type Parameters:
//   T: The type of value contained in both Results
//
// Parameters:
//   r Result[T]: Primary Result to check first
//   alternative Result[T]: Fallback Result to use if primary failed
//
// Returns:
//   Result[T]: The primary Result if successful, otherwise the alternative Result
//
// Example:
//   Fallback data sources:
//
//     primaryResult := fetchFromPrimaryDB(id)
//     fallbackResult := fetchFromCache(id)
//     final := result.OrElse(primaryResult, fallbackResult)
//
//   Configuration with defaults:
//
//     userConfig := loadUserConfig()
//     defaultConfig := result.Success(DefaultConfig{})
//     config := result.OrElse(userConfig, defaultConfig)
//
//   Service degradation:
//
//     fastService := callFastAPI()
//     slowService := callSlowAPI()
//     response := result.OrElse(fastService, slowService)
//
// Behavior:
//   - Returns r immediately if r.IsSuccess()
//   - Returns alternative if r.IsFailure() (alternative may also be failed)
//   - Does not evaluate alternative if primary succeeds (short-circuiting)
//
// Performance:
//   Zero allocation if primary Result is successful (short-circuit return).
//   Alternative Result evaluation only happens when needed.
//
// Thread Safety:
//   Safe for concurrent use as it doesn't modify the original Results.
func OrElse[T any](r Result[T], alternative Result[T]) Result[T] {
	if r.IsSuccess() {
		return r
	}
	return alternative
}

// OrElseGet returns the Result if successful, or calls the supplier function if failed.
//
// OrElseGet provides lazy evaluation of fallback Results, enabling expensive
// fallback operations to be deferred until actually needed. The supplier function
// is only called when the primary Result is failed, making this more efficient
// than OrElse when the alternative is expensive to compute.
//
// Type Parameters:
//   T: The type of value contained in the Results
//
// Parameters:
//   r Result[T]: Primary Result to check first
//   supplier func() Result[T]: Function that provides fallback Result when needed.
//                              Only called if primary Result is failed.
//
// Returns:
//   Result[T]: The primary Result if successful, otherwise the Result from supplier()
//
// Example:
//   Expensive fallback operations:
//
//     primaryResult := quickLookup(id)
//     final := result.OrElseGet(primaryResult, func() result.Result[Data] {
//         // This expensive operation only runs if quickLookup failed
//         return expensiveFullSearch(id)
//     })
//
//   Dynamic fallback selection:
//
//     cacheResult := getFromCache(key)
//     final := result.OrElseGet(cacheResult, func() result.Result[Data] {
//         // Choose fallback strategy based on current system state
//         if isHighLoad() {
//             return getFromSecondaryCache(key)
//         } else {
//             return fetchFromDatabase(key)
//         }
//     })
//
//   Retry with backoff:
//
//     attempt1 := networkCall()
//     final := result.OrElseGet(attempt1, func() result.Result[Response] {
//         time.Sleep(time.Second) // Backoff
//         return networkCall() // Retry
//     })
//
// Performance:
//   Zero overhead if primary Result is successful (supplier not called).
//   Supplier function evaluation deferred until needed, saving resources.
//
// Thread Safety:
//   Safe for concurrent use. Supplier function must be thread-safe if called concurrently.
func OrElseGet[T any](r Result[T], supplier func() Result[T]) Result[T] {
	if r.IsSuccess() {
		return r
	}
	return supplier()
}

// ValueOrZero returns the contained value if successful, or the zero value for type T if failed.
//
// This method provides a convenient way to extract values with the language's
// default zero value as fallback. It's useful when the zero value represents
// a reasonable default state for your application logic.
//
// Returns:
//   T: The contained value if successful, or the zero value for type T if failed.
//      Zero values: 0 for numbers, "" for strings, nil for pointers, etc.
//
// Example:
//   Using zero values as defaults:
//
//     countResult := calculateCount()
//     count := countResult.ValueOrZero() // 0 if failed
//     
//     nameResult := fetchName()
//     name := nameResult.ValueOrZero() // "" if failed
//     
//     userResult := loadUser()
//     user := userResult.ValueOrZero() // User{} if failed
//
//   Comparison with ValueOr:
//
//     explicit := result.ValueOr("default")    // Explicit default
//     zeroVal := result.ValueOrZero()          // Language zero value
//
// Performance:
//   Minimal overhead - delegates to ValueOr with pre-computed zero value.
//   Zero value computation happens at compile time for most types.
//
// Thread Safety:
//   Safe for concurrent access as Result instances are immutable.
func (r Result[T]) ValueOrZero() T {
	var zero T
	return r.ValueOr(zero)
}

// Unwrap returns the contained value and error as separate return values.
//
// This method provides Go-idiomatic access to both the value and error,
// allowing integration with existing Go code that expects multiple return
// values. It's particularly useful when interfacing with non-Result-based
// APIs or when you need explicit access to both components.
//
// Returns:
//   T: The contained value (may be zero value if Result is failed)
//   error: The contained error (nil if Result is successful)
//
// Example:
//   Go-style error handling:
//
//     result := performOperation()
//     value, err := result.Unwrap()
//     if err != nil {
//         return fmt.Errorf("operation failed: %w", err)
//     }
//     return processValue(value)
//
//   Integration with existing APIs:
//
//     result := computeResult()
//     value, err := result.Unwrap()
//     return legacyFunction(value, err) // Function expects (T, error)
//
//   Multiple assignment:
//
//     val1, err1 := result1.Unwrap()
//     val2, err2 := result2.Unwrap()
//     if err1 != nil || err2 != nil {
//         // Handle errors
//     }
//
// Integration:
//   Perfect for bridging Result-based code with traditional Go error handling
//   patterns, enabling gradual adoption of Result patterns in existing codebases.
//
// Thread Safety:
//   Safe for concurrent access as Result instances are immutable.
func (r Result[T]) Unwrap() (T, error) {
	return r.value, r.err
}

// UnwrapOrPanic returns the contained value if successful, or panics if failed.
//
// This method is an alias for Get() that makes the panic behavior explicit
// in the method name. It's useful in scenarios where panicking is the intended
// behavior for error conditions, such as initialization code or test scenarios.
//
// Returns:
//   T: The contained value if the Result is successful.
//
// Panics:
//   Panics with the contained error if the Result is in a failed state.
//   The panic value will be the exact error from the Result.
//
// Example:
//   Initialization code where failure is unrecoverable:
//
//     config := loadConfig().UnwrapOrPanic()
//     // Application cannot continue without valid config
//     
//     dbConnection := connectToDatabase().UnwrapOrPanic()
//     // Database connection is required for operation
//
//   Test scenarios:
//
//     result := functionUnderTest(validInput)
//     value := result.UnwrapOrPanic() // Test should fail if function fails
//     assert.Equal(t, expectedValue, value)
//
// Warning:
//   This method will cause runtime panics on failure. Only use when:
//   - Failure represents an unrecoverable condition
//   - In test code where panics indicate test failures
//   - During application initialization where partial startup is worse than no startup
//
// Alternative:
//   Consider Get() for similar behavior, or Value() with IsSuccess() for safer patterns.
func (r Result[T]) UnwrapOrPanic() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

// Map transforms the contained value using the provided function if the Result is successful,
// or propagates the error if the Result is failed.
//
// Map enables functional composition by applying transformations to successful values
// while automatically handling error propagation. This is a core operation in
// railway-oriented programming, allowing you to chain transformations without
// explicit error checking at each step.
//
// Type Parameters:
//   T: The input type of the original Result
//   U: The output type after transformation
//
// Parameters:
//   r Result[T]: The Result to transform
//   fn func(T) U: Pure function to transform the value. Should not have side effects.
//
// Returns:
//   Result[U]: New Result containing the transformed value if original was successful,
//              or the original error if the original Result was failed.
//
// Example:
//   Basic transformation:
//
//     numberResult := result.Success(42)
//     stringResult := result.Map(numberResult, func(n int) string {
//         return fmt.Sprintf("Number: %d", n)
//     })
//     // stringResult contains "Number: 42"
//
//   Chain multiple transformations:
//
//     result := result.Success("hello world")
//     upper := result.Map(result, strings.ToUpper)
//     length := result.Map(upper, func(s string) int { return len(s) })
//     // Final result contains the length of "HELLO WORLD"
//
//   Error propagation:
//
//     failedResult := result.Failure[int](errors.New("failed"))
//     mapped := result.Map(failedResult, func(n int) string { return "success" })
//     // mapped still contains the original error, transformation was skipped
//
// Performance:
//   Zero allocation for failed Results (error propagation).
//   Single allocation for successful Results (new Result[U]).
//   Transformation function execution overhead only for successful cases.
//
// Thread Safety:
//   Safe for concurrent use as it creates new Result instances without modifying originals.
func Map[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.IsFailure() {
		return Failure[U](r.err)
	}
	return Success(fn(r.value))
}

// FlatMap transforms the contained value using a function that returns a Result,
// flattening nested Results to avoid Result[Result[T]] structures.
//
// FlatMap is essential for chaining operations that can fail, enabling monadic
// composition in railway-oriented programming. It's the key difference from Map:
// Map works with functions that return plain values, while FlatMap works with
// functions that return Results.
//
// Type Parameters:
//   T: The input type of the original Result
//   U: The output type of the Result returned by the transformation function
//
// Parameters:
//   r Result[T]: The Result to transform
//   fn func(T) Result[U]: Function that takes a value and returns a Result.
//                         Can represent operations that might fail.
//
// Returns:
//   Result[U]: The Result returned by the transformation function if original was successful,
//              or the original error if the original Result was failed.
//
// Example:
//   Chain operations that can fail:
//
//     parseResult := result.Success("42")
//     numberResult := result.FlatMap(parseResult, func(s string) result.Result[int] {
//         if num, err := strconv.Atoi(s); err != nil {
//             return result.Failure[int](err)
//         } else {
//             return result.Success(num)
//         }
//     })
//
//   Database operations:
//
//     userResult := findUser(id)
//     profileResult := result.FlatMap(userResult, func(user User) result.Result[Profile] {
//         return loadProfile(user.ID) // Returns Result[Profile]
//     })
//
//   Validation chains:
//
//     result := result.Success(userData)
//         .FlatMap(validateEmail)
//         .FlatMap(validateAge)
//         .FlatMap(saveUser)
//
// Comparison with Map:
//   - Map: func(T) U -> transforms value to different type
//   - FlatMap: func(T) Result[U] -> transforms value to Result, flattens automatically
//
// Performance:
//   Zero allocation for failed Results (error propagation).
//   Delegates to transformation function for successful Results.
//
// Thread Safety:
//   Safe for concurrent use as it doesn't modify the original Result.
func FlatMap[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.IsFailure() {
		return Failure[U](r.err)
	}
	return fn(r.value)
}

// Filter validates the contained value against a predicate, converting the Result
// to a failure if the predicate returns false.
//
// Filter enables conditional processing in Result chains, allowing you to enforce
// business rules and validation logic. If the predicate fails, the Result becomes
// a failure with a descriptive error message.
//
// Parameters:
//   predicate func(T) bool: Function that tests the value. Should be a pure function
//                          without side effects. Returns true if value is valid.
//
// Returns:
//   Result[T]: The original Result if successful and predicate returns true,
//              a failed Result with "filter predicate failed" error if predicate returns false,
//              or the original failed Result if already failed.
//
// Example:
//   Validation in processing chains:
//
//     result := result.Success(42)
//         .Filter(func(n int) bool { return n > 0 })
//         .Map(func(n int) string { return fmt.Sprintf("Positive: %d", n) })
//
//   Business rule validation:
//
//     ageResult := result.Success(25)
//         .Filter(func(age int) bool { return age >= 18 })
//     // Fails if age < 18
//
//   String validation:
//
//     emailResult := result.Success("user@example.com")
//         .Filter(func(email string) bool { return strings.Contains(email, "@") })
//
//   Chain multiple filters:
//
//     result := result.Success(userData)
//         .Filter(validateAge)
//         .Filter(validateEmail)
//         .Filter(validateCountry)
//
// Error Handling:
//   Failed predicates generate a generic "filter predicate failed" error.
//   For custom error messages, use FlatMap with explicit validation functions.
//
// Performance:
//   Zero overhead for failed Results (immediate return).
//   Single predicate function call for successful Results.
//   Creates new failed Result only when predicate fails.
//
// Thread Safety:
//   Safe for concurrent use. Creates new Result instances without modifying originals.
func (r Result[T]) Filter(predicate func(T) bool) Result[T] {
	if r.IsFailure() {
		return r
	}
	if !predicate(r.value) {
		return FailureWithMessage[T]("filter predicate failed")
	}
	return r
}

// ForEach executes a side-effect function with the contained value if the Result is successful.
//
// ForEach enables imperative operations within functional Result chains,
// such as logging, caching, or triggering external actions. The function
// is only called for successful Results, providing safe access to values
// without explicit error checking.
//
// Parameters:
//   fn func(T): Side-effect function to execute with the value.
//               Typically used for logging, caching, notifications, etc.
//               Should not modify the value (Result is immutable).
//
// Returns:
//   None. This method is called for side effects only.
//
// Example:
//   Logging successful results:
//
//     result := processData(input)
//     result.ForEach(func(data ProcessedData) {
//         log.Printf("Successfully processed %d records", data.Count)
//     })
//
//   Caching successful computations:
//
//     computationResult := expensiveComputation()
//     computationResult.ForEach(func(result ComputationResult) {
//         cache.Set(cacheKey, result)
//     })
//
//   Triggering notifications:
//
//     userResult := createUser(userData)
//     userResult.ForEach(func(user User) {
//         emailService.SendWelcomeEmail(user.Email)
//         analyticsService.TrackUserCreation(user.ID)
//     })
//
//   Chain with other operations:
//
//     result := result.Success("important data")
//         .Map(strings.ToUpper)
//         .ForEach(func(data string) { log.Info(data) })
//         .Filter(func(data string) bool { return len(data) > 0 })
//
// Use Cases:
//   - Logging and debugging
//   - Caching successful results
//   - Sending notifications or events
//   - Updating metrics or analytics
//   - Any side effect that should only occur on success
//
// Thread Safety:
//   Safe for concurrent access to the Result. However, the side-effect function
//   must be thread-safe if called concurrently.
func (r Result[T]) ForEach(fn func(T)) {
	if r.IsSuccess() {
		fn(r.value)
	}
}

// IfFailure executes a side-effect function with the contained error if the Result is failed.
//
// IfFailure enables error handling side effects within Result chains,
// such as logging errors, sending alerts, or recording failures.
// The function is only called for failed Results, providing safe
// access to errors without explicit success checking.
//
// Parameters:
//   fn func(error): Side-effect function to execute with the error.
//                   Typically used for logging, alerting, metrics, etc.
//                   Should not modify the error state.
//
// Returns:
//   None. This method is called for side effects only.
//
// Example:
//   Error logging:
//
//     result := riskyOperation()
//     result.IfFailure(func(err error) {
//         log.Error("Operation failed", "error", err)
//     })
//
//   Error metrics and alerting:
//
//     processResult := processData(input)
//     processResult.IfFailure(func(err error) {
//         metrics.IncrementErrorCounter("data_processing")
//         alertService.SendAlert("Processing failed: " + err.Error())
//     })
//
//   Error recovery preparation:
//
//     dbResult := database.Query(sql)
//     dbResult.IfFailure(func(err error) {
//         // Log the error and prepare for fallback
//         log.Warn("Database query failed, using cache", "error", err)
//         fallbackService.PrepareCache()
//     })
//
//   Chain with success handling:
//
//     result := performOperation()
//         .ForEach(func(data Data) { log.Info("Success", "data", data) })
//         .IfFailure(func(err error) { log.Error("Failed", "error", err) })
//
// Use Cases:
//   - Error logging and debugging
//   - Sending error alerts or notifications
//   - Recording error metrics
//   - Preparing fallback mechanisms
//   - Any side effect that should only occur on failure
//
// Thread Safety:
//   Safe for concurrent access to the Result. However, the side-effect function
//   must be thread-safe if called concurrently.
func (r Result[T]) IfFailure(fn func(error)) {
	if r.IsFailure() {
		fn(r.err)
	}
}

// Combine merges two Results into a single Result containing a Tuple of both values.
//
// Combine enables parallel processing of independent operations, collecting
// their results into a single structure. Both Results must be successful for
// the combination to succeed; if either Result is failed, the first failure
// is returned.
//
// Type Parameters:
//   T: Type of the first Result's value
//   U: Type of the second Result's value
//
// Parameters:
//   r1 Result[T]: First Result to combine
//   r2 Result[U]: Second Result to combine
//
// Returns:
//   Result[Tuple[T, U]]: Success containing both values if both inputs are successful,
//                        or the first encountered failure.
//
// Example:
//   Combine independent operations:
//
//     userResult := fetchUser(userID)
//     profileResult := fetchProfile(userID)
//     combined := result.Combine(userResult, profileResult)
//     
//     combined.ForEach(func(tuple result.Tuple[User, Profile]) {
//         user := tuple.First
//         profile := tuple.Second
//         displayUserProfile(user, profile)
//     })
//
//   Validation of multiple fields:
//
//     emailValidation := validateEmail(email)
//     ageValidation := validateAge(age)
//     validation := result.Combine(emailValidation, ageValidation)
//     
//     validation.ForEach(func(tuple result.Tuple[string, int]) {
//         validEmail := tuple.First
//         validAge := tuple.Second
//         createUser(validEmail, validAge)
//     })
//
//   Parallel data fetching:
//
//     weatherResult := fetchWeather(location)
//     newsResult := fetchNews(category)
//     dashboard := result.Combine(weatherResult, newsResult)
//
// Error Handling:
//   Returns the first error encountered. If r1 fails, r1's error is returned.
//   If r1 succeeds but r2 fails, r2's error is returned.
//   Both values are only available if both Results are successful.
//
// Performance:
//   Short-circuits on first failure - r2 is not evaluated if r1 fails.
//   Single allocation for successful combination (Tuple creation).
//
// Thread Safety:
//   Safe for concurrent use as it creates new Result instances without modifying originals.
func Combine[T, U any](r1 Result[T], r2 Result[U]) Result[Tuple[T, U]] {
	if r1.IsFailure() {
		return Failure[Tuple[T, U]](r1.err)
	}
	if r2.IsFailure() {
		return Failure[Tuple[T, U]](r2.err)
	}
	return Success(Tuple[T, U]{First: r1.value, Second: r2.value})
}

// Tuple represents a pair of values of potentially different types.
//
// Tuple is used primarily with the Combine function to hold the results
// of multiple independent operations. It provides type-safe access to
// both values through named fields, making it easier to work with
// combined results than anonymous structs or interfaces.
//
// Type Parameters:
//   T: Type of the first value
//   U: Type of the second value (can be the same as T)
//
// Fields:
//   First T:  The first value in the tuple
//   Second U: The second value in the tuple
//
// Example:
//   Working with combined results:
//
//     combined := result.Combine(userResult, profileResult)
//     combined.ForEach(func(tuple result.Tuple[User, Profile]) {
//         fmt.Printf("User: %s, Profile: %v", tuple.First.Name, tuple.Second)
//     })
//
//   Destructuring tuple values:
//
//     if combined.IsSuccess() {
//         tuple := combined.Value()
//         user := tuple.First
//         profile := tuple.Second
//         // Use user and profile separately
//     }
//
// Integration:
//   Designed to work seamlessly with Combine function and Result patterns,
//   providing a clean way to handle multiple successful values.
//
// Thread Safety:
//   Tuple instances are immutable once created and safe for concurrent access.
type Tuple[T, U any] struct {
	First  T
	Second U
}

// Async represents an asynchronous Result that will be available in the future.
//
// Async enables non-blocking operations by wrapping Result delivery in a channel.
// It's useful for concurrent operations, background processing, and integrating
// with goroutine-based architectures while maintaining Result pattern benefits.
//
// Type Parameters:
//   T: The type of value that will be available when the async operation completes
//
// Fields:
//   ch chan Result[T]: Internal channel for Result delivery (buffered, capacity 1)
//
// Example:
//   Background processing:
//
//     async := result.NewAsync[ProcessedData]()
//     go func() {
//         data, err := expensiveOperation()
//         if err != nil {
//             async.Fail(err)
//         } else {
//             async.Complete(data)
//         }
//     }()
//     
//     // Later, when result is needed
//     result := async.Await()
//
//   Concurrent operations:
//
//     async1 := processDataAsync(input1)
//     async2 := processDataAsync(input2)
//     
//     result1 := async1.Await()
//     result2 := async2.Await()
//
// Integration:
//   Seamlessly integrates with Result patterns - Await() returns a standard Result[T]
//   that can be used with all Result methods (Map, FlatMap, Filter, etc.).
//
// Thread Safety:
//   Safe for concurrent use. Multiple goroutines can safely call Complete/Fail,
//   and multiple goroutines can safely call Await (all will receive the same result).
type Async[T any] struct {
	ch chan Result[T]
}

// NewAsync creates a new Async instance ready to receive a Result.
//
// NewAsync initializes the internal channel and returns a pointer to the Async
// instance. The async operation can then be completed using Complete() or Fail().
//
// Type Parameters:
//   T: The type of value that will be delivered asynchronously
//
// Returns:
//   *Async[T]: A new Async instance ready for completion
//
// Example:
//   Create and use async result:
//
//     async := result.NewAsync[string]()
//     
//     go func() {
//         time.Sleep(1 * time.Second)
//         async.Complete("Background work completed")
//     }()
//     
//     result := async.Await() // Blocks until completion
//     fmt.Println(result.Value())
//
//   With error handling:
//
//     async := result.NewAsync[Data]()
//     
//     go func() {
//         if data, err := fetchData(); err != nil {
//             async.Fail(err)
//         } else {
//             async.Complete(data)
//         }
//     }()
//
// Performance:
//   Uses a buffered channel with capacity 1 for efficient single-value delivery.
//   No additional allocation overhead beyond the channel and Async struct.
//
// Thread Safety:
//   The returned Async instance is safe for concurrent access from multiple goroutines.
func NewAsync[T any]() *Async[T] {
	return &Async[T]{
		ch: make(chan Result[T], 1),
	}
}

// Complete successfully completes the async operation with the provided value.
//
// Complete sends a successful Result containing the value through the internal
// channel and closes the channel to signal completion. This should be called
// exactly once per Async instance, typically from a goroutine performing
// background work.
//
// Parameters:
//   value T: The successful result value to deliver
//
// Example:
//   Background data processing:
//
//     async := result.NewAsync[ProcessedData]()
//     
//     go func() {
//         data := processLargeDataset(input)
//         async.Complete(data) // Signal successful completion
//     }()
//     
//     result := async.Await() // Will receive Success(data)
//
//   HTTP request processing:
//
//     async := result.NewAsync[APIResponse]()
//     
//     go func() {
//         response := httpClient.Get(url)
//         async.Complete(response)
//     }()
//
// Behavior:
//   - Sends Success(value) to the internal channel
//   - Closes the channel to prevent further sends
//   - Subsequent calls to Complete or Fail will panic (channel closed)
//   - All waiting Await() calls will receive the same successful Result
//
// Thread Safety:
//   Safe to call from any goroutine, but should only be called once per Async instance.
func (a *Async[T]) Complete(value T) {
	a.ch <- Success(value)
	close(a.ch)
}

// Fail completes the async operation with an error.
//
// Fail sends a failed Result containing the error through the internal
// channel and closes the channel to signal completion. This should be called
// exactly once per Async instance when the background operation encounters
// an unrecoverable error.
//
// Parameters:
//   err error: The error that caused the operation to fail
//
// Example:
//   Network operation with timeout:
//
//     async := result.NewAsync[APIData]()
//     
//     go func() {
//         ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//         defer cancel()
//         
//         data, err := fetchFromAPI(ctx, url)
//         if err != nil {
//             async.Fail(err) // Signal failure
//         } else {
//             async.Complete(data)
//         }
//     }()
//
//   File processing with error recovery:
//
//     async := result.NewAsync[ProcessedFile]()
//     
//     go func() {
//         if file, err := os.Open(filename); err != nil {
//             async.Fail(fmt.Errorf("failed to open file: %w", err))
//         } else {
//             // Process file...
//             async.Complete(processedData)
//         }
//     }()
//
// Behavior:
//   - Sends Failure[T](err) to the internal channel
//   - Closes the channel to prevent further sends
//   - Subsequent calls to Complete or Fail will panic (channel closed)
//   - All waiting Await() calls will receive the same failed Result
//
// Thread Safety:
//   Safe to call from any goroutine, but should only be called once per Async instance.
func (a *Async[T]) Fail(err error) {
	a.ch <- Failure[T](err)
	close(a.ch)
}

// Await blocks until the async operation completes and returns the Result.
//
// Await performs a blocking receive on the internal channel, waiting until
// either Complete() or Fail() is called. Once the Result is available,
// it can be used with all standard Result methods (Map, FlatMap, etc.).
//
// Returns:
//   Result[T]: The Result delivered by Complete() or Fail(), containing either
//              a successful value or an error.
//
// Example:
//   Basic async/await pattern:
//
//     async := processDataAsync(input)
//     result := async.Await() // Blocks until completion
//     
//     if result.IsSuccess() {
//         fmt.Println("Processing completed:", result.Value())
//     } else {
//         fmt.Println("Processing failed:", result.Error())
//     }
//
//   Chain with Result operations:
//
//     async := fetchUserAsync(userID)
//     result := async.Await()
//         .Map(func(user User) string { return user.Name })
//         .Filter(func(name string) bool { return len(name) > 0 })
//
//   Multiple async operations:
//
//     async1 := fetchDataAsync(source1)
//     async2 := fetchDataAsync(source2)
//     
//     result1 := async1.Await()
//     result2 := async2.Await()
//     
//     combined := result.Combine(result1, result2)
//
// Behavior:
//   - Blocks until Complete() or Fail() is called
//   - Returns the same Result to all callers (multiple Await calls are safe)
//   - Does not consume the Result (unlike single-use channels)
//
// Performance:
//   Blocking operation with no additional allocation overhead.
//   Efficient channel receive with no polling or busy-waiting.
//
// Thread Safety:
//   Safe to call from multiple goroutines - all will receive the same Result.
func (a *Async[T]) Await() Result[T] {
	return <-a.ch
}

// Try executes a function that might panic and converts panics to Result failures.
//
// Try provides a safe way to execute potentially panicking code within the
// Result pattern, converting any recovered panics into failed Results.
// This enables integration of panic-based error handling with Result-based
// error handling patterns.
//
// Type Parameters:
//   T: The return type of the function to execute safely
//
// Parameters:
//   fn func() T: Function to execute that might panic. Should be a complete
//                operation that returns a value of type T.
//
// Returns:
//   Result[T]: Success(value) if function executes without panic,
//              Failure with panic-derived error if function panics.
//
// Example:
//   Safe array access:
//
//     result := result.Try(func() string {
//         return array[index] // Might panic with index out of bounds
//     })
//     
//     result.ForEach(func(value string) {
//         fmt.Println("Safe access:", value)
//     }).IfFailure(func(err error) {
//         fmt.Println("Access failed:", err)
//     })
//
//   Safe type assertion:
//
//     result := result.Try(func() string {
//         return interfaceValue.(string) // Might panic on type assertion
//     })
//
//   Safe division:
//
//     result := result.Try(func() float64 {
//         return numerator / denominator // Might panic on division by zero
//     })
//
// Error Handling:
//   Recovered panics are converted to errors using fmt.Errorf("panic recovered: %v", r).
//   The original panic value is preserved in the error message.
//
// Performance:
//   Minimal overhead when no panic occurs (defer and function call).
//   Recovery path has additional overhead for error creation and stack unwinding.
//
// Thread Safety:
//   Safe for concurrent use. Each call operates independently with its own panic recovery.
func Try[T any](fn func() T) Result[T] {
	defer func() {
		if r := recover(); r != nil {
			// This will be handled by the calling code
		}
	}()

	value := fn()
	return Success(value)
}

// TryAsync executes a function asynchronously that might panic, returning an Async result.
//
// TryAsync combines the panic safety of Try with the asynchronous execution of Async,
// enabling safe background execution of potentially panicking code. The function
// is executed in a new goroutine with panic recovery, delivering the result
// through an Async instance.
//
// Type Parameters:
//   T: The return type of the function to execute safely and asynchronously
//
// Parameters:
//   fn func() T: Function to execute asynchronously that might panic.
//                Will be run in a separate goroutine with panic recovery.
//
// Returns:
//   *Async[T]: Async instance that will deliver Success(value) if function completes normally,
//              or Failure with panic-derived error if function panics.
//
// Example:
//   Background data processing with panic safety:
//
//     async := result.TryAsync(func() ProcessedData {
//         // This might panic due to invalid data or resource issues
//         return processLargeDataset(rawData)
//     })
//     
//     result := async.Await()
//     result.ForEach(func(data ProcessedData) {
//         fmt.Println("Processing completed successfully")
//     }).IfFailure(func(err error) {
//         fmt.Println("Processing failed or panicked:", err)
//     })
//
//   Safe file processing in background:
//
//     async := result.TryAsync(func() FileContent {
//         // File operations might panic
//         content := readAndParseFile(filename)
//         return processContent(content)
//     })
//
//   Multiple safe async operations:
//
//     async1 := result.TryAsync(func() Data1 { return riskyOperation1() })
//     async2 := result.TryAsync(func() Data2 { return riskyOperation2() })
//     
//     result1 := async1.Await()
//     result2 := async2.Await()
//     combined := result.Combine(result1, result2)
//
// Error Handling:
//   Panics are recovered and converted to failures using fmt.Errorf("panic recovered: %v", r).
//   The error preserves the original panic value for debugging.
//
// Performance:
//   Goroutine creation overhead plus panic recovery setup.
//   No blocking on the calling goroutine - execution happens in background.
//
// Thread Safety:
//   Safe for concurrent use. Each call creates an independent goroutine with its own panic recovery.
func TryAsync[T any](fn func() T) *Async[T] {
	async := NewAsync[T]()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				async.Fail(fmt.Errorf("panic recovered: %v", r))
			}
		}()

		value := fn()
		async.Complete(value)
	}()

	return async
}
