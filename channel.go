package stream_chat

import (
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"time"
)

type ChannelMember struct {
	UserID      string `json:"user_id,omitempty"`
	User        *User  `json:"user,omitempty"`
	IsModerator bool   `json:"is_moderator,omitempty"`

	Invited          bool       `json:"invited,omitempty"`
	InviteAcceptedAt *time.Time `json:"invite_accepted_at,omitempty"`
	InviteRejectedAt *time.Time `json:"invite_rejected_at,omitempty"`
	Role             string     `json:"role,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Channel struct {
	ID          string
	Type        string
	CID         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   User
	Frozen      bool
	MemberCount int
	Config      map[string]interface{}
	Messages    []Message
	Members     []ChannelMember
	client      *Client
}

func (ch *Channel) fromMap(hmap map[string]interface{}) {
	if id, ok := hmap["id"].(string); ok {
		ch.ID = id
	}
	if _type, ok := hmap["type"].(string); ok {
		ch.Type = _type
	}
	if cid, ok := hmap["cid"].(string); ok {
		ch.CID = cid
	}
	if created, ok := hmap["cid"].(time.Time); ok {
		ch.CreatedAt = created
	}
	if updated, ok := hmap["cid"].(time.Time); ok {
		ch.UpdatedAt = updated
	}
	// todo: user
	if frozen, ok := hmap["frozen"].(bool); ok {
		ch.Frozen = frozen
	}

	if count, ok := hmap["member_count"].(float64); ok {
		ch.MemberCount = int(count)
	}
}

func addUserID(hash map[string]interface{}, userID string) map[string]interface{} {
	hash["user"] = userID
	return hash
}

// SendMessage sends a message to this channel
//
// message: the Message object
// userID: the ID of the user that created the message
func (ch *Channel) SendMessage(message Message, userID string) error {
	data := map[string]interface{}{
		"message": addUserID(message.toHash(), userID),
	}

	p := path.Join("channels", ch.Type, ch.ID, "message")

	return ch.client.makeRequest(http.MethodPost, p, nil, data, nil)
}

// SendEvent sends an event on this channel
//
// event: event data, ie {type: 'message.read'}
// userID: the ID of the user sending the event
func (ch *Channel) SendEvent(event Event, userID string) error {
	data := map[string]interface{}{
		"event": addUserID(event.toHash(), userID),
	}

	p := path.Join("channels", ch.Type, ch.ID, "event")

	return ch.client.makeRequest(http.MethodPost, p, nil, data, nil)
}

// SendReaction sends a reaction about a message
//
// messageID: the message id
// reaction: the reaction object, ie {type: 'love'}
// userID: the ID of the user that created the reaction
func (ch *Channel) SendReaction(messageID string, reaction Reaction, userID string) error {
	data := map[string]interface{}{
		"reaction": addUserID(reaction.toHash(), userID),
	}

	p := path.Join("messages", messageID, "reaction")

	return ch.client.makeRequest(http.MethodPost, p, nil, data, nil)
}

// DeleteReaction removes a reaction by user and type
//
// messageID: the id of the message from which te remove the reaction
// reaction_type: the type of reaction that should be removed
// userID: the id of the user
func (ch *Channel) DeleteReaction(messageID string, reactionType string, userID string) error {
	if messageID == "" {
		return errors.New("message ID must be not empty")
	}
	if reactionType == "" {
		return errors.New("reaction type must be not empty")
	}

	p := path.Join("messages", messageID, "reaction", reactionType)

	params := map[string][]string{
		"user_id": {userID},
	}

	return ch.client.makeRequest(http.MethodDelete, p, params, nil, nil)
}

// query makes request to channel api and updates channel internal state
func (ch *Channel) query(options map[string]interface{}, data map[string]interface{}) (err error) {
	payload := map[string]interface{}{
		"state": true,
	}

	for k, v := range options {
		payload[k] = v
	}

	if data == nil {
		data = map[string]interface{}{}
	}

	data["created_by"] = map[string]interface{}{"id": ch.CreatedBy.ID}

	payload["data"] = data

	p := path.Join("channels", ch.Type, ch.ID, "query")

	var resp struct {
		Channel  map[string]interface{}
		Messages []Message
		Members  []ChannelMember
	}

	err = ch.client.makeRequest(http.MethodPost, p, nil, payload, &resp)
	if err != nil {
		return err
	}

	ch.fromMap(resp.Channel)
	ch.Members = resp.Members
	ch.Messages = resp.Messages

	return nil
}

// Update edits the channel's custom properties
//
// options: the object to update the custom properties of this channel with
// message: optional update message
func (ch *Channel) Update(options map[string]interface{}, message string) error {
	payload := map[string]interface{}{
		"data":    options,
		"message": message,
	}

	p := path.Join("channels", ch.Type, ch.ID)

	return ch.client.makeRequest(http.MethodPost, p, nil, payload, nil)
}

// Delete removes the channel. Messages are permanently removed.
func (ch *Channel) Delete() error {
	p := path.Join("channels", ch.Type, ch.ID)

	return ch.client.makeRequest(http.MethodDelete, p, nil, nil, nil)
}

// Truncate removes all messages from the channel
func (ch *Channel) Truncate() error {
	p := path.Join("channels", ch.Type, ch.ID, "truncate")

	return ch.client.makeRequest(http.MethodPost, p, nil, nil, nil)
}

// Adds members to the channel
//
// users: user IDs to add as members
func (ch *Channel) AddMembers(users []string) error {
	data := map[string]interface{}{
		"add_members": users,
	}

	p := path.Join("channels", ch.Type, ch.ID)

	return ch.client.makeRequest(http.MethodPost, p, nil, data, nil)
}

//  RemoveMembers deletes members with given IDs from the channel
func (ch *Channel) RemoveMembers(userIDs []string) error {
	data := map[string]interface{}{
		"remove_members": userIDs,
	}

	p := path.Join("channels", ch.Type, ch.ID)

	return ch.client.makeRequest(http.MethodPost, p, nil, data, nil)
}

// AddModerators adds moderators with given IDs to the channel
func (ch *Channel) AddModerators(userIDs []string) error {
	data := map[string]interface{}{
		"add_moderators": userIDs,
	}

	p := path.Join("channels", ch.Type, ch.ID)

	return ch.client.makeRequest(http.MethodPost, p, nil, data, nil)
}

// DemoteModerators moderators with given IDs from the channel
func (ch *Channel) DemoteModerators(userIDs []string) error {
	data := map[string]interface{}{
		"demote_moderators": userIDs,
	}

	p := path.Join("channels", ch.Type, ch.ID)

	return ch.client.makeRequest(http.MethodPost, p, nil, data, nil)
}

//  MarkRead send the mark read event for this user, only works if the `read_events` setting is enabled
//
//  userID: the user ID for the event
//  options: additional data, ie {"messageID": last_messageID}
func (ch *Channel) MarkRead(userID string, options map[string]interface{}) error {
	if userID == "" {
		return errors.New("user ID must be not empty")
	}

	p := path.Join("channels", ch.Type, ch.ID, "read")

	options = addUserID(options, userID)

	return ch.client.makeRequest(http.MethodPost, p, nil, options, nil)
}

// GetReplies returns list of the message replies for a parent message
//
// parenID: The message parent id, ie the top of the thread
// options: Pagination params, ie {limit:10, idlte: 10}
func (ch *Channel) GetReplies(parentID string, options map[string][]string) (replies []Message, err error) {
	if parentID == "" {
		return nil, errors.New("parent ID must be not empty")
	}

	p := path.Join("messages", parentID, "replies")

	var resp json.RawMessage

	err = ch.client.makeRequest(http.MethodGet, p, options, nil, &resp)

	return replies, err
}

// GetReactions returns list of the reactions, supports pagination
//
// messageID: The message id
// options: Pagination params, ie {"limit":10, "idlte": 10}
func (ch *Channel) GetReactions(messageID string, options map[string][]string) ([]Reaction, error) {
	if messageID == "" {
		return nil, errors.New("messageID must be not empty")
	}

	p := path.Join("messages", messageID, "reactions")

	var resp struct {
		Reactions []Reaction `json:"reactions"`
	}

	err := ch.client.makeRequest(http.MethodGet, p, options, nil, &resp)

	return resp.Reactions, err
}

// BanUser bans target user ID from this channel
// options: additional ban options, ie {"timeout": 3600, "reason": "offensive language is not allowed here"}
func (ch *Channel) BanUser(targetID string, options map[string]interface{}) error {
	if targetID == "" {
		return errors.New("targetID must be not empty")
	}

	options["type"] = ch.Type
	options["id"] = ch.ID

	return ch.client.BanUser(targetID, options)
}

// UnBanUser removes the ban for target user ID on this channel
func (ch *Channel) UnBanUser(targetID string, options map[string]string) error {
	if targetID == "" {
		return errors.New("target ID must be not empty")
	}

	options["type"] = ch.Type
	options["id"] = ch.ID

	return ch.client.UnBanUser(targetID, options)
}

// CreateChannel creates new channel of given type and id or returns already created one
func (c *Client) CreateChannel(chanType string, chanID string, userID string, data map[string]interface{}) (*Channel, error) {
	ch := &Channel{
		Type:      chanType,
		ID:        chanID,
		client:    c,
		CreatedBy: User{ID: userID},
	}

	options := map[string]interface{}{
		"watch":    false,
		"state":    false,
		"presence": false,
	}

	err := ch.query(options, data)

	return ch, err
}
