package network

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/icon-project/goloop/module"
)

type protocolHandler struct {
	m            *manager
	protocol     protocolInfo
	subProtocols map[protocolInfo]module.ProtocolInfo
	reactor      module.Reactor
	name         string
	priority     uint8
	receiveQueue Queue
	eventQueue   Queue
	failureQueue Queue
	//log
	log *logger
}

func newProtocolHandler(
	m *manager,
	pi protocolInfo,
	spiList []module.ProtocolInfo,
	r module.Reactor,
	name string,
	priority uint8) *protocolHandler {

	ph := &protocolHandler{
		m:            m,
		protocol:     pi,
		subProtocols: make(map[protocolInfo]module.ProtocolInfo),
		reactor:      r,
		name:         name,
		priority:     priority,
		receiveQueue: NewQueue(DefaultReceiveQueueSize),
		eventQueue:   NewQueue(DefaultEventQueueSize),
		failureQueue: NewQueue(DefaultFailureQueueSize),
		log: newLogger("ProtocolHandler",
			fmt.Sprintf("%s.%s.%s",
				m.Channel(),
				hex.EncodeToString(m.PeerID().Bytes()[:DefaultSimplePeerIDSize]),
				name)),
	}
	for _, sp := range spiList {
		k := protocolInfo(sp.Uint16())
		if _, ok := ph.subProtocols[k]; ok {
			ph.log.Println("Warning", "newProtocolHandler", "already registered protocol", ph.name, ph.protocol, sp)
		}
		ph.subProtocols[k] = sp
	}

	ph.log.excludes = []string{
		"onEvent",
		//"onFailure",
		"onPacket",
		"Unicast",
		"Multicast",
		"Broadcast",
	}

	go ph.receiveRoutine()
	go ph.eventRoutine()
	go ph.failureRoutine()
	return ph
}

func (ph *protocolHandler) receiveRoutine() {
	for {
		<-ph.receiveQueue.Wait()
		for {
			ctx := ph.receiveQueue.Pop()
			if ctx == nil {
				break
			}
			pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
			p := ctx.Value(p2pContextKeyPeer).(*Peer)
			pi := ph.subProtocols[pkt.subProtocol]
			// ph.log.Println("receiveRoutine", pi, p.ID)
			r, err := ph.reactor.OnReceive(pi, pkt.payload, p.ID())
			if err != nil {
				// ph.log.Println("receiveRoutine", err)
			}

			if r && pkt.ttl != 1 && pkt.dest != p2pDestPeer {
				if err := ph.m.relay(pkt); err != nil {
					ph.onFailure(err, pkt, nil)
				}
			}
		}
	}
}

//callback from PeerToPeer.onPacket() in Peer.onReceiveRoutine
func (ph *protocolHandler) onPacket(pkt *Packet, p *Peer) {
	ph.log.Println("onPacket", pkt, p)

	k := pkt.subProtocol
	if _, ok := ph.subProtocols[k]; ok {
		ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
		ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
		if ok := ph.receiveQueue.Push(ctx); !ok {
			ph.log.Println("Warning", "onPacket", "receiveQueue Push failure", ph.name, pkt.protocol, pkt.subProtocol, p.ID())
		}
	} else {
		//ph.log.Println("Warning", "onPacket", "not registered protocol", ph.name, pkt.protocol, pkt.subProtocol, p.ID())
	}
}

