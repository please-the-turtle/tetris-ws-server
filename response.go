package main

const (
	ResponseStatusOK           = "ok"
	ResponseStatusError        = "error"
	ResponseStatusLobbyCreated = "lobby-created"
)

type Response struct {
	Content string `json:"content"`
	Status  string `json:"status"`
	Error   string `json:"error"`
}

func NewResponse(status string, content string) *Response {
	return &Response{
		Content: content,
		Status:  status,
		Error:   "",
	}
}

func NewErrorResponse(content string, err string) *Response {
	return &Response{
		Content: content,
		Status:  ResponseStatusError,
		Error:   err,
	}
}

func (e *Response) Encode() []byte {
	return EncodeToJSON(e)
}
