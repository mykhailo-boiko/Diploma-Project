package httpresponse

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Data  any   `json:"data,omitempty"`
	Meta  *Meta `json:"meta,omitempty"`
	Error *Error `json:"error,omitempty"`
}

type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func OK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Response{Data: data})
}

func Created(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, Response{Data: data})
}

func List(w http.ResponseWriter, data any, total, limit, offset int) {
	writeJSON(w, http.StatusOK, Response{
		Data: data,
		Meta: &Meta{Total: total, Limit: limit, Offset: offset},
	})
}

func Err(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, Response{
		Error: &Error{Code: code, Message: message},
	})
}

func ErrWithData(w http.ResponseWriter, status int, code, message string, data map[string]any) {
	writeJSON(w, status, Response{
		Error: &Error{Code: code, Message: message, Data: data},
	})
}

func NotFound(w http.ResponseWriter, code, message string) {
	Err(w, http.StatusNotFound, code, message)
}

func BadRequest(w http.ResponseWriter, code, message string) {
	Err(w, http.StatusBadRequest, code, message)
}

func Forbidden(w http.ResponseWriter, code, message string) {
	Err(w, http.StatusForbidden, code, message)
}

func Unauthorized(w http.ResponseWriter, code, message string) {
	Err(w, http.StatusUnauthorized, code, message)
}

func InternalError(w http.ResponseWriter, code, message string) {
	Err(w, http.StatusInternalServerError, code, message)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
