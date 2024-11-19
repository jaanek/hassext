package chromecast

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gogo/protobuf/proto"
	pb "github.com/jaanek/hassext/chromecast/proto"
	"github.com/jaanek/hassext/jblbar"
	"github.com/pkg/errors"
	"github.com/zerodha/logf"
)

const (
	LIVING_ROOM_JBL = "JBL BAR 500"
)

var (
	// Global request id
	requestID uint64
)

const (
	// 'CC1AD845' seems to be a predefined app; check link
	// https://gist.github.com/jloutsenhizer/8855258
	defaultChromecastAppID = "CC1AD845"

	defaultSender = "sender-0"
	defaultRecv   = "receiver-0"

	NamespaceConn  = "urn:x-cast:com.google.cast.tp.connection"
	NamespaceRecv  = "urn:x-cast:com.google.cast.receiver"
	NamespaceMedia = "urn:x-cast:com.google.cast.media"
)

const (
	dialerTimeout   = time.Second * 3
	dialerKeepAlive = time.Second * 30
)

type Chromecast interface {
	DNSEntry() CastDNSEntry
	SetDNSEntry(CastDNSEntry)
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Volume() (*Volume, error)
	IncVolume(inc float32) error
	SetVolume(value float32) error
	SetMuted(value bool) error
	GetReceiverStatus() (*ReceiverStatusResponse, error)
	CallCommand(command jblbar.Command, value *jblbar.CommandPayload, result any) error
}

type cc struct {
	prefix            string
	log               logf.Logger
	entry             CastDNSEntry
	conn              *tls.Conn
	receiveLoopCancel context.CancelFunc
	recvMsgChan       chan *pb.CastMessage
	resultChanMap     map[uint64]chan *pb.CastMessage // Internal mapping of request id to result channel
	closeChanOnce     sync.Once
	jblBar            jblbar.JblBar
}

func New(log logf.Logger, entry CastDNSEntry, cert *tls.Certificate) Chromecast {
	return &cc{
		prefix:        "[chromecast] ",
		log:           log,
		entry:         entry,
		recvMsgChan:   make(chan *pb.CastMessage, 5),
		resultChanMap: make(map[uint64]chan *pb.CastMessage),
		jblBar:        jblbar.New(log, "https://"+entry.GetAddr(), cert),
	}
}

func (c *cc) DNSEntry() CastDNSEntry {
	return c.entry
}

func (c *cc) SetDNSEntry(e CastDNSEntry) {
	c.entry = e
}

func (c *cc) Connect(ctx context.Context) error {
	var err error
	dialer := &net.Dialer{
		Timeout:   dialerTimeout,
		KeepAlive: dialerKeepAlive,
	}
	var addr, port = c.entry.GetAddr(), c.entry.GetPort()
	c.conn, err = tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:%d", addr, port), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to connect to chromecast at '%s:%d'", addr, port)
	}

	// start listening async response messages
	ctx, c.receiveLoopCancel = context.WithCancel(context.Background())
	go c.receiveLoop(ctx)

	// send connect header
	if err := c.sendDefault(&ConnectHeader, NamespaceConn); err != nil {
		return errors.Wrap(err, "unable to connect to chromecast")
	}
	return nil
}

func (c *cc) Disconnect(ctx context.Context) error {
	err := c.sendDefault(&CloseHeader, NamespaceConn)
	if err != nil {
		c.log.Error(c.prefix+"Error while sending CLOSE payload to chromecast.", "name", c.entry.GetName())
	}
	if c.receiveLoopCancel != nil {
		c.receiveLoopCancel()
	}
	defer c.closeChanOnce.Do(func() {
		close(c.recvMsgChan)
	})
	return c.conn.Close()
}

func (c *cc) Volume() (*Volume, error) {
	status, err := c.GetReceiverStatus()
	if err != nil {
		return nil, err
	}
	return &status.Status.Volume, nil
}

func (c *cc) IncVolume(inc float32) error {
	vol, err := c.Volume()
	if err != nil {
		return err
	}
	var value = vol.Level + inc
	return c.SetVolume(value)
}

