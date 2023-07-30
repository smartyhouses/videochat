package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/guregu/null"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"nkonev.name/chat/auth"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"time"
)

const ReservedPublicallyAvailableForSearchChats = "__AVAILABLE_FOR_SEARCH"

// db model
type Chat struct {
	Id                 int64
	Title              string
	LastUpdateDateTime time.Time
	TetATet            bool
	CanResend          bool
	Avatar             null.String
	AvatarBig          null.String
	AvailableToSearch  bool
	Pinned             bool
	Blog               bool
}

type Blog struct {
	Id             int64
	Title          string
	CreateDateTime time.Time
	Avatar         null.String
}

type ChatWithParticipants struct {
	Chat
	ParticipantsIds   []int64
	ParticipantsCount int
	IsAdmin           bool
}

// CreateChat creates a new chat.
// Returns an error if user is invalid or the tx fails.
func (tx *Tx) CreateChat(u *Chat) (int64, *time.Time, error) {
	// Validate the input.
	if u == nil {
		return 0, nil, tracerr.Wrap(errors.New("chat required"))
	} else if u.Title == "" {
		return 0, nil, tracerr.Wrap(errors.New("title required"))
	}

	// https://stackoverflow.com/questions/4547672/return-multiple-fields-as-a-record-in-postgresql-with-pl-pgsql/6085167#6085167
	res := tx.QueryRow(`SELECT chat_id, last_update_date_time FROM CREATE_CHAT($1, $2, $3, $4, $5) AS (chat_id BIGINT, last_update_date_time TIMESTAMP)`, u.Title, u.TetATet, u.CanResend, u.AvailableToSearch, u.Blog)
	var id int64
	var lastUpdateDateTime time.Time
	if err := res.Scan(&id, &lastUpdateDateTime); err != nil {
		return 0, nil, tracerr.Wrap(err)
	}

	return id, &lastUpdateDateTime, nil
}

func (tx *Tx) CreateTetATetChat(behalfUserId int64, toParticipantId int64) (int64, error) {
	tetATetChatName := fmt.Sprintf("tet_a_tet_%v_%v", behalfUserId, toParticipantId)
	chatId, _, err := tx.CreateChat(&Chat{Title: tetATetChatName, TetATet: true, CanResend: viper.GetBool("canResendFromTetATet")})
	return chatId, err
}

func (tx *Tx) IsExistsTetATet(participant1 int64, participant2 int64) (bool, int64, error) {
	res := tx.QueryRow("select b.chat_id from (select a.count >= 2 as exists, a.chat_id from ( (select cp.chat_id, count(cp.user_id) from chat_participant cp join chat ch on ch.id = cp.chat_id where ch.tet_a_tet = true and (cp.user_id = $1 or cp.user_id = $2) group by cp.chat_id)) a) b where b.exists is true;", participant1, participant2)
	var chatId int64
	if err := res.Scan(&chatId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// there were no rows, but otherwise no error occurred
			return false, 0, nil
		}
		return false, 0, tracerr.Wrap(err)
	}
	return true, chatId, nil
}

// expects $1 is userId
func selectChatClause() string {
	return `SELECT 
				ch.id, 
				ch.title, 
				ch.avatar, 
				ch.avatar_big,
				ch.last_update_date_time,
				ch.tet_a_tet,
				ch.can_resend,
				ch.available_to_search,
				cp.user_id IS NOT NULL as pinned,
				ch.blog
			FROM chat ch 
				LEFT JOIN chat_pinned cp on (ch.id = cp.chat_id and cp.user_id = $1)
`
}

func provideScanToChat(chat *Chat) []any {
	return []any{
		&chat.Id,
		&chat.Title,
		&chat.Avatar,
		&chat.AvatarBig,
		&chat.LastUpdateDateTime,
		&chat.TetATet,
		&chat.CanResend,
		&chat.AvailableToSearch,
		&chat.Pinned,
		&chat.Blog,
	}
}


