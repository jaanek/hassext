package snapcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/httpclient"
	"github.com/jaanek/hassext/snapcast/jrpc"
	"github.com/zerodha/logf"
)

type Snapcast interface {
	RpcCall(any) ([]byte, error)
	GroupSetName(string, string) error
	ClientGetStatus(clientId string) (*Client, error)
	ClientIncVolume(clientId string, incStep int) error
	ClientChangeStream(clientId string, up bool) error
	SendRequest(any)
	Run(context.Context)
}

type Group struct {
	Id       string
	Name     string
	Muted    bool
	StreamId string
}

type Client struct {
	Group     *Group
	Id        string
	Connected bool
	Muted     bool
	Volume    int64
}

type Stream struct {
	Id     string
	Status string
}

type ServerStatus struct {
	Groups         map[string]*Group
	Clients        map[string]*Client
	Streams        []Stream
	PlayingStreams []Stream
}

type snapcast struct {
	lo       logf.Logger
	http     httpclient.HttpClient
	params   *HttpClientParams
	errors   chan error
	requests chan any
	Status   ServerStatus
}

type HttpClientParams struct {
	ApiUrl string
}

func New(lo logf.Logger, params *HttpClientParams) Snapcast {
	return &snapcast{
		lo:       lo,
		http:     httpclient.New(getApiDefaultRetryCheckPolicy(lo, params), defaultRetryWaitDelay),
		params:   params,
		errors:   make(chan error, 10),
		requests: make(chan any, 10),
		Status: ServerStatus{
			Groups:         map[string]*Group{},
			Clients:        map[string]*Client{},
			Streams:        make([]Stream, 0),
			PlayingStreams: make([]Stream, 0),
		},
	}
}

func (s *snapcast) RpcCall(args any) ([]byte, error) {
	params, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	body, err := s.Post(s.params.ApiUrl+"/jsonrpc", params, httpclient.HttpReqCallback, httpclient.HttpRespCallback)
	return body, err
}

func (s *snapcast) GroupGetStatus(groupId string) (*Group, error) {
	body, err := s.RpcCall(jrpc.NewGroupGetStatus(groupId))
	if err != nil {
		return nil, err
	}
	result, err := jrpc.RpcParseResult(body)
	if err != nil {
		return nil, err
	}
	errors := &data.Errors{}
	id := result.GetString("$.result.group.id", errors)
	muted := result.GetBool("$.result.group.muted", errors)
	name := result.GetString("$.result.group.name", errors)
	streamId := result.GetString("$.result.group.stream_id", errors)
	if errors.HasAny() {
		return nil, fmt.Errorf(errors.Error())
	}
	s.lo.Info("GroupGetStatus success", "result", string(body))
	return &Group{
		Id:       *id,
		Name:     *name,
		Muted:    *muted,
		StreamId: *streamId,
	}, nil
}

func (s *snapcast) GroupSetName(groupId, name string) error {
	body, err := s.RpcCall(jrpc.NewGroupSetName(groupId, name))
	if err != nil {
		return err
	}
	_, err = jrpc.RpcParseResult(body)
	if err != nil {
		return err
	}
	s.lo.Info("GroupSetName success", "result", string(body))
	return nil
}

func (s *snapcast) ClientGetStatus(clientId string) (*Client, error) {
	body, err := s.RpcCall(jrpc.NewClientGetStatus(clientId))
	if err != nil {
		return nil, err
	}
	result, err := jrpc.RpcParseResult(body)
	if err != nil {
		return nil, err
	}
	errors := &data.Errors{}
	connected := result.GetBool("$.result.client.connected", errors)
	muted := result.GetBool("$.result.client.config.volume.muted", errors)
	volume := result.GetInt64("$.result.client.config.volume.percent", errors)
	if errors.HasAny() {
		return nil, fmt.Errorf(errors.Error())
	}
	s.lo.Info("ClientGetStatus success", "result", string(body))
	return &Client{
		Id:        clientId,
		Connected: *connected,
		Muted:     *muted,
		Volume:    *volume,
	}, nil
}

