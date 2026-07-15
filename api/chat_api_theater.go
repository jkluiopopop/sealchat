package api

import (
	"encoding/json"
	"errors"
	"strings"

	"sealchat/model"
	"sealchat/protocol"
	"sealchat/service"
)

type theaterSubscribeRequest struct {
	WorldID       string `json:"worldId"`
	ChannelID     string `json:"channelId"`
	KnownRevision int64  `json:"knownRevision"`
}

type theaterSnapshotDescriptor struct {
	Type    protocol.EventName     `json:"type"`
	Payload theaterSnapshotPayload `json:"payload"`
}

type theaterSubscriptionSync struct {
	Subscribed bool                       `json:"subscribed"`
	Mode       string                     `json:"mode"`
	Revision   int64                      `json:"revision"`
	Events     []service.TheaterEvent     `json:"events,omitempty"`
	Snapshot   *theaterSnapshotDescriptor `json:"snapshot,omitempty"`
	RoomID     string                     `json:"roomId"`
	Checksum   string                     `json:"checksum"`
}

func prepareTheaterSubscription(ctx *ChatContext, request theaterSubscribeRequest) (*theaterSubscriptionSync, error) {
	if ctx == nil || ctx.User == nil || ctx.ConnInfo == nil {
		return nil, errors.New("theater subscription requires authenticated websocket")
	}
	if ctx.User.IsBot {
		return nil, errors.New("BOT theater subscription is not enabled")
	}
	request.WorldID = strings.TrimSpace(request.WorldID)
	request.ChannelID = strings.TrimSpace(request.ChannelID)
	if request.WorldID == "" || request.ChannelID == "" || request.KnownRevision < 0 {
		return nil, errors.New("invalid theater subscription scope")
	}
	if ctx.ConnInfo.ChannelId != request.ChannelID {
		return nil, errors.New("theater subscription channel does not match active channel")
	}
	if ctx.IsObserver() && ctx.ObserverWorldID() != request.WorldID {
		return nil, errors.New("theater subscription world does not match observer scope")
	}
	if ctx.IsObserver() {
		world, _, resolveErr := service.ResolveWorldObserverLink(ctx.ConnInfo.ObserverSlug)
		if resolveErr != nil || world == nil || world.ID != request.WorldID {
			return nil, errors.New("theater observer link is no longer valid")
		}
	}
	if !ctx.IsObserver() && ctx.ConnInfo.WorldId != "" && ctx.ConnInfo.WorldId != request.WorldID {
		return nil, errors.New("theater subscription world does not match active world")
	}
	var snapshot *service.TheaterSnapshotResult
	var err error
	if ctx.IsObserver() {
		snapshot, err = service.GetTheaterSnapshotForObserver(nil, ctx.ObserverWorldID(), request.ChannelID, service.TheaterSnapshotOptions{})
	} else {
		snapshot, err = service.GetTheaterSnapshot(nil, ctx.User.ID, request.WorldID, request.ChannelID, service.TheaterSnapshotOptions{})
	}
	if err != nil {
		return nil, err
	}
	result := &theaterSubscriptionSync{
		Subscribed: true, Mode: "current", Revision: snapshot.Revision,
		Events: []service.TheaterEvent{}, RoomID: snapshot.RoomID, Checksum: snapshot.Checksum,
	}
	ctx.ConnInfo.setTheaterSubscription(&theaterSubscription{WorldID: request.WorldID, ChannelID: request.ChannelID, KnownRevision: request.KnownRevision})
	if request.KnownRevision == snapshot.Revision {
		return result, nil
	}
	if request.KnownRevision > snapshot.Revision {
		result.Mode = "snapshot"
		result.Snapshot = theaterSnapshotDescriptorFor(snapshot, "gap", observerSlugForContext(ctx))
		return result, nil
	}
	var events *service.TheaterEventsResult
	if ctx.IsObserver() {
		events, err = service.ListTheaterEventsForObserver(nil, ctx.ObserverWorldID(), request.ChannelID, request.KnownRevision, 200)
	} else {
		events, err = service.ListTheaterEvents(nil, ctx.User.ID, request.WorldID, request.ChannelID, request.KnownRevision, 200)
	}
	if err != nil {
		if service.IsTheaterErrorCode(err, service.TheaterErrorHistoryExpired) {
			result.Mode = "snapshot"
			result.Snapshot = theaterSnapshotDescriptorFor(snapshot, "history-expired", observerSlugForContext(ctx))
			return result, nil
		}
		return nil, err
	}
	if events.HasMore || events.ToRevision != snapshot.Revision {
		result.Mode = "snapshot"
		result.Snapshot = theaterSnapshotDescriptorFor(snapshot, "gap", observerSlugForContext(ctx))
		return result, nil
	}
	result.Mode = "events"
	result.Events = events.Events
	return result, nil
}

