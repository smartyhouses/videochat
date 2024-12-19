package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.46

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/montag451/go-eventbus"
	"go.opentelemetry.io/otel/attribute"
	"nkonev.name/event/auth"
	"nkonev.name/event/dto"
	"nkonev.name/event/graph/model"
	"nkonev.name/event/logger"
	"nkonev.name/event/rabbitmq"
	"nkonev.name/event/utils"
)

// Ping is the resolver for the ping field.
func (r *queryResolver) Ping(ctx context.Context) (*bool, error) {
	res := true
	return &res, nil
}

// ChatEvents is the resolver for the chatEvents field.
func (r *subscriptionResolver) ChatEvents(ctx context.Context, chatID int64) (<-chan *model.ChatEvent, error) {
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}

	hasAccess, err := r.HttpClient.CheckAccess(ctx, authResult.UserId, chatID)
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during checking participant user %v, chat %v", authResult.UserId, chatID)
		return nil, err
	}
	if !hasAccess {
		logger.GetLogEntry(ctx, r.Lgr).Infof("User %v is not participant of chat %v", authResult.UserId, chatID)
		return nil, errors.New("Unauthorized")
	}
	logger.GetLogEntry(ctx, r.Lgr).Infof("Subscribing to chatEvents channel as user %v", authResult.UserId)

	var cam = make(chan *model.ChatEvent)
	subscribeHandler, err := r.Bus.Subscribe(dto.CHAT_EVENTS, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx, r.Lgr).Errorf("In processing ChatEvents panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.ChatEvent:
			if isReceiverOfEvent(typedEvent.UserId, authResult) && typedEvent.ChatId == chatID {
				_, span := r.Tr.Start(rabbitmq.DeserializeValues(r.Lgr, typedEvent.TraceString), fmt.Sprintf("subscription.%s", typedEvent.EventType))
				defer span.End()
				span.SetAttributes(
					attribute.Int64("userId", typedEvent.UserId),
					attribute.Int64("chatId", typedEvent.ChatId),
				)

				cam <- convertToChatEvent(&typedEvent)
			}
			break
		default:
			logger.GetLogEntry(ctx, r.Lgr).Debugf("Skipping %v as is no mapping here for this type, user %v, chat %v", typedEvent, authResult.UserId, chatID)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during creating eventbus subscription user %v, chat %v", authResult.UserId, chatID)
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.GetLogEntry(ctx, r.Lgr).Infof("Closing chatEvents channel for user %v", authResult.UserId)
				err := r.Bus.Unsubscribe(subscribeHandler)
				if err != nil {
					logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during unsubscribing from bus in chatEvents channel for user %v", authResult.UserId)
				}
				close(cam)
				return
			}
		}
	}()

	return cam, nil
}

// GlobalEvents is the resolver for the globalEvents field.
func (r *subscriptionResolver) GlobalEvents(ctx context.Context) (<-chan *model.GlobalEvent, error) {
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}
	logger.GetLogEntry(ctx, r.Lgr).Infof("Subscribing to globalEvents channel as user %v", authResult.UserId)

	var cam = make(chan *model.GlobalEvent)
	globalSubscribeHandler, err := r.Bus.Subscribe(dto.GLOBAL_USER_EVENTS, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx, r.Lgr).Errorf("In processing GlobalEvents panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.GlobalUserEvent:
			if isReceiverOfEvent(typedEvent.UserId, authResult) {
				_, span := r.Tr.Start(rabbitmq.DeserializeValues(r.Lgr, typedEvent.TraceString), fmt.Sprintf("subscription.%s", typedEvent.EventType))
				defer span.End()
				span.SetAttributes(
					attribute.Int64("userId", typedEvent.UserId),
				)

				cam <- convertToGlobalEvent(&typedEvent)
			}
			break
		default:
			logger.GetLogEntry(ctx, r.Lgr).Debugf("Skipping %v as is no mapping here for this type, user %v", typedEvent, authResult.UserId)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during creating eventbus subscription user %v", authResult.UserId)
		return nil, err
	}
	killSessionsSubscribeHandler, err := r.Bus.Subscribe(dto.AAA_KILL_SESSIONS, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx, r.Lgr).Errorf("In processing GlobalEvents panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.UserSessionsKilledEvent:
			if isReceiverOfEvent(typedEvent.UserId, authResult) {
				_, span := r.Tr.Start(rabbitmq.DeserializeValues(r.Lgr, typedEvent.TraceString), fmt.Sprintf("subscription.%s", typedEvent.EventType))
				defer span.End()
				span.SetAttributes(
					attribute.Int64("userId", typedEvent.UserId),
				)

				cam <- convertToUserSessionsKilledEvent(&typedEvent)
			}
			break
		default:
			logger.GetLogEntry(ctx, r.Lgr).Debugf("Skipping %v as is no mapping here for this type, user %v", typedEvent, authResult.UserId)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during creating eventbus subscription user %v", authResult.UserId)
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():

				logger.GetLogEntry(ctx, r.Lgr).Infof("Closing globalEvents channel for user %v", authResult.UserId)
				err := r.Bus.Unsubscribe(globalSubscribeHandler)
				if err != nil {
					logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during unsubscribing from bus in globalEvents channel for user %v", authResult.UserId)
				}

				logger.GetLogEntry(ctx, r.Lgr).Infof("Closing killSessionsSubscribeHandler channel for user %v", authResult.UserId)
				err = r.Bus.Unsubscribe(killSessionsSubscribeHandler)
				if err != nil {
					logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during unsubscribing from bus in UserVideoStatus channel for user %v", authResult.UserId)
				}

				close(cam)
				return
			}
		}
	}()

	return cam, nil
}