func (s *snapcast) ClientIncVolume(clientId string, incStep int) error {
	client, err := s.ClientGetStatus(clientId)
	if err != nil {
		return err
	}
	muted := client.Muted
	volume := client.Volume + int64(incStep)
	if volume > 100 {
		return fmt.Errorf("volume > 100 not allowed")
	} else if volume < 0 {
		return fmt.Errorf("volume < 0 not allowed")
	}

	// increase volume
	body, err := s.RpcCall(jrpc.NewClientSetVolume(clientId, muted, int(volume)))
	if err != nil {
		return err
	}
	_, err = jrpc.RpcParseResult(body)
	if err != nil {
		return err
	}
	s.lo.Info("ClientIncVolume success", "result", string(body))
	return nil
}

func (s *snapcast) ClientChangeStream(clientId string, up bool) error {
	client, err := s.ClientGetStatus(clientId)
	if err != nil {
		return err
	}
	muted := client.Muted
	volume := client.Volume

	// unmute the client if muted
	if muted {
		body, err := s.RpcCall(jrpc.NewClientSetVolume(clientId, false, int(volume)))
		if err != nil {
			return err
		}
		_, err = jrpc.RpcParseResult(body)
		if err != nil {
			return err
		}
		s.lo.Info("ClientUnmuted success", "result", string(body))
		return nil
	}

	// change the group stream
	cachedClient := s.Status.Clients[clientId]
	if cachedClient == nil {
		return fmt.Errorf("No cached client found! Client id: %s", clientId)
	}
	cachedGroup := cachedClient.Group
	group, err := s.GroupGetStatus(cachedGroup.Id)
	if err != nil {
		return err
	}
	streams := s.Status.PlayingStreams

	// find the next stream id to set
	idx := 0
	for i := 0; i < len(streams); i++ {
		stream := streams[i]
		if group.StreamId == stream.Id {
			idx = i
		}
	}
	if up {
		if idx+1 >= len(streams) {
			idx = 0
		} else {
			idx += 1
		}
	} else {
		if idx-1 < 0 {
			idx = len(streams) - 1
		} else {
			idx -= 1
		}
	}
	nextStream := streams[idx]
	body, err := s.RpcCall(jrpc.NewGroupSetStream(group.Id, nextStream.Id))
	if err != nil {
		return err
	}
	_, err = jrpc.RpcParseResult(body)
	if err != nil {
		return err
	}
	s.lo.Info("GroupSetStream success", "result", string(body))
	return nil
}

func (s *snapcast) SendRequest(req any) {
	s.requests <- req
}

