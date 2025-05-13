package gui

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type RunServerRequest struct {
	Name string `json:"name"`
}

type CustomCommandRequest struct {
	Command string `json:"command"`
}

type ServerInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	State     string `json:"state"`
	Transport string `json:"transport"`
	ToolType  string `json:"tool_type,omitempty"`
	Port      int    `json:"port"`
	URL       string `json:"url"`
}

func NewSuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

func NewErrorResponse(err error) Response {
	return Response{
		Success: false,
		Error:   err.Error(),
	}
}
