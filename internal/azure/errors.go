package azure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type ErrorClass string

const (
	ErrorUnknown             ErrorClass = "unknown"
	ErrorAuthentication      ErrorClass = "authentication"
	ErrorPermissionDenied    ErrorClass = "permission_denied"
	ErrorMarketplaceRequired ErrorClass = "marketplace_required"
	ErrorNotFound            ErrorClass = "not_found"
	ErrorValidation          ErrorClass = "validation"
)

type OperationError struct {
	Class     ErrorClass `json:"class"`
	Operation string     `json:"operation"`
	Scope     string     `json:"scope"`
	Code      string     `json:"code"`
	Retryable bool       `json:"retryable"`
	RequestID string     `json:"requestId"`
	Message   string     `json:"message"`
}

func (e *OperationError) Error() string {
	if e.RequestID == "" {
		return fmt.Sprintf("%s: %s", e.Operation, e.Message)
	}
	return fmt.Sprintf("%s: %s (request ID %s)", e.Operation, e.Message, e.RequestID)
}

func ClassifyError(err error, operation, scope string) *OperationError {
	if err == nil {
		return nil
	}
	result := &OperationError{
		Class:     ErrorUnknown,
		Operation: operation,
		Scope:     scope,
		Message:   "Azure operation failed.",
	}

	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) {
		result.Code = responseErr.ErrorCode
		if responseErr.RawResponse != nil {
			result.RequestID = responseErr.RawResponse.Header.Get("x-ms-request-id")
		}
		result.Retryable = responseErrorRetryable(responseErr)
		switch responseErr.StatusCode {
		case 401:
			result.Class = ErrorAuthentication
			result.Message = "Azure sign-in is no longer valid. Sign in again."
		case 403:
			result.Class = ErrorPermissionDenied
			result.Message = "The signed-in account is not authorized for this Azure operation."
		case 404:
			result.Class = ErrorNotFound
			result.Message = "The selected Azure resource was not found. Refresh the selection."
		case 408, 429:
			result.Message = "Azure temporarily rejected this operation. Retry it after the service becomes available."
		default:
			if responseErr.StatusCode >= 500 {
				result.Message = "Azure temporarily failed this operation. Retry it later."
			} else if responseErr.StatusCode >= 400 && responseErr.StatusCode < 500 {
				result.Class = ErrorValidation
				result.Message = "Azure rejected the request as invalid. Review the selected resource and deployment."
			} else {
				result.Message = "Azure returned an error for this operation."
			}
		}
	}
	var authenticationErr *azidentity.AuthenticationFailedError
	if errors.As(err, &authenticationErr) {
		result.Class = ErrorAuthentication
		result.Message = "Azure sign-in is no longer valid. Sign in again."
		return result
	}

	text := strings.ToLower(err.Error())
	if strings.Contains(text, "marketplace") || strings.Contains(text, "purchase") || strings.Contains(text, "agreement") {
		result.Class = ErrorMarketplaceRequired
		result.Message = "This model requires an existing Azure Marketplace agreement. Bridge does not purchase or sign agreements."
	}
	if strings.Contains(text, "invalid") || strings.Contains(text, "validation") {
		result.Class = ErrorValidation
		result.Message = "Azure rejected the request as invalid. Review the selected resource and deployment."
	}
	return result
}

func responseErrorRetryable(err *azcore.ResponseError) bool {
	if err == nil {
		return false
	}
	if err.StatusCode == 408 || err.StatusCode == 429 || err.StatusCode >= 500 {
		return true
	}
	code := strings.ToLower(err.ErrorCode)
	return strings.Contains(code, "operationinprogress") ||
		strings.Contains(code, "anotheroperationinprogress") ||
		strings.Contains(code, "temporarilyunavailable") ||
		strings.Contains(code, "throttl")
}

func RequireOperation(err error, operation, scope string) error {
	if err == nil {
		return nil
	}
	return ClassifyError(err, operation, scope)
}
