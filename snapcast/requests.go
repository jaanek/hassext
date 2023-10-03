package snapcast

type Request interface {
	GetClientID() string
}

type IncVolumeReq struct {
	ClientId string
	IncStep  int
}

func (r IncVolumeReq) GetClientID() string {
	return r.ClientId
}

type ChangeStreamReq struct {
	ClientId string
	Up       bool
}

func (r ChangeStreamReq) GetClientID() string {
	return r.ClientId
}

type MuteOnOffReq struct {
	ClientId string
}

func (r MuteOnOffReq) GetClientID() string {
	return r.ClientId
}

type SetDefaultChannelReq struct {
	ClientId string
	StreamId string
}

func (r SetDefaultChannelReq) GetClientID() string {
	return r.ClientId
}

func (s *snapcast) processRequest(req Request) error {
	// first check if client is muted, if so then first action behaves as unmute
	client, err := s.ClientGetStatus(req.GetClientID())
	if err != nil {
		return err
	}
	if client.Muted {
		return s.ClientMute(client, false)
	}

	// client is on now check the action
	switch r := req.(type) {
	case MuteOnOffReq:
		return s.ClientMuteOnOff(r.ClientId)
	case SetDefaultChannelReq:
		return s.ClientSetDefaultStream(r.ClientId, r.StreamId)
	case IncVolumeReq:
		return s.ClientIncVolume(r.ClientId, r.IncStep)
	case ChangeStreamReq:
		return s.ClientChangeStream(r.ClientId, r.Up)
	}
	return nil
}
