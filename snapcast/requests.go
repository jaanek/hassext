package snapcast

type IncVolumeReq struct {
	ClientId string
	IncStep  int
}

type ChangeStreamReq struct {
	ClientId string
	Up       bool
}

type OnOffMuteReq struct {
	ClientId string
}

func (s *snapcast) processRequest(req any) error {
	switch r := req.(type) {
	case OnOffMuteReq:
		return s.ClientOnOffMute(r.ClientId)
	case IncVolumeReq:
		return s.ClientIncVolume(r.ClientId, r.IncStep)
	case ChangeStreamReq:
		return s.ClientChangeStream(r.ClientId, r.Up)
	}
	return nil
}
