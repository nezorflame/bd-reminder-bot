package slack

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Slack consts
const (
	TypeMessage     = "message"
	UserInviteLimit = 30
)

const (
	postMessageURL          = "https://slack.com/api/chat.postMessage"
	conversationsCreateURL  = "https://slack.com/api/conversations.create"
	conversationsInfoURL    = "https://slack.com/api/conversations.info"
	conversationsInviteURL  = "https://slack.com/api/conversations.invite"
	conversationsListURL    = "https://slack.com/api/conversations.list"
	conversationsMembersURL = "https://slack.com/api/conversations.members"
	imListURL               = "https://slack.com/api/im.list"
	userProfileURL          = "https://slack.com/api/users.profile.get"

	alreadyInChannelErrorMsg = "already_in_channel"
	cantInviteSelfErrorMsg   = "cant_invite_self"
)

// SendAPIMessage sends a message with Web API
func SendAPIMessage(token, chanID, message string) error {
	var response struct {
		OK      bool   `json:"ok"`
		Error   string `json:"error,omitempty"`
		TS      string `json:"ts,omitempty"`
		Message struct {
			Username string `json:"username"`
			BotID    string `json:"bot_id"`
			Type     string `json:"type"`
			Subtype  string `json:"subtype"`
		} `json:"message,omitempty"`
	}

	request := struct {
		Conversation string `json:"channel"`
		Text         string `json:"text"`
		AsUser       bool   `json:"as_user"`
		Username     string `json:"username"`
		IconEmoji    string `json:"icon_emoji"`
	}{
		Conversation: chanID,
		Text:         message,
		AsUser:       false,
		Username:     "Birthday in RnD",
		IconEmoji:    ":cake:",
	}
	body, err := json.Marshal(request)
	if err != nil {
		return errors.Wrap(err, "unable to marshal request")
	}
	headers := map[string]string{"Authorization": "Bearer " + token}

	body, err = makeRequest(postMessageURL, methodPOST, contentJSON, body, nil, headers)
	if err != nil {
		return errors.Wrap(err, "unable to make POST request")
	}

	if err = json.Unmarshal(body, &response); err != nil {
		return errors.Wrap(err, "unable to unmarshal response")
	}

	if !response.OK {
		return errors.Errorf("API error: %s", response.Error)
	}

	return nil
}

// CreateNewConversation creates new Slack conversation and returns its ID and error, if any
func CreateNewConversation(token, conversationName string, isPrivate bool) (string, error) {
	var response struct {
		OK           bool         `json:"ok"`
		Error        string       `json:"error"`
		Conversation Conversation `json:"channel"`
	}

	params := map[string]string{
		"token":      token,
		"name":       conversationName,
		"is_private": strconv.FormatBool(isPrivate),
	}
	respBody, err := makeRequest(conversationsCreateURL, methodGET, contentEncoded, nil, params, nil)
	if err != nil {
		return "", errors.Wrap(err, "unable to make GET request")
	}

	if err = json.Unmarshal(respBody, &response); err != nil {
		return "", errors.Wrap(err, "unable to unmarshal response")
	}

	if !response.OK {
		return "", errors.Errorf("API error: %s", response.Error)
	}

	return response.Conversation.ID, nil
}

// GetConversationInfo returns info about the Slack conversation by its ID
func GetConversationInfo(token, chanID string) (*Conversation, error) {
	var response struct {
		OK           bool         `json:"ok"`
		Error        string       `json:"error"`
		Conversation Conversation `json:"channel"`
	}

	params := map[string]string{"token": token, "channel": chanID}
	respBody, err := makeRequest(conversationsInfoURL, methodGET, contentEncoded, nil, params, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to make GET request")
	}

	if err = json.Unmarshal(respBody, &response); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal response")
	}

	if !response.OK {
		return nil, errors.Errorf("API error: %s", response.Error)
	}

	return &response.Conversation, nil
}

// GetConversations returns info about all of the public and private Slack conversations in the workspace
func GetConversations(token string, withArchived bool) ([]Conversation, error) {
	var response struct {
		OK            bool           `json:"ok"`
		Error         string         `json:"error"`
		Conversations []Conversation `json:"channels"`
	}

	params := map[string]string{
		"token":            token,
		"limit":            "10000",
		"types":            "public_channel,private_channel",
		"exclude_archived": fmt.Sprintf("%t", !withArchived),
	}
	respBody, err := makeRequest(conversationsListURL, methodGET, contentEncoded, nil, params, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to make GET request")
	}

	if err = json.Unmarshal(respBody, &response); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal response")
	}

	if !response.OK {
		return nil, errors.Errorf("API error: %s", response.Error)
	}

	return response.Conversations, nil
}

