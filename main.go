package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"
	"github.com/andersfylling/snowflake/v5"
)

const (
	guildid = "449043801344966666" // basically only used for auto-freeing users (too lazy to add anything to databse)
	prefix  = "-"
)

var (
	BotID           string
	rsplit          = regexp.MustCompile(`([^\\])( )`)
	lastReaction    *disgord.MessageReactionAdd
	currentJailRole string
)

func checkOnJailedUsers(client *disgord.Client) {
	gsnow := snowflake.ParseSnowflakeString(guildid)
	for {
		time.Sleep(10 * time.Second)
		err := freeFreeableUsers(gsnow, client)
		if err != nil {
			fmt.Println("Error checking on jailed users:", err)
		}
	}
}

func main() {
	//initialize database
	InitDB()

	//set jail role to current
	jroleid, err := GetJailRole()
	if err != nil {
		panic(err)
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
		// grab bot ID
		botuser, err := client.CurrentUser().Get()
		if err != nil {
			panic(err)
		}
		BotID = botuser.ID.String()

		// start checking on jailed users
		go checkOnJailedUsers(client)

		// successfully started
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
			// This is in a different thread, I checked
			parseCommand(evt.Message, &s, client)
		})

	//on message reaction
	client.Gateway().
		MessageReactionAdd(func(s disgord.Session, evt *disgord.MessageReactionAdd) { // on reaction
			lastReaction = evt
		})

	client.Gateway().
		GuildMemberAdd(func(s disgord.Session, evt *disgord.GuildMemberAdd) { // on guild member join
			err := rejailAlreadyJailedUser(snowflake.ParseSnowflakeString(guildid), client, evt.Member.UserID)
			if err != nil {
				fmt.Println("Error checking & rejailing user upon join:", err)
			}
		})
}

