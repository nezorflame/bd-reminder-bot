package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nezorflame/bd-reminder-bot/slack"

	goage "github.com/bearbin/go-age"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	ws "golang.org/x/net/websocket"
)

const (
	commandHi       = "hi"
	commandBirthday = "birthday"
	commandShutdown = "turnoff"

	errorMsgNameTaken = "API error: name_taken"
)

func msgWatcher(ctx context.Context, conn *ws.Conn, c *config, msgs *messages) error {
	for {
		select {
		case <-ctx.Done():
			logrus.Warnln("Stopping message watcher")
			return nil
		default:
			// read each incoming message
			m, err := slack.GetWSMessage(conn)
			if err != nil {
				// ignore timeout errors, log others
				if netErr, ok := err.(net.Error); ok {
					if netErr.Timeout() || netErr.Temporary() {
						continue
					}
				}
				// exit to try to recover from this error
				logrus.WithError(err).Debugln("Not a message")
				return err
			}

			// skip non-messages
			if m.Type != slack.TypeMessage {
				continue
			}

			// see if we're mentioned
			if strings.HasPrefix(m.Text, "<@"+c.BotUID+">") {
				// if so try to parse it
				mText := strings.Replace(m.Text, "<@"+c.BotUID+">", "", 1)
				mText = strings.Trim(mText, " ")
				logrus.Debugln(mText)
				switch strings.ToLower(mText) {
				case commandHi:
					go func(m slack.Message) {
						m.Text = "<@" + m.User + "> hello!"
						if err := slack.SendWSMessage(conn, m); err != nil {
							logrus.WithError(err).Errorln("Unable to send message to Slack")
						}
					}(m)
				case commandBirthday:
					go func(m slack.Message) {
						user, err := slack.GetUserProfile(c.LegacyToken, m.User)
						if err != nil {
							logrus.WithError(err).Error("Unable to get user profile")
							m.Text = fmt.Sprintf(msgs.ProfileError, m.User)
							if err := slack.SendWSMessage(conn, m); err != nil {
								logrus.WithError(err).Errorln("Unable to send message to Slack")
							}
							return
						}

						days, err := getUserBDInfo(time.Now().In(c.Location), user.Skype)
						if err != nil {
							logrus.WithError(err).Error("Unable to get user BD info")
							m.Text = fmt.Sprintf(msgs.BDParseError, user.ID)
						} else if days > 0 {
							logrus.Infof("User %s: %d days left", user.ID, days)
							m.Text = fmt.Sprintf(msgs.PersonalIncoming, user.ID, days)
						} else {
							logrus.Infof("User %s: birthday is today", user.ID)
							m.Text = fmt.Sprintf(msgs.PersonalToday, user.ID)
						}
						if err := slack.SendWSMessage(conn, m); err != nil {
							logrus.WithError(err).Errorln("Unable to send message to Slack")
						}
					}(m)
				case commandShutdown:
					if m.User != c.ManagerID {
						m.Text = msgs.ShutdownError
						if err := slack.SendWSMessage(conn, m); err != nil {
							logrus.WithError(err).Errorln("Unable to send message to Slack")
						}
						continue
					}
					if msgs.ShutdownAnnounce != "" {
						m.Text = msgs.ShutdownAnnounce
						if err := slack.SendWSMessage(conn, m); err != nil {
							logrus.WithError(err).Errorln("Unable to send message to Slack")
						}
					}
					return nil
				default:
					// ignore this
					continue
				}
			}
		}
	}
}

func bdWatcher(ctx context.Context, db *DB, c *config, m *messages) error {
	// first start
	mgrDM, err := slack.FindDMByUserID(c.LegacyToken, c.ManagerID)
	if err != nil {
		return errors.Wrap(err, "unable to find manager's DM")
	}
	c.ManagerDM = mgrDM

	now := time.Now().In(c.Location)
	logrus.Infoln("Starting first birthday check at", now.Format(time.RFC1123))
	if now.Hour() >= c.WorkdayStart && now.Hour() <= c.WorkdayEnd {
		if err := announceBirthdays(db, c, m); err != nil {
			return errors.Wrap(err, "unable to print birthdays")
		}
	} else {
		logrus.Info("Too soon, skipping first check")
	}

	// run the watcher on hourly basis
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			logrus.Warn("Stopping birthday watcher")
			ticker.Stop()
			return nil
		case t := <-ticker.C:
			// check if we can send it now
			t = t.In(c.Location)
			logrus.Infoln("Starting new birthday check at", t.Format(time.RFC1123))
			if t.Hour() < c.WorkdayStart || t.Hour() > c.WorkdayEnd {
				logrus.Info("Too soon to check, skipping")
				continue
			}

			if err := announceBirthdays(db, c, m); err != nil {
				ticker.Stop()
				return errors.Wrap(err, "unable to print birthdays")
			}
		}
	}
}