func theaterSnapshotDescriptorFor(snapshot *service.TheaterSnapshotResult, reason, observerSlug string) *theaterSnapshotDescriptor {
	return &theaterSnapshotDescriptor{Type: protocol.EventTheaterSnapshot, Payload: theaterSnapshotPayload{
		Revision: snapshot.Revision, SchemaVersion: snapshot.SchemaVersion, Checksum: snapshot.Checksum, Reason: reason,
		SnapshotURL: theaterSnapshotURL(snapshot.WorldID, snapshot.ChannelID, observerSlug),
	}}
}

func observerSlugForContext(ctx *ChatContext) string {
	if ctx == nil || !ctx.IsObserver() || ctx.ConnInfo == nil {
		return ""
	}
	return strings.TrimSpace(ctx.ConnInfo.ObserverSlug)
}

func apiTheaterSubscribeWs(ctx *ChatContext, msg []byte) {
	var envelope struct {
		Data theaterSubscribeRequest `json:"data"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		writeTheaterWSResponse(ctx, nil, err)
		return
	}
	result, err := prepareTheaterSubscription(ctx, envelope.Data)
	writeTheaterWSResponse(ctx, result, err)
	if err != nil || result == nil {
		return
	}
	queue := ctx.ConnInfo.ensureTheaterQueue()
	if queue == nil {
		return
	}
	gap := theaterSnapshotEventForConnection(ctx.ConnInfo, envelope.Data.WorldID, envelope.Data.ChannelID, result.RoomID, result.Revision, model.TheaterSchemaVersion, result.Checksum, "gap")
	if result.Snapshot != nil {
		queue.Enqueue(theaterSnapshotEventWithURL(envelope.Data.WorldID, envelope.Data.ChannelID, result.RoomID, result.Revision, result.Snapshot.Payload.SchemaVersion, result.Checksum, result.Snapshot.Payload.Reason, result.Snapshot.Payload.SnapshotURL), gap)
		return
	}
	for _, item := range result.Events {
		payload := map[string]any{
			"mutationId": item.MutationID, "revisionBefore": item.RevisionBefore,
			"revision": item.Revision, "type": item.Type, "payload": item.Payload,
		}
		queue.Enqueue(theaterGatewayEvent(protocol.EventTheaterMutationApplied, envelope.Data.WorldID, envelope.Data.ChannelID, result.RoomID, item.Revision, item.MutationID, payload), gap)
	}
}

func apiTheaterUnsubscribeWs(ctx *ChatContext, _ []byte) {
	if ctx != nil && ctx.ConnInfo != nil {
		ctx.ConnInfo.closeTheaterQueue()
	}
	writeTheaterWSResponse(ctx, map[string]any{"subscribed": false}, nil)
}

func writeTheaterWSResponse(ctx *ChatContext, data any, err error) {
	if ctx == nil || ctx.Conn == nil {
		return
	}
	if err != nil {
		_ = ctx.Conn.WriteJSON(map[string]any{"echo": ctx.Echo, "err": err.Error(), "data": data})
		return
	}
	_ = ctx.Conn.WriteJSON(map[string]any{"echo": ctx.Echo, "data": data})
}