func (c *cc) SetVolume(value float32) error {
	if value > 1 || value < 0 {
		return ErrVolumeOutOfRange
	}
	return c.sendDefault(&SetVolume{
		PayloadHeader: VolumeHeader,
		Volume: Volume{
			Level: value,
		},
	}, NamespaceRecv)
}

func (c *cc) SetMuted(value bool) error {
	return c.sendDefault(&SetVolume{
		PayloadHeader: VolumeHeader,
		Volume: Volume{
			Muted: value,
		},
	}, NamespaceRecv)
}

func (c *cc) GetReceiverStatus() (*ReceiverStatusResponse, error) {
	apiMessage, err := c.sendDefaultAndWait(&GetStatusHeader, NamespaceRecv)
	if err != nil {
		return nil, err
	}
	return parseReceiverStatus(apiMessage)
}

func (c *cc) CallCommand(command jblbar.Command, value *jblbar.CommandPayload, result any) error {
	return c.jblBar.CallCommand(command, value, result)
}

func (c *cc) getNextRequestId() uint64 {
	// NOTE: Not concurrent safe, but currently only synchronous flow is possible
	// TODO(vishen): just make concurrent safe regardless of current flow
	requestID += 1
	return requestID
}

func (c *cc) sendDefault(payload Payload, namespace string) error {
	var reqId = c.getNextRequestId()
	return c.send(reqId, payload, defaultSender, defaultRecv, namespace)
}

func (c *cc) sendDefaultAndWait(payload Payload, namespace string) (*pb.CastMessage, error) {
	return c.sendAndWait(payload, defaultSender, defaultRecv, namespace)
}

func (c *cc) sendAndWait(payload Payload, sourceID, destinationID, namespace string) (*pb.CastMessage, error) {
	// Set a timeout to wait for the response
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// TODO(vishen): not concurrent safe. Not a problem at the moment
	// because only synchronous flow currently allowed.
	var reqId = c.getNextRequestId()
	resultChan := make(chan *pb.CastMessage, 1)
	c.resultChanMap[reqId] = resultChan
	defer func() {
		delete(c.resultChanMap, requestID)
	}()

	err := c.send(reqId, payload, sourceID, destinationID, namespace)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		return result, nil
	}
}

func (c *cc) send(reqId uint64, payload Payload, sourceID, destinationID, namespace string) error {
	payload.SetRequestId(reqId)
	return c.Send(payload, sourceID, destinationID, namespace)
}

func (c *cc) Send(payload Payload, sourceID, destinationID, namespace string) error {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "unable to marshal json payload")
	}
	payloadUtf8 := string(payloadJson)
	message := &pb.CastMessage{
		ProtocolVersion: pb.CastMessage_CASTV2_1_0.Enum(),
		SourceId:        &sourceID,
		DestinationId:   &destinationID,
		Namespace:       &namespace,
		PayloadType:     pb.CastMessage_STRING.Enum(),
		PayloadUtf8:     &payloadUtf8,
	}
	proto.SetDefaults(message)
	data, err := proto.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "unable to marshal proto payload")
	}

	c.log.Info(c.prefix+"Send payload", "requestID", payload.GetRequestId(), "sourceID", sourceID, "destinationID", destinationID, "namespace", namespace, "payloadJson", payloadJson)

	if err := binary.Write(c.conn, binary.BigEndian, uint32(len(data))); err != nil {
		return errors.Wrap(err, "unable to write binary format")
	}
	if _, err := c.conn.Write(data); err != nil {
		return errors.Wrap(err, "unable to send data")
	}
	return nil
}