// UserStatusEvents is the resolver for the userStatusEvents field.
func (r *subscriptionResolver) UserStatusEvents(ctx context.Context, userIds []int64) (<-chan []*model.UserStatusEvent, error) {
	// user online
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}
	logger.GetLogEntry(ctx, r.Lgr).Infof("Subscribing to UserOnline channel as user %v", authResult.UserId)

	var cam = make(chan []*model.UserStatusEvent)

	subscribeHandlerUserOnline, err := r.Bus.Subscribe(dto.USER_ONLINE, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx, r.Lgr).Errorf("In processing UserOnline panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.ArrayUserOnline:
			var batch = []*model.UserStatusEvent{}
			for _, userOnline := range typedEvent.UserOnlines {
				if utils.Contains(userIds, userOnline.UserId) {
					_, span := r.Tr.Start(rabbitmq.DeserializeValues(r.Lgr, typedEvent.TraceString), fmt.Sprintf("subscription.%s", "user_online"))
					defer span.End()
					span.SetAttributes(
						attribute.Int64("userId", userOnline.UserId),
					)

					batch = append(batch, convertToUserOnline(userOnline))
				}
			}
			if len(batch) > 0 {
				cam <- batch
			}
			break
		default:
			logger.GetLogEntry(ctx, r.Lgr).Debugf("Skipping %v as is no mapping here for this type, user %v", typedEvent, authResult.UserId)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during creating eventbus subscription user %v", authResult.UserId)
		return nil, err
	}

	subscribeHandlerVideoCallStatus, err := r.Bus.Subscribe(dto.GENERAL, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx, r.Lgr).Errorf("In processing UserVideoStatus panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.GeneralEvent:
			var videoCallUsersCallStatusChangedEvent = typedEvent.VideoCallUsersCallStatusChangedEvent
			if videoCallUsersCallStatusChangedEvent != nil {
				var batch = []*model.UserStatusEvent{}
				for _, userCallStatus := range videoCallUsersCallStatusChangedEvent.Users {
					if utils.Contains(userIds, userCallStatus.UserId) {
						_, span := r.Tr.Start(rabbitmq.DeserializeValues(r.Lgr, typedEvent.TraceString), fmt.Sprintf("subscription.%s", typedEvent.EventType))
						defer span.End()
						span.SetAttributes(
							attribute.Int64("userId", userCallStatus.UserId),
						)

						batch = append(batch, convertToUserCallStatusChanged(typedEvent, userCallStatus))
					}
				}
				if len(batch) > 0 {
					cam <- batch
				}
			}
			break
		default:
			logger.GetLogEntry(ctx, r.Lgr).Debugf("Skipping %v as is no mapping here for this type, user %v", typedEvent, authResult.UserId)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during creating eventbus subscription user %v", authResult.UserId)
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.GetLogEntry(ctx, r.Lgr).Infof("Closing UserOnline channel for user %v", authResult.UserId)
				err := r.Bus.Unsubscribe(subscribeHandlerUserOnline)
				if err != nil {
					logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during unsubscribing from bus in UserOnline channel for user %v", authResult.UserId)
				}

				logger.GetLogEntry(ctx, r.Lgr).Infof("Closing UserVideoStatus channel for user %v", authResult.UserId)
				err = r.Bus.Unsubscribe(subscribeHandlerVideoCallStatus)
				if err != nil {
					logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during unsubscribing from bus in UserVideoStatus channel for user %v", authResult.UserId)
				}

				close(cam)
				return
			}
		}
	}()

	return cam, nil
}