func (db *DB) GetChatCount(itemId, userId int64, previous bool) (int, error) {

	var sqlQ = `
		SELECT 
			ROW_NUMBER () OVER (ORDER BY (cp.user_id is not null, ch.last_update_date_time, ch.id) DESC),
			ch.id,
			ch.title
		FROM 
			chat ch 
			LEFT JOIN chat_pinned cp 
				on (ch.id = cp.chat_id and cp.user_id = 1) 
		WHERE ch.id IN ( SELECT chat_id FROM chat_participant WHERE user_id = 1 )
	`

	var theQuery string
	if (previous) {
		theQuery = fmt.Sprintf(sqlQ, "<")
	} else {
		theQuery = fmt.Sprintf(sqlQ, ">")
	}

	var count int
	row := db.QueryRow(theQuery, userId, itemId)
	err := row.Scan(&count)
	if err != nil {
		return 0, tracerr.Wrap(err)
	} else {
		return count, nil
	}
}

func (db *DB) GetChatsByLimitOffset(participantId int64, limit int, offset int) ([]*Chat, error) {
	var rows *sql.Rows
	var err error
	rows, err = db.Query(selectChatClause()+` WHERE ch.id IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 ) ORDER BY (cp.user_id is not null, ch.last_update_date_time, ch.id) DESC LIMIT $2 OFFSET $3`, participantId, limit, offset)
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		list := make([]*Chat, 0)
		for rows.Next() {
			chat := Chat{}
			if err := rows.Scan(provideScanToChat(&chat)[:]...); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func getBlogPostsByLimitOffsetCommon(co CommonOperations, limit int, offset int) ([]*Blog, error) {
	var rows *sql.Rows
	var err error
	rows, err = co.Query(`SELECT 
				ch.id, 
				ch.title,
				ch.create_date_time,
				ch.avatar
			FROM chat ch WHERE ch.blog is TRUE ORDER BY (ch.create_date_time, ch.id) DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		list := make([]*Blog, 0)
		for rows.Next() {
			chat := Blog{}
			if err := rows.Scan(&chat.Id, &chat.Title, &chat.CreateDateTime, &chat.Avatar); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func (tx *Tx) GetBlogPostsByLimitOffset(limit int, offset int) ([]*Blog, error) {
	return getBlogPostsByLimitOffsetCommon(tx, limit, offset)
}

func (db *DB) GetBlogPostsByLimitOffset(limit int, offset int) ([]*Blog, error) {
	return getBlogPostsByLimitOffsetCommon(db, limit, offset)
}

type BlogPost struct {
	ChatId    int64
	MessageId int64
	OwnerId   int64
	Text      string
}

func blogPostsCommon(co CommonOperations, ids []int64) ([]*BlogPost, error) {
	var builder = ""
	var first = true
	for _, chatId := range ids {
		if !first {
			builder += " union "
		}
		builder += fmt.Sprintf("(select %v, id, owner_id, text from message_chat_%v where blog_post is true order by id limit 1)", chatId, chatId)

		first = false
	}

	var rows *sql.Rows
	var err error
	rows, err = co.Query(builder)
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		list := make([]*BlogPost, 0)
		for rows.Next() {
			chat := BlogPost{}
			if err := rows.Scan(&chat.ChatId, &chat.MessageId, &chat.OwnerId, &chat.Text); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func (tx *Tx) BlogPosts(ids []int64) ([]*BlogPost, error) {
	return blogPostsCommon(tx, ids)
}

func (db *DB) BlogPosts(ids []int64) ([]*BlogPost, error) {
	return blogPostsCommon(db, ids)
}

func (db *DB) GetChatsByLimitOffsetSearch(participantId int64, limit int, offset int, searchString string, additionalFoundUserIds []int64) ([]*Chat, error) {
	var rows *sql.Rows
	var err error
	searchStringWithPercents := "%" + searchString + "%"

	var additionalUserIds = ""
	first := true
	for _, userId := range additionalFoundUserIds {
		if !first {
			additionalUserIds = additionalUserIds + ","
		}
		additionalUserIds = additionalUserIds + utils.Int64ToString(userId)
		first = false
	}
	var additionalUserIdsClause = ""
	if len(additionalFoundUserIds) > 0 {
		additionalUserIdsClause = fmt.Sprintf(" OR ( ch.tet_a_tet IS true AND ch.id IN ( SELECT chat_id FROM chat_participant WHERE user_id IN (%s) ) ) ", additionalUserIds)
	}

	rows, err = db.Query(selectChatClause()+fmt.Sprintf(`
		WHERE 
		    ( ( ch.id IN ( SELECT chat_id FROM chat_participant WHERE user_id = $1 ) OR ( ch.available_to_search IS TRUE ) ) AND ( $5 = '%s' or ch.title ILIKE $4 %s ) ) 
			ORDER BY (cp.user_id is not null, ch.last_update_date_time, ch.id) DESC 
			LIMIT $2 OFFSET $3
	`, ReservedPublicallyAvailableForSearchChats, additionalUserIdsClause), participantId, limit, offset, searchStringWithPercents, searchString)
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		list := make([]*Chat, 0)
		for rows.Next() {
			chat := Chat{}
			if err := rows.Scan(provideScanToChat(&chat)[:]...); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, &chat)
			}
		}
		return list, nil
	}
}

func convertToWithParticipants(db CommonOperations, chat *Chat, behalfUserId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	if ids, err := db.GetParticipantIds(chat.Id, participantsSize, participantsOffset); err != nil {
		return nil, err
	} else {
		admin, err := db.IsAdmin(behalfUserId, chat.Id)
		if err != nil {
			return nil, err
		}
		participantsCount, err := db.GetParticipantsCount(chat.Id)
		if err != nil {
			return nil, err
		}
		ccc := &ChatWithParticipants{
			Chat:              *chat,
			ParticipantsIds:   ids,
			IsAdmin:           admin,
			ParticipantsCount: participantsCount,
		}
		return ccc, nil
	}
}

func convertToWithoutParticipants(db CommonOperations, chat *Chat, behalfUserId int64) (*ChatWithParticipants, error) {
	admin, err := db.IsAdmin(behalfUserId, chat.Id)
	if err != nil {
		return nil, err
	}
	ccc := &ChatWithParticipants{
		Chat:              *chat,
		ParticipantsIds:   []int64{}, // to be set in callee
		IsAdmin:           admin,
		ParticipantsCount: 0, // to be set in callee
	}
	return ccc, nil
}

type ChatQueryByLimitOffset struct {
	Limit  int
	Offset int
}

type ChatQueryByIds struct {
	Ids []int64
}

type ParticipantIds struct {
	ChatId         int64
	ParticipantIds []int64
}

func convertToWithParticipantsBatch(chat *Chat, participantIdsBatch []*ParticipantIds, isAdminBatch map[int64]bool, participantsCountBatch map[int64]int) (*ChatWithParticipants, error) {
	participantsCount := participantsCountBatch[chat.Id]

	var participantsIds []int64 = make([]int64, 0)
	for _, pidsb := range participantIdsBatch {
		if pidsb.ChatId == chat.Id {
			participantsIds = pidsb.ParticipantIds
			break
		}
	}

	admin := isAdminBatch[chat.Id]

	ccc := &ChatWithParticipants{
		Chat:              *chat,
		ParticipantsIds:   participantsIds,
		IsAdmin:           admin,
		ParticipantsCount: participantsCount,
	}
	return ccc, nil
}

func (db *DB) GetChatsWithParticipants(participantId int64, limit, offset int, searchString string, additionalFoundUserIds []int64, userPrincipalDto *auth.AuthResult, participantsSize, participantsOffset int) ([]*ChatWithParticipants, error) {
	var err error
	var chats []*Chat

	if searchString == "" {
		chats, err = db.GetChatsByLimitOffset(participantId, limit, offset)
	} else {
		chats, err = db.GetChatsByLimitOffsetSearch(participantId, limit, offset, searchString, additionalFoundUserIds)
	}

	if err != nil {
		return nil, err
	} else {
		var chatIds []int64 = make([]int64, 0)
		for _, cc := range chats {
			chatIds = append(chatIds, cc.Id)
		}

		fixedParticipantsSize := utils.FixSize(participantsSize)
		participantIdsBatch, err := db.GetParticipantIdsBatch(chatIds, fixedParticipantsSize)
		if err != nil {
			return nil, err
		}

		isAdminBatch, err := db.IsAdminBatch(userPrincipalDto.UserId, chatIds)
		if err != nil {
			return nil, err
		}

		participantsCountBatch, err := db.GetParticipantsCountBatch(chatIds)
		if err != nil {
			return nil, err
		}

		list := make([]*ChatWithParticipants, 0)

		for _, cc := range chats {
			if ccc, err := convertToWithParticipantsBatch(cc, participantIdsBatch, isAdminBatch, participantsCountBatch); err != nil {
				return nil, err
			} else {
				list = append(list, ccc)
			}
		}
		return list, nil
	}
}

func (tx *Tx) GetChatWithParticipants(behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(tx, behalfParticipantId, chatId, participantsSize, participantsOffset)
}

func (db *DB) GetChatWithParticipants(behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	return getChatWithParticipantsCommon(db, behalfParticipantId, chatId, participantsSize, participantsOffset)
}

func getChatWithParticipantsCommon(commonOps CommonOperations, behalfParticipantId, chatId int64, participantsSize, participantsOffset int) (*ChatWithParticipants, error) {
	if chat, err := commonOps.GetChat(behalfParticipantId, chatId); err != nil {
		return nil, err
	} else if chat == nil {
		return nil, nil
	} else {
		return convertToWithParticipants(commonOps, chat, behalfParticipantId, participantsSize, participantsOffset)
	}
}

func (tx *Tx) GetChatWithoutParticipants(behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	return getChatWithoutParticipantsCommon(tx, behalfParticipantId, chatId)
}

func (db *DB) GetChatWithoutParticipants(behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	return getChatWithoutParticipantsCommon(db, behalfParticipantId, chatId)
}

func getChatWithoutParticipantsCommon(commonOps CommonOperations, behalfParticipantId, chatId int64) (*ChatWithParticipants, error) {
	if chat, err := commonOps.GetChat(behalfParticipantId, chatId); err != nil {
		return nil, err
	} else if chat == nil {
		return nil, nil
	} else {
		return convertToWithoutParticipants(commonOps, chat, behalfParticipantId)
	}
}

func (db *DB) CountChats() (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM chat")
	err := row.Scan(&count)
	if err != nil {
		return 0, tracerr.Wrap(err)
	} else {
		return count, nil
	}
}

func (db *DB) CountChatsPerUser(userId int64) (int64, error) {
	var count int64
	row := db.QueryRow("SELECT count(*) FROM chat_participant WHERE user_id = $1", userId)
	err := row.Scan(&count)
	if err != nil {
		return 0, tracerr.Wrap(err)
	} else {
		return count, nil
	}
}

func (tx *Tx) DeleteChat(id int64) error {
	if _, err := tx.Exec(fmt.Sprintf(`DROP TABLE message_chat_%v;`, id)); err != nil {
		return tracerr.Wrap(err)
	}

	if _, err := tx.Exec(fmt.Sprintf(`DROP SEQUENCE message_chat_id_%v;`, id)); err != nil {
		return tracerr.Wrap(err)
	}

	if res, err := tx.Exec("DELETE FROM chat WHERE id = $1", id); err != nil {
		return tracerr.Wrap(err)
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			return tracerr.Wrap(err)
		}
		if affected == 0 {
			return tracerr.Wrap(errors.New("No rows affected"))
		}
		return nil
	}
}

func (tx *Tx) EditChat(id int64, newTitle string, avatar, avatarBig null.String, canResend bool, availableToSearch bool, blog bool) (*time.Time, error) {

	if res, err := tx.Exec(`UPDATE chat SET title = $2, avatar = $3, avatar_big = $4, last_update_date_time = utc_now(), can_resend = $5, available_to_search = $6, blog = $7 WHERE id = $1`, id, newTitle, avatar, avatarBig, canResend, availableToSearch, blog); err != nil {
		Logger.Errorf("Error during editing chat id %v", err)
		return nil, err
	} else {
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		if affected == 0 {
			return nil, tracerr.Wrap(errors.New("No rows affected"))
		}
	}

	var lastUpdateDateTime time.Time
	res := tx.QueryRow(`SELECT last_update_date_time FROM chat WHERE id = $1`, id)
	if err := res.Scan(&lastUpdateDateTime); err != nil {
		return nil, tracerr.Wrap(err)
	}

	return &lastUpdateDateTime, nil
}

func getChatCommon(co CommonOperations, participantId, chatId int64) (*Chat, error) {
	row := co.QueryRow(selectChatClause()+` WHERE ch.id in (SELECT chat_id FROM chat_participant WHERE user_id = $1 AND chat_id = $2)`, participantId, chatId)
	chat := Chat{}
	err := row.Scan(provideScanToChat(&chat)[:]...)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		return &chat, nil
	}
}

func (db *DB) GetChat(participantId, chatId int64) (*Chat, error) {
	return getChatCommon(db, participantId, chatId)
}

func (tx *Tx) GetChat(participantId, chatId int64) (*Chat, error) {
	return getChatCommon(tx, participantId, chatId)
}

func getChatBasicCommon(co CommonOperations, chatId int64) (*BasicChatDto, error) {
	row := co.QueryRow(`SELECT 
				ch.id, 
				ch.title, 
				ch.tet_a_tet,
				ch.can_resend,
				ch.available_to_search,
				ch.blog,
				ch.create_date_time
			FROM chat ch 
			WHERE ch.id = $1
`, chatId)
	chat := BasicChatDto{}
	err := row.Scan(&chat.Id, &chat.Title, &chat.IsTetATet, &chat.CanResend, &chat.AvailableToSearch, &chat.IsBlog, &chat.CreateDateTime)
	if errors.Is(err, sql.ErrNoRows) {
		// there were no rows, but otherwise no error occurred
		return nil, nil
	}
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		return &chat, nil
	}
}

func (db *DB) GetChatBasic(chatId int64) (*BasicChatDto, error) {
	return getChatBasicCommon(db, chatId)
}

func (tx *Tx) GetChatBasic(chatId int64) (*BasicChatDto, error) {
	return getChatBasicCommon(tx, chatId)
}

func getChatsBasicCommon(co CommonOperations, chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
	result := map[int64]*BasicChatDtoExtended{}
	if len(chatIds) == 0 {
		return result, nil
	}

	inClause := ""
	first := true
	for chatId, val := range chatIds {
		if val {
			dtl := ""
			if !first {
				dtl = ","
			}
			dtl += utils.Int64ToString(chatId)
			inClause = inClause + dtl
		}

		first = false
	}
	rows, err := co.Query(fmt.Sprintf(`
		SELECT 
			c.id, 
			c.title,
			(cp.user_id is not null),
			c.tet_a_tet,
			c.can_resend,
			c.available_to_search,
			c.blog,
			c.create_date_time
		FROM chat c 
		    LEFT JOIN chat_participant cp 
		        ON (c.id = cp.chat_id AND cp.user_id = $1) 
		WHERE c.id IN (%s)`, inClause),
		behalfParticipantId)
	if err != nil {
		return nil, tracerr.Wrap(err)
	} else {
		defer rows.Close()
		list := make([]*BasicChatDtoExtended, 0)
		for rows.Next() {
			dto := new(BasicChatDtoExtended)
			if err := rows.Scan(&dto.Id, &dto.Title, &dto.BehalfUserIsParticipant, &dto.IsTetATet, &dto.CanResend, &dto.AvailableToSearch, &dto.IsBlog, &dto.CreateDateTime); err != nil {
				return nil, tracerr.Wrap(err)
			} else {
				list = append(list, dto)
			}
		}
		for _, bc := range list {
			result[bc.Id] = bc
		}
		return result, nil
	}
}

func (db *DB) GetChatsBasic(chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
	return getChatsBasicCommon(db, chatIds, behalfParticipantId)
}

func (tx *Tx) GetChatsBasic(chatIds map[int64]bool, behalfParticipantId int64) (map[int64]*BasicChatDtoExtended, error) {
	return getChatsBasicCommon(tx, chatIds, behalfParticipantId)
}

type BasicChatDto struct {
	Id                int64
	Title             string
	IsTetATet         bool
	CanResend         bool
	AvailableToSearch bool
	IsBlog            bool
	CreateDateTime    time.Time
}

type BasicChatDtoExtended struct {
	BasicChatDto
	BehalfUserIsParticipant bool
}

func (tx *Tx) UpdateChatLastDatetimeChat(id int64) error {
	if _, err := tx.Exec("UPDATE chat SET last_update_date_time = utc_now() WHERE id = $1", id); err != nil {
		return tracerr.Wrap(err)
	} else {
		return nil
	}
}

func (tx *Tx) GetChatLastDatetimeChat(chatId int64) (time.Time, error) {
	row := tx.QueryRow(`SELECT last_update_date_time FROM chat WHERE id = $1`, chatId)
	var lastUpdateDateTime time.Time
	err := row.Scan(&lastUpdateDateTime)
	if err != nil {
		return lastUpdateDateTime, tracerr.Wrap(err)
	} else {
		return lastUpdateDateTime, nil
	}
}

func (db *DB) IsChatExists(chatId int64) (bool, error) {
	row := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM chat WHERE id = $1)`, chatId)
	exists := false
	err := row.Scan(&exists)
	if err != nil {
		return false, tracerr.Wrap(err)
	} else {
		return exists, nil
	}

}

func pinChatCommon(co CommonOperations, chatId int64, userId int64, pin bool) error {
	if pin {
		_, err := co.Exec("insert into chat_pinned(user_id, chat_id) values ($1, $2) on conflict do nothing", userId, chatId)
		if err != nil {
			return tracerr.Wrap(err)
		}
	} else {
		_, err := co.Exec("delete from chat_pinned where user_id = $1 and chat_id = $2", userId, chatId)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (db *DB) PinChat(chatId int64, userId int64, pin bool) error {
	return pinChatCommon(db, chatId, userId, pin)
}

func (tx *Tx) PinChat(chatId int64, userId int64, pin bool) error {
	return pinChatCommon(tx, chatId, userId, pin)
}

func (tx *Tx) IsChatPinned(chatId int64, behalfUserId int64) (bool, error) {
	var res bool
	row := tx.QueryRow(`SELECT 
    	cp.user_id IS NOT NULL as pinned 
		FROM chat ch 
		    LEFT JOIN chat_pinned cp on (ch.id = cp.chat_id and cp.user_id = $1) WHERE ch.id = $2`,
		behalfUserId,
		chatId,
	)
	if row.Err() != nil {
		return false, tracerr.Wrap(row.Err())
	}
	if err := row.Scan(&res); err != nil {
		return false, tracerr.Wrap(err)
	}
	return res, nil
}

func (tx *Tx) RenameChat(chatId int64, title string) error {
	_, err := tx.Exec("update chat set title = $1 where id = $2", title, chatId)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (db *DB) DeleteAllParticipants() error {
	// see aaa/src/main/resources/db/demo/V32000__demo.sql
	// 1 admin
	// 2 nikita
	// 3 alice
	// 4 bob
	// 5 John Smith
	_, err := db.Exec("DELETE FROM chat_participant WHERE user_id > 5")
	return tracerr.Wrap(err)
}