func (ph *protocolHandler) failureRoutine() {
	for {
		<-ph.failureQueue.Wait()
		for {
			ctx := ph.failureQueue.Pop()
			if ctx == nil {
				break
			}
			err := ctx.Value(p2pContextKeyError).(error)
			pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
			c := ctx.Value(p2pContextKeyCounter).(*Counter)

			k := pkt.subProtocol
			if pi, ok := ph.subProtocols[k]; ok {
				var netErr module.NetworkError
				if pkt.sender == nil {
					switch pkt.dest {
					case p2pDestPeer:
						netErr = NewUnicastError(err, pkt.destPeer)
					case p2pDestAny:
						if pkt.ttl == 1 {
							netErr = NewBroadcastError(err, module.BROADCAST_NEIGHBOR)
						} else {
							netErr = NewBroadcastError(err, module.BROADCAST_ALL)
						}
					default: //p2pDestPeerGroup < dest < p2pDestPeer
						netErr = NewMulticastError(err, ph.m.getRoleByDest(pkt.dest))
					}
					ph.reactor.OnFailure(netErr, pi, pkt.payload)
				} else {
					//TODO retry relay
					ph.log.Println("Warning", "receiveRoutine", "relay", err, c)
					//netErr = newNetworkError(err, "relay", pkt)
					//ph.reactor.OnFailure(netErr, pi, pkt.payload)
				}
			}
		}
	}
}

func (ph *protocolHandler) onFailure(err error, pkt *Packet, c *Counter) {
	ph.log.Println("onFailure", err, pkt, c)
	ctx := context.WithValue(context.Background(), p2pContextKeyError, err)
	ctx = context.WithValue(ctx, p2pContextKeyPacket, pkt)
	ctx = context.WithValue(ctx, p2pContextKeyCounter, c)
	if ok := ph.failureQueue.Push(ctx); !ok {
		ph.log.Println("Warning", "onFailure", "failureQueue Push failure", pkt)
	}
}

func (ph *protocolHandler) eventRoutine() {
	for {
		<-ph.eventQueue.Wait()
		for {
			ctx := ph.eventQueue.Pop()
			if ctx == nil {
				break
			}
			evt := ctx.Value(p2pContextKeyEvent).(string)
			p := ctx.Value(p2pContextKeyPeer).(*Peer)
			ph.log.Println("eventRoutine", evt, p.ID())
			switch evt {
			case p2pEventJoin:
				ph.reactor.OnJoin(p.ID())
			case p2pEventLeave:
				ph.reactor.OnLeave(p.ID())
			case p2pEventDuplicate:
				ph.reactor.OnLeave(p.ID())
			}
		}
	}
}

func (ph *protocolHandler) onEvent(evt string, p *Peer) {
	ph.log.Println("onEvent", evt, p)
	ctx := context.WithValue(context.Background(), p2pContextKeyEvent, evt)
	ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
	if ok := ph.eventQueue.Push(ctx); !ok {
		ph.log.Println("Warning", "onEvent", "eventQueue Push failure", evt, p.ID())
	}
}

func (ph *protocolHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	spi := protocolInfo(pi.Uint16())
	if _, ok := ph.subProtocols[spi]; !ok {
		return ErrNotRegisteredProtocol
	}
	ph.log.Println("Unicast", pi, len(b), id)
	if err := ph.m.unicast(ph.protocol, spi, b, id); err != nil {
		return NewUnicastError(err, id)
	}
	return nil
}

//TxMessage,PrevoteMessage, Send to Validators
func (ph *protocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	spi := protocolInfo(pi.Uint16())
	if _, ok := ph.subProtocols[spi]; !ok {
		return ErrNotRegisteredProtocol
	}
	ph.log.Println("Multicast", pi, len(b), role)
	if err := ph.m.multicast(ph.protocol, spi, b, role); err != nil {
		return NewMulticastError(err, role)
	}
	return nil
}

//ProposeMessage,PrecommitMessage,BlockMessage, Send to Citizen
func (ph *protocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	spi := protocolInfo(pi.Uint16())
	if _, ok := ph.subProtocols[spi]; !ok {
		return ErrNotRegisteredProtocol
	}
	ph.log.Println("Broadcast", pi, len(b), bt)
	if err := ph.m.broadcast(ph.protocol, spi, b, bt); err != nil {
		return NewBroadcastError(err, bt)
	}
	return nil
}
