package main

import "time"

type config struct {
	WorkdayStart int
	WorkdayEnd   int

	Location *time.Location

	BotUID        string
	LegacyToken   string
	MainChannelID string
	ManagerID     string
	ManagerDM     string

	BDHighTreshold int
	BDLowTreshold  int

	Blacklist []string
}

type messages struct {
	ShutdownAnnounce string
	ShutdownError    string
	ProfileError     string
	BDParseError     string
	PersonalIncoming string
	PersonalToday    string
	ManagerAnnounce  string
	ChannelAnnounce  string
}

type bdInfo struct {
	RealName string
	Surname  string
	Birthday string
	DaysLeft int
}