// UserAccountEvents is the resolver for the userAccountEvents field.
func (r *subscriptionResolver) UserAccountEvents(ctx context.Context, userIdsFilter []int64) (<-chan *model.UserAccountEvent, error) {
	authResult, ok := ctx.Value(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok {
		return nil, errors.New("Unable to get auth context")
	}
	logger.GetLogEntry(ctx, r.Lgr).Infof("Subscribing to UserAccount channel as user %v", authResult.UserId)

	var cam = make(chan *model.UserAccountEvent)

	subscribeHandlerAaaChange, err := r.Bus.Subscribe(dto.AAA_CHANGE, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx, r.Lgr).Errorf("In processing UserAccount panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.UserAccountEventChanged:
			if filter(typedEvent.UserId, userIdsFilter) {
				var anEvent = r.prepareUserAccountEvent(ctx, authResult.UserId, typedEvent.EventType, typedEvent.User)
				if anEvent != nil {
					_, span := r.Tr.Start(rabbitmq.DeserializeValues(r.Lgr, typedEvent.TraceString), fmt.Sprintf("subscription.%s", typedEvent.EventType))
					defer span.End()
					span.SetAttributes(
						attribute.Int64("userId", typedEvent.UserId),
					)

					cam <- anEvent
				}
			}
			break
		default:
			logger.GetLogEntry(ctx, r.Lgr).Debugf("Skipping %v as is no mapping here for this type, user %v", typedEvent, authResult.UserId)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during creating eventbus subscription user %v", authResult.UserId)
		return nil, err
	}

	subscribeHandlerAaaCreate, err := r.Bus.Subscribe(dto.AAA_CREATE, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx, r.Lgr).Errorf("In processing UserAccount panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.UserAccountEventCreated:
			if filter(typedEvent.UserId, userIdsFilter) {
				var anEvent = r.prepareUserAccountEvent(ctx, authResult.UserId, typedEvent.EventType, typedEvent.User)
				if anEvent != nil {
					_, span := r.Tr.Start(rabbitmq.DeserializeValues(r.Lgr, typedEvent.TraceString), fmt.Sprintf("subscription.%s", typedEvent.EventType))
					defer span.End()
					span.SetAttributes(
						attribute.Int64("userId", typedEvent.UserId),
					)

					cam <- anEvent
				}
			}
			break
		default:
			logger.GetLogEntry(ctx, r.Lgr).Debugf("Skipping %v as is no mapping here for this type, user %v", typedEvent, authResult.UserId)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during creating eventbus subscription user %v", authResult.UserId)
		return nil, err
	}

	subscribeHandlerAaaDelete, err := r.Bus.Subscribe(dto.AAA_DELETE, func(event eventbus.Event, t time.Time) {
		defer func() {
			if err := recover(); err != nil {
				logger.GetLogEntry(ctx, r.Lgr).Errorf("In processing UserAccount panic recovered: %v", err)
			}
		}()

		switch typedEvent := event.(type) {
		case dto.UserAccountEventDeleted:
			if filter(typedEvent.UserId, userIdsFilter) {
				var anEvent = convertUserAccountDeletedEvent(typedEvent.EventType, typedEvent.UserId)
				if anEvent != nil {
					_, span := r.Tr.Start(rabbitmq.DeserializeValues(r.Lgr, typedEvent.TraceString), fmt.Sprintf("subscription.%s", typedEvent.EventType))
					defer span.End()
					span.SetAttributes(
						attribute.Int64("userId", typedEvent.UserId),
					)

					cam <- anEvent
				}
			}
		default:
			logger.GetLogEntry(ctx, r.Lgr).Debugf("Skipping %v as is no mapping here for this type, user %v", typedEvent, authResult.UserId)
		}
	})
	if err != nil {
		logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during creating eventbus subscription user %v", authResult.UserId)
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.GetLogEntry(ctx, r.Lgr).Infof("Closing UserAccount change channel for user %v", authResult.UserId)
				err := r.Bus.Unsubscribe(subscribeHandlerAaaChange)
				if err != nil {
					logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during unsubscribing from bus in UserAccount change channel for user %v", authResult.UserId)
				}

				logger.GetLogEntry(ctx, r.Lgr).Infof("Closing UserAccount create channel for user %v", authResult.UserId)
				err = r.Bus.Unsubscribe(subscribeHandlerAaaCreate)
				if err != nil {
					logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during unsubscribing from bus in UserAccount create channel for user %v", authResult.UserId)
				}

				logger.GetLogEntry(ctx, r.Lgr).Infof("Closing UserAccount delete channel for user %v", authResult.UserId)
				err = r.Bus.Unsubscribe(subscribeHandlerAaaDelete)
				if err != nil {
					logger.GetLogEntry(ctx, r.Lgr).Errorf("Error during unsubscribing from bus in UserAccount delete channel for user %v", authResult.UserId)
				}

				close(cam)
				return
			}
		}
	}()

	return cam, nil
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// Subscription returns SubscriptionResolver implementation.
func (r *Resolver) Subscription() SubscriptionResolver { return &subscriptionResolver{r} }

