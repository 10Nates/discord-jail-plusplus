package main

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/snowflake/v5"
)

var rusrtag = regexp.MustCompile(`<@!?(\d{17,}?)>`)

func baseReply(msg *disgord.Message, s *disgord.Session, reply string) {
	msg.Reply(context.Background(), *s, reply)
}

func helpReply(msg *disgord.Message, s *disgord.Session) {
	helpe := &disgord.Embed{
		Title:       "Command Help",
		Description: "----------",
		Color:       0xff0000, //red
		Fields: []*disgord.EmbedField{
			{
				Name:  prefix + "help",
				Value: "Summons a help list.",
			},
			{
				Name:  "-jail",
				Value: "Requires ban permissions. Puts a user in jail.\n\n*Format (with ID):* `-jail [userID/mention] [time] [reason]`\n*Format (search):* `-jail search [query] [time] [reason]`\n*Format (reply):* `-jail [time] [reason]`",
			},
			{
				Name:  "-unjail / -free",
				Value: "Requires ban permissions. Takes the user out of jail.\n\n*Format (with ID):* `-unjail [userID/mention]`\n*Format (search):* `-unjail search [query]`\n*Format (reply):* `-unjail`",
			},
			{
				Name:  "-jailreason",
				Value: "Requires ban permissions. Shows the reason a given user was jailed, as well as the time they have left.\n\n*Format (with ID):* `-jailreason [userID/mention]`\n*Format (search):* `-jailreason search [query]`\n*Format (reply):* `-jailreason`",
			},
			{
				Name:  "-setjailrole",
				Value: "Requires admin permissions. Sets the role given to jailed users. Does not change the role for already jailed users.\n\n*Format:* `-setjailrole [roleID]`",
			},
		},
	}

	msg.Reply(context.Background(), *s, helpe)
}

func findUser(msg *disgord.Message, s *disgord.Session, client *disgord.Client, search bool, query string) (user *disgord.User, err error) {

	if search {
		// search for usertag
		members, err := client.Guild(msg.GuildID).GetMembers(&disgord.GetMembers{Limit: 0})
		if err != nil {
			return nil, fmt.Errorf("an error occured fetching members, please try again")
		}

		var match *disgord.Member

		for i := 0; i < len(members); i++ {
			if strings.Contains(strings.ToLower(members[i].Nick), strings.ToLower(query)) {
				match = members[i]
			}
		}

		if match == nil {
			return nil, fmt.Errorf("could not find a match, please try again")
		}

		avatar, err := match.User.AvatarURL(256, false)
		if err != nil {
			avatar = "https://cdn.discordapp.com/embed/avatars/1.png?size=256"
		}

		viewuser := &disgord.Embed{
			Title:     "Is this the user you meant?",
			Thumbnail: &disgord.EmbedThumbnail{URL: avatar},
			Color:     0xff0000, //red
			Fields: []*disgord.EmbedField{
				{
					Name:  match.User.Tag(),
					Value: match.User.ID.String(),
				},
				{
					Name:  "Please react with YES :white_check_mark: / NO :x:",
					Value: "\u200B",
				},
			},
		}

		newmsg, err := msg.Reply(context.Background(), *s, viewuser)
		if err != nil {
			return nil, fmt.Errorf("error making confimation message, please try again")
		}
		newmsg.React(context.Background(), *s, "✅")
		time.Sleep(time.Second)
		newmsg.React(context.Background(), *s, "❌")

		emojiChan := make(chan string, 1)

		go func(emojiChan chan string) {

			for i := 11; i > 0; i-- { //wait 10 seconds for reaction
				time.Sleep(time.Second)

				if lastReaction.MessageID == newmsg.ID && lastReaction.UserID == msg.Author.ID {
					if lastReaction.PartialEmoji.Name == "✅" {
						emojiChan <- "check"
					} else if lastReaction.PartialEmoji.Name == "❌" {
						emojiChan <- "x"
					} else {
						emojiChan <- "none" // bad reaction
					}
					break
				}

				if i == 1 {
					emojiChan <- "none" // bad reaction
					break
				}
			}
		}(emojiChan)

		emoji := <-emojiChan
		err = client.Channel(newmsg.ChannelID).Message(newmsg.ID).Delete()
		if err != nil {
			newmsg.Reply(context.Background(), *s, "Could not delete message. Please check permissions.")
		}

		switch emoji {
		case "check":
			return match.User, nil
		case "x":
			return nil, fmt.Errorf("could not find user, please try again")
		default:
			return nil, fmt.Errorf("did not react in time, please try again")
		}

	} else {
		// discover user ID without mention
		queryres := rusrtag.FindStringSubmatch(query)
		var parsedquery string
		if len(queryres) > 1 {
			parsedquery = queryres[1]
		} else {
			parsedquery = query
		}

		userMentionedNum, errUserMentionedNum := strconv.ParseUint(parsedquery, 10, 64)
		if errUserMentionedNum != nil {
			return nil, fmt.Errorf("invalid user ID, please try again")
		}
		userMentioned, errUserMentioned := client.User(snowflake.NewSnowflake(userMentionedNum)).Get()
		if errUserMentioned != nil {
			return nil, fmt.Errorf("invalid user ID, please try again")
		}
		return userMentioned, nil
	}

}

