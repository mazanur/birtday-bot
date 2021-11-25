package handler

import (
	"context"
	"github.com/rs/zerolog/log"
	"runtime/debug"
)

type botError struct {
	cause error
	msg   string
}

type ErrorHandler struct {
	errorChanel chan botError
}

func NewErrorHandler() *ErrorHandler {
	errors := make(chan botError)
	return &ErrorHandler{errors}
}

func (h *ErrorHandler) HandleError(err error) {
	h.errorChanel <- botError{cause: err}
}

func (h *ErrorHandler) HandleErrorWithMsg(err error, msg string) {
	h.errorChanel <- botError{cause: err, msg: msg}
}

func (h *ErrorHandler) Do(ctx context.Context) {
	for {
		select {

		case <-ctx.Done():
			return

		case err, ok := <-h.errorChanel:
			if !ok {
				return
			}
			log.Error().Err(err.cause).Stack().Msgf("Error bot err: %v, msg: %s,\n stack: %s", err.cause, err.msg, string(debug.Stack()))
		}
	}
}