type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func filter(userFromBus int64, userIdsFilter []int64) bool {
	if len(userIdsFilter) == 0 {
		return true
	}
	return utils.Contains(userIdsFilter, userFromBus)
}
func (sr *subscriptionResolver) prepareUserAccountEvent(ctx context.Context, myUserId int64, eventType string, user *dto.UserAccountEvent) *model.UserAccountEvent {
	if user == nil {
		logger.GetLogEntry(ctx, sr.Lgr).Errorf("Logical mistake")
		return nil
	}

	extended, err := sr.HttpClient.GetUserExtended(ctx, user.Id, myUserId)
	if err != nil {
		logger.GetLogEntry(ctx, sr.Lgr).Errorf("error during getting user extended: %v", err)
		return nil
	}

	ret := model.UserAccountEvent{}
	ret.EventType = eventType
	ret.UserAccountEvent = convertUserAccountExtended(myUserId, user, extended)
	return &ret
}
func convertUserAccountDeletedEvent(eventType string, userId int64) *model.UserAccountEvent {
	ret := model.UserAccountEvent{}
	ret.UserAccountEvent = &model.UserDeletedDto{ID: userId}
	ret.EventType = eventType
	return &ret
}
func convertUserAccountExtended(myUserId int64, user *dto.UserAccountEvent, aDto *dto.UserAccountExtended) *model.UserAccountExtendedDto {
	userAccountEvent := &model.UserAccountExtendedDto{
		ID:                aDto.Id,
		Login:             aDto.Login,
		Avatar:            aDto.Avatar,
		AvatarBig:         aDto.AvatarBig,
		ShortInfo:         aDto.ShortInfo,
		LastLoginDateTime: aDto.LastLoginDateTime,
		Oauth2Identifiers: convertOauth2Identifiers(aDto.Oauth2Identifiers),
		CanLock:           aDto.CanLock,
		CanEnable:         aDto.CanEnable,
		CanDelete:         aDto.CanDelete,
		CanChangeRole:     aDto.CanChangeRole,
		CanConfirm:        aDto.CanConfirm,
		LoginColor:        aDto.LoginColor,
		CanRemoveSessions: aDto.CanRemoveSessions,
		Ldap:              aDto.Ldap,
		CanSetPassword:    aDto.CanSetPassword,
	}
	if myUserId == aDto.Id {
		userAccountEvent.Email = user.Email.Ptr()
		userAccountEvent.AwaitingForConfirmEmailChange = &user.AwaitingForConfirmEmailChange
	}
	if aDto.AdditionalData != nil {
		userAccountEvent.AdditionalData = &model.DataDto{
			Enabled:   aDto.AdditionalData.Enabled,
			Expired:   aDto.AdditionalData.Expired,
			Locked:    aDto.AdditionalData.Locked,
			Confirmed: aDto.AdditionalData.Confirmed,
			Roles:     aDto.AdditionalData.Roles,
		}
	}

	return userAccountEvent
}
func convertOauth2Identifiers(identifiers *dto.Oauth2Identifiers) *model.OAuth2Identifiers {
	if identifiers == nil {
		return nil
	}
	return &model.OAuth2Identifiers{
		FacebookID:  identifiers.FacebookId,
		VkontakteID: identifiers.VkontakteId,
		GoogleID:    identifiers.GoogleId,
		KeycloakID:  identifiers.KeycloakId,
	}
}
func convertToUserCallStatusChanged(event dto.GeneralEvent, u dto.VideoCallUserCallStatusChangedDto) *model.UserStatusEvent {
	return &model.UserStatusEvent{
		EventType: event.EventType,
		UserID:    u.UserId,
		IsInVideo: &u.IsInVideo,
	}
}
func convertToUserOnline(userOnline dto.UserOnline) *model.UserStatusEvent {
	return &model.UserStatusEvent{
		EventType: "user_online",
		UserID:    userOnline.UserId,
		Online:    &userOnline.Online,
	}
}
func convertToChatEvent(e *dto.ChatEvent) *model.ChatEvent {
	var result = &model.ChatEvent{
		EventType: e.EventType,
	}
	messageDto := e.MessageNotification
	if messageDto != nil {
		result.MessageEvent = convertDisplayMessageDto(messageDto)
	}

	messageDeleted := e.MessageDeletedNotification
	if messageDeleted != nil {
		result.MessageDeletedEvent = &model.MessageDeletedDto{
			ID:     messageDeleted.Id,
			ChatID: messageDeleted.ChatId,
		}
	}

	userTypingEvent := e.UserTypingNotification
	if userTypingEvent != nil {
		result.UserTypingEvent = &model.UserTypingDto{
			Login:         userTypingEvent.Login,
			ParticipantID: userTypingEvent.ParticipantId,
		}
	}

	messageBroadcast := e.MessageBroadcastNotification
	if messageBroadcast != nil {
		result.MessageBroadcastEvent = &model.MessageBroadcastNotification{
			Login:  messageBroadcast.Login,
			UserID: messageBroadcast.UserId,
			Text:   messageBroadcast.Text,
		}
	}

	fileUploadedEvent := e.PreviewCreatedEvent
	if fileUploadedEvent != nil {
		result.PreviewCreatedEvent = &model.PreviewCreatedEvent{
			ID:            fileUploadedEvent.Id,
			URL:           fileUploadedEvent.Url,
			PreviewURL:    fileUploadedEvent.PreviewUrl,
			AType:         fileUploadedEvent.Type,
			CorrelationID: &fileUploadedEvent.CorrelationId,
		}
	}

	participants := e.Participants
	if participants != nil {
		result.ParticipantsEvent = convertParticipantsWithAdmin(*participants)
	}

	promotePinnedMessageEvent := e.PromoteMessageNotification
	if promotePinnedMessageEvent != nil {
		result.PromoteMessageEvent = convertPinnedMessageEvent(promotePinnedMessageEvent)
	}

	publishedMessageEvent := e.PublishedMessageNotification
	if publishedMessageEvent != nil {
		result.PublishedMessageEvent = convertPublishedMessageEvent(publishedMessageEvent)
	}

	fileEvent := e.FileEvent
	if fileEvent != nil {
		result.FileEvent = &model.WrappedFileInfoDto{
			FileInfoDto: &model.FileInfoDto{
				ID:             fileEvent.FileInfoDto.Id,
				Filename:       fileEvent.FileInfoDto.Filename,
				URL:            fileEvent.FileInfoDto.Url,
				PublicURL:      fileEvent.FileInfoDto.PublicUrl,
				PreviewURL:     fileEvent.FileInfoDto.PreviewUrl,
				Size:           fileEvent.FileInfoDto.Size,
				CanDelete:      fileEvent.FileInfoDto.CanDelete,
				CanEdit:        fileEvent.FileInfoDto.CanEdit,
				CanShare:       fileEvent.FileInfoDto.CanShare,
				LastModified:   fileEvent.FileInfoDto.LastModified,
				OwnerID:        fileEvent.FileInfoDto.OwnerId,
				Owner:          convertParticipant(fileEvent.FileInfoDto.Owner),
				CanPlayAsVideo: fileEvent.FileInfoDto.CanPlayAsVideo,
				CanShowAsImage: fileEvent.FileInfoDto.CanShowAsImage,
				CanPlayAsAudio: fileEvent.FileInfoDto.CanPlayAsAudio,
				FileItemUUID:   fileEvent.FileInfoDto.FileItemUuid,
				CorrelationID:  fileEvent.FileInfoDto.CorrelationId,
			},
		}
	}

	reactionChangedEvent := e.ReactionChangedEvent
	if reactionChangedEvent != nil {
		result.ReactionChangedEvent = &model.ReactionChangedEvent{
			MessageID: reactionChangedEvent.MessageId,
			Reaction:  convertReaction(&reactionChangedEvent.Reaction),
		}
	}

	return result
}
func convertDisplayMessageDto(messageDto *dto.DisplayMessageDto) *model.DisplayMessageDto {
	var result = &model.DisplayMessageDto{ // dto.DisplayMessageDto
		ID:             messageDto.Id,
		Text:           messageDto.Text,
		ChatID:         messageDto.ChatId,
		OwnerID:        messageDto.OwnerId,
		CreateDateTime: messageDto.CreateDateTime,
		EditDateTime:   messageDto.EditDateTime.Ptr(),
		Owner:          convertParticipant(messageDto.Owner),
		CanEdit:        messageDto.CanEdit,
		CanDelete:      messageDto.CanDelete,
		FileItemUUID:   messageDto.FileItemUuid,
		Pinned:         messageDto.Pinned,
		BlogPost:       messageDto.BlogPost,
		PinnedPromoted: messageDto.PinnedPromoted,
		Published:      messageDto.Published,
		CanPublish:     messageDto.CanPublish,
		CanPin:         messageDto.CanPin,
	}
	embedMessageDto := messageDto.EmbedMessage
	if embedMessageDto != nil {
		result.EmbedMessage = &model.EmbedMessageResponse{
			ID:            embedMessageDto.Id,
			ChatID:        embedMessageDto.ChatId,
			ChatName:      embedMessageDto.ChatName,
			Text:          embedMessageDto.Text,
			Owner:         convertParticipant(embedMessageDto.Owner),
			EmbedType:     embedMessageDto.EmbedType,
			IsParticipant: embedMessageDto.IsParticipant,
		}
	}
	reactions := messageDto.Reactions
	if reactions != nil {
		result.Reactions = convertReactions(reactions)
	}
	return result
}
func convertReactions(reactions []dto.Reaction) []*model.Reaction {
	ret := make([]*model.Reaction, 0)
	for _, r := range reactions {
		rr := r
		ret = append(ret, convertReaction(&rr))
	}
	return ret
}
func convertReaction(r *dto.Reaction) *model.Reaction {
	return &model.Reaction{
		Count:    r.Count,
		Reaction: r.Reaction,
		Users:    convertParticipants(r.Users),
	}
}
func convertPinnedMessageEvent(e *dto.PinnedMessageEvent) *model.PinnedMessageEvent {
	return &model.PinnedMessageEvent{
		Message: convertPinnedMessageDto(&e.Message),
		Count:   e.TotalCount,
	}
}
func convertPublishedMessageEvent(e *dto.PublishedMessageEvent) *model.PublishedMessageEvent {
	return &model.PublishedMessageEvent{
		Message: convertPublishedMessageDto(&e.Message),
		Count:   e.TotalCount,
	}
}
func convertPublishedMessageDto(e *dto.PublishedMessageDto) *model.PublishedMessageDto {
	return &model.PublishedMessageDto{
		ID:             e.Id,
		Text:           e.Text,
		ChatID:         e.ChatId,
		OwnerID:        e.OwnerId,
		Owner:          convertParticipant(e.Owner),
		CanPublish:     e.CanPublish,
		CreateDateTime: e.CreateDateTime,
	}
}
func convertPinnedMessageDto(e *dto.PinnedMessageDto) *model.PinnedMessageDto {
	return &model.PinnedMessageDto{
		ID:             e.Id,
		Text:           e.Text,
		ChatID:         e.ChatId,
		OwnerID:        e.OwnerId,
		Owner:          convertParticipant(e.Owner),
		PinnedPromoted: e.PinnedPromoted,
		CreateDateTime: e.CreateDateTime,
		CanPin:         e.CanPin,
	}
}
func convertToGlobalEvent(e *dto.GlobalUserEvent) *model.GlobalEvent {
	//eventType string, chatDtoWithAdmin *dto.ChatDtoWithAdmin
	var ret = &model.GlobalEvent{
		EventType: e.EventType,
	}
	chatEvent := e.ChatNotification
	if chatEvent != nil {
		ret.ChatEvent = &model.ChatDto{
			ID:                                  chatEvent.Id,
			Name:                                chatEvent.Name,
			Avatar:                              chatEvent.Avatar.Ptr(),
			AvatarBig:                           chatEvent.AvatarBig.Ptr(),
			ShortInfo:                           chatEvent.ShortInfo.Ptr(),
			LastUpdateDateTime:                  chatEvent.LastUpdateDateTime,
			ParticipantIds:                      chatEvent.ParticipantIds,
			CanEdit:                             chatEvent.CanEdit.Ptr(),
			CanDelete:                           chatEvent.CanDelete.Ptr(),
			CanLeave:                            chatEvent.CanLeave.Ptr(),
			UnreadMessages:                      chatEvent.UnreadMessages,
			CanBroadcast:                        chatEvent.CanBroadcast,
			CanVideoKick:                        chatEvent.CanVideoKick,
			CanAudioMute:                        chatEvent.CanAudioMute,
			CanChangeChatAdmins:                 chatEvent.CanChangeChatAdmins,
			TetATet:                             chatEvent.IsTetATet,
			ParticipantsCount:                   chatEvent.ParticipantsCount,
			Participants:                        convertParticipants(chatEvent.Participants),
			CanResend:                           chatEvent.CanResend,
			AvailableToSearch:                   chatEvent.AvailableToSearch,
			IsResultFromSearch:                  chatEvent.IsResultFromSearch.Ptr(),
			Pinned:                              chatEvent.Pinned,
			Blog:                                chatEvent.Blog,
			LoginColor:                          chatEvent.LoginColor.Ptr(),
			RegularParticipantCanPublishMessage: chatEvent.RegularParticipantCanPublishMessage,
			LastLoginDateTime:                   chatEvent.LastLoginDateTime.Ptr(),
			RegularParticipantCanPinMessage:     chatEvent.RegularParticipantCanPinMessage,
		}
	}

	chatDeleted := e.ChatDeletedDto
	if chatDeleted != nil {
		ret.ChatDeletedEvent = &model.ChatDeletedDto{
			ID: chatDeleted.Id,
		}
	}

	userProfileDto := e.CoChattedParticipantNotification
	if userProfileDto != nil {
		ret.CoChattedParticipantEvent = &model.Participant{
			ID:         userProfileDto.Id,
			Login:      userProfileDto.Login,
			Avatar:     userProfileDto.Avatar.Ptr(),
			ShortInfo:  userProfileDto.ShortInfo.Ptr(),
			LoginColor: userProfileDto.LoginColor.Ptr(),
		}
	}

	videoUserCountEvent := e.VideoCallUserCountEvent
	if videoUserCountEvent != nil {
		ret.VideoUserCountChangedEvent = &model.VideoUserCountChangedDto{
			UsersCount: videoUserCountEvent.UsersCount,
			ChatID:     videoUserCountEvent.ChatId,
		}
	}

	videoCallScreenShareChangedEvent := e.VideoCallScreenShareChangedDto
	if videoCallScreenShareChangedEvent != nil {
		ret.VideoCallScreenShareChangedDto = &model.VideoCallScreenShareChangedDto{
			ChatID:          videoCallScreenShareChangedEvent.ChatId,
			HasScreenShares: videoCallScreenShareChangedEvent.HasScreenShares,
		}
	}

	videoRecordingEvent := e.VideoCallRecordingEvent
	if videoRecordingEvent != nil {
		ret.VideoRecordingChangedEvent = &model.VideoRecordingChangedDto{
			RecordInProgress: videoRecordingEvent.RecordInProgress,
			ChatID:           videoRecordingEvent.ChatId,
		}
	}

	videoChatInvite := e.VideoChatInvitation
	if videoChatInvite != nil {
		ret.VideoCallInvitation = &model.VideoCallInvitationDto{
			ChatID:   videoChatInvite.ChatId,
			ChatName: videoChatInvite.ChatName,
			Status:   videoChatInvite.Status,
			Avatar:   videoChatInvite.Avatar,
		}
	}

	videoDial := e.VideoParticipantDialEvent
	if videoDial != nil {
		ret.VideoParticipantDialEvent = &model.VideoDialChanges{
			ChatID: videoDial.ChatId,
			Dials:  convertDials(videoDial.Dials),
		}
	}

	unreadMessages := e.UnreadMessagesNotification
	if unreadMessages != nil {
		ret.UnreadMessagesNotification = &model.ChatUnreadMessageChanged{
			ChatID:             unreadMessages.ChatId,
			UnreadMessages:     unreadMessages.UnreadMessages,
			LastUpdateDateTime: unreadMessages.LastUpdateDateTime,
		}
	}

	allUnreadMessages := e.AllUnreadMessagesNotification
	if allUnreadMessages != nil {
		ret.AllUnreadMessagesNotification = &model.AllUnreadMessages{
			AllUnreadMessages: allUnreadMessages.MessagesCount,
		}
	}

	userNotification := e.UserNotificationEvent
	if userNotification != nil {
		ret.NotificationEvent = &model.WrapperNotificationDto{
			Count: userNotification.TotalCount,
			NotificationDto: &model.NotificationDto{
				ID:               userNotification.NotificationDto.Id,
				ChatID:           userNotification.NotificationDto.ChatId,
				MessageID:        userNotification.NotificationDto.MessageId,
				NotificationType: userNotification.NotificationDto.NotificationType,
				Description:      userNotification.NotificationDto.Description,
				CreateDateTime:   userNotification.NotificationDto.CreateDateTime,
				ByUserID:         userNotification.NotificationDto.ByUserId,
				ByLogin:          userNotification.NotificationDto.ByLogin,
				ByAvatar:         userNotification.NotificationDto.ByAvatar,
				ChatTitle:        userNotification.NotificationDto.ChatTitle,
			},
		}
	}

	hasUnreadMessagesChanged := e.HasUnreadMessagesChanged
	if hasUnreadMessagesChanged != nil {
		ret.HasUnreadMessagesChanged = &model.HasUnreadMessagesChangedEvent{
			HasUnreadMessages: hasUnreadMessagesChanged.HasUnreadMessages,
		}
	}

	browserNotification := e.BrowserNotification
	if browserNotification != nil {
		ret.BrowserNotification = &model.BrowserNotification{
			ChatID:      browserNotification.ChatId,
			ChatName:    browserNotification.ChatName,
			ChatAvatar:  browserNotification.ChatAvatar,
			MessageID:   browserNotification.MessageId,
			MessageText: browserNotification.MessageText,
			OwnerID:     browserNotification.OwnerId,
			OwnerLogin:  browserNotification.OwnerLogin,
		}
	}

	return ret
}
func convertToUserSessionsKilledEvent(aDto *dto.UserSessionsKilledEvent) *model.GlobalEvent {
	var ret = &model.GlobalEvent{
		EventType:   aDto.EventType,
		ForceLogout: &model.ForceLogoutEvent{ReasonType: aDto.ReasonType},
	}

	return ret
}
func convertParticipant(owner *dto.User) *model.Participant {
	if owner == nil {
		return nil
	}
	return &model.Participant{
		ID:         owner.Id,
		Login:      owner.Login,
		Avatar:     owner.Avatar.Ptr(),
		ShortInfo:  owner.ShortInfo.Ptr(),
		LoginColor: owner.LoginColor.Ptr(),
	}
}
func convertParticipants(participants []*dto.User) []*model.Participant {
	if participants == nil {
		return nil
	}
	usrs := []*model.Participant{}
	for _, user := range participants {
		usrs = append(usrs, convertParticipant(user))
	}
	return usrs
}
func convertParticipantWithAdmin(owner *dto.UserWithAdmin) *model.ParticipantWithAdmin {
	if owner == nil {
		return nil
	}
	return &model.ParticipantWithAdmin{
		ID:         owner.Id,
		Login:      owner.Login,
		Avatar:     owner.Avatar.Ptr(),
		Admin:      owner.Admin,
		ShortInfo:  owner.ShortInfo.Ptr(),
		LoginColor: owner.LoginColor.Ptr(),
	}
}
func convertParticipantsWithAdmin(participants []*dto.UserWithAdmin) []*model.ParticipantWithAdmin {
	if participants == nil {
		return nil
	}
	usrs := []*model.ParticipantWithAdmin{}
	for _, user := range participants {
		usrs = append(usrs, convertParticipantWithAdmin(user))
	}
	return usrs
}
func convertDials(dials []*dto.VideoDialChanged) []*model.VideoDialChanged {
	if dials == nil {
		return nil
	}
	dls := []*model.VideoDialChanged{}
	for _, dl := range dials {
		dls = append(dls, convertDial(dl))
	}
	return dls
}
func convertDial(dl *dto.VideoDialChanged) *model.VideoDialChanged {
	if dl == nil {
		return nil
	}
	return &model.VideoDialChanged{
		UserID: dl.UserId,
		Status: dl.Status,
	}
}
func isReceiverOfEvent(userId int64, authResult *auth.AuthResult) bool {
	return userId == authResult.UserId
}
