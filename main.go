package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"
)

const (
	BotID  = "462051981863682048"
	prefix = "-"
)

var (
	rsplit          = regexp.MustCompile(`([^\\])( )`)
	lastReaction    *disgord.MessageReactionAdd
	currentJailRole string
)

func main() {
	//initialize database
	InitDB()

	//set jail role to current
	jroleid, err := GetJailRole()
	if err != nil {
		panic("Could not get jail role")
	}
	currentJailRole = jroleid

	//load client
	client := disgord.New(disgord.Config{
		BotToken: os.Getenv("Token"),
		Intents:  disgord.IntentGuildMessages | disgord.IntentGuildMembers | disgord.IntentGuildMessageReactions,
	})
	defer client.Gateway().StayConnectedUntilInterrupted()

	//startup message
	client.Gateway().BotReady(func() {
		fmt.Println("Bot started @ " + time.Now().Local().Format(time.RFC1123))
		client.UpdateStatusString(prefix + "help")
	})
	//filter out unwanted messages
	content, err := std.NewMsgFilter(context.Background(), client)
	if err != nil {
		panic(err)
	}
	content.NotByBot(client)
	//content.ContainsBotMention(client)
	content.HasPrefix(prefix)

	//on message with mention
	client.Gateway().
		WithMiddleware(content.NotByBot, content.HasPrefix).                // filter
		MessageCreate(func(s disgord.Session, evt *disgord.MessageCreate) { // on message

			go parseCommand(evt.Message, &s, client)
		})

	//on message reaction
	client.Gateway().
		MessageReactionAdd(func(s disgord.Session, h *disgord.MessageReactionAdd) { // on reaction
			lastReaction = h
		})
}

func parseCommand(msg *disgord.Message, s *disgord.Session, client *disgord.Client) {
	cstr := msg.Content
	if !strings.HasPrefix(cstr, "-") {
		return // this should be automatic but it isn't for some reason
	}

	rsplitstr := rsplit.ReplaceAllString(cstr, "$1\n")
	carr := strings.Split(rsplitstr, "\n")

	args := []string{}
	argsl := []string{}

	for i := 0; i < len(carr); i++ {
		if !strings.Contains(carr[i], BotID) {
			args = append(args, carr[i])
			argsl = append(argsl, strings.ToLower(carr[i]))
		}
	}

	if len(args) < 1 {
		args = append(args, "")
		argsl = append(argsl, "")
	}

	// fetch author permissions
	authorperms, err := msg.Member.GetPermissions(context.Background(), client)
	if err != nil {
		baseReply(msg, s, "Error fetching user permissions. Please try again.")
		return
	}

	switch argsl[0][1:] {
	case "help":
		helpReply(msg, s)
	case "jail":
		// owners & administrators are noninclusive
		if !authorperms.Contains(disgord.PermissionBanMembers) && !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}

		var member *disgord.User
		var err error
		var timestr string
		var reason string
		if len(args) > 1 && args[1] == "search" && len(args) > 2 { // search for user instead of ID
			member, err = findUser(msg, s, client, true, args[2])

			//has to be done here because it shifts depending on previous args
			if len(args) > 3 {
				timestr = args[3] // unparsed time
			}
			if len(args) > 4 {
				reason = strings.Join(args[4:], " ") // reason
			}
		} else if msg.Type == disgord.MessageTypeReply {
			member = msg.ReferencedMessage.Author

			//has to be done here because it shifts depending on previous args
			if len(args) > 1 {
				timestr = args[1] // unparsed time
			}
			if len(args) > 2 {
				reason = strings.Join(args[2:], " ") // reason
			}
		} else if len(args) > 1 {
			member, err = findUser(msg, s, client, false, args[1])

			//has to be done here because it shifts depending on previous args
			if len(args) > 2 {
				timestr = args[2] // unparsed time
			}
			if len(args) > 3 {
				reason = strings.Join(args[3:], " ") // reason
			}
		} else {
			baseReply(msg, s, "Please provide a user to jail.")
			return
		}
		if err != nil {
			baseReply(msg, s, "Could not find user. Please try again.")
			return
		}

		// found user, continue
		if timestr == "" {
			baseReply(msg, s, "Please provide an amount of time to be jailed.")
			return
		}
		if reason == "" {
			baseReply(msg, s, "Please provide a reason for the user to jailed.")
			return
		}

		jailtime, inf, err := timeParser(timestr)
		if err != nil {
			baseReply(msg, s, "An error occured while parsing the time provided. Please try again.")
			return
		}

		if inf {
			jailtime = math.MaxInt64
		}

		if jailtime == 0 {
			baseReply(msg, s, "Invalid amount of time provided. Please try again.")
			return
		}

		baseReply(msg, s, "User found: "+member.Tag()+" | Timestamp: "+jailtime.String()+" | Reason: "+reason)

	case "unjail":
		fallthrough
	case "free":
		// owners & administrators are noninclusive
		if !authorperms.Contains(disgord.PermissionBanMembers) && !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}

		var member *disgord.User
		var err error
		if len(args) > 2 { // search for user instead of ID
			member, err = findUser(msg, s, client, true, args[2])
		} else if len(args) > 1 {
			member, err = findUser(msg, s, client, false, args[1])
		} else if msg.Type == disgord.MessageTypeReply {
			member = msg.ReferencedMessage.Author
		} else {
			baseReply(msg, s, "Please provide a user to free.")
			return
		}
		if err != nil {
			baseReply(msg, s, "Could not find user. Please try again.")
			return
		}

		// found user, continue
		baseReply(msg, s, "User found: "+member.Tag())

	case "jailreason":
		// owners & administrators are noninclusive
		if !authorperms.Contains(disgord.PermissionBanMembers) && !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}

		var member *disgord.User
		var err error
		if len(args) > 2 { // search for user instead of ID
			member, err = findUser(msg, s, client, true, args[2])
		} else if len(args) > 1 {
			member, err = findUser(msg, s, client, false, args[1])
		} else if msg.Type == disgord.MessageTypeReply {
			member = msg.ReferencedMessage.Author
		} else {
			baseReply(msg, s, "Please provide a user to view.")
			return
		}
		if err != nil {
			baseReply(msg, s, "Could not find user. Please try again.")
			return
		}

		// found user, continue
		baseReply(msg, s, "User found: "+member.Tag())

	case "setjailrole":
		// owners are noninclusive
		if !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}
	default:
		return
	}
}
