# Go Discord Jail

----------------------------------

> Version 1.0.0

> Made by Nathan Hedge @ https://almostd.one/

----------------------------------

__A basic jail bot made for a friend.__

Allows you to mention a user, remove all of their roles, give them a special role, and then rename them to only their User ID.

----------------------------------

<br>
<br>

__LIST OF COMMANDS__
---
### `@bot help`
> Summons a help list.
 
### `@bot jail`
> Requires ban permissions. Puts a user in jail.
> Format (with ID): `@bot jail [userID/mention] [time] [reason]`
> Format (search): `@bot jail search [query] [time] [reason]`
> Format (reply): `@bot jail [time] [reason]`
 
### `@bot unjail` or `@bot free`
> Requires ban permissions. Takes the user out of jail.
> Format (with ID): `@bot unjail [userID/mention]`
> Format (search): `@bot unjail search [query]`
> Format (reply): `@bot unjail`

### `@bot setjailrole`
> Requires admin permissions. Sets the role given to jailed users. Does not change the role for already jailed users.
> Format: `@bot setjailrole [roleID]`

### Default response
> Responds "Hello!"

<br>
<br>

----------------------------------

__GENERAL DETAILS__
---
> Data is stored in Sqlite.

> Time is parsed as ?Y?M?w?d?h?m?s

> ALL commands (except for some modifiers) work with aNY CapItaLIzATIoN.

> Commands do not work in DMs.

> Spaces can be included in individual arguments by using backspaces / Ex: `This\ is\ one\ argument, but\ this\ is\ the\ second`

----------------------------------

<br>
<br>

## Copyright (C) 2022 Nathan Hedge (@10Nates)