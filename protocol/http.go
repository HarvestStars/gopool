package protocol

// Req is for lavad
type Req struct {
	JSONRPC string        `json:"jsonrpc" form:"jsonrpc"`
	ID      string        `json:"id" form:"id"`
	Method  string        `json:"method" form:"method"`
	Params  []interface{} `json:"params" form:"params"`
}

// Resp ...
type Resp struct {
	Result interface{} `json:"result" form:"result"`
	Err    error       `json:"error" form:"error"`
	ID     string      `json:"id" form:"id"`
}

// Accept ...
type Accept struct {
	Accept bool `json:"accept"`
}