// GetConversationMembers returns the Slack conversation member list by conversation ID
func GetConversationMembers(token, chanID string) ([]string, error) {
	var response struct {
		OK      bool     `json:"ok"`
		Error   string   `json:"error"`
		Members []string `json:"members"`
	}

	params := map[string]string{"token": token, "channel": chanID}
	respBody, err := makeRequest(conversationsMembersURL, methodGET, contentEncoded, nil, params, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to make GET request")
	}

	if err = json.Unmarshal(respBody, &response); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal response")
	}

	if !response.OK {
		return nil, errors.Errorf("API error: %s", response.Error)
	}

	return response.Members, nil
}

// GetUserProfile returns the Slack user's profile by user ID
func GetUserProfile(token, userID string) (*UserProfile, error) {
	var response struct {
		OK      bool        `json:"ok"`
		Error   string      `json:"error"`
		Profile UserProfile `json:"profile"`
	}

	params := map[string]string{"token": token, "user": userID}
	respBody, err := makeRequest(userProfileURL, methodGET, contentEncoded, nil, params, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to make GET request")
	}

	if err = json.Unmarshal(respBody, &response); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal response")
	}

	if !response.OK {
		return nil, errors.Errorf("API error: %s", response.Error)
	}

	response.Profile.ID = userID
	return &response.Profile, nil
}

// FindDMByUserID returns Slack IM ID for the provided user ID
func FindDMByUserID(token, userID string) (string, error) {
	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		IMs   []IM   `json:"ims"`
	}

	params := map[string]string{"token": token}
	respBody, err := makeRequest(imListURL, methodGET, contentEncoded, nil, params, nil)
	if err != nil {
		return "", errors.Wrap(err, "unable to make GET request")
	}

	if err = json.Unmarshal(respBody, &response); err != nil {
		return "", errors.Wrap(err, "unable to unmarshal response")
	}

	if !response.OK {
		return "", errors.Errorf("API error: %s", response.Error)
	}

	for _, im := range response.IMs {
		if im.User == userID {
			return im.ID, nil
		}
	}

	return "", errors.New("user not found")
}

// InviteMembersToConversation adds members from the slice to the Slack conversation
func InviteMembersToConversation(token, chanID string, memberIDs []string) error {
	var response struct {
		OK           bool         `json:"ok"`
		Error        string       `json:"error,omitempty"`
		Conversation Conversation `json:"channel,omitempty"`
		Errors       []struct {
			User  string `json:"user"`
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		} `json:"errors,omitempty"`
	}

	var request struct {
		Conversation string `json:"channel"`
		Users        string `json:"users"`
	}

	request.Conversation = chanID
	for _, id := range memberIDs {
		request.Users = id
		logrus.Debugln("Adding user", request.Users)

		body, err := json.Marshal(request)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal request for user %s", id)
		}
		headers := map[string]string{"Authorization": "Bearer " + token}

		body, err = makeRequest(conversationsInviteURL, methodPOST, contentJSON, body, nil, headers)
		if err != nil {
			return errors.Wrapf(err, "unable to make POST request for user %s", id)
		}

		if err = json.Unmarshal(body, &response); err != nil {
			return errors.Wrapf(err, "unable to unmarshal response for user %s", id)
		}

		if !response.OK {
			if len(response.Errors) > 0 {
				err = errors.Errorf("total error count: %d", len(response.Errors))
				flag := false
				for _, e := range response.Errors {
					if !e.OK {
						switch e.Error {
						case alreadyInChannelErrorMsg, cantInviteSelfErrorMsg:
							// ignore and do nothing
						case "":
							flag = true
							err = errors.Wrapf(err, "user %s - unknown error", e.User)
						default:
							flag = true
							err = errors.Wrapf(err, "user %s - %s", e.User, e.Error)
						}
					}
				}
				if flag {
					return errors.Wrapf(err, "multiple API errors")
				}
			} else {
				switch response.Error {
				case alreadyInChannelErrorMsg, cantInviteSelfErrorMsg:
					// ignore and do nothing
				case "":
					return errors.Errorf("unknown API error for user %s", id)
				default:
					return errors.Errorf("API error for user %s: %s", id, response.Error)
				}
			}
		}
	}

	return nil
}