func (s *snapcast) Run(ctx context.Context) {
	// log errors if they happen
	go func() {
		for {
			select {
			case err := <-s.errors:
				s.lo.Error("Error while fetching snapcast status data", "error", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// start fetching data
	ticker := time.NewTicker(30 * time.Second)
	for {
		// fetch data
		result, err := s.fetchServerStatus()
		if err != nil {
			werr := fmt.Errorf("fetch error %w", err)
			s.errors <- werr
			s.lo.Error("snapcast", "error", werr)
		} else {
			s.parseServerStatus(result)
		}

		// wait next tick
		select {
		case action := <-s.requests:
			err := s.processRequest(action)
			if err != nil {
				s.lo.Error("Processing queue request", "error", err)
			}
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (s *snapcast) fetchServerStatus() (*data.Data, error) {
	body, err := s.RpcCall(jrpc.NewCall("Server.GetStatus"))
	result, err := jrpc.RpcParseResult(body)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *snapcast) parseServerStatus(result *data.Data) {
	// clear the status
	s.Status.Groups = make(map[string]*Group)
	s.Status.Clients = make(map[string]*Client)
	s.Status.Streams = make([]Stream, 0)
	s.Status.PlayingStreams = make([]Stream, 0)

	// parse groups and clients
	groupsArr := result.GetArray("$.result.server.groups.*")
	if groupsArr != nil {
		// s.lo.Info("Data", "length", len(groupsArr), "groups", groupsArr)
		for _, groupMap := range groupsArr {
			// parse data
			groupData := data.Data{Value: groupMap}
			groupErrors := &data.Errors{}
			id := groupData.GetString("$.id", groupErrors)
			streamId := groupData.GetString("$.stream_id", groupErrors)
			groupMuted := groupData.GetBool("$.muted", groupErrors)
			groupName := groupData.GetString("$.name", groupErrors)
			clientIds := groupData.GetArray("$.clients[*].id")
			clientConnectedArr := groupData.GetArray("$.clients[*].connected")
			clientMutedArr := groupData.GetArray("$.clients[*].config.volume.muted")
			clientVolumeArr := groupData.GetArray("$.clients[*].config.volume.percent")
			// s.lo.Info("Group", "id", id, "stream_id", streamId, "clientIds", clientIds, "muted", clientMutedArr, "volume", clientVolumeArr)
			if groupErrors.HasAny() {
				s.lo.Error("Error while parsing group! ", "errors", groupErrors.Error())
			} else {
				// s.lo.Info("Group", "id", *id, "stream_id", *streamId, "clientIds", clientIds, "muted", clientMutedArr, "volume", clientVolumeArr)
				// set group
				group := &Group{
					Id:       *id,
					Name:     *groupName,
					Muted:    *groupMuted,
					StreamId: *streamId,
				}
				s.Status.Groups[*id] = group
				// s.lo.Info("Parsed", "group", group)

				// set clients
				for i := 0; i < len(clientIds); i++ {
					clientErrors := data.Errors{}
					cid, ok := clientIds[i].(string)
					if !ok {
						clientErrors = append(clientErrors, fmt.Errorf("client id is not string"))
					}
					connected, ok := clientConnectedArr[i].(bool)
					if !ok {
						clientErrors = append(clientErrors, fmt.Errorf("client connected is not bool"))
					}
					muted, ok := clientMutedArr[i].(bool)
					if !ok {
						clientErrors = append(clientErrors, fmt.Errorf("client muted is not bool"))
					}
					volume, ok := clientVolumeArr[i].(int64)
					if !ok {
						clientErrors = append(clientErrors, fmt.Errorf("volume muted is not int"))
					}
					s.Status.Clients[cid] = &Client{
						Group:     group,
						Id:        cid,
						Connected: connected,
						Muted:     muted,
						Volume:    volume,
					}
					if clientErrors.HasAny() {
						s.lo.Error("Client parsing", "errors", clientErrors.Error())
					}
					// s.lo.Info("Parsed", "client", s.Status.Clients[cid])
				}
			}
		}
	}

	// get streams
	streamIdsArr := result.GetArray("$.result.server.streams[*].id")
	streamStatusArr := result.GetArray("$.result.server.streams[*].status")
	if streamIdsArr != nil {
		// convert from any[] => []string
		for i := 0; i < len(streamIdsArr); i++ {
			idAny := streamIdsArr[i]
			statusAny := streamStatusArr[i]
			id, idOk := idAny.(string)
			status, statusOk := statusAny.(string)
			if idOk && statusOk {
				stream := Stream{
					Id:     strings.Trim(id, " "),
					Status: strings.ToLower(strings.Trim(status, " ")),
				}
				s.Status.Streams = append(s.Status.Streams, stream)
				if stream.Status == "playing" {
					s.Status.PlayingStreams = append(s.Status.PlayingStreams, stream)
				}
			}
		}
		s.lo.Info("Data", "streams", s.Status.Streams, "playing streams", s.Status.PlayingStreams)
	}
	return
}

func (s *snapcast) Get(url string, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Get(s.http, url, setReq, getResp)
}

func (s *snapcast) Post(url string, data []byte, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Post(s.http, url, data, setReq, getResp)
}
