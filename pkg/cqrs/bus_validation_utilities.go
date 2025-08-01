// Package cqrs provides CQRS utilities for command and query bus validation
// DRY PRINCIPLE: Eliminates 27-line duplication (mass=141) between command_bus.go and query_bus.go
package cqrs

import (
	"context"
	"reflect"

	"github.com/flext/flexcore/pkg/result"
	"github.com/flext/flexcore/pkg/errors"
)

// BusValidationUtilities provides shared validation logic for CQRS buses
// DRY PRINCIPLE: Eliminates massive duplication between command and query buses
type BusValidationUtilities struct{}

// NewBusValidationUtilities creates new bus validation utilities
func NewBusValidationUtilities() *BusValidationUtilities {
	return &BusValidationUtilities{}
}

// IsValidCommandHandler checks if a type implements the CommandHandler interface
// DRY PRINCIPLE: Eliminates 27-line duplication from command_bus.go:108-134
func (utils *BusValidationUtilities) IsValidCommandHandler(handlerType reflect.Type) bool {
	return utils.isValidHandler(handlerType, "Command")
}

// IsValidQueryHandler checks if a type implements the QueryHandler interface  
// DRY PRINCIPLE: Eliminates 27-line duplication from query_bus.go:95-121
func (utils *BusValidationUtilities) IsValidQueryHandler(handlerType reflect.Type) bool {
	return utils.isValidHandler(handlerType, "Query")
}

// isValidHandler checks if a type implements a Handler interface (Command or Query)
// DRY PRINCIPLE: Shared implementation eliminating 27 lines of duplicate validation logic
// SOLID SRP: Reduced from 6 returns to 1 return using Result pattern and ValidationOrchestrator
func (utils *BusValidationUtilities) isValidHandler(handlerType reflect.Type, interfaceType string) bool {
	orchestrator := utils.createValidationOrchestrator(handlerType, interfaceType)
	validationResult := orchestrator.ValidateHandlerInterface()
	
	if validationResult.IsFailure() {
		return false
	}

	return validationResult.Value()
}

// HandlerValidationOrchestrator handles complete handler validation with Result pattern
// SOLID SRP: Single responsibility for handler validation orchestration
type HandlerValidationOrchestrator struct {
	handlerType   reflect.Type
	interfaceType string
}

// createValidationOrchestrator creates a specialized validation orchestrator
// SOLID SRP: Factory method for creating specialized orchestrators
func (utils *BusValidationUtilities) createValidationOrchestrator(handlerType reflect.Type, interfaceType string) *HandlerValidationOrchestrator {
	return &HandlerValidationOrchestrator{
		handlerType:   handlerType,
		interfaceType: interfaceType,
	}
}

// ValidateHandlerInterface validates handler interface with centralized error handling
// SOLID SRP: Single responsibility for complete handler interface validation
func (orchestrator *HandlerValidationOrchestrator) ValidateHandlerInterface() result.Result[bool] {
	// Phase 1: Check Handle method exists
	methodResult := orchestrator.validateHandleMethodExists()
	if methodResult.IsFailure() {
		return result.Failure[bool](methodResult.Error())
	}

	method := methodResult.Value()

	// Phase 2: Validate method signature
	signatureResult := orchestrator.validateMethodSignature(method.Type)
	if signatureResult.IsFailure() {
		return result.Failure[bool](signatureResult.Error())
	}

	methodType := method.Type

	// Phase 3: Validate context parameter
	contextResult := orchestrator.validateContextParameter(methodType)
	if contextResult.IsFailure() {
		return result.Failure[bool](contextResult.Error())
	}

	// Phase 4: Validate command/query parameter
	parameterResult := orchestrator.validateCommandQueryParameter(methodType)
	if parameterResult.IsFailure() {
		return result.Failure[bool](parameterResult.Error())
	}

	return result.Success(true)
}