// compile regex on start
var regyears = regexp.MustCompile(`(\d+?)y(ears)?`)
var regmonths = regexp.MustCompile(`(\d+?)(M([^i]|$)|mon)`)
var regweeks = regexp.MustCompile(`(\d+?)w`)
var regdays = regexp.MustCompile(`(\d+?)d`)
var reghours = regexp.MustCompile(`(\d+?)h`)
var regmins = regexp.MustCompile(`(\d+?)m([^o]|$)`)
var regsecs = regexp.MustCompile(`(\d+?)s`)

// inline solution for determining of something exists and then parsing it
func ITEN(condition bool, str []string, f int64) (time.Duration, error) {
	if condition {
		a, err := strconv.ParseInt(str[1], 10, 64)
		if err == nil {
			return time.Duration(a), err
		} else {
			return 0, err
		}
	}
	return time.Duration(f), nil
}

func timeParser(datestring string) (time.Duration, bool, error) {
	datestringlow := strings.ToLower(datestring)

	if strings.Contains(datestringlow, "inf") || strings.Contains(datestringlow, "ever") {
		return math.MaxInt64, true, nil // return as infinite, skip further analysis
	}

	years_str := regyears.FindStringSubmatch(datestringlow)
	years, err1 := ITEN(len(years_str) > 1, years_str, 0)

	if years > 200 {
		return math.MaxInt64, true, nil // return as infinite, skip further analysis
	}

	months_str := regmonths.FindStringSubmatch(datestring)
	months, err2 := ITEN(len(months_str) > 1, months_str, 0)

	if months > 2400 { // 200 * 12
		return math.MaxInt64, true, nil // return as infinite, skip further analysis
	}

	weeks_str := regweeks.FindStringSubmatch(datestringlow)
	weeks, err3 := ITEN(len(weeks_str) > 1, weeks_str, 0)

	if months > 10400 { // weeks in 200 years
		return math.MaxInt64, true, nil // return as infinite, skip further analysis
	}

	days_str := regdays.FindStringSubmatch(datestringlow)
	days, err4 := ITEN(len(days_str) > 1, days_str, 0)

	if days > 73000 { // days in 200 years
		return math.MaxInt64, true, nil // return as infinite, skip further analysis
	}

	hours_str := reghours.FindStringSubmatch(datestringlow)
	hours, err5 := ITEN(len(hours_str) > 1, hours_str, 0)

	mins_str := regmins.FindStringSubmatch(datestring)
	mins, err6 := ITEN(len(mins_str) > 1, mins_str, 0)

	secs_str := regsecs.FindStringSubmatch(datestringlow)
	secs, err7 := ITEN(len(secs_str) > 1, secs_str, 0)

	// hacky error handling
	if err1 != nil {
		return 0, false, err1
	}
	if err2 != nil {
		return 0, false, err2
	}
	if err3 != nil {
		return 0, false, err3
	}
	if err4 != nil {
		return 0, false, err4
	}
	if err5 != nil {
		return 0, false, err5
	}
	if err6 != nil {
		return 0, false, err6
	}
	if err7 != nil {
		return 0, false, err7
	}

	duration := time.Duration(0)
	duration += years * 525960 * time.Minute // that many minutes in a year
	duration += months * 43830 * time.Minute
	if duration < 0 {
		return math.MaxInt64, true, nil // return as infinite since there was rollover
	}
	duration += weeks * 10080 * time.Minute
	duration += days * 1440 * time.Minute
	if duration < 0 {
		return math.MaxInt64, true, nil // return as infinite since there was rollover
	}
	duration += hours * 60 * time.Minute
	if duration < 0 {
		return math.MaxInt64, true, nil // return as infinite since there was rollover
	}
	duration += mins * time.Minute
	duration += secs * time.Second
	if duration < 0 {
		return math.MaxInt64, true, nil // return as infinite since there was rollover
	}

	return time.Duration(duration), false, nil
}