func (c *cc) receiveLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Fallthrough if not done
		}
		var length uint32
		if c.conn == nil {
			continue
		}
		if err := binary.Read(c.conn, binary.BigEndian, &length); err != nil {
			c.log.Error(c.prefix+"failed to binary read payload", "error", err)
			break
		}
		if length == 0 {
			c.log.Warn(c.prefix + "empty payload received")
			continue
		}

		payload := make([]byte, length)
		i, err := io.ReadFull(c.conn, payload)
		if err != nil {
			c.log.Error(c.prefix+"failed to read payload", "error", err)
			continue
		}

		if i != int(length) {
			c.log.Error(c.prefix+"invalid payload", "wanted", length, "but read", i)
			continue
		}

		message := &pb.CastMessage{}
		if err := proto.Unmarshal(payload, message); err != nil {
			c.log.Error(c.prefix+"failed to unmarshal proto cast message", "payload", payload, "error", err)
			continue
		}
		// Get the requestID from the message to use in the log. We don't really care if this fails.
		requestID, _ := jsonparser.GetInt([]byte(*message.PayloadUtf8), "requestId")
		if requestID == 0 {
			requestID = -1
		}
		// Cast to int, losing information, but unlilely we will
		// ever send that many messages in a single run.
		requestIDi := uint64(requestID)
		c.log.Info(c.prefix+"Received payload", "requestID", requestIDi, "DestinationId", *message.DestinationId, "SourceId", *message.SourceId, "Namespace", *message.Namespace, "Payload", *message.PayloadUtf8)

		// notify sendAndWait listener if there is any
		if resultChan, ok := c.resultChanMap[requestIDi]; ok {
			resultChan <- message
		}

		// listen async messages
		var headers PayloadHeader
		if err := json.Unmarshal([]byte(*message.PayloadUtf8), &headers); err != nil {
			c.log.Error(c.prefix+"failed to unmarshal proto message header", "error", err)
			continue
		}
		c.handleMessage(requestIDi, message, &headers)
	}
}

func (c *cc) handleMessage(requestID uint64, msg *pb.CastMessage, headers *PayloadHeader) {
	messageBytes := []byte(*msg.PayloadUtf8)
	messageType, err := jsonparser.GetString(messageBytes, "type")
	if err != nil {
		c.log.Error(c.prefix+"could not find 'type' key in response message", "request_id", requestID, "PayloadUtf8", *msg.PayloadUtf8, "error", err)
		return
	}
	switch messageType {
	case "PING":
		if err := c.Send(&PongHeader, *msg.SourceId, *msg.DestinationId, *msg.Namespace); err != nil {
			c.log.Error(c.prefix+"unable to respond to 'PING'", "error", err)
		}
	case "LOAD_FAILED":
		// c.MediaFinished()
	case "MEDIA_STATUS":
		resp := MediaStatusResponse{}
		if err := json.Unmarshal(messageBytes, &resp); err == nil {
			c.log.Info(c.prefix+"Media status", "status", resp)
			for _, status := range resp.Status {
				// The LoadingItemId is only set when there is a playlist and there
				// is an item being loaded to play next.
				if status.IdleReason == "FINISHED" && status.LoadingItemId == 0 {
					// c.MediaFinished()
				} else if status.IdleReason == "INTERRUPTED" && status.Media.ContentId == "" {
					// This can happen when we go "next" in a playlist when it
					// is playing the last track.
					// c.MediaFinished()
				}
			}
		}
	case "RECEIVER_STATUS":
		resp, err := parseReceiverStatus(msg)
		if err != nil {
			c.log.Info(c.prefix+"Receiver status", "status", resp)
			// Check to see if the application on the device has changed,
			// if it has it is likely not this running instance that changed
			// it because that currently isn't possible.
			// for _, app := range resp.Status.Applications {
			// 	if app.AppId != c.application.AppId {
			// 		// c.MediaFinished()
			// 	}
			// 	c.application = &app
			// }
			// c.volumeReceiver = &resp.Status.Volume
		}
	case "CLOSE":
		// c.MediaFinished()
		c.log.Info(c.prefix + "Closed")
	}
}

func parseReceiverStatus(apiMessage *pb.CastMessage) (*ReceiverStatusResponse, error) {
	messageBytes := []byte(*apiMessage.PayloadUtf8)
	resp := ReceiverStatusResponse{}
	if err := json.Unmarshal(messageBytes, &resp); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling json")
	}
	return &resp, nil
}