func parseCommand(msg *disgord.Message, s *disgord.Session, client *disgord.Client) {
	if msg.GuildID.String() != guildid {
		return // only one database, only one server
	}

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

		if jailtime == 0 {
			baseReply(msg, s, "Invalid amount of time provided. Please try again.")
			return
		}

		// correct input, continue

		realmember, err := client.Guild(msg.GuildID).Member(member.ID).Get() // need the user member for roles later down the line
		if err != nil {
			baseReply(msg, s, "An error occured while gathering data for the user. Please try again.")
			return
		}

		juser, err := convertToJailedUser(client, realmember, !inf, jailtime, reason, msg.Author)
		if err != nil {
			baseReply(msg, s, "An error occured while gathering data for the user. Please try again.")
			return
		}

		//check if exists already
		user, _ := FetchJailedUser(uint64(member.ID))
		if user != nil {
			baseReply(msg, s, "User selected is already in jail. Please try again.")
			return
		}

		// do the thing
		err = jailUser(msg, client, realmember, juser)
		if err != nil {
			baseReply(msg, s, "An error occured jailing user. Please check permissions and try again.\nError: "+err.Error())
			return
		}

		if inf {
			baseReply(msg, s, "User has been jailed successfully and will not be freed.")
		} else {
			baseReply(msg, s, "User has been jailed successfully and will be freed in "+jailtime.String()+".")
		}

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
		if len(args) > 1 && args[1] == "search" && len(args) > 2 { // search for user instead of ID
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
		user, err := FetchJailedUser(uint64(member.ID))
		if err != nil {
			baseReply(msg, s, err.Error())
			return
		}

		// found user in jailed database, continue
		err = freeUser(msg.GuildID, client, user)
		if err != nil {
			baseReply(msg, s, "An error occured freeing user. Please check permissions and try again.\nError: "+err.Error())
		}

		baseReply(msg, s, "User has been freed successfully.")

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
		user, err := FetchJailedUser(uint64(member.ID))
		if err != nil {
			baseReply(msg, s, err.Error())
			return
		}

		// found user in jailed database, continue
		err = displayJailedUser(msg, s, user)
		if err != nil {
			baseReply(msg, s, "An error occured displaying the user. Please try again.")
			return
		}

	case "setjailrole":
		// owners are noninclusive
		if !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}

		if len(args) > 1 {

			_, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				baseReply(msg, s, "Please provide a valid role ID.") // pretest
				return
			}

			if currentJailRole == args[1] {
				baseReply(msg, s, "That is already the jail role.")
				return
			}

			roles, err := client.Guild(msg.GuildID).GetRoles()
			if err != nil {
				baseReply(msg, s, "Error fetching roles. Please try again.")
				return
			}

			roleExists := false
			for i := 0; i < len(roles); i++ {
				if roles[i].ID.String() == args[1] {
					roleExists = true
					break
				}
			}

			if !roleExists {
				baseReply(msg, s, "Please provide a valid role ID.")
				return
			}

			// role exists, continue
			err = SetJailRole(args[1])
			if err != nil {
				baseReply(msg, s, "Error setting role. Please try again.\nError: "+err.Error())
				return
			}
			currentJailRole = args[1]

			baseReply(msg, s, "Successfully changed jail role to "+args[1])

		} else {
			baseReply(msg, s, "Please provide a role ID to change the jail role to.")
		}

	case "mark":
		// owners & administrators are noninclusive
		if !authorperms.Contains(disgord.PermissionBanMembers) && !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}

		var member *disgord.User
		var markname string
		var err error
		if len(args) > 1 && args[1] == "search" && len(args) > 2 { // search for user instead of ID
			member, err = findUser(msg, s, client, true, args[2])

			if len(args) > 3 {
				markname = argsl[3]
			}
		} else if len(args) > 1 {
			member, err = findUser(msg, s, client, false, args[1])

			if len(args) > 2 {
				markname = argsl[2]
			}
		} else if msg.Type == disgord.MessageTypeReply {
			member = msg.ReferencedMessage.Author

			if len(args) > 1 {
				markname = argsl[1]
			}
		} else {
			baseReply(msg, s, "Please provide a user to mark.")
			return
		}
		if err != nil {
			baseReply(msg, s, "Could not find user. Please try again.")
			return
		}

		// found user, continue
		realmember, err := client.Guild(msg.GuildID).Member(member.ID).Get() // need the user member for roles later down the line
		if err != nil {
			baseReply(msg, s, "An error occured while gathering data for the user. Please try again.")
			return
		}

		// get mark object & validate
		mark, _, err := FetchMarkByName(markname)
		if err != nil {
			baseReply(msg, s, err.Error())
			return
		}

		// get markeduser object + roles
		markedUser, err := convertToMarkedUser(client, realmember, msg.Author, mark)
		if err != nil {
			baseReply(msg, s, err.Error())
			return
		}

		_, code, err := FetchMarkedUser(uint64(member.ID))
		if err != nil && code != 0 {
			baseReply(msg, s, err.Error())
			return
		} else if err == nil && code == 0 {
			baseReply(msg, s, "User is already marked. Please unmark the user and then mark them again.")
			return
		}

		// user is not already marked, mark is valid, we're set
		err = markUser(msg, client, realmember, markedUser)
		if err != nil {
			baseReply(msg, s, "An error occured marking user. Please check permissions and try again.\nError: "+err.Error())
			return
		}

		baseReply(msg, s, "User has been successfully marked.")

	case "unmark":
		// owners & administrators are noninclusive
		if !authorperms.Contains(disgord.PermissionBanMembers) && !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}

		var member *disgord.User
		var err error
		if len(args) > 1 && args[1] == "search" && len(args) > 2 { // search for user instead of ID
			member, err = findUser(msg, s, client, true, args[2])
		} else if len(args) > 1 {
			member, err = findUser(msg, s, client, false, args[1])
		} else if msg.Type == disgord.MessageTypeReply {
			member = msg.ReferencedMessage.Author
		} else {
			baseReply(msg, s, "Please provide a user to unmark.")
			return
		}
		if err != nil {
			baseReply(msg, s, "Could not find user. Please try again.")
			return
		}

		// found user, continue
		user, _, err := FetchMarkedUser(uint64(member.ID))
		if err != nil {
			baseReply(msg, s, err.Error())
			return
		}

		// found user in marked database, continue
		err = unMarkUser(msg.GuildID, client, user)
		if err != nil {
			baseReply(msg, s, "An error occured unmarking user. Please check permissions and try again.\nError: "+err.Error())
		}

		baseReply(msg, s, "User has been successfully unmarked.")

	case "markroles":
		// owners & administrators are noninclusive
		if !authorperms.Contains(disgord.PermissionBanMembers) && !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}

		if len(args) > 1 && argsl[1] == "list" {

			listRoles, err := getMarksFormatted()

			if err != nil {
				baseReply(msg, s, "Error fetching roles. Please try again.\nError: "+err.Error())
				return
			}

			baseReply(msg, s, "List of Mark Roles:"+listRoles)

		} else if len(args) > 3 && argsl[1] == "add" {

			roleid, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				baseReply(msg, s, "Please provide a valid role ID.") // pretest
				return
			}

			//check if role is taken
			_, code, err := FetchMarkByID(roleid)
			if err == nil {
				baseReply(msg, s, "Mark already exists with the given ID.") // already exists
				return
			} else if err != nil && code != 0 {
				baseReply(msg, s, err.Error()) // something went wrong
				return
			}

			markname := argsl[3]

			//check if role is taken part 2
			_, code, err = FetchMarkByName(markname)
			if err != nil && code == 0 {
				baseReply(msg, s, "Mark already exists with the given name.") // already exists
				return
			} else if err != nil {
				baseReply(msg, s, err.Error()) // something went wrong
				return
			}

			roles, err := client.Guild(msg.GuildID).GetRoles()
			if err != nil {
				baseReply(msg, s, "Error fetching roles. Please try again.")
				return
			}

			roleExists := false
			for i := 0; i < len(roles); i++ {
				if roles[i].ID.String() == args[1] {
					roleExists = true
					break
				}
			}

			if !roleExists {
				baseReply(msg, s, "Please provide a valid role ID.")
				return
			}

			// role is valid & mark is not taken
			_, err = AddMark(roleid, markname)
			if err != nil {
				baseReply(msg, s, "Error creating mark. Please try again.\nError: "+err.Error())
				return
			}

			baseReply(msg, s, "Mark successfully added.")

		} else if len(args) > 2 && argsl[1] == "remove" {

			roleid, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				baseReply(msg, s, "Please provide a valid role ID.") // pretest
				return
			}

			//check if role is taken
			markToBeRemoved, _, err := FetchMarkByID(roleid)
			if err != nil {
				baseReply(msg, s, "Mark does not exist. Please try again.") // Doesn't exists
				return
			}

			//No need to verify if role actually exists, that would actually be a bad thing for recovery
			_, err = DeleteMark(markToBeRemoved.id)
			if err != nil {
				// the most common error is likely to be SQL telling you that the mark is in use (I hope)
				baseReply(msg, s, "An error occured deleting the mark. Please try again.\nError: "+err.Error())
				return
			}

			baseReply(msg, s, "Mark removed successfully.")

		} else {
			baseReply(msg, s, "Please provide what you would like to do with mark roles.")
		}

	case "markremovedroles":
		// owners are noninclusive
		if !authorperms.Contains(disgord.PermissionAll) && !authorperms.Contains(disgord.PermissionAdministrator) {
			baseReply(msg, s, "You do not have permissions to use this command.")
			return
		}

		baseReply(msg, s, "TODO")

	default:
		return
	}
}