func convertToJailedUser(client *disgord.Client, member *disgord.Member, release bool, releasetime time.Duration, reason string, jailer *disgord.User) (*JailedUser, error) {

	timenow := time.Now()
	rtime := timenow.Add(releasetime)

	avatarurl, err := member.User.AvatarURL(1024, false)
	if err != nil {
		return nil, err
	}

	mroles := member.Roles
	roles := ""

	for i := 0; i < len(mroles); i++ {
		roles += mroles[i].String() + " "
	}

	roles = strings.TrimSpace(roles) // last space

	newuser := &JailedUser{
		id:          uint64(member.UserID),
		releasable:  release,
		jailedTime:  timenow,
		releaseTime: rtime,
		reason:      reason,
		jailer:      uint64(jailer.ID),
		oldnick:     member.Nick,
		oldpfpurl:   avatarurl,
		oldroles:    roles,
		jailrole:    currentJailRole,
	}

	return newuser, nil
}

func jailUser(msg *disgord.Message, client *disgord.Client, member *disgord.Member, user *JailedUser) error {
	memberbuilder := client.Guild(msg.GuildID).Member(member.UserID)

	memberid := member.UserID.String() // easier than converting user ID to string

	_, err := memberbuilder.Update(&disgord.UpdateMember{
		Nick:           &memberid,
		Roles:          &[]snowflake.Snowflake{snowflake.ParseSnowflakeString(user.jailrole)},
		AuditLogReason: "User jailed by " + snowflake.Snowflake(user.jailer).String() + " for " + user.reason,
	})
	if err != nil {
		return err
	}

	_, err = JailNewUser(user)
	if err != nil {
		return err
	}

	fmt.Println("Jailed user\t", user.id)

	return nil
}

func freeUser(guildID snowflake.Snowflake, client *disgord.Client, user *JailedUser) error {
	memberbuilder := client.Guild(guildID).Member(snowflake.Snowflake(user.id))

	member, _ := memberbuilder.Get() // user still in server

	rolesStrArray := strings.Split(user.oldroles, " ")
	roles := []snowflake.Snowflake{}

	for i := 0; i < len(rolesStrArray); i++ {
		roles = append(roles, snowflake.ParseSnowflakeString(rolesStrArray[i]))
	}

	if member != nil {
		_, err := memberbuilder.Update(&disgord.UpdateMember{
			Nick:           &user.oldnick,
			Roles:          &roles,
			AuditLogReason: "User freed either manually or automatically",
		})

		if err != nil {
			return err
		}
	} else {
		fmt.Println("User gone\t", user.id)
	}

	_, err := RemoveJailedUser(user.id)
	if err != nil {
		return err
	}

	fmt.Println("Freed user\t", user.id)

	return nil
}

func displayJailedUser(msg *disgord.Message, s *disgord.Session, user *JailedUser) error {

	release := user.releaseTime.UTC().Format(time.RFC1123)
	if !user.releasable {
		release = "Never"
	}

	onick := user.oldnick
	if onick == "" {
		onick = "None"
	}

	viewuser := &disgord.Embed{
		Title:     "Jailed User " + snowflake.Snowflake(user.id).String(),
		Thumbnail: &disgord.EmbedThumbnail{URL: user.oldpfpurl},
		Color:     0xff0000,
		Fields: []*disgord.EmbedField{
			{
				Name:   "Reason",
				Value:  user.reason,
				Inline: false,
			},
			{
				Name:   "Release",
				Value:  release,
				Inline: true,
			},
			{
				Name:   "Jailer",
				Value:  snowflake.Snowflake(user.jailer).String(),
				Inline: true,
			},
			{
				Name:   "Old Nick",
				Value:  onick,
				Inline: true,
			},
		},
	}

	_, err := msg.Reply(context.Background(), *s, viewuser)
	if err != nil {
		return err
	}

	return nil
}

