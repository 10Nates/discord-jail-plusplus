# Go Discord Jail++

----------------------------------

> Version 1.0.2

> Made by Nathan Hedge @ https://almostd.one/

----------------------------------

__An (originally) jail bot made for a friend.__

Allows you to mention a user, remove all of their roles, give them a special role, and then rename them to only their User ID. Will later be expanded into more than just jailing.

----------------------------------

<br>
<br>

__LIST OF COMMANDS__
---
*Using prefix `-`*

<br>
<br>

### `-help`
Summons a help list.
 
### `-jail`
Requires ban permissions. Puts a user in jail.
> Format (with ID): `-jail [userID/mention] [time] [reason]`
> 
> Format (search): `-jail search [query] [time] [reason]`
> 
> Format (reply): `-jail [time] [reason]`
 
### `-unjail` or `-free`
Requires ban permissions. Takes the user out of jail. 
> Format (with ID): `-unjail [userID/mention]`
> 
> Format (search): `-unjail search [query]`
> 
> Format (reply): `-unjail`

### `-jailreason`
Requires ban permissions. Shows the reason a given user was jailed, as well as the time they have left.
> Format (with ID): `-jailreason [userID/mention]`
> 
> Format (search): `-jailreason search [query]`
> 
> Format (reply): `-jailreason`

### `-setjailrole`
Requires admin permissions. Sets the role given to jailed users. Does not change the role for already jailed users. 
> Format: `-setjailrole [roleID]`

### `-mark`
Requires manage message permissions. Marks a user.
> Format (with ID): `-mark [userID/mention] [markname]`
> 
> Format (search): `-mark search [query] [markname]`
> 
> Format (reply): `-mark [markname]`

### `-unmark`
Requires manage message permissions. Marks a user.
> Format (with ID): `-unmark [userID/mention]`
> 
> Format (search):* `-unmark search [query]`
> 
> Format (reply):* `-unmark`

### `-markroles` or `-managemarks`
Requires admin permissions. Adds & removes marks.
> Format (list): `-markroles list`
> 
> Format (add): `-markroles add [roleID] [markname]`
> 
> Format (remove): `-markroles remove [roleID]`

### `-markremovedroles`
Requires admin permissions. Adds & removes mark-removed roles, roles that get removed when a user is marked.
> Format (list): `-markremovedroles list`
> 
> Format (add/remove): `-markremovedroles [add|remove] [roleID]`"


<br>
<br>

----------------------------------

<br>
<br>

__GENERAL DETAILS__
---
> Data is stored in SQLite.

> Time is parsed as ?Y?M?w?d?h?m?s

> ALL commands (except for some modifiers) work with aNY CapItaLIzATIoN.

> Commands do not work in DMs.

> Spaces can be included in individual arguments by using backspaces / Ex: `This\ is\ one\ argument, but\ this\ is\ the\ second`

<br>
<br>

----------------------------------

<br>
<br>

__BUILD AND RUN__
---

> Build the program with `go build -ldflags="-s -w" -o dist/djpp main.go commands.go database.go` (just the standard go build command)

> Program requires `Token` environment variable (the Discord bot token)

<br>
<br>

----------------------------------

<br>
<br>

## Copyright (C) 2022 Nathan Hedge (@10Nates)