// validateHandleMethodExists validates Handle method exists
// SOLID SRP: Single responsibility for method existence validation
func (orchestrator *HandlerValidationOrchestrator) validateHandleMethodExists() result.Result[reflect.Method] {
	method, exists := orchestrator.handlerType.MethodByName("Handle")
	if !exists {
		return result.Failure[reflect.Method](errors.ValidationError("handler does not have Handle method"))
	}
	return result.Success(method)
}

// validateMethodSignature validates method signature correctness
// SOLID SRP: Single responsibility for method signature validation
func (orchestrator *HandlerValidationOrchestrator) validateMethodSignature(methodType reflect.Type) result.Result[bool] {
	if methodType.NumIn() != 3 || methodType.NumOut() != 1 {
		return result.Failure[bool](errors.ValidationError("Handle method must have signature: Handle(ctx context.Context, cmd/query T) result.Result[R]"))
	}
	return result.Success(true)
}

// validateContextParameter validates context parameter type
// SOLID SRP: Single responsibility for context parameter validation
func (orchestrator *HandlerValidationOrchestrator) validateContextParameter(methodType reflect.Type) result.Result[bool] {
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !methodType.In(1).Implements(contextType) {
		return result.Failure[bool](errors.ValidationError("first parameter must implement context.Context"))
	}
	return result.Success(true)
}

// validateCommandQueryParameter validates command/query parameter interface
// SOLID SRP: Single responsibility for command/query parameter validation
func (orchestrator *HandlerValidationOrchestrator) validateCommandQueryParameter(methodType reflect.Type) result.Result[bool] {
	interfaceResult := orchestrator.getTargetInterface()
	if interfaceResult.IsFailure() {
		return result.Failure[bool](interfaceResult.Error())
	}

	targetInterface := interfaceResult.Value()
	if !methodType.In(2).Implements(targetInterface) {
		return result.Failure[bool](errors.ValidationError("second parameter must implement " + orchestrator.interfaceType + "Type() method"))
	}

	return result.Success(true)
}

// getTargetInterface gets the target interface type based on interface type
// SOLID SRP: Single responsibility for target interface resolution
func (orchestrator *HandlerValidationOrchestrator) getTargetInterface() result.Result[reflect.Type] {
	switch orchestrator.interfaceType {
	case "Command":
		targetInterface := reflect.TypeOf((*interface{ CommandType() string })(nil)).Elem()
		return result.Success(targetInterface)
	case "Query":
		targetInterface := reflect.TypeOf((*interface{ QueryType() string })(nil)).Elem()
		return result.Success(targetInterface)
	default:
		return result.Failure[reflect.Type](errors.ValidationError("unsupported interface type: " + orchestrator.interfaceType))
	}
}

// ValidateHandlerRegistration performs shared validation for handler registration
// DRY PRINCIPLE: Eliminates duplicate validation logic in RegisterHandler methods
func (utils *BusValidationUtilities) ValidateHandlerRegistration(
	entity interface{}, 
	handler interface{}, 
	entityTypeName string,
) error {
	if entity == nil {
		return errors.ValidationError(entityTypeName + " cannot be nil")
	}

	if handler == nil {
		return errors.ValidationError("handler cannot be nil")
	}

	// Validate entity type name
	switch entityTypeName {
	case "command", "query":
		// Valid entity types
	default:
		return errors.ValidationError("unsupported entity type: " + entityTypeName)
	}

	// Validate handler implements appropriate interface
	handlerType := reflect.TypeOf(handler)
	var isValid bool
	switch entityTypeName {
	case "command":
		isValid = utils.IsValidCommandHandler(handlerType)
	case "query":
		isValid = utils.IsValidQueryHandler(handlerType)
	}

	if !isValid {
		return errors.ValidationError("handler must implement " + entityTypeName + "Handler interface")
	}

	return nil
}

