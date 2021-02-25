package server

/*
type Request struct {
	Id      string   `json:"id"`
	Command Commands `json:"command"`
	Method  Methods  `json:"method"`
	Data    *Data    `json:"data,omitempty"`
}

type Methods int

const (
	None Methods = iota
	Get
	Set
)

func (request *Request) Json(data interface{}) (*Request, error) {
	if data == nil {
		return request, nil
	}
	request.Data = &Data{
		ContentType: Json,
	}
	return request.marshal(data)
}

func (request *Request) Xml(data interface{}) (*Request, error) {
	if data == nil {
		return request, nil
	}
	request.Data = &Data{
		ContentType: Xml,
	}
	return request.marshal(data)
}

func (request *Request) Text(data interface{}) (*Request, error) {
	if data == nil {
		return request, nil
	}
	request.Data = &Data{
		ContentType: Text,
	}
	return request.marshal(data)
}

func (request *Request) marshal(data interface{}) (*Request, error) {
	tmp, err := request.Data.ContentType.Marshal(data)
	if err != nil {
		return nil, err
	}
	request.Data.Body = string(tmp)
	return request, nil
}

func (request *Request) String() string {
	data := "null"
	if request.Data != nil {
		data = request.Data.String()
	}
	return fmt.Sprintf("id: %s, command: %v, method: %v, data: %s", request.Id, request.Command, request.Method, data)
}*/