func announceBirthdays(db *DB, c *config, m *messages) error {
	now := time.Now().In(c.Location)

	chMembers, err := slack.GetConversationMembers(c.LegacyToken, c.MainChannelID)
	if err != nil {
		return errors.Wrap(err, "unable to get channel")
	}

	if len(chMembers) == 0 {
		return errors.New("Slack channel is empty")
	}

	logrus.Debugln("Members before blacklisting:", len(chMembers))
	for i := 0; i < len(chMembers); i++ {
		// remove blacklisted items
		if stringInSlice(chMembers[i], c.Blacklist) {
			logrus.Debugln("Blacklisting", chMembers[i])
			chMembers = append(chMembers[:i], chMembers[i+1:]...)
			i--
		}
	}
	logrus.Debugln("Members after blacklisting:", len(chMembers))

	// create worker goroutine and gather results
	var profiles []*slack.UserProfile
	ch := make(chan *slack.UserProfile)
	var wg sync.WaitGroup
	wg.Add(len(chMembers))
	for i := range chMembers {
		go func(i int) {
			defer wg.Done()
			user, err := slack.GetUserProfile(c.LegacyToken, chMembers[i])
			if err != nil {
				logrus.WithError(err).Errorf("Unable to get user %s", chMembers[i])
				return
			}
			logrus.Debugln("Adding", user.ID, user.RealName)
			ch <- user
		}(i)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for p := range ch {
		profiles = append(profiles, p)
	}
	logrus.Infof("Main channel contains %d valid members", len(profiles))

	managerAnnounceMap := make(map[string]bdInfo)
	channelAnnounceMap := make(map[string]bdInfo)
	for _, p := range profiles {
		logrus.Debugln(p.ID, p.RealName, p.Skype)
		days, err := getUserBDInfo(now, p.Skype)
		if err != nil {
			// we can ignore this error, just log in debug mode
			logrus.Debug(err)
			continue
		}

		year, _, _ := now.Date()
		currentBD := p.Skype + strconv.Itoa(year)

		// adding only the people who have BD in less than bdTreshold days
		if days <= c.BDHighTreshold && days > c.BDLowTreshold {
			logrus.Infof("Checking manager cache for user %s", p.ID)
			ok, err := db.CheckUserBDInCache(db.ManagerBucketName, p.ID, currentBD)
			if err != nil {
				logrus.WithError(err).Errorf("Unable to check user %s in cache", p.ID)
			} else if ok {
				logrus.Infof("User %s is present in manager cache, skipping", p.ID)
				continue
			}

			logrus.Infof("Informing manager about user %s (%d day(s) left)", p.ID, days)
			managerAnnounceMap[p.ID] = bdInfo{p.RealName, strings.ToLower(p.LastName), currentBD, days}
		} else if days <= c.BDLowTreshold {
			logrus.Infof("Checking channel cache for user %s", p.ID)
			ok, err := db.CheckUserBDInCache(db.ChannelBucketName, p.ID, currentBD)
			if err != nil {
				logrus.WithError(err).Errorf("Unable to check user %s in cache", p.ID)
			} else if ok {
				logrus.Infof("User %s is present in channel cache, skipping", p.ID)
				continue
			}

			logrus.Infof("Creating channel about user %s (%d day(s) left)", p.ID, days)
			channelAnnounceMap[p.ID] = bdInfo{p.RealName, strings.ToLower(p.LastName), currentBD, days}
		}
	}

	for id, info := range managerAnnounceMap {
		if err := slack.SendAPIMessage(
			c.LegacyToken, c.ManagerDM, fmt.Sprintf(m.ManagerAnnounce, id, info.DaysLeft),
		); err != nil {
			logrus.WithError(err).Errorf("Unable to send message to user %s", id)
			continue
		}

		// add to cache
		if err := db.SaveUserBDToCache(db.ManagerBucketName, id, info.Birthday); err != nil {
			logrus.WithError(err).Errorf("Unable to save birthday in manager cache for user %s", id)
			continue
		}
		logrus.Infoln("Saved birthday in manager cache for user", id)
	}

	if len(channelAnnounceMap) > 0 {
		if err := sendBDsToNewChannels(db, c, m.ChannelAnnounce, channelAnnounceMap); err != nil {
			logrus.WithError(err).Errorf("Unable to send birthdays to channels")
			return err
		}
	}

	logrus.Infoln("Finished check, sleeping")
	return nil
}

func sendBDsToNewChannels(db *DB, c *config, announce string, userInfoMap map[string]bdInfo) error {
	for id, info := range userInfoMap {
		// form the channel name
		year := time.Now().In(c.Location).Year()
		chanName := strings.Replace(fmt.Sprintf("%s-bd-%d", info.Surname, year), ".", "", -1)
		logrus.Debugln("Creating new channel", chanName)

		// create new private channel
		chanID, err := slack.CreateNewConversation(c.LegacyToken, chanName, true)
		if err != nil {
			if err.Error() != errorMsgNameTaken {
				return errors.Wrapf(err, "unable to create channel with name %s", chanName)
			}
			logrus.Infof("Channel %s already exists, getting its ID...", chanName)

			conversations, err := slack.GetConversations(c.LegacyToken, false)
			if err != nil {
				return errors.Wrap(err, "unable to get conversations")
			}

			for _, c := range conversations {
				if c.Name == chanName {
					chanID = c.ID
					break
				}
			}
		}
		// if empty, something went wrong
		if chanID == "" {
			return errors.Errorf("channel with name %s not found", chanName)
		}

		// get main channel member list
		members, err := slack.GetConversationMembers(c.LegacyToken, c.MainChannelID)
		if err != nil {
			return errors.Wrap(err, "unable to get main channel members")
		}

		// check blacklist
		// skip manager, if it's not his/her birthday
		logrus.Debugln("Members before blacklisting:", len(members))
		for i := 0; i < len(members); i++ {
			if stringInSlice(members[i], c.Blacklist) && members[i] != c.ManagerID || members[i] == id {
				logrus.Debugln("Blacklisting", members[i])
				members = append(members[:i], members[i+1:]...)
				i--
			}
		}
		logrus.Debugln("Members after blacklisting:", len(members))

		// invite main channel members
		err = slack.InviteMembersToConversation(c.LegacyToken, chanID, members)
		if err != nil {
			return errors.Wrapf(err, "unable to invite members to channel %s", chanName)
		}

		// send the greeting message
		bdDate := info.Birthday[:2] + "." + info.Birthday[2:4] + "." + info.Birthday[4:]
		if err := slack.SendAPIMessage(
			c.LegacyToken, chanID, fmt.Sprintf(announce, id, info.RealName, bdDate, c.ManagerID),
		); err != nil {
			return errors.Wrapf(err, "unable to send message to channel with ID %s", chanID)
		}

		// add to cache
		if err := db.SaveUserBDToCache(db.ChannelBucketName, id, info.Birthday); err != nil {
			logrus.WithError(err).Errorf("Unable to save birthday in channel cache for user %s", id)
		} else {
			logrus.Infoln("Saved birthday in channel cache for user", id)
		}

		logrus.Infof("Posted birthday message for the user %s in the channel %s", id, chanName)
	}

	return nil
}

func getUserBDInfo(now time.Time, userBD string) (days int, err error) {
	// we assume that people fill their BD date in the DDMM format
	if len(userBD) != 4 {
		return -1, errors.New("Skype field has wrong amount of symbols")
	}

	bd, err := time.Parse("02012006", userBD+"1900")
	if err != nil {
		return -1, errors.Wrap(err, "unable to parse birthday")
	}
	// use the last ms of the day
	bd = bd.Add(23*time.Hour + 59*time.Minute + 59*time.Second + 999*time.Millisecond)
	logrus.Debugf("Got user's birthday: %s", bd)

	ageBD := bd.AddDate(goage.Age(bd), 0, 0)
	if now.After(ageBD) {
		ageBD = ageBD.AddDate(1, 0, 0)
	}

	days = int(ageBD.Sub(now).Hours()) / 24
	logrus.Debugf("Days left: %d", days)
	return
}

func stringInSlice(s string, ss []string) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}