// GetEntityType extracts type string from command or query using reflection
// DRY PRINCIPLE: Eliminates duplicate type extraction logic
func (utils *BusValidationUtilities) GetEntityType(entity interface{}) (string, error) {
	if entity == nil {
		return "", errors.ValidationError("entity cannot be nil")
	}

	entityValue := reflect.ValueOf(entity)
	
	// Try CommandType() method first
	if method := entityValue.MethodByName("CommandType"); method.IsValid() {
		results := method.Call(nil)
		if len(results) == 1 {
			if str, ok := results[0].Interface().(string); ok {
				return str, nil
			}
		}
	}

	// Try QueryType() method second
	if method := entityValue.MethodByName("QueryType"); method.IsValid() {
		results := method.Call(nil)
		if len(results) == 1 {
			if str, ok := results[0].Interface().(string); ok {
				return str, nil
			}
		}
	}

	return "", errors.ValidationError("entity must implement CommandType() or QueryType() method")
}

// ValidateEntityForExecution validates entity before execution
// DRY PRINCIPLE: Eliminates duplicate validation in Execute methods
func (utils *BusValidationUtilities) ValidateEntityForExecution(
	entity interface{}, 
	entityTypeName string,
) error {
	if entity == nil {
		return errors.ValidationError(entityTypeName + " cannot be nil")
	}

	// Validate entity has correct type method
	_, err := utils.GetEntityType(entity)
	if err != nil {
		return errors.Wrap(err, "invalid "+entityTypeName)
	}

	return nil
}

// CheckHandlerExists validates handler exists in registry
// DRY PRINCIPLE: Eliminates duplicate handler existence checking
func (utils *BusValidationUtilities) CheckHandlerExists(
	handlers map[string]interface{},
	entityType string,
	entityTypeName string,
) (interface{}, error) {
	handler, exists := handlers[entityType]
	if !exists {
		return nil, errors.NotFoundError("handler for " + entityTypeName + " type " + entityType)
	}
	return handler, nil
}

// BusRegistrationResult represents result of handler registration
type BusRegistrationResult struct {
	EntityType string
	Success    bool
	Error      error
}

// BusExecutionContext provides context for bus operations
type BusExecutionContext struct {
	EntityType string
	Handler    interface{}
	Entity     interface{}
}

// CreateExecutionContext creates execution context for bus operations
// DRY PRINCIPLE: Standardizes execution context creation
func (utils *BusValidationUtilities) CreateExecutionContext(
	entity interface{},
	handler interface{},
	entityTypeName string,
) (*BusExecutionContext, error) {
	entityType, err := utils.GetEntityType(entity)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get "+entityTypeName+" type")
	}

	return &BusExecutionContext{
		EntityType: entityType,
		Handler:    handler,
		Entity:     entity,
	}, nil
}

// RegisterHandlerGeneric provides shared handler registration logic for CQRS buses
// DRY PRINCIPLE: Eliminates 21-line duplication (mass=117) between command_bus.go and query_bus.go RegisterHandler methods
func (utils *BusValidationUtilities) RegisterHandlerGeneric(
	entity interface{},
	handler interface{},
	entityTypeName string,
	handlers map[string]interface{},
	handlerErrorPrefix string,
) error {
	// DRY: Use shared validation logic
	if err := utils.ValidateHandlerRegistration(entity, handler, entityTypeName); err != nil {
		return err
	}

	entityType, err := utils.GetEntityType(entity)
	if err != nil {
		return errors.Wrap(err, "failed to get "+entityTypeName+" type")
	}

	// Check for existing handler
	if _, exists := handlers[entityType]; exists {
		return errors.AlreadyExistsError("handler for " + entityTypeName + " type " + entityType)
	}

	// Register the handler
	handlers[entityType] = handler
	return nil
}

