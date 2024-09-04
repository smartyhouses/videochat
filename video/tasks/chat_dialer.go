package tasks

import (
	"context"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"nkonev.name/video/client"
	"nkonev.name/video/config"
	"nkonev.name/video/db"
	"nkonev.name/video/dto"
	. "nkonev.name/video/logger"
	"nkonev.name/video/producer"
	"nkonev.name/video/services"
	"nkonev.name/video/utils"
	"time"
)

type ChatDialerService struct {
	database   			    *db.DB
	conf                    *config.ExtendedConfig
	rabbitMqInvitePublisher *producer.RabbitInvitePublisher
	dialStatusPublisher     *producer.RabbitDialStatusPublisher
	chatClient              *client.RestClient
	tracer             trace.Tracer
	stateChangedEventService *services.StateChangedEventService
}

func NewChatDialerService(
	database   			  *db.DB,
	conf *config.ExtendedConfig,
	rabbitMqInvitePublisher *producer.RabbitInvitePublisher,
	dialStatusPublisher *producer.RabbitDialStatusPublisher,
	chatClient *client.RestClient,
	stateChangedEventService *services.StateChangedEventService,
) *ChatDialerService {
	trcr := otel.Tracer("scheduler/chat-dialer")
	return &ChatDialerService{
		database:                database,
		conf:                    conf,
		rabbitMqInvitePublisher: rabbitMqInvitePublisher,
		dialStatusPublisher:     dialStatusPublisher,
		chatClient:              chatClient,
		tracer:             trcr,
		stateChangedEventService: stateChangedEventService,
	}
}

func (srv *ChatDialerService) doJob() {

	ctx, span := srv.tracer.Start(context.Background(), "scheduler.ChatDialer")
	defer span.End()

	GetLogEntry(ctx).Debugf("Invoked periodic ChatDialer")

	srv.makeDial(ctx)

	GetLogEntry(ctx).Debugf("End of ChatNotifier")
}

func (srv *ChatDialerService) makeDial(ctx context.Context) {
	err := db.Transact(srv.database, func(tx *db.Tx) error {
		offset := int64(0)
		hasMoreElements := true
		for hasMoreElements {
			// here we use order by owner_id
			batchUserStates, err := tx.GetAllUserStatesOrderByOwnerAndChat(utils.DefaultSize, offset)
			if err != nil {
				GetLogEntry(ctx).Errorf("error during reading user states %v", err)
				continue
			}

			// prepare batch
			// chat:owner:[UserCallState]
			byChatAndOwner := map[int64]map[int64][]dto.UserCallState{}
			// in order to process a case when we have different owners, chats in the same batch
			for _, st := range batchUserStates {
				owner := utils.OwnerIdToNoUser(st.OwnerUserId)
				if _, ok := byChatAndOwner[st.ChatId]; !ok {
					byChatAndOwner[st.ChatId] = map[int64][]dto.UserCallState{}
				}
				byChatAndOwner[st.ChatId][owner] = append(byChatAndOwner[st.ChatId][owner], st)
			}

			// process batch
			for chat, maps := range byChatAndOwner {
				for owner, states := range maps {
					srv.processBatch(ctx, tx, chat, owner, states)
				}
			}

			hasMoreElements = len(batchUserStates) == utils.DefaultSize
			offset += utils.DefaultSize
		}
		return nil
	})
	if err != nil {
		GetLogEntry(ctx).Errorf("Error during processing: %v", err)
	}
}

func (srv *ChatDialerService) processBatch(ctx context.Context, tx *db.Tx, chatId, ownerId int64, batchUserStates []dto.UserCallState) {
	if len(batchUserStates) == 0 {
		return
	}

	userIds := []int64{}
	for _, st := range batchUserStates {
		userIds = append(userIds, st.UserId)
	}

	inviteNames, err := srv.chatClient.GetChatNameForInvite(ctx, chatId, ownerId, userIds)
	if err != nil {
		GetLogEntry(ctx).Error(err, "Failed during getting chat invite names")
	}

	for _, st := range batchUserStates {
		// cleanNotNeededAnymoreDialData - should be before status == services.CallStatusNotFound exit
		srv.cleanNotNeededAnymoreDialData(ctx, tx, chatId, st)

		m := map[int64]string{st.UserId: st.Status}
		realOwnerId := ownerId
		if realOwnerId == db.NoUser {
			realOwnerId = st.UserId
		}
		srv.stateChangedEventService.SendDialEvents(ctx, chatId, m, realOwnerId, utils.NullToEmpty(st.OwnerAvatar), st.ChatTetATet, inviteNames)
	}
}

func (srv *ChatDialerService) cleanNotNeededAnymoreDialData(
	ctx context.Context,
	tx *db.Tx,
	chatId int64,
	state dto.UserCallState,
) {
	if db.IsTemporary(state.Status) { // cleanup "normally" created temporary statuses
		if state.MarkedForRemoveAt != nil &&
			time.Now().UTC().Sub(*state.MarkedForRemoveAt) > viper.GetDuration("schedulers.chatDialerTask.removeTemporaryUserCallStatusAfter") {

			GetLogEntry(ctx).Infof("Removing temporary in status %v of user tokenId %v, userId %v, chat %v", state.Status, state.TokenId, state.UserId, chatId)
			err := tx.Remove(dto.UserCallStateId{
				TokenId: state.TokenId,
				UserId:  state.UserId,
			})
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable invoke RemoveFromDialList, user tokenId %v, userId %v, chat %v, error %v", state.TokenId, state.UserId, chatId, err)
				return
			}
		}
	} else if state.Status == db.CallStatusBeingInvited { // clean "dangling" beingInvited
		if time.Now().UTC().Sub(state.CreateDateTime) > viper.GetDuration("schedulers.chatDialerTask.removeDanglingCallStatusBeingInvitedAfter") {

			GetLogEntry(ctx).Infof("Removing dangling in status %v of user tokenId %v, userId %v, chat %v", state.Status, state.TokenId, state.UserId, chatId)
			err := tx.Remove(dto.UserCallStateId{
				TokenId: state.TokenId,
				UserId:  state.UserId,
			})
			if err != nil {
				GetLogEntry(ctx).Errorf("Unable invoke RemoveFromDialList, user tokenId %v, userId %v, chat %v, error %v", state.TokenId, state.UserId, chatId, err)
				return
			}

		}
	}
}


type ChatDialerTask struct {
	*gointerlock.GoInterval
}

func ChatDialerScheduler(
	redisConnector *redisV8.Client,
	service *ChatDialerService,
	conf *config.ExtendedConfig,
) *ChatDialerTask {
	var interv = viper.GetDuration("schedulers.chatDialerTask.dialPeriod")
	Logger.Infof("Created chats dialer with interval %v", interv)
	return &ChatDialerTask{&gointerlock.GoInterval{
		Name:           "chatDialerTask",
		Interval:       interv,
		Arg:            service.doJob,
		RedisConnector: redisConnector,
	}}
}