func freeFreeableUsers(guildID snowflake.Snowflake, client *disgord.Client) error {
	userlist, err := FetchAllJailedUsers(true)
	if err != nil {
		return err
	}

	for i := 0; i < len(userlist); i++ {
		user := userlist[i]
		if time.Now().After(user.releaseTime) {
			err = freeUser(guildID, client, user)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func rejailAlreadyJailedUser(guildID snowflake.Snowflake, client *disgord.Client, userid snowflake.Snowflake) error {

	user, err := FetchJailedUser(uint64(userid))
	if err != nil {
		return nil // user not jailed, skip
	}

	memberbuilder := client.Guild(guildID).Member(userid)

	userstring := userid.String()

	_, err = memberbuilder.Update(&disgord.UpdateMember{
		Nick:           &userstring,
		Roles:          &[]snowflake.Snowflake{snowflake.ParseSnowflakeString(user.jailrole)},
		AuditLogReason: "User left & rejoined before being freed",
	})

	if err != nil {
		return err
	}

	fmt.Println("Jailed user rejoined\t", user.id)

	return nil
}

func convertToMarkedUser(client *disgord.Client, member *disgord.Member, marker *disgord.User, mark *Mark) (*MarkedUser, error) {
	mroles := member.Roles
	roles := ""

	for i := 0; i < len(mroles); i++ {
		roles += mroles[i].String() + " "
	}

	roles = strings.TrimSpace(roles) // last space

	newuser := &MarkedUser{
		id:       uint64(member.UserID),
		marker:   uint64(marker.ID),
		markrole: mark.id,
		oldroles: roles,
	}

	return newuser, nil
}

func markUser(msg *disgord.Message, client *disgord.Client, member *disgord.Member, user *MarkedUser) error {

	removedRolesString, err := GetMarkedRemovedRoles()
	if err != nil {
		return err
	}

	markRoleSnowflake := snowflake.NewSnowflake(user.markrole)

	var newRoles []snowflake.Snowflake

	for i := 0; i < len(member.Roles); i++ {
		if !strings.Contains(removedRolesString, member.Roles[i].String()) {
			newRoles = append(newRoles, member.Roles[i])
		}
	}

	newRoles = append(newRoles, markRoleSnowflake)

	memberbuilder := client.Guild(msg.GuildID).Member(member.UserID)
	_, err = memberbuilder.Update(&disgord.UpdateMember{
		Roles:          &newRoles,
		AuditLogReason: "User marked by " + snowflake.Snowflake(user.marker).String(),
	})
	if err != nil {
		return err
	}

	_, err = MarkNewUser(user)
	if err != nil {
		return err
	}

	fmt.Println("Marked user\t", user.id)

	return nil
}

func unMarkUser(guildID snowflake.Snowflake, client *disgord.Client, user *MarkedUser) error {
	memberbuilder := client.Guild(guildID).Member(snowflake.Snowflake(user.id))

	member, _ := memberbuilder.Get() // user still in server

	rolesStrArray := strings.Split(user.oldroles, " ")
	roles := []snowflake.Snowflake{}

	for i := 0; i < len(rolesStrArray); i++ {
		roles = append(roles, snowflake.ParseSnowflakeString(rolesStrArray[i]))
	}

	if member != nil {
		_, err := memberbuilder.Update(&disgord.UpdateMember{
			Roles:          &roles,
			AuditLogReason: "User unmarked",
		})

		if err != nil {
			return err
		}
	} else {
		fmt.Println("User gone\t", user.id)
	}

	_, err := DeleteMarkfromUser(user.id)
	if err != nil {
		return err
	}

	fmt.Println("Unmarked user\t", user.id)

	return nil
}

func getMarksFormatted() (string, error) {
	marks, err := GetAllMarks()
	if err != nil {
		return "", err
	}

	list := ""

	for _, m := range marks {
		list += "\n" + m.name + " - " + snowflake.NewSnowflake(m.id).String()
	}

	if list == "" {
		return "", fmt.Errorf("no marks in system")
	}

	return list, nil

}