// ExecuteGeneric provides shared execution logic for CQRS buses
// DRY PRINCIPLE: Eliminates 21-line duplication (mass=148) between command_bus.go and query_bus.go Execute methods
func (utils *BusValidationUtilities) ExecuteGeneric(
	ctx context.Context,
	entity interface{},
	entityTypeName string,
	handlers map[string]interface{},
	invoker *HandlerInvoker,
) result.Result[interface{}] {
	// DRY: Use shared validation logic
	if err := utils.ValidateEntityForExecution(entity, entityTypeName); err != nil {
		return result.Failure[interface{}](err)
	}

	entityType, err := utils.GetEntityType(entity)
	if err != nil {
		return result.Failure[interface{}](errors.Wrap(err, "failed to get "+entityTypeName+" type"))
	}

	// DRY: Use shared handler existence check
	handler, err := utils.CheckHandlerExists(handlers, entityType, entityTypeName)
	if err != nil {
		return result.Failure[interface{}](err)
	}

	// DRY: Use shared handler invocation
	invocationResult := invoker.InvokeHandler(ctx, handler, entity, entityTypeName)
	return invocationResult.ToResult()
}

// HandlerInvoker provides shared reflection-based handler invocation for CQRS buses
// DRY PRINCIPLE: Eliminates 21-line duplication between command and query buses
type HandlerInvoker struct {
	validationUtils *BusValidationUtilities
}

// NewHandlerInvoker creates a new handler invoker
func NewHandlerInvoker() *HandlerInvoker {
	return &HandlerInvoker{
		validationUtils: NewBusValidationUtilities(),
	}
}

// HandlerInvocationResult represents the result of handler invocation
type HandlerInvocationResult struct {
	Value interface{}
	Error error
}

// InvokeHandler invokes a handler using reflection with shared logic
// DRY PRINCIPLE: Eliminates duplicate reflection code in command_bus.go and query_bus.go
func (invoker *HandlerInvoker) InvokeHandler(
	ctx context.Context,
	handler interface{},
	entity interface{},
	handlerType string,
) HandlerInvocationResult {
	// Validate handler has Handle method
	handlerValue := reflect.ValueOf(handler)
	handlerTypeReflect := reflect.TypeOf(handler)

	method, exists := handlerTypeReflect.MethodByName("Handle")
	if !exists {
		return HandlerInvocationResult{
			Error: errors.InternalError("handler does not have Handle method"),
		}
	}

	// Prepare arguments for reflection call
	args := []reflect.Value{
		handlerValue,
		reflect.ValueOf(ctx),
		reflect.ValueOf(entity),
	}

	// Call the handler method
	results := method.Func.Call(args)
	if len(results) != 1 {
		return HandlerInvocationResult{
			Error: errors.InternalError("handler returned unexpected number of values"),
		}
	}

	// Extract and process the result
	resultValue := results[0].Interface()
	return invoker.processHandlerResult(resultValue, handlerType)
}

// processHandlerResult processes different types of handler results
// SOLID SRP: Single responsibility for result processing
func (invoker *HandlerInvoker) processHandlerResult(resultValue interface{}, handlerType string) HandlerInvocationResult {
	switch v := resultValue.(type) {
	case result.Result[interface{}]:
		// Already a Result type, return as-is
		return HandlerInvocationResult{Value: v}
	case error:
		// Handle error returns from handlers
		if v != nil {
			return HandlerInvocationResult{Error: v}
		}
		// nil error means success with no value
		return HandlerInvocationResult{Value: result.Success[interface{}](nil)}
	default:
		// For commands: expect Result type, anything else is error
		if handlerType == "command" {
			return HandlerInvocationResult{
				Error: errors.InternalError("command handler returned unexpected result type"),
			}
		}
		// For queries: wrap non-Result values as Success
		return HandlerInvocationResult{Value: result.Success[interface{}](v)}
	}
}

// ToResult converts HandlerInvocationResult to result.Result[interface{}]
// SOLID OCP: Open for extension with different result types
func (hir HandlerInvocationResult) ToResult() result.Result[interface{}] {
	if hir.Error != nil {
		return result.Failure[interface{}](hir.Error)
	}
	if resultValue, ok := hir.Value.(result.Result[interface{}]); ok {
		return resultValue
	}
	return result.Success[interface{}](hir.Value)
}