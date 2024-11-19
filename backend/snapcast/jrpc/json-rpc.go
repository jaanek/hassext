package jrpc

import (
	"fmt"

	"github.com/jaanek/hassext/data"
	"github.com/ohler55/ojg/oj"
)

// https://github.com/badaix/snapcast/blob/develop/doc/json_rpc_api/control.md

var requestId = 0

func newId() int {
	requestId += 1
	return requestId
}

func RpcParseResult(body []byte) (result data.DataValue, error error) {
	parsed, error := oj.Parse(body)
	result = data.NewDataValue(parsed)
	if error != nil {
		return
	}
	errMsg := result.GetString("$.error.message", nil)
	if errMsg != "" {
		error = fmt.Errorf("%s", errMsg)
	}
	return
}

func NewCall(method string) Call {
	return Call{
		Id:      newId(),
		JsonRpc: "2.0",
		Method:  method,
	}
}

func NewClientGetStatus(clientId string) ClientGetStatusCall {
	return ClientGetStatusCall{
		Call: NewCall("Client.GetStatus"),
		Params: ParamsId{
			Id: clientId,
		},
	}
}

func NewClientSetVolume(clientId string, muted bool, percent int) ClientSetVolumeCall {
	return ClientSetVolumeCall{
		Call: NewCall("Client.SetVolume"),
		Params: ClientSetVolume{
			Id: clientId,
			Volume: ClientVolume{
				Muted:   muted,
				Percent: percent,
			},
		},
	}
}

func NewGroupGetStatus(groupId string) GroupGetStatusCall {
	return GroupGetStatusCall{
		Call: NewCall("Group.GetStatus"),
		Params: ParamsId{
			Id: groupId,
		},
	}
}

func NewGroupSetName(groupId string, name string) GroupSetNameCall {
	return GroupSetNameCall{
		Call: NewCall("Group.SetName"),
		Params: GroupSetName{
			Id:   groupId,
			Name: name,
		},
	}
}

func NewGroupSetStream(groupId string, streamId string) GroupSetStreamCall {
	return GroupSetStreamCall{
		Call: NewCall("Group.SetStream"),
		Params: GroupSetStream{
			Id:       groupId,
			StreamId: streamId,
		},
	}
}

func NewGroupSetMute(groupId string, mute bool) GroupSetMuteCall {
	return GroupSetMuteCall{
		Call: NewCall("Group.SetMute"),
		Params: GroupSetMute{
			Id:   groupId,
			Mute: mute,
		},
	}
}

type Call struct {
	Id      int    `json:"id"`
	JsonRpc string `json:"jsonrpc"`
	Method  string `json:"method"`
}

type ParamsId struct {
	Id string `json:"id"`
}

type ClientGetStatusCall struct {
	Call
	Params ParamsId `json:"params"`
}

type (
	ClientSetVolumeCall struct {
		Call
		Params ClientSetVolume `json:"params"`
	}
	ClientSetVolume struct {
		Id     string       `json:"id"`
		Volume ClientVolume `json:"volume"`
	}
	ClientVolume struct {
		Muted   bool `json:"muted"`
		Percent int  `json:"percent"`
	}
)

type GroupGetStatusCall struct {
	Call
	Params ParamsId `json:"params"`
}

type (
	GroupSetNameCall struct {
		Call
		Params GroupSetName `json:"params"`
	}
	GroupSetName struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}
)

type (
	GroupSetStreamCall struct {
		Call
		Params GroupSetStream `json:"params"`
	}
	GroupSetStream struct {
		Id       string `json:"id"`
		StreamId string `json:"stream_id"`
	}
)

type (
	GroupSetMuteCall struct {
		Call
		Params GroupSetMute `json:"params"`
	}
	GroupSetMute struct {
		Id   string `json:"id"`
		Mute bool   `json:"mute"`
	}
)
