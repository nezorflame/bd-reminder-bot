/*
bd-reminder-bot - Slack bot in Go for birthday reminders
Inspired by 'mybot' from RapidLoop at https://github.com/rapidloop/mybot
*/

package main // import "github.com/nezorflame/bd-reminder-bot"

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nezorflame/bd-reminder-bot/slack"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	// get flags
	debugPtr := flag.Bool("debug", false, "debug level for logs")
	configPtr := flag.String("config", "config", "config file name")
	dbPtr := flag.String("db", "./bolt.db", "BoltDB file location")
	flag.Parse()

	// set log level
	if *debugPtr {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// set config path
	viper.SetConfigName(*configPtr)
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatal(err)
	}

	// parse config
	mBucket, cBucket, botToken, c, m, err := parseConfig()
	if err != nil {
		logrus.WithError(err).Fatalf("Unable to init config")
	}

	// connect to BoltDB
	db, err := openDB(dbPtr, mBucket, cBucket, DefaultDBTimeout)
	if err != nil {
		logrus.WithError(err).Fatalf("Unable to open DB")
	}
	defer db.Close()

	// connect to Slack
	wsConfig, botUID, err := slack.InitWSConfig(botToken)
	if err != nil {
		logrus.WithError(err).Fatalf("Unable to get Slack WS config")
	}
	c.BotUID = botUID

	// launch the message and birthday watchers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		// error counter and retry limit
		errCount := 0
		retryLimit := 3
		for {
			// check error counter
			if errCount == retryLimit {
				logrus.Warnln("Error limit reached, exiting")
				break
			}

			// dial websocket
			wsConn, err := slack.DialWS(wsConfig)
			if err != nil {
				errCount++
				logrus.WithError(err).WithField("try", errCount).Errorf("Unable to connect to Slack's websocket, retrying")
				continue
			}
			defer wsConn.Close() // not interested in this error, so skipping
			errCount = 0         // resetting the counter

			// launch message watcher
			if err := msgWatcher(ctx, wsConn, c, m); err != nil {
				errCount++
				wsConn.Close() // not interested in this error, so skipping
				logrus.WithError(err).WithField("try", errCount).Warnln("Message watcher failed, trying to reconnect")
				continue
			}
		}
		cancel()
		wg.Done()
	}()
	go func() {
		if err := bdWatcher(ctx, db, c, m); err != nil {
			logrus.WithError(err).Errorln("Birthday watcher failed")
		}
		cancel()
		wg.Done()
	}()
	logrus.Infoln("Bot is ready with user ID", botUID)

	// watch the OS signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-ctx.Done():
			logrus.Warnln("Shutting down")
			wg.Wait()
			return
		case <-sigs:
			logrus.Warnln("Exiting program on Ctrl+C")
			cancel()
			wg.Wait()
			return
		}
	}
}

func parseConfig() (mBucket, cBucket, bToken string, c *config, m *messages, err error) {
	// base settings
	if mBucket = viper.GetString("manager_bucket"); mBucket == "" {
		err = errors.New("manager_bucket can't be empty")
		return
	}

	if cBucket = viper.GetString("channel_bucket"); cBucket == "" {
		err = errors.New("channel_bucket can't be empty")
		return
	}

	// init the config variables
	c = &config{}

	c.WorkdayStart = viper.GetInt("workday_start")
	c.WorkdayEnd = viper.GetInt("workday_end")
	if c.WorkdayStart >= c.WorkdayEnd {
		err = errors.New("workday_start can't be higher than or equal to workday_end")
		return
	}

	loc := viper.GetString("location")
	if loc == "" {
		loc = "UTC"
	}
	if c.Location, err = time.LoadLocation(loc); err != nil {
		err = errors.New("location is wrong")
		return
	}

	// init Slack variables
	slackSection := viper.Sub("slack")

	if bToken = slackSection.GetString("bot_token"); bToken == "" {
		err = errors.New("bot_token can't be empty")
		return
	}

	if c.LegacyToken = slackSection.GetString("legacy_token"); c.LegacyToken == "" {
		err = errors.New("legacy_token can't be empty")
		return
	}

	if c.MainChannelID = slackSection.GetString("main_channel_id"); c.MainChannelID == "" {
		err = errors.New("main_channel_id can't be empty")
		return
	}

	if c.ManagerID = slackSection.GetString("manager_id"); c.ManagerID == "" {
		err = errors.New("manager_id can't be empty")
		return
	}

	if c.BDHighTreshold = slackSection.GetInt("bd_treshold_high"); c.BDHighTreshold == 0 {
		err = errors.New("bd_treshold_high can't be zero")
		return
	}

	if c.BDLowTreshold = slackSection.GetInt("bd_treshold_low"); c.BDLowTreshold == 0 {
		err = errors.New("bd_treshold_low can't be zero")
		return
	}
	if c.BDHighTreshold < c.BDLowTreshold {
		err = errors.New("bd_treshold_low can't be higher than bd_treshold_high")
		return
	}

	if c.Blacklist = slackSection.GetStringSlice("blacklist"); len(c.Blacklist) == 0 {
		logrus.Warnln("blacklist is empty")
	}

	// init the message texts
	m = &messages{}
	msgSection := viper.Sub("messages")

	m.ShutdownAnnounce = msgSection.GetString("shutdown_announce") // can be empty, why not

	if m.ShutdownError = msgSection.GetString("shutdown_error"); m.ShutdownError == "" {
		err = errors.New("messages.shutdown_error can't be empty")
		return
	}

	if m.ProfileError = msgSection.GetString("profile_error"); m.ProfileError == "" {
		err = errors.New("messages.profile_error can't be empty")
		return
	}

	if m.BDParseError = msgSection.GetString("bd_parse_error"); m.BDParseError == "" {
		err = errors.New("messages.bd_parse_error can't be empty")
		return
	}

	if m.PersonalIncoming = msgSection.GetString("personal_incoming"); m.PersonalIncoming == "" {
		err = errors.New("messages.personal_incoming can't be empty")
		return
	}

	if m.PersonalToday = msgSection.GetString("personal_today"); m.PersonalToday == "" {
		err = errors.New("messages.personal_today can't be empty")
		return
	}

	if m.ManagerAnnounce = msgSection.GetString("manager_announce"); m.ManagerAnnounce == "" {
		err = errors.New("messages.manager_announce can't be empty")
		return
	}

	if m.ChannelAnnounce = msgSection.GetString("channel_announce"); m.ChannelAnnounce == "" {
		err = errors.New("messages.channel_announce can't be empty")
		return
	}

	return
}